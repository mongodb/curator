package repobuilder

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
)

// rpmRepoBuilder contains specific implementation for building RPM
// repositories using createrepo.
type rpmRepoBuilder struct {
	*repoBuilderJob
	mutex sync.Mutex
}

func setupRPMJob(j *repoBuilderJob) {
	r := &rpmRepoBuilder{repoBuilderJob: j}
	r.builder = r
}

func (j *rpmRepoBuilder) injectPackage(local, repoName string) (string, error) {
	repoPath := filepath.Join(local, repoName, j.Arch)
	err := j.linkPackages(filepath.Join(repoPath, "RPMS"))

	return repoPath, errors.Wrapf(err, "linking packages for %s", repoPath)
}

func (j *rpmRepoBuilder) rebuildRepo(workingDir string) error {
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

	if j.Conf.DryRun {
		output = "no output: dry run"
	} else {
		// remove olddata before running createrepo or else the command
		// will fail if it exists.
		if err = os.RemoveAll(".olddata"); err != nil {
			return errors.Wrap("problem removing .olddata dir", err)
		}

		grip.Notice(message.Fields{
			"cmd":       strings.Join(cmd.Args, " "),
			"message":   "building repo with operation",
			"job_id":    j.ID(),
			"job_scope": j.Scopes(),
		})
		out, err = cmd.CombinedOutput()
		output = string(out)
		if err != nil {
			grip.Error(message.WrapError(err, message.Fields{
				"cmd":       strings.Join(cmd.Args, " "),
				"job_id":    j.ID(),
				"job_scope": j.Scopes(),
				"output":    output,
			}))
			return errors.Wrap(err, "problem building repo")
		}
	}

	grip.Info(message.Fields{
		"distro":    j.Distro.Name,
		"dry_run":   j.Conf.DryRun,
		"cmd":       strings.Join(cmd.Args, " "),
		"job_id":    j.ID(),
		"job_scope": j.Scopes(),
		"dir":       workingDir,
		"output":    output,
	})

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
