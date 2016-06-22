package repobuilder

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/job"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/curator"
	"github.com/tychoish/grip"
)

const releaseMetaDataDateFormat = "Wed, 15 Jun 2006 00:02:52 UTC"

type BuildDEBRepoJob struct {
	Job
}

func init() {
	registry.AddJobType("build-rpm-repo", func() amboy.Job {
		return &BuildDEBRepoJob{buildRepoJob()}
	})
}

// TODO: need to find some way to create the arch directories if they
// don't exist, which probably means a configuration change. Currently
// arch directories are created when attempting to build the repos,
// which means that there's a condition where we add a new
// architecture (e.g. a community arm build,) and until we've pushed
// packages to all repos the repo-metadata will be "ahead" of the
// state of the repo. Should correct itself when everything
// pushes. Unclear if current solution is susceptible to this.

func NewBuildDEBRepo(conf *RepositoryConfig, distro *RepositoryDefinition, version, arch, profile string, pkgs ...string) (*BuildDEBRepoJob, error) {
	var err error
	r := &BuildDEBRepoJob{buildRepoJob()}

	r.release, err = curator.NewMongoDBVersion(version)
	if err != nil {
		return nil, err
	}

	r.WorkSpace, err = os.Getwd()
	if err != nil {
		grip.Errorln("system error: cannot determine the current working directory.",
			"not creating a job object.")
		return nil, err
	}

	r.JobType = amboy.JobType{
		Name:    "build-deb-repo",
		Version: 0,
	}
	r.grip = grip.NewJournaler("curator.repobuilder.deb")
	r.grip.CloneSender(grip.Sender())
	r.Name = fmt.Sprintf("build-deb-repo.%d", job.GetNumber())
	r.Distro = distro
	r.Conf = conf
	r.PackagePaths = pkgs
	r.Version = version
	r.Arch = arch
	r.Profile = profile
	return r, nil
}

func (j *BuildDEBRepoJob) injectNewPackages(local, target string) ([]string, error) {
	catcher := grip.NewCatcher()
	var changedRepos []string

	repoName := filepath.Join(local, target)
	arch := "binary-" + j.Arch

	if j.release.IsRelease() {
		seriesRepoPath := filepath.Join(repoName, j.release.Series(), "main")
		changedRepos = append(changedRepos, seriesRepoPath)
		catcher.Add(j.linkPackages(filepath.Join(seriesRepoPath, arch)))
	}

	if j.release.IsStableSeries() {
		mirror, ok := j.Conf.Mirrors[j.release.Series()]
		if ok && mirror == "stable" {
			stableRepoPath := filepath.Join(repoName, "stable", "main")
			changedRepos = append(changedRepos, stableRepoPath)
			catcher.Add(j.linkPackages(filepath.Join(stableRepoPath, arch)))
		}
	}

	if j.release.IsDevelopmentSeries() {
		mirror, ok := j.Conf.Mirrors[j.release.Series()]
		if ok && mirror == "unstable" {
			devRepoPath := filepath.Join(repoName, "unstable", "main")
			changedRepos = append(changedRepos, devRepoPath)
			catcher.Add(j.linkPackages(filepath.Join(devRepoPath, arch)))
		}
	}

	if j.release.IsReleaseCandidate() || j.release.IsDevelopmentBuild() {
		testingRepoPath := filepath.Join(repoName, "testing", "main")
		changedRepos = append(changedRepos, testingRepoPath)
		catcher.Add(j.linkPackages(filepath.Join(testingRepoPath, arch)))
	}

	return changedRepos, catcher.Resolve()
}

func gzipAndWriteToFile(fileName string, content []byte) error {
	var gz bytes.Buffer

	w, err := gzip.NewWriterLevel(&gz, flate.BestCompression)

	if err != nil {
		return err
	}
	_, err = w.Write(content)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fileName, gz.Bytes(), 0644)
}

func (j *BuildDEBRepoJob) rebuildRepo(workingDir string, catcher *grip.MultiCatcher, wg *sync.WaitGroup) {
	defer wg.Done()

	arch := "binary-" + j.Arch

	// start by running dpkg-scanpackages to generate a packages file
	// in the source.
	cmd := exec.Command("dpkg-scanpackages", "--multiversion", arch)
	cmd.Dir = workingDir
	out, err := cmd.Output()
	catcher.Add(err)
	if err != nil {
		return
	}

	// Write the packages file to disk.
	pkgsFile := filepath.Join(workingDir, arch, "Packages")
	err = ioutil.WriteFile(pkgsFile, out, 0644)
	catcher.Add(err)
	if err != nil {
		return
	}

	// Compress/gzip the packages file
	catcher.Add(gzipAndWriteToFile(pkgsFile+".gz", out))

	// Continue by building the Releases file, first by using the
	// template, and then by
	releaseTmplSrc, ok := j.Conf.Templates.Deb[j.Distro.Edition]
	if !ok {
		catcher.Add(fmt.Errorf("no 'Release' template defined for %s", j.Distro.Edition))
		return
	}

	// initialize the template.
	tmpl, err := template.New("Releases").Parse(releaseTmplSrc)
	catcher.Add(err)
	if err != nil {
		return
	}

	// open a file that we're going to write the releases file to.
	relFileName := filepath.Join(workingDir, "Release")
	relFile, err := os.Create(relFileName)
	catcher.Add(err)
	if err != nil {
		return
	}
	err = tmpl.Execute(relFile, struct {
		Name          string
		CodeName      string
		Date          string
		Architectures string
	}{
		Name:          j.Distro.Name,
		CodeName:      j.Distro.CodeName,
		Date:          time.Now().Format(releaseMetaDataDateFormat),
		Architectures: strings.Join(j.Distro.Architectures, " "),
	})
	catcher.Add(err)
	if err != nil {
		return
	}

	// This builds a Release file that includes the header
	// information from all repositories.
	cmd = exec.Command("apt-ftparchive", "release", ".")
	cmd.Dir = workingDir
	out, err = cmd.Output()
	catcher.Add(err)
	j.grip.Debug(string(out))
	if err != nil {
		return
	}

	// add the output of apt-ftparchive to template.
	_, err = relFile.Write(out)
	catcher.Add(err)
	if err != nil {
		return
	}

	// close the file so that the notary client can sign it.
	err = relFile.Close()
	catcher.Add(err)
	if err != nil {
		return
	}

	// sign the file using the notary service. To remove the
	// MongoDB-specificity we could make this configurable, or
	// offer ways of specifying different signing option.
	output, err := j.signFile(relFileName, workingDir)
	j.grip.Debug(output)
	j.mutex.Lock()
	j.Output["sign-release-file-"+workingDir] = output
	j.mutex.Unlock()
	catcher.Add(err)
	if err != nil {
		return
	}

	j.Conf.BuildIndexPageForDirectory(workingDir, j.Distro.Bucket)
}
