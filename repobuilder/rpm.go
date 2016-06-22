package repobuilder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/dependency"
	"github.com/mongodb/amboy/job"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/curator"
	"github.com/mongodb/curator/sthree"
	"github.com/satori/go.uuid"
	"github.com/tychoish/grip"
)

type BuildRPMRepoJob struct {
	Name         string                `bson:"name" json:"name" yaml:"name"`
	IsComplete   bool                  `bson:"completed" json:"completed" yaml:"completed"`
	T            amboy.JobType         `bson:"job_type" json:"job_type" yaml:"job_type"`
	D            dependency.Manager    `bson:"dependency" json:"dependency" yaml:"dependency"`
	Config       *RepositoryDefinition `bson:"repo" json:"repo", yaml:"repo"`
	Output       map[string]string     `bson:"output" json:"output" yaml:"output"`
	Version      string                `bson:"version" json:"version" yaml:"version"`
	Arch         string                `bson:"arch" json:"arch" yaml:"arch"`
	PackagePaths []string              `bson:"package_paths" json:"package_paths" yaml:"package_paths"`
	DryRun       bool                  `bson:"dry_run" json:"dry_run" yaml:"dry_run"`
	Profile      string                `bson:"aws_profile" json:"aws_profile" yaml:"aws_profile"`
	release      *curator.MongoDBVersion
	grip         grip.Journaler
	workingDirs  []string
	mutex        sync.Mutex
}

func init() {
	registry.AddJobType("build-rpm-repo", func() amboy.Job {
		return buildRPMRepoJob()
	})
}

func buildRPMRepoJob() *BuildRPMRepoJob {
	return &BuildRPMRepoJob{
		D:      dependency.NewAlways(),
		Output: make(map[string]string),
		grip:   grip.NewJournaler("curator.rpm.builder"),
		T: amboy.JobType{
			Name:    "build-rpm-repo",
			Version: 0,
		},
	}
}

func NewBuildRPMRepo(c *RepositoryDefinition, version, arch, profile string, pkgs ...string) (*BuildRPMRepoJob, error) {
	var err error
	r := buildRPMRepoJob()
	r.grip.CloneSender(grip.Sender())
	r.Name = fmt.Sprintf("build-rpm-repo.%d", job.GetNumber())
	r.Config = c
	r.PackagePaths = pkgs
	r.Version = version
	r.Arch = arch
	r.Profile = profile
	r.release, err = curator.NewMongoDBVersion(version)
	return r, err
}

func (j *BuildRPMRepoJob) ID() string {
	return j.Name
}

func (j *BuildRPMRepoJob) Completed() bool {
	return j.IsComplete
}

func (j *BuildRPMRepoJob) Type() amboy.JobType {
	return j.T
}

func (j *BuildRPMRepoJob) Dependency() dependency.Manager {
	return j.D
}

func (j *BuildRPMRepoJob) SetDependency(d dependency.Manager) {
	if d.Type().Name == dependency.AlwaysRun {
		j.D = d
	} else {
		j.grip.Warning("repo building jobs should take 'always'-run dependencies.")
	}
}

func (j *BuildRPMRepoJob) markComplete() {
	j.IsComplete = true
}

func (j *BuildRPMRepoJob) linkPackages(dest string) error {
	catcher := grip.NewCatcher()
	for _, pkg := range j.PackagePaths {
		if j.DryRun {
			j.grip.Noticef("dry-run: would link %s in %s", pkg, dest)
			continue
		}

		err := os.MkdirAll(dest, 0744)
		if err != nil {
			catcher.Add(err)
			continue
		}
		j.grip.Infof("copying package %s to local staging error", pkg)
		catcher.Add(os.Link(pkg, filepath.Join(dest, filepath.Base(pkg))))
	}
	return catcher.Resolve()
}

