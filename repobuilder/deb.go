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
	registry.AddJobType("build-deb-repo", func() amboy.Job {
		return &BuildDEBRepoJob{*buildRepoJob()}
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
	r := &BuildDEBRepoJob{Job: *buildRepoJob()}

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

	if arch == "x86_64" {
		r.Arch = "amd64"
	} else {
		r.Arch = arch
	}

	r.grip = grip.NewJournaler("curator.repobuilder.deb")
	r.grip.CloneSender(grip.Sender())
	r.grip.SetThreshold(grip.ThresholdLevel())
	r.Name = fmt.Sprintf("build-deb-repo.%d", job.GetNumber())
	r.Distro = distro
	r.Conf = conf
	r.PackagePaths = pkgs
	r.Version = version
	r.Profile = profile
	return r, nil
}

func (j *BuildDEBRepoJob) createArchDirs(basePath string) ([]string, error) {
	catcher := grip.NewCatcher()
	var changedPaths []string

	for _, arch := range j.Distro.Architectures {
		catcher.Add(os.MkdirAll(filepath.Join(basePath, "binary-"+arch), 0755))
	}

	return changedPaths, catcher.Resolve()
}

func (j *BuildDEBRepoJob) injectNewPackages(local string) ([]string, error) {
	catcher := grip.NewCatcher()
	var changedRepos []string

	arch := "binary-" + j.Arch

	if j.release.IsRelease() {
		seriesRepoPath := filepath.Join(local, j.release.Series(), "main")
		changedRepos = append(changedRepos, seriesRepoPath)
		catcher.Add(j.linkPackages(filepath.Join(seriesRepoPath, arch)))

		extraPaths, err := j.createArchDirs(seriesRepoPath)
		catcher.Add(err)
		changedRepos = append(changedRepos, extraPaths...)
	}

	if j.release.IsStableSeries() {
		mirror, ok := j.Conf.Mirrors[j.release.Series()]
		if ok && mirror == "stable" {
			stableRepoPath := filepath.Join(local, "stable", "main")
			changedRepos = append(changedRepos, stableRepoPath)
			catcher.Add(j.linkPackages(filepath.Join(stableRepoPath, arch)))

			extraPaths, err := j.createArchDirs(stableRepoPath)
			catcher.Add(err)
			changedRepos = append(changedRepos, extraPaths...)
		}
	}

	if j.release.IsDevelopmentSeries() {
		mirror, ok := j.Conf.Mirrors[j.release.Series()]
		if ok && mirror == "unstable" {
			devRepoPath := filepath.Join(local, "unstable", "main")
			changedRepos = append(changedRepos, devRepoPath)
			catcher.Add(j.linkPackages(filepath.Join(devRepoPath, arch)))

			extraPaths, err := j.createArchDirs(devRepoPath)
			catcher.Add(err)
			changedRepos = append(changedRepos, extraPaths...)
		}
	}

	if j.release.IsReleaseCandidate() || j.release.IsDevelopmentBuild() {
		testingRepoPath := filepath.Join(local, "testing", "main")
		changedRepos = append(changedRepos, testingRepoPath)
		catcher.Add(j.linkPackages(filepath.Join(testingRepoPath, arch)))

		extraPaths, err := j.createArchDirs(testingRepoPath)
		catcher.Add(err)
		changedRepos = append(changedRepos, extraPaths...)
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

	err = ioutil.WriteFile(fileName, gz.Bytes(), 0644)
	if err != nil {
		return err
	}

	grip.Noticeln("wrote zipped packages file to:", fileName)
	return nil
}

func (j *BuildDEBRepoJob) rebuildRepo(workingDir string, catcher *grip.MultiCatcher, wg *sync.WaitGroup) {
	defer wg.Done()

	arch := "binary-" + j.Arch

	// start by running dpkg-scanpackages to generate a packages file
	// in the source.
	dirParts := strings.Split(workingDir, string(filepath.Separator))
	cmd := exec.Command("dpkg-scanpackages", "--multiversion", filepath.Join(filepath.Join(dirParts[3:8]...), arch))
	cmd.Dir = filepath.Join(dirParts[:3]...)
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
	j.grip.Noticeln("wrote packages file to:", pkgsFile)

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

	buffer := bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buffer, struct {
		CodeName      string
		Architectures string
	}{
		CodeName:      j.Distro.CodeName,
		Architectures: strings.Join(j.Distro.Architectures, " "),
	})
	catcher.Add(err)
	if err != nil {
		return
	}

	// This builds a Release file using the header info generated
	// from the template above.
	cmd = exec.Command("apt-ftparchive", "release", "../")
	cmd.Dir = workingDir
	out, err = cmd.Output()
	catcher.Add(err)
	outString := string(out)
	j.grip.Debug(outString)
	if err != nil {
		return
	}

	// get the content from the template and add the output of
	// apt-ftparchive there.
	releaseContent := buffer.Bytes()
	releaseContent = append(releaseContent, out...)

	// tracking the output is useful. we'll do that here.
	j.mutex.Lock()
	j.Output["sign-release-file-"+workingDir] = outString
	j.mutex.Unlock()
	catcher.Add(err)

	// write the content of the release file to disk.
	relFileName := filepath.Join(workingDir, "Release")
	err = ioutil.WriteFile(relFileName, releaseContent, 0644)
	catcher.Add(err)
	if err != nil {
		return
	}
	j.grip.Noticeln("wrote release files to:", relFileName)

	// sign the file using the notary service. To remove the
	// MongoDB-specificity we could make this configurable, or
	// offer ways of specifying different signing option.
	output, err := j.signFile(relFileName, false)
	catcher.Add(err)
	if err != nil {
		j.grip.DebugWhen(false, output)
	}

	catcher.Add(j.Conf.BuildIndexPageForDirectory(workingDir, j.Distro.Bucket))
}
