package sthree

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/goamz/goamz/s3"
	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/job"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

// Not making this job public or registering it with amboy because it
// doesn't make sense to run these jobs in a shared queue or a
// distributed queue. These are implementation details of the Bucket
// type and its methods.

// syncFromJob implements amboy.Job and is used in conjunction with
// Bucket's internal method to support paralleled sync operations. See
// the documentation of the Run method for information about the
// behavior of the job.
type syncFromJob struct {
	withDelete bool
	localPath  string
	remoteFile s3.Key
	b          *Bucket

	*amboy.JobBase
}

func newSyncFromJob(b *Bucket, localPath string, remoteFile s3.Key, withDelete bool) *syncFromJob {
	j := &syncFromJob{
		remoteFile: remoteFile,
		withDelete: withDelete,
		localPath:  localPath,
		b:          b,
		JobBase: &amboy.JobBase{
			JobType: amboy.JobType{
				Name:    "s3-sync-from",
				Version: 1,
			},
		},
	}

	j.SetID(fmt.Sprintf("sync-from_%s.%d", localPath, job.GetNumber()))

	return j
}

func (j *syncFromJob) doGet() error {
	err := j.b.Get(j.remoteFile.Key, j.localPath)

	if err != nil {
		return errors.Wrap(err, "problem with s3 get during sync")
	}

	return nil
}

// Run executes the synchronization job. If the local file doesn't
// exist, pulls down the remote file, otherwise hashes the local file
// and compares that hash to the remote file's hash. If they differ,
// pull the remote file.
func (j *syncFromJob) Run() {
	defer j.MarkComplete()

	// if the remote file doesn't exist, we should return early here.
	if j.remoteFile.Key == "" {
		return
	}

	// if the remote file has disappeared, we should return early here.
	exists, err := j.b.Exists(j.remoteFile.Key)
	if err != nil {
		j.AddError(errors.Wrapf(err, "problem checking if the file '%s' exists",
			j.remoteFile.Key))
		return
	}
	if !exists {
		if j.withDelete && !j.b.dryRun {
			err = os.RemoveAll(j.localPath)
			if err != nil {
				j.AddError(errors.Wrapf(err,
					"problem removing local file %s, during sync from bucket %s with delete",
					j.localPath, j.b.name))
				return
			}
			grip.Debugf("removed local file %s during sync from bucket %s with delete",
				j.localPath, j.b.name)
			return
		}

		grip.NoticeWhenf(j.b.dryRun,
			"dry-run: would remove local file %s from because it doesn't exist in bucket %s",
			j.remoteFile.Key, j.b.name)

		// if we get here the file doesn't exist so we should try to copy it.
		grip.WarningWhenf(!j.b.dryRun, "file %s disappeared during sync pull operation. "+
			"Doing nothing because *not* in delete-mode",
			j.remoteFile.Key)

		return
	}

	// if the local file doesn't exist, download the remote file and return.
	if _, err = os.Stat(j.localPath); os.IsNotExist(err) {
		err = j.doGet()
		if err != nil {
			j.AddError(errors.Wrap(err, "problem downloading file during sync"))
		}
		return
	}

	// if both the remote and local files exist, then we should
	// compare md5 checksums between these file and download the
	// remote file if they differ.

	// Start by reading the file.
	data, err := ioutil.ReadFile(j.localPath)
	if err != nil {
		j.AddError(errors.Wrap(err, "problem reading file before hashing for sync operation"))
	}

	remoteChecksum := strings.Trim(j.remoteFile.ETag, "\" ")
	if fmt.Sprintf("%x", md5.Sum(data)) != remoteChecksum {
		grip.Debugf("hashes aren't the same: [op=pull, file=%s, local=%x, remote=%s]",
			j.remoteFile.Key, md5.Sum(data), remoteChecksum)
		err := j.doGet()
		if err != nil {
			j.AddError(errors.Wrapf(err, "problem fetching file '%s' during sync",
				j.remoteFile.Key))
			return
		}
	}

	return
}
