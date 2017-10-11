package repobuilder

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

// BuildRPMRepoJob contains specific implementation for building RPM
// repositories using createrepo.
type BuildRPMRepoJob struct {
	*Job
}

func setupRPMJob(j *Job) {
	r := &BuildRPMRepoJob{j}
	r.Job.builder = r
}

func (j *BuildRPMRepoJob) injectPackage(local, repoName string) (string, error) {
	repoPath := filepath.Join(local, repoName, j.Arch)
	err := j.linkPackages(filepath.Join(repoPath, "RPMS"))

	return repoPath, errors.Wrapf(err, "linking packages for %s", repoPath)
}

func (j *BuildRPMRepoJob) rebuildRepo(workingDir string) error {
	var output string
	var err error
	var out []byte

	// We want to ensure that we don't run more than one
	// createrepo operation at a time. Eventually it would be nice
	// if there were a pure-Go way to build repositories that we
	// knew was safe, but we've seen a number of racey-failures
	// because of running createrepo at the same time.
	j.mutex.Lock()
	defer j.mutex.Unlock()

	cmd := exec.Command("createrepo", "-d", "-s", "sha", workingDir)

	if j.DryRun {
		grip.Noticeln("[dry-run] would run:", strings.Join(cmd.Args, " "))
		output = "no output: dry run"
	} else {
		grip.Noticeln("building repo with operation:", strings.Join(cmd.Args, " "))
		out, err = cmd.CombinedOutput()
		output = string(out)
		if err != nil {
			grip.Error(err)
			grip.Info(output)
			return errors.Wrap(err, "problem building repo")
		}
		grip.Debug(output)
	}

	grip.Infoln("rebuilt repo for:", workingDir)
	j.Output[workingDir] = output

	metaDataFile := filepath.Join(workingDir, "repodata", "repomd.xml")

	// signFile(name, extension, overwrite)
	if err = j.signFile(metaDataFile, "asc", false); err != nil {
		return errors.Wrapf(err, "signing release metadata for %s", workingDir)
	}

	if err = j.Conf.BuildIndexPageForDirectory(workingDir, j.Distro.Bucket); err != nil {
		return errors.Wrapf(err, "building index.html pages for %s", workingDir)
	}

	return nil
}
