package sthree

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

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
	withDelete bool
	remoteFile s3.Key
	b          *Bucket
	t          amboy.JobType
	name       string
	localPath  string
	errors     []error
}

func newSyncToJob(b *Bucket, localPath string, remoteFile s3.Key, withDelete bool) *syncToJob {
	return &syncToJob{
		name:       fmt.Sprintf("%s.%d.sync-to", remoteFile.Key, job.GetNumber()),
		withDelete: withDelete,
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
	err := j.b.Put(j.localPath, j.remoteFile.Key)

	if err != nil {
		return errors.Wrap(err, "s3 error with put during sync")
	}

	return nil
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
// remote file exists, compares the hashes between these files and
// uploads the local file if it differs from the remote file.
func (j *syncToJob) Run() {
	defer j.markComplete()

	// if the local file doesn't exist or has disappeared since
	// the job was created, there's nothing to do, we can return early
	if _, err := os.Stat(j.localPath); os.IsNotExist(err) {
		if j.withDelete && !j.b.dryRun {
			err := j.b.Delete(j.remoteFile.Key)
			if err != nil {
				j.addError(errors.Wrapf(err,
					"problem deleting %s from bucket %s",
					j.remoteFile.Key, j.b.name))
				return
			}
			grip.Debugf("deleted file %s from bucket %s", j.remoteFile.Key, j.b.name)
			return
		} else {
			grip.NoticeWhenf(j.b.dryRun,
				"dry-run: would delete remote file %s from bucket %s because it doesn't exist locally",
				j.remoteFile.Key, j.b.name)

			grip.DebugWhenf(!j.b.dryRun,
				"local file %s does not exist, so we can't upload it", j.localPath)

			return
		}
	}

	// first double check that it doesn't exist (s3 is eventually
	// consistent.) if the file has appeared since we created the
	// task can safely fall through this case and compare hashes,
	// otherwise we should put it here.
	exists, err := j.b.Exists(j.remoteFile.Key)
	if err != nil {
		j.addError(errors.Wrapf(err,
			"problem checking if the file '%s' exists in the bucket %s",
			j.localPath, j.b.name))
		return
	}
	if !exists {
		grip.Debugf("uploading %s because remote file %s/%s does not exist",
			j.localPath, j.b.name, j.remoteFile.Key)
		err = j.doPut()
		if err != nil {
			j.addError(errors.Wrapf(err, "problem uploading file %s -> %s",
				j.localPath, j.remoteFile.Key))
			return
		}
	}

	remoteChecksum := strings.Trim(j.remoteFile.ETag, "\" ")

	// if s3 doesn't know what the hash of the remote file is or
	// returns it to us, then we don't need to hash it locally,
	// because we'll always upload it in that situation.
	if remoteChecksum == "" {
		grip.Debugf("s3 does not report a hash for %s, uploading file", j.remoteFile.Key)
		err = j.doPut()
		if err != nil {
			j.addError(errors.Wrapf(err, "problem uploading file '%s' during sync",
				j.remoteFile.Key))
		}
		return
	}

	// if the remote object exists, then we should compare md5
	// checksums between the local and remote objects and upload
	// the local file if they differ.
	data, err := ioutil.ReadFile(j.localPath)
	if err != nil {
		j.addError(errors.Wrap(err,
			"problem reading file before hashing for sync operation"))
		return
	}

	if fmt.Sprintf("%x", md5.Sum(data)) != remoteChecksum {
		grip.Debugf("hashes aren't the same: [op=push, file=%s, local=%x, remote=%s]",
			j.remoteFile.Key, md5.Sum(data), remoteChecksum)
		err = j.doPut()
		if err != nil {
			j.addError(errors.Wrapf(err, "problem uploading file '%s' during sync",
				j.remoteFile.Key))
		}
	}
}

func (j *syncToJob) Dependency() dependency.Manager {
	return dependency.NewAlways()
}

func (j *syncToJob) SetDependency(_ dependency.Manager) {
	return
}
