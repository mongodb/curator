package sthree

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/dependency"
	"github.com/mongodb/amboy/job"
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
	remoteFile *s3Item
	b          *Bucket
	t          amboy.JobType
	name       string
	localPath  string
}

func newSyncToJob(localPath string, remoteFile *s3Item, b *Bucket) *syncToJob {
	return &syncToJob{
		name:       fmt.Sprintf("%s.%d.sync-to", remoteFile.Name, job.GetNumber()),
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

	catcher := grip.NewCatcher()

	// if the local file doesn't exist or has disappeared since
	// the job was created, there's nothing to do, we can return early
	if _, err := os.Stat(j.localPath); os.IsNotExist(err) {
		return nil
	}

	// first double check that it doesn't exist (s3 is eventually
	// consistent.) if the file has appeared since we created the
	// task can safely fall through this case and compare hashes,
	// otherwise we should put it here.
	exists, err := j.b.bucket.Exists(j.localPath)
	catcher.Add(err)

	if err == nil && !exists {
		catcher.Add(j.b.Put(j.localPath, j.remoteFile.Name))
		return nil
	}

	// if s3 doesn't know what the hash of the remote file is or
	// returns it to us, then we don't need to hash it locally,
	// because we'll always upload it in that situation.
	if j.remoteFile.MD5 == "" {
		catcher.Add(j.b.Put(j.localPath, j.remoteFile.Name))
		return nil
	}

	// if the remote object exists, then we should compare md5
	// checksums between the local and remote objects and upload
	// the local file if they differ.
	data, err := ioutil.ReadFile(j.localPath)
	catcher.Add(err)

	if err == nil {
		if fmt.Sprintf("%x", md5.Sum(data)) != j.remoteFile.MD5 {
			catcher.Add(j.b.Put(j.localPath, j.remoteFile.Name))
		}
	}

	return catcher.Resolve()
}

func (j *syncToJob) Dependency() dependency.Manager {
	return dependency.NewAlways()
}

func (j *syncToJob) SetDependency(_ dependency.Manager) {
	return
}
