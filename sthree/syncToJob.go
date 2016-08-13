package sthree

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/goamz/goamz/s3"
	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/dependency"
	"github.com/mongodb/amboy/job"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

// Not making this job public or registering it with amboy because it
// doesn't make sense to run these jobs in a shared queue or a
// distributed queue. These are implementation details of the Bucket
// type and its methods.

// syncToJob implements amboy.Job and is used in conjunction with
// Bucket's internal method to support paralleled sync operations. See
// the documentation of the Run method for information about the
// behavior of the job.
type syncToJob struct {
	isComplete bool
	remoteFile s3.Key
	b          *Bucket
	t          amboy.JobType
	name       string
	localPath  string
	errors     []error
}

const timeFormat = "2006-01-02T15:04:05.000Z07:00"

func newSyncToJob(localPath string, remoteFile s3.Key, b *Bucket) *syncToJob {
	return &syncToJob{
		name:       fmt.Sprintf("%s.%d.sync-to", remoteFile.Key, job.GetNumber()),
		remoteFile: remoteFile,
		localPath:  localPath,
		b:          b,
		t: amboy.JobType{
			Name:    "s3-sync-to",
			Version: 0,
		},
	}
}

func (j *syncToJob) ID() string {
	return j.name
}

func (j *syncToJob) Type() amboy.JobType {
	return j.t
}

func (j *syncToJob) Completed() bool {
	return j.isComplete
}

func (j *syncToJob) markComplete() {
	j.isComplete = true
}

func (j *syncToJob) doPut() error {
	return j.b.Put(j.localPath, j.remoteFile.Key)
}

func (j *syncToJob) addError(err error) {
	if err != nil {
		j.errors = append(j.errors, err)
	}
}

func (j *syncToJob) Error() error {
	if len(j.errors) == 0 {
		return nil
	}

	var outputs []string

	for _, err := range j.errors {
		outputs = append(outputs, fmt.Sprintf("%+v", err))
	}

	return errors.New(strings.Join(outputs, "\n"))
}

// Run executes the synchronization job. If local file doesn't exist
// this operation becomes a noop. Otherwise, will always upload the
// local file if a remote file exists, and if both the local and
// remote file exists, compares the timestamp and size, and if the
// local file is newer and of different size, it uploads the current
// file. A previous implementation attempted to compare hashes, but
// the s3 API would not reliably return checksums.
func (j *syncToJob) Run() {
	defer j.markComplete()

	// if the local file doesn't exist or has disappeared since
	// the job was created, there's nothing to do, we can return early
	stat, err := os.Stat(j.localPath)
	if os.IsNotExist(err) {
		grip.Debugf("local file %s does not exist, so we can't upload it", j.localPath)
		return
	}

	// first double check that it doesn't exist (s3 is eventually
	// consistent.) if the file has appeared since we created the
	// task can safely fall through this case and compare hashes,
	// otherwise we should put it here.
	exists, err := j.b.bucket.Exists(j.localPath)
	if err != nil {
		j.addError(errors.Wrapf(err,
			"problem checking if the file '%s' exists in the bucket %s",
			j.localPath, j.b.name))
		return
	}
	if !exists {
		err = j.doPut()
		if err != nil {
			j.addError(errors.Wrapf(err, "problem uploading file %s -> %s",
				j.localPath, j.remoteFile.Key))
			return
		}
	}

	objModTime, err := time.Parse(timeFormat, j.remoteFile.LastModified)
	if err != nil {
		grip.Warningf("could not identify the modification time for '%s', uploading local copy",
			j.remoteFile.Key)
		err = j.doPut()
		if err != nil {
			j.addError(errors.Wrapf(err, "problem uploading file '%s' during sync",
				j.remoteFile.Key))
		}
		return
	}

	if stat.ModTime().After(objModTime) && stat.Size() != j.remoteFile.Size {
		err = j.doPut()
		if err != nil {
			j.addError(errors.Wrapf(err, "problem uploading file '%s' during sync",
				j.remoteFile.Key))
		}
		return
	}
}

func (j *syncToJob) Dependency() dependency.Manager {
	return dependency.NewAlways()
}

func (j *syncToJob) SetDependency(_ dependency.Manager) {
	return
}
