package repobuilder

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/evergreen-ci/bond"
	"github.com/evergreen-ci/pail"
	"github.com/evergreen-ci/utility"
	"github.com/google/uuid"
	"github.com/mholt/archiver"
	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/dependency"
	"github.com/mongodb/amboy/job"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
)

type jobImpl interface {
	rebuildRepo(string) error
	injectPackage(string, string) (string, error)
}

// repoBuilderJob provides the common structure for a repository building Job.
type repoBuilderJob struct {
	Distro       *RepositoryDefinition `bson:"distro" json:"distro" yaml:"distro"`
	Conf         *RepositoryConfig     `bson:"conf" json:"conf" yaml:"conf"`
	Output       map[string]string     `bson:"output" json:"output" yaml:"output"`
	Version      string                `bson:"version" json:"version" yaml:"version"`
	Arch         string                `bson:"arch" json:"arch" yaml:"arch"`
	AWSProfile   string                `bson:"aws_profile" json:"aws_profile" yaml:"aws_profile"`
	AWSKey       string                `bson:"aws_key" json:"aws_key" yaml:"aws_key"`
	AWSSecret    string                `bson:"aws_secret" json:"aws_secret" yaml:"aws_secret"`
	AWSToken     string                `bson:"aws_token" json:"aws_token" yaml:"aws_token"`
	NotaryKey    string                `bson:"notary_key" json:"notary_key" yaml:"notary_key"`
	NotaryToken  string                `bson:"notary_token" json:"notary_token" yaml:"notary_token"`
	PackagePaths []string              `bson:"package_paths" json:"package_paths" yaml:"package_paths"`
	*job.Base    `bson:"metadata" json:"metadata" yaml:"metadata"`

	tmpdir      string
	client      *http.Client
	workingDirs []string
	release     bond.MongoDBVersion
	builder     jobImpl
}

const jobName = "build-repo"

func init() {
	registry.AddJobType(jobName, func() amboy.Job { return buildRepoJob() })
}

func buildRepoJob() *repoBuilderJob {
	j := &repoBuilderJob{
		Output: make(map[string]string),
		Base: &job.Base{
			JobType: amboy.JobType{
				Name:    jobName,
				Version: 4,
			},
		},
	}

	j.SetDependency(dependency.NewAlways())

	return j
}

// NewBuildRepoJob constructs a repository building job, which
// implements the amboy.Job interface. Provides a legacy interface for
// NewRepoBuilderJob.
func NewBuildRepoJob(conf *RepositoryConfig, distro *RepositoryDefinition, version, arch, profile string, pkgs ...string) (amboy.Job, error) {
	return NewRepoBuilderJob(JobOptions{
		Configuration: conf,
		Distro:        distro,
		Version:       version,
		Arch:          arch,
		AWSProfile:    profile,
		Packages:      pkgs,
		JobID:         uuid.New().String(),
	})
}

// JobOptions describes the options to construct a
// RepoBuilderJob.
type JobOptions struct {
	Configuration *RepositoryConfig     `bson:"conf" json:"conf" yaml:"conf"`
	Distro        *RepositoryDefinition `bson:"distro" json:"distro" yaml:"distro"`
	Version       string                `bson:"version" json:"version" yaml:"version"`
	Arch          string                `bson:"arch" json:"arch" yaml:"arch"`
	Packages      []string              `bson:"packages" json:"packages" yaml:"packages"`
	JobID         string                `bson:"job_id" json:"job_id" yaml:"job_id"`

	AWSProfile string `bson:"aws_profile" json:"aws_profile" yaml:"aws_profile"`
	AWSKey     string `bson:"aws_key" json:"aws_key" yaml:"aws_key"`
	AWSSecret  string `bson:"aws_secret" json:"aws_secret" yaml:"aws_secret"`
	AWSToken   string `bson:"aws_token" json:"aws_token" yaml:"aws_token"`

	NotaryKey   string `bson:"notary_key" json:"notary_key" yaml:"notary_key"`
	NotaryToken string `bson:"notary_token" json:"notary_token" yaml:"notary_token"`

	release bond.MongoDBVersion
}

// Validate returns an error if the job options struct is not
// logically valid.
func (opts *JobOptions) Validate() error {
	catcher := grip.NewBasicCatcher()
	catcher.NewWhen(opts.Configuration == nil, "configuration must not be nil")
	catcher.NewWhen(opts.Distro == nil, "distro specification must not be nil")

	release, err := bond.CreateMongoDBVersion(opts.Version)
	catcher.Add(err)
	opts.release = release

	return catcher.Resolve()
}

