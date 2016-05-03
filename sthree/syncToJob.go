package sthree

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/tychoish/amboy"
	"github.com/tychoish/amboy/dependency"
	"github.com/tychoish/amboy/job"
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
	remoteFile *s3Item
	b          *Bucket
	t          amboy.JobType
	name       string
	localPath  string
}

func newSyncToJob(localPath string, remoteFile *s3Item, b *Bucket) *syncToJob {
	return &syncToJob{
		name:       fmt.Sprintf("%s.%d.sync-to", remoteFile.Name, job.GetJobNumber()),
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

// Run executes the synchronization job. If local file doesn't exist
// this operation becomes a noop. Otherwise, will always upload the
// local file if a remote file exists, and if both the local and
// remote file exists, compares the hashes between these files and
// uploads the local file if it differs from the remote file.
func (j *syncToJob) Run() error {
	defer j.markComplete()

	// if the local file doesn't exist or has disappeared since
	// the job was created, there's nothing to do, we can return early
	if _, err := os.Stat(j.localPath); os.IsNotExist(err) {
		return nil
	}

	if j.remoteFile == nil {
		j.b.catcher.Add(j.b.Put(j.localPath, j.remoteFile.Name))
		return nil
	}

	if j.remoteFile.Name == "" {
		// first double check that it doesn't exist (s3 is
		// eventually consistent.) if the file has appeared we
		// can safely fall through this case and compare hashes.

		exists, err := j.b.bucket.Exists(j.remoteFile.Name)
		j.b.catcher.Add(err)

		if err == nil && !exists {
			j.b.catcher.Add(j.b.Put(j.localPath, j.remoteFile.Name))
			return nil
		}
	}

	// if the remote object exists, then we should compare md5
	// checksums between the local and remote objects and upload
	// the local file if they differ.
	data, err := ioutil.ReadFile(j.localPath)
	j.b.catcher.Add(err)
	if err == nil {
		if fmt.Sprintf("%x", md5.Sum(data)) != j.remoteFile.MD5 {
			j.b.catcher.Add(j.b.Put(j.localPath, j.remoteFile.Name))
		}
	}

	return nil
}

func (j *syncToJob) Dependency() dependency.Manager {
	return dependency.NewAlwaysDependency()
}

func (j *syncToJob) SetDependency(_ dependency.Manager) {
	return
}
