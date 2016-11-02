package recall

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mholt/archiver"
	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/dependency"
	"github.com/mongodb/amboy/job"
	"github.com/mongodb/amboy/registry"
	"github.com/pkg/errors"
	"github.com/tychoish/bond"
	"github.com/tychoish/grip"
)

// DownloadFileJob is an amboy.Job implementation that supports
// downloading a a file to the local file system.
type DownloadFileJob struct {
	URL       string `bson:"url" json:"url" yaml:"url"`
	Directory string `bson:"dir" json:"dir" yaml:"dir"`
	FileName  string `bson:"file" json:"file" yaml:"file"`
	*job.Base `bson:"metadata" json:"metadata" yaml:"metadata"`
}

func init() {
	registry.AddJobType("bond-recall-download-file", func() amboy.Job {
		return newDownloadJob()
	})
}

func newDownloadJob() *DownloadFileJob {
	return &DownloadFileJob{
		Base: &job.Base{
			JobType: amboy.JobType{
				Name:    "bond-recall-download-file",
				Version: 0,
				Format:  amboy.JSON,
			},
		},
	}
}

// NewDownloadJob constructs a DownloadFileJob. The job has a
// dependency on the downloaded file, and will only execute if that
// file does not exist.
func NewDownloadJob(url, path string, force bool) (*DownloadFileJob, error) {
	j := newDownloadJob()
	if err := j.setURL(url); err != nil {
		return nil, errors.Wrap(err, "problem constructing Job object (url)")
	}

	if err := j.setDirectory(path); err != nil {
		return nil, errors.Wrap(err, "problem constructing Job object (directory)")
	}

	fn := j.getFileName()
	j.SetID(fmt.Sprintf("%s-%d",
		strings.Replace(fn, string(filepath.Separator), "-", -1),
		job.GetNumber()))

	if force {
		j.SetDependency(dependency.NewAlways())
	} else {
		j.SetDependency(dependency.NewCreatesFile(fn))
	}

	return j, nil
}

// Run implements the main action of the Job. This implementation
// checks the job directly and returns early if the downloaded file
// exists. This behavior may be redundant in the case that the queue
// skips jobs with "passed" jobs.
func (j *DownloadFileJob) Run() {
	defer j.MarkComplete()

	fn := j.getFileName()
	defer attemptTimestampUpdate(fn)

	// in theory the queue should do this next check, but most do not
	if state := j.Dependency().State(); state == dependency.Passed {
		grip.Noticef("file %s is already downloaded", fn)
		return
	}

	if err := bond.DownloadFile(j.URL, fn); err != nil {
		j.handleError(errors.Wrapf(err, "problem downloading file %s", fn))
		return
	}
	grip.Noticef("downloaded %s file", fn)

	var err error
	if strings.HasSuffix(fn, ".tgz") {
		// there is no tar.gz because we renamed it in setURL()
		err = archiver.TarGz.Open(fn, filepath.Dir(fn))
	} else if strings.HasSuffix(fn, ".zip") {
		err = archiver.Zip.Open(fn, filepath.Dir(fn))
	} else {
		err = errors.Errorf("file %s is in unsupported archive format", fn)
	}

	if err != nil {
		j.handleError(errors.Wrap(err, "problem extracting artifacts"))
		return
	}
}

//
// Internal Methods
//

func attemptTimestampUpdate(fn string) {
	// update the timestamps so we playwell with the cache. These
	// operations are logged but don't impact the tasks error
	// state if they fail.
	now := time.Now()
	if err := os.Chtimes(fn, now, now); err != nil {
		grip.Debug(err)
	}

	// hopefully directory names in archives are the same are the
	// same as the filenames. Wnwinding the assumption will
	// probably require a different archiver tool.
	dirname := fn[0 : len(fn)-len(filepath.Ext(fn))]
	if err := os.Chtimes(dirname, now, now); err != nil {
		grip.Debug(err)
	}
}

func (j *DownloadFileJob) handleError(err error) {
	j.AddError(err)
	grip.CatchError(err)
	grip.CatchDebug(os.RemoveAll(j.getFileName())) // cleanup
}

func (j *DownloadFileJob) getFileName() string {
	return filepath.Join(j.Directory, j.FileName)
}

func (j *DownloadFileJob) setDirectory(path string) error {
	if stat, err := os.Stat(path); !os.IsNotExist(err) && !stat.IsDir() {
		// if the path exists and isn't a directory, then we
		// won't be able to download into it:
		return errors.Errorf("%s is not a directory, cannot download files into it",
			path)
	}

	j.Directory = path
	return nil
}

func (j *DownloadFileJob) setURL(url string) error {
	if !strings.HasPrefix(url, "http") {
		return errors.Errorf("%s is not a valid url", url)
	}

	if strings.HasSuffix(url, "/") {
		return errors.Errorf("%s does not contain a valid filename component", url)
	}

	j.URL = url
	j.FileName = filepath.Base(url)

	if strings.HasSuffix(url, ".tar.gz") {
		j.FileName = filepath.Ext(filepath.Ext(j.FileName)) + ".tgz"
	}

	return nil
}