// NewRepoBuilderJob produces a new repo job.
func NewRepoBuilderJob(opts JobOptions) (amboy.Job, error) {
	err := opts.Validate()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	j := buildRepoJob()

	j.Conf = opts.Configuration
	if j.Conf.WorkSpace == "" {
		j.Conf.WorkSpace, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	j.Arch = opts.Distro.getArchForDistro(opts.Arch)
	j.Distro = opts.Distro
	j.PackagePaths = opts.Packages
	j.Version = opts.Version
	j.AWSProfile = opts.AWSProfile
	j.AWSKey = opts.AWSKey
	j.AWSSecret = opts.AWSSecret
	j.AWSToken = opts.AWSToken
	j.NotaryKey = opts.NotaryKey
	j.NotaryToken = opts.NotaryToken
	j.release = opts.release
	j.SetID(fmt.Sprintf("%s.distro.%s.repo.%s", jobName, opts.Distro.Type, opts.JobID))

	repoName := j.getPackageLocation()
	scopes := []string{}
	for _, repo := range opts.Distro.Repos {
		switch opts.Distro.Type {
		case RPM:
			scopes = append(scopes, path.Join(j.Distro.Bucket, repo, repoName, opts.Arch))
		case DEB:
			scopes = append(scopes, path.Join(j.Distro.Bucket, repo, repoName))
		default:
			return nil, errors.Errorf("repo type %s is not supported", opts.Distro.Type)
		}
	}

	j.SetScopes(scopes)
	if j.release.IsRelease() {
		j.SetPriority(10)
	}

	return j, nil
}

func (j *repoBuilderJob) setup() {
	if j.builder != nil {
		return
	}

	if j.Distro == nil {
		j.AddError(errors.New("invalid job definition, missing distro"))
		return
	}

	if j.Distro.Type == DEB {
		setupDEBJob(j)
	} else if j.Distro.Type == RPM {
		setupRPMJob(j)
	} else {
		j.AddError(errors.Errorf("invalid distro definition '%s'", j.Distro.Type))
	}

	if j.client == nil {
		j.client = utility.GetDefaultHTTPRetryableClient()
	}

	var err error

	if j.release == nil {
		j.release, err = bond.CreateMongoDBVersion(j.Version)
		if err != nil {
			j.AddError(err)
		}
	}

	j.tmpdir, err = ioutil.TempDir(j.Conf.TempSpace, j.ID())
	if err != nil {
		j.AddError(errors.Wrap(err, "problem making tempdir"))
	}

	if j.NotaryKey == "" {
		j.NotaryKey = os.Getenv("NOTARY_KEY_NAME")
		if j.NotaryKey == "" && j.release != nil {
			if j.Distro.Type == DEB && (j.release.Series() == "3.0" || j.release.Series() == "2.6") {
				j.NotaryKey = "richard"
				j.NotaryToken = os.Getenv("NOTARY_TOKEN_DEB_LEGACY")
			} else if j.release.IsLTS() || j.release.IsContinuous() {
				j.NotaryKey = "server-" + j.release.Series()
			} else {
				j.NotaryKey = "server-" + j.release.StableReleaseSeries()
			}
		}
	}

	if j.NotaryToken == "" {
		j.NotaryToken = os.Getenv("NOTARY_TOKEN")
	}

	if j.NotaryKey == "" {
		j.AddError(errors.New("the notary service auth token (NOTARY_TOKEN) is not defined in the environment"))
	}
}

func (j *repoBuilderJob) cleanup() {
	utility.PutHTTPClient(j.client)
	j.AddError(os.RemoveAll(j.tmpdir))
}

func (j *repoBuilderJob) linkPackages(dest string) error {
	catcher := grip.NewCatcher()
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	for _, pkg := range j.PackagePaths {
		if j.Distro.Type == DEB && !strings.HasSuffix(pkg, ".deb") {
			// the Packages files generated by the compile
			// task are caught in this glob. It's
			// harmless, as we regenerate these files
			// later, but just to be careful and more
			// clear, we should skip these files.
			continue
		}
		if j.Distro.Type == RPM && !strings.HasSuffix(pkg, ".rpm") {
			continue
		}

		if _, err := os.Stat(dest); os.IsNotExist(err) {
			grip.Noticeln("creating directory:", dest)
			if err := os.MkdirAll(dest, 0744); err != nil {
				catcher.Add(errors.Wrapf(err, "problem creating directory %s",
					dest))
				continue
			}
		}

		mirror := filepath.Join(dest, filepath.Base(pkg))
		if j.release.IsDevelopmentBuild() || (j.release.IsLTS() && j.release.IsDevelopmentSeries()) || j.release.IsContinuous() {
			new := strings.Replace(mirror, j.release.String(), j.release.Series(), 1)
			if new != mirror {
				grip.Debug(message.Fields{
					"op":        "renaming development packages",
					"from":      mirror,
					"to":        new,
					"job_id":    j.ID(),
					"job_scope": j.Scopes(),
					"repo":      j.Distro.Name,
					"version":   j.release.String(),
					"package":   pkg,
				})
				if err := os.Rename(mirror, new); err != nil {
					return errors.Wrap(err, "problem renaming development release")
				}
				mirror = new
			}
		}

		if _, err := os.Stat(mirror); os.IsNotExist(err) {
			grip.Debug(message.Fields{
				"op":        "copying packages to local staging",
				"from":      pkg,
				"to":        dest,
				"job_id":    j.ID(),
				"job_scope": j.Scopes(),
				"repo":      j.Distro.Name,
				"version":   j.release.String(),
			})

			if err = os.Link(pkg, mirror); err != nil {
				catcher.Add(errors.Wrapf(err, "problem copying package %s to %s",
					pkg, mirror))
				continue
			}

			if j.Distro.Type == RPM {
				wg.Add(1)
				go func(toSign string) {
					// sign each package, overwriting the package with the signed package.
					catcher.Add(errors.Wrapf(j.signFile(toSign, "", true), // (name, extension, overwrite)
						"problem signing file %s", toSign))
					wg.Done()
				}(mirror)
			}

		}
	}

	return catcher.Resolve()
}

func (j *repoBuilderJob) injectNewPackages(local string) (string, error) {
	return j.builder.injectPackage(local, j.getPackageLocation())
}

func (j *repoBuilderJob) getPackageLocation() string {
	if j.release.IsReleaseCandidate() {
		// release candidates go into the testing repo.
		return "testing"
	} else if j.release.IsDevelopmentBuild() || (j.release.IsLTS() && j.release.IsDevelopmentRelease()) || j.release.IsContinuous() {
		// nightlies and continuous releases go into the "development" repo.
		return "development"
	} else {
		// stable releases and LTS releases have their own repos.
		return j.release.Series()
	}
}

// signFile wraps the python notary-client.py script. Pass it the name
// of a file to sign, the "archiveExtension" (which only impacts
// non-package files, as defined by the notary service and client,)
// and an "overwrite" bool. Overwrite: forces package signing to
// overwrite the existing file, removing the archive's
// signature. Using overwrite=true and a non-nil string is not logical
// and returns a warning, but is passed to the client.
func (j *repoBuilderJob) signFile(fileName, archiveExtension string, overwrite bool) error {
	// In the future it would be nice if we could talk to the
	// notary service directly rather than shelling out here. The
	// final option controls if we overwrite this file.

	args := []string{
		"notary-client.py",
		"--key-name", j.NotaryKey,
		"--auth-token", j.NotaryToken,
		"--comment", "\"curator package signing\"",
		"--notary-url", j.Conf.Services.NotaryURL,
		"--archive-file-ext", archiveExtension,
		"--outputs", "sig",
	}

	grip.AlertWhen(strings.HasPrefix(archiveExtension, "."),
		message.Fields{
			"job_id":    j.ID(),
			"job_scope": j.Scopes(),
			"repo":      j.Distro.Name,
			"version":   j.release.String(),
			"extension": archiveExtension,
			"message":   "extension has leading dot, which is usually problem",
		})

	grip.CriticalWhen(overwrite && len(archiveExtension) != 0,
		message.Fields{
			"job_id":    j.ID(),
			"job_scope": j.Scopes(),
			"repo":      j.Distro.Name,
			"version":   j.release.String(),
			"extension": archiveExtension,
			"message":   "specified overwrite with an archive extension",
			"impact":    "no package impact",
		})

	if overwrite {
		args = append(args, "--package-file-suffix", "")
	} else {
		// if we're not overwriting the unsigned source file
		// with the signed file, then we should remove the
		// signed artifact before. Unclear if this is needed,
		// the cronjob did this.
		grip.Warning(message.WrapError(os.Remove(fileName+"."+archiveExtension),
			message.Fields{
				"mesage":    "problem removing file",
				"filename":  fileName + "." + archiveExtension,
				"job_id":    j.ID(),
				"job_scope": j.Scopes(),
				"repo":      j.Distro.Name,
				"version":   j.release.String(),
			}))
	}

	args = append(args, filepath.Base(fileName))
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = filepath.Dir(fileName)

	grip.Info(message.Fields{
		"message":   "running notary-client command",
		"cmd":       strings.Replace(strings.Join(cmd.Args, " "), j.NotaryToken, "XXXXX", -1),
		"job_id":    j.ID(),
		"job_scope": j.Scopes(),
		"repo":      j.Distro.Name,
		"version":   j.release.String(),
	})

	out, err := cmd.CombinedOutput()
	output := strings.Trim(string(out), " \n\t")

	if err != nil {
		grip.Warning(message.WrapError(err,
			message.Fields{
				"message":   "error signing file",
				"path":      fileName,
				"output":    output,
				"job_id":    j.ID(),
				"job_scope": j.Scopes(),
				"repo":      j.Distro.Name,
				"version":   j.release.String(),
			}))
		return errors.Wrap(err, "problem with notary service client signing file")
	}

	grip.Info(message.Fields{
		"message":   "signed file",
		"path":      fileName,
		"output":    output,
		"job_id":    j.ID(),
		"job_scope": j.Scopes(),
		"repo":      j.Distro.Name,
		"version":   j.release.String(),
	})

	return nil
}

func (j *repoBuilderJob) processPackages(ctx context.Context) error {
	paths := []string{}
	catcher := grip.NewBasicCatcher()
	for idx, path := range j.PackagePaths {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if !strings.HasPrefix(path, "http") {
			paths = append(paths, path)
			continue
		}
		req, err := http.NewRequest(http.MethodGet, path, nil)
		if err != nil {
			catcher.Add(err)
			continue
		}
		req = req.WithContext(ctx)

		resp, err := j.client.Do(req)
		if err != nil {
			catcher.Add(err)
			continue
		}
		if resp.StatusCode != 200 {
			catcher.Errorf("got status %d downloading packages", resp.StatusCode)
			continue
		}
		localPath := filepath.Join(j.tmpdir, filepath.Base(path))
		defer func() { catcher.Add(resp.Body.Close()) }()
		file, err := os.Create(localPath)
		if err != nil {
			catcher.Add(errors.Wrapf(err, "failed to create local path '%s'", localPath))
			break
		}
		_, err = io.Copy(file, resp.Body)
		if err != nil {
			catcher.Add(err)
			catcher.Add(file.Close())
			break
		}

		if err = file.Close(); err != nil {
			catcher.Add(err)
			break
		}

		if strings.HasSuffix(localPath, ".deb") || strings.HasSuffix(localPath, ".rpm") {
			paths = append(paths, localPath)
			continue
		}

		var expandedPath string
		for archiverName, ff := range archiver.SupportedFormats {
			if !ff.Match(localPath) {
				continue
			}

			expandedPath = filepath.Join(j.tmpdir, fmt.Sprintf("extracted-%d-%s", idx, archiverName))
			if err := ff.Open(localPath, expandedPath); err != nil {
				catcher.Add(err)
			}

			break
		}
		if expandedPath == "" {
			catcher.Errorf("could not expand archive for %s", localPath)
		}
		catcher.Add(filepath.Walk(expandedPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".deb") || strings.HasSuffix(path, ".rpm") {
				paths = append(paths, path)
			}
			return nil
		}))
	}

	grip.Info(message.Fields{
		"message":    "processed paths",
		"job_scope":  j.Scopes(),
		"repo":       j.Distro.Name,
		"version":    j.release.String(),
		"paths":      paths,
		"input":      j.PackagePaths,
		"has_errors": catcher.HasErrors(),
	})

	if !catcher.HasErrors() {
		j.PackagePaths = paths
	}

	return catcher.Resolve()
}