func (j *BuildRPMRepoJob) injectNewPackages(local, target, arch string) ([]string, error) {
	catcher := grip.NewCatcher()
	var changedRepos []string

	repoName := filepath.Join([]string{local, target}...)

	seriesRepoPath := filepath.Join(repoName, j.release.Series())
	changedRepos = append(changedRepos, seriesRepoPath)
	catcher.Add(j.linkPackages(filepath.Join(seriesRepoPath, arch, "RPMS")))

	if j.release.IsStableSeries() {
		stableRepoPath := filepath.Join(repoName, "stable")
		changedRepos = append(changedRepos, stableRepoPath)
		catcher.Add(j.linkPackages(filepath.Join(stableRepoPath, arch, "RPMS")))
	}

	if j.release.IsDevelopmentSeries() {
		devRepoPath := filepath.Join(repoName, "unstable")
		changedRepos = append(changedRepos, devRepoPath)
		catcher.Add(j.linkPackages(filepath.Join(devRepoPath, arch, "RPMS")))
	}

	if j.release.IsReleaseCandidate() || j.release.IsDevelopmentBuild() {
		testingRepoPath := filepath.Join(repoName, "testing")
		changedRepos = append(changedRepos, testingRepoPath)
		catcher.Add(j.linkPackages(filepath.Join(testingRepoPath, arch, "RPMS")))
	}

	return changedRepos, catcher.Resolve()
}

func (j *BuildRPMRepoJob) rebuildRepo(workingDir string, catcher *grip.MultiCatcher, wg *sync.WaitGroup) {
	var output string

	cmd := exec.Command("createrepo", "-d", "-s", "sha .")
	cmd.Dir = workingDir

	if j.DryRun {
		output = "no output: dry run"
		j.grip.Noticeln("[dry-run] would run:", strings.Join(cmd.Args, " "))
	} else {
		out, err := cmd.CombinedOutput()
		catcher.Add(err)
		output = string(out)
		j.grip.Debug(output)
	}

	j.grip.Infoln("rebuilt repo for:", workingDir)
	j.mutex.Lock()
	j.Output[workingDir] = output
	j.mutex.Unlock()

	// TODO add regeneration of HTML pages here for MAKE-6
	wg.Done()
}

func (j *BuildRPMRepoJob) Run() error {
	localStaging, err := os.Getwd()
	if err != nil {
		return err
	}

	bucket := sthree.GetBucketWithProfile(j.Config.Bucket, j.Profile)
	err = bucket.Open()
	defer bucket.Close()
	if err != nil {
		return err
	}

	defer j.markComplete()
	wg := &sync.WaitGroup{}
	catcher := grip.NewCatcher()

	for _, remote := range j.Config.Repos {
		wg.Add(1)
		go func(repo *RepositoryDefinition, local, remote string) {
			j.grip.Infof("rebuilding %s.%s", bucket, remote)
			defer wg.Done()

			var err error
			local = filepath.Join(local, uuid.NewV4().String())
			j.workingDirs = append(j.workingDirs, local)

			if j.DryRun {
				j.grip.Noticef("dry-run: would create '%s' directory", filepath.Abs(local))
				j.grip.Noticef("dry-run: would download from %s to %s", remote, local)
			} else {
				err = os.MkdirAll(local, 0755)
				if err != nil {
					catcher.Add(err)
					return
				}

				err = bucket.SyncFrom(local, remote)
				if err != nil {
					catcher.Add(err)
					return
				}
			}

			changedRepos, err := j.injectNewPackages(local, remote, j.Arch)
			if err != nil {
				catcher.Add(err)
				return
			}

			rWg := &sync.WaitGroup{}
			rCatcher := grip.NewCatcher()
			for _, dir := range changedRepos {
				rWg.Add(1)
				go j.rebuildRepo(dir, rCatcher, rWg)
			}
			rWg.Wait()

			if rCatcher.HasErrors() {
				j.grip.Errorf("encountered error rebuilding %s (%s). Uploading no data",
					remote, local)
				catcher.Add(rCatcher.Resolve())
				return
			}

			if j.DryRun {
				j.grip.Noticef("in dry run mode. otherwise would have built %s (%s)",
					remote, local)
			} else {
				// don't need to return early here, only
				// because this is the last operation.
				catcher.Add(bucket.SyncTo(local, remote))
				j.grip.Noticef("completed rebuilding repo %s (%s)", remote, local)
			}
		}(j.Config, localStaging, remote)
	}
	wg.Wait()

	j.grip.Notice("completed rebuilding all repositories")
	return catcher.Resolve()
}
