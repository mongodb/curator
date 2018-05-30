package sthree

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/goamz/goamz/s3"
	"github.com/mongodb/amboy/job"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SyncFromSuite contains tests of some of the more specific behaviors
// of the SyncFrom operation, and it's backing amboy.Job
// Implementation. While this is comparatively slim, sync operations
// are thoroughly tested: in the BucketJobSuite, we check basic
// amboy.Job features and support, and in the BucketSuite we test the
// job in integration with the bucket, as in the public interface.
type SyncFromSuite struct {
	bucket   *Bucket
	job      *syncFromJob
	require  *require.Assertions
	uuid     string
	toDelete []string
	suite.Suite
}

func TestSyncFromSuite(t *testing.T) {
	suite.Run(t, new(SyncFromSuite))
}

func (s *SyncFromSuite) SetupSuite() {
	s.require = s.Require()

	id := uuid.NewV4()
	s.uuid = id.String()
}

func (s *SyncFromSuite) SetupTest() {
	s.bucket = GetBucket("build-test-curator")
	s.job = &syncFromJob{b: s.bucket, Base: &job.Base{}}
}

func (s *SyncFromSuite) TearDownTest() {
	s.bucket.Close()
}

func (s *SyncFromSuite) TearDownSuite() {
	s.NoError(s.bucket.DeletePrefix(s.uuid))
	for _, fn := range s.toDelete {
		if _, err := os.Stat(fn); !os.IsNotExist(err) {
			s.NoError(os.Remove(fn))
		}
	}
}

func (s *SyncFromSuite) TestSyncIsNoopWithoutDeleteIfNoRemoteFile() {
	s.job.withDelete = false
	s.job.remoteFile = s3.Key{Key: "NO-EXISTS"}
	s.False(s.bucket.dryRun)
	s.job.Run(context.TODO())
	s.NoError(s.job.Error())
}

func (s *SyncFromSuite) TestSyncIsNoopInDryRunWithDeleteIfNoRemoteFile() {
	s.bucket.dryRun = true
	s.job.withDelete = true
	s.job.remoteFile = s3.Key{Key: "NO-EXISTS"}
	s.True(s.bucket.dryRun)
	s.job.Run(context.TODO())
	s.NoError(s.job.Error())
}

func (s *SyncFromSuite) writeToTempFile() string {
	f, err := ioutil.TempFile("", s.uuid)
	if !s.NoError(err) {
		return f.Name()
	}

	if !s.NoError(ioutil.WriteFile(f.Name(), uuid.NewV4().Bytes(), 0644)) {
		return f.Name()
	}

	s.toDelete = append(s.toDelete, f.Name())

	return f.Name()
}

func (s *SyncFromSuite) TestSyncWithDeleteRemovesFile() {
	s.job.withDelete = true
	s.job.localPath = s.writeToTempFile()
	s.job.remoteFile = s3.Key{Key: "NO-EXISTS"}

	s.False(s.bucket.dryRun)
	s.job.Run(context.TODO())
	s.NoError(s.job.Error())

	_, err := os.Stat(s.job.localPath)
	s.True(os.IsNotExist(err))
}

func (s *SyncFromSuite) TestSyncWithoutDeleteDoesNotRemoveFile() {
	s.job.withDelete = false
	s.job.localPath = s.writeToTempFile()
	s.job.remoteFile = s3.Key{Key: "NO-EXISTS"}

	exists, err := s.bucket.Exists(s.job.remoteFile.Key)
	s.NoError(err)
	s.False(exists)

	s.False(s.bucket.dryRun)
	s.job.Run(context.TODO())
	s.NoError(s.job.Error())

	exists, err = s.bucket.Exists(s.job.remoteFile.Key)
	s.NoError(err)
	s.False(exists)
	_, err = os.Stat(s.job.localPath)
	s.False(os.IsNotExist(err))
}