// Run is the main execution entry point into repository building, and is a component
func (j *repoBuilderJob) Run(ctx context.Context) {
	defer j.MarkComplete()

	j.setup()
	if j.HasErrors() {
		return
	}

	defer j.cleanup()
	opts := pail.S3Options{
		Region:                   j.Distro.Region,
		SharedCredentialsProfile: j.AWSProfile,
		Name:                     j.Distro.Bucket,
		DryRun:                   j.Conf.DryRun,
		Verbose:                  j.Conf.Verbose,
		UseSingleFileChecksums:   true,
		DeleteOnPull:             true,
		Permissions:              pail.S3PermissionsPublicRead,
		MaxRetries:               10,
	}

	if j.AWSKey != "" {
		opts.Credentials = pail.CreateAWSCredentials(j.AWSKey, j.AWSSecret, j.AWSToken)
	}

	bucket, err := pail.NewS3Bucket(opts)
	if err != nil {
		j.AddError(errors.Wrapf(err, "problem getting s3 bucket %s", j.Distro.Bucket))
		return
	}

	bucket, err = pail.NewParallelSyncBucket(pail.ParallelBucketOptions{Workers: runtime.NumCPU() * 2, DeleteOnPull: true}, bucket)
	if err != nil {
		j.AddError(errors.Wrap(err, "problem constructing parallel bucket"))
		return
	}

	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		timeout := 120 * time.Minute

		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	if err = j.processPackages(ctx); err != nil {
		j.AddError(err)
		return
	}

	// at the moment there is only multiple repos for RPM distros
	for _, remote := range j.Distro.Repos {
		j.workingDirs = append(j.workingDirs, remote)
		grip.Debug(message.Fields{
			"message":   "rebuilding repo",
			"job_id":    j.ID(),
			"job_scope": j.Scopes(),
			"repo":      j.Distro.Name,
			"version":   j.release.String(),
			"remote":    remote,
			"bucket":    j.Distro.Bucket,
		})

		local := filepath.Join(j.Conf.WorkSpace, j.Distro.Bucket, remote)

		var err error

		if err = os.MkdirAll(local, 0755); err != nil {
			j.AddError(errors.Wrapf(err, "problem creating directory %s", local))
			return
		}
		grip.Debug(message.Fields{
			"message":   "downloading package",
			"job_id":    j.ID(),
			"job_scope": j.Scopes(),
			"repo":      j.Distro.Name,
			"version":   j.release.String(),
			"remote":    remote,
			"local":     local,
		})

		pkgLocation := j.getPackageLocation()
		syncOpts := pail.SyncOptions{
			Local:  filepath.Join(local, pkgLocation),
			Remote: filepath.Join(remote, pkgLocation),
		}
		if err = bucket.Pull(ctx, syncOpts); err != nil {
			j.AddError(errors.Wrapf(err, "problem syncing from %s to %s", remote, local))
			return
		}

		grip.Info(message.Fields{
			"message":   "copying new packages into local staging area",
			"job_id":    j.ID(),
			"job_scope": j.Scopes(),
			"repo":      j.Distro.Name,
			"version":   j.release.String(),
		})

		changed, err := j.injectNewPackages(local)
		if err != nil {
			j.AddError(errors.Wrap(err, "problem copying packages into staging repos"))
			return
		}

		// rebuildRepo may hold the lock (and does for
		// the bulk of the operation with RPM
		// distros.)
		if err = j.builder.rebuildRepo(changed); err != nil {
			j.AddError(errors.Wrapf(err, "problem building repo in '%s'", changed))
			return
		}

		var syncSource string
		var changedComponent string

		if j.Distro.Type == DEB {
			changedComponent = filepath.Dir(changed[len(local)+1:])
			syncSource = filepath.Dir(changed)
		} else if j.Distro.Type == RPM {
			changedComponent = changed[len(local)+1:]
			syncSource = changed
		} else {
			j.AddError(errors.Errorf("curator does not support uploading '%s' repos",
				j.Distro.Type))
			return
		}

		// do the sync. It's ok,
		syncOpts = pail.SyncOptions{
			Local:  syncSource,
			Remote: filepath.Join(remote, changedComponent),
		}
		err = bucket.Push(ctx, syncOpts)
		if err != nil {
			j.AddError(errors.Wrapf(err, "problem uploading %s to %s/%s",
				syncSource, j.Distro.Bucket, changedComponent))
			return
		}
	}

	msg := message.Fields{
		"message":   "completed rebuilding repositories",
		"job_id":    j.ID(),
		"job_scope": j.Scopes(),
		"repo":      j.Distro.Name,
		"version":   j.release.String(),
	}

	if j.HasErrors() {
		msg["outcome"] = "encountered problem"
		grip.Warning(message.WrapError(j.Error(), msg))
	} else {
		grip.Info(msg)
	}
}
