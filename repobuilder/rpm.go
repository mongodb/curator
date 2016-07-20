package repobuilder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/job"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/curator"
	"github.com/tychoish/grip"
)

type BuildRPMRepoJob struct {
	Job
}

func init() {
	registry.AddJobType("build-rpm-repo", func() amboy.Job {
		return &BuildRPMRepoJob{*buildRepoJob()}
	})
}

func NewBuildRPMRepo(conf *RepositoryConfig, distro *RepositoryDefinition, version, arch, profile string, pkgs ...string) (*BuildRPMRepoJob, error) {
	var err error
	r := &BuildRPMRepoJob{*buildRepoJob()}

	r.WorkSpace, err = os.Getwd()
	if err != nil {
		grip.Errorln("system error: cannot determine the current working directory.",
			"not creating a job object.")
		return nil, err
	}

	r.release, err = curator.NewMongoDBVersion(version)
	if err != nil {
		return nil, err
	}

	r.JobType = amboy.JobType{
		Name:    "build-rpm-repo",
		Version: 0,
	}
	r.grip = grip.NewJournaler("curator.repobuilder.rpm")
	r.grip.CloneSender(grip.Sender())
	r.grip.SetThreshold(grip.ThresholdLevel())
	r.Name = fmt.Sprintf("build-rpm-repo.%d", job.GetNumber())
	r.Distro = distro
	r.Conf = conf
	r.PackagePaths = pkgs
	r.Version = version
	r.Arch = arch
	r.Profile = profile

	return r, nil
}

func (j *BuildRPMRepoJob) injectNewPackages(local string) ([]string, error) {
	catcher := grip.NewCatcher()
	var changedRepos []string

	if j.release.IsRelease() {
		seriesRepoPath := filepath.Join(local, j.release.Series(), j.Arch)
		changedRepos = append(changedRepos, seriesRepoPath)
		catcher.Add(j.linkPackages(filepath.Join(seriesRepoPath, "RPMS")))
	}

	if j.release.IsStableSeries() {
		mirror, ok := j.Conf.Mirrors[j.release.Series()]
		if ok && mirror == "stable" {
			stableRepoPath := filepath.Join(local, "stable", j.Arch)
			changedRepos = append(changedRepos, stableRepoPath)
			catcher.Add(j.linkPackages(filepath.Join(stableRepoPath, "RPMS")))
		}
	}

	if j.release.IsDevelopmentSeries() {
		mirror, ok := j.Conf.Mirrors[j.release.Series()]
		if ok && mirror == "unstable" {
			devRepoPath := filepath.Join(local, "unstable", j.Arch)
			changedRepos = append(changedRepos, devRepoPath)
			catcher.Add(j.linkPackages(filepath.Join(devRepoPath, "RPMS")))
		}
	}

	if j.release.IsReleaseCandidate() || j.release.IsDevelopmentBuild() {
		testingRepoPath := filepath.Join(local, "testing", j.Arch)
		changedRepos = append(changedRepos, testingRepoPath)
		catcher.Add(j.linkPackages(filepath.Join(testingRepoPath, "RPMS")))
	}

	return changedRepos, catcher.Resolve()
}

func (j *BuildRPMRepoJob) rebuildRepo(workingDir string, catcher *grip.MultiCatcher, wg *sync.WaitGroup) {
	defer wg.Done()

	var output string

	cmd := exec.Command("createrepo", "-d", "-s", "sha", workingDir)
	j.grip.Infoln("building repo with operation:", strings.Join(cmd.Args, " "))

	if j.DryRun {
		output = "no output: dry run"
		j.grip.Noticeln("[dry-run] would run:", strings.Join(cmd.Args, " "))
	} else {
		j.grip.Noticeln("building repo with operation:", strings.Join(cmd.Args, " "))
		out, err := cmd.CombinedOutput()
		catcher.Add(err)
		output = string(out)
		if err != nil {
			j.grip.Error(err)
			j.grip.Info(output)
		} else {
			j.grip.Debug(output)
		}
	}

	j.grip.Infoln("rebuilt repo for:", workingDir)
	j.mutex.Lock()
	j.Output[workingDir] = output
	j.mutex.Unlock()

	catcher.Add(j.Conf.BuildIndexPageForDirectory(workingDir, j.Distro.Bucket))
}
