package sthree

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goamz/goamz/s3"
	"github.com/mongodb/amboy/job"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SyncToSuite contains tests of some of the more specific behaviors
// of the SyncTo operation, and it's backing amboy.Job
// Implementation. While this is comparatively slim, sync operations
// are thoroughly tested: in the BucketJobSuite, we check basic
// amboy.Job features and support, and in the BucketSuite we test the
// job in integration with the bucket, as in the public interface.
type SyncToSuite struct {
	bucket   *Bucket
	job      *syncToJob
	require  *require.Assertions
	uuid     string
	tmpDir   string
	toDelete []string
	suite.Suite
}

func TestSyncToSuite(t *testing.T) {
	suite.Run(t, new(SyncToSuite))
}

func (s *SyncToSuite) SetupSuite() {
	s.require = s.Require()

	id := uuid.Must(uuid.NewV4())
	s.uuid = id.String()
	tmpDir, err := ioutil.TempDir("", s.uuid)
	s.require.NoError(err)
	s.tmpDir = tmpDir
}

func (s *SyncToSuite) SetupTest() {
	s.bucket = GetBucket("build-test-curator")
	s.job = &syncToJob{b: s.bucket, Base: &job.Base{}}
}

func (s *SyncToSuite) TearDownTest() {
	s.NoError(s.bucket.DeletePrefix(s.uuid))
	s.bucket.Close()
}

func (s *SyncToSuite) TearDownSuite() {
	s.NoError(s.bucket.DeletePrefix(s.uuid))
	for _, fn := range s.toDelete {
		if _, err := os.Stat(fn); !os.IsNotExist(err) {
			s.NoError(os.Remove(fn))
		}
	}
	s.NoError(s.bucket.DeletePrefix(s.uuid))
}

func (s *SyncToSuite) writeToTempFile() string {
	f, err := ioutil.TempFile("", s.uuid)
	if !s.NoError(err) {
		return f.Name()
	}

	if !s.NoError(ioutil.WriteFile(f.Name(), uuid.Must(uuid.NewV4()).Bytes(), 0644)) {
		return f.Name()
	}

	s.toDelete = append(s.toDelete, f.Name())

	return f.Name()
}

func (s *SyncToSuite) TestSyncWithoutDeleteLeavesRemoteFile() {
	file := s.writeToTempFile()
	key := filepath.Join(s.uuid, file)
	s.NoError(s.bucket.Put(file, key))

	s.NoError(s.job.Error())
	s.job.remoteFile = s3.Key{Key: key}
	s.job.localPath = filepath.Join(s.tmpDir, file)

	s.False(s.job.withDelete)
	s.False(s.job.b.dryRun)
	s.job.Run(context.TODO())
	s.NoError(s.job.Error())

	exists, err := s.bucket.Exists(key)
	s.NoError(err)
	s.True(exists)
}

func (s *SyncToSuite) TestSyncWithDeleteRemovesRemoteFile() {
	s.job.withDelete = true
	s.bucket.dryRun = false
	file := s.writeToTempFile()
	key := filepath.Join(s.uuid, file)
	s.NoError(s.bucket.Put(file, key))

	s.NoError(s.job.Error())
	s.job.remoteFile = s3.Key{Key: key}
	s.job.localPath = filepath.Join(s.tmpDir, file)

	s.True(s.job.withDelete)
	s.False(s.job.b.dryRun)
	s.job.Run(context.TODO())
	s.NoError(s.job.Error())

	exists, err := s.bucket.Exists(key)
	s.NoError(err)
	s.False(exists)
}

func (s *SyncToSuite) TestSyncWithDeleteAndDryRunDoesNotRemovesRemoteFile() {
	s.job.withDelete = true
	s.job.b.dryRun = true
	file := s.writeToTempFile()
	key := filepath.Join(s.uuid, file)
	s.NoError(s.bucket.Put(file, key))

	s.NoError(s.job.Error())
	s.job.remoteFile = s3.Key{Key: key}
	s.job.localPath = ".DOES-NOT-EXIST"

	s.True(s.job.withDelete)
	s.True(s.job.b.dryRun)
	s.job.Run(context.TODO())
	s.NoError(s.job.Error())

	exists, err := s.bucket.Exists(key)
	s.NoError(err)
	s.False(exists)
}

func (s *SyncToSuite) TestSyncWithoutDeleteAndDryRunDoesNotRemovesRemoteFile() {
	s.job.withDelete = false
	file := s.writeToTempFile()
	key := filepath.Join(s.uuid, file)
	s.NoError(s.bucket.Put(file, key))

	s.NoError(s.job.Error())
	s.job.remoteFile = s3.Key{Key: key}
	s.job.localPath = file

	s.job.b.dryRun = true
	s.False(s.job.withDelete)
	s.True(s.job.b.dryRun)
	s.job.Run(context.TODO())
	s.NoError(s.job.Error())

	exists, err := s.bucket.Exists(key)
	s.NoError(err)
	s.True(exists)
}

func (s *SyncToSuite) TestSyncPutsNewFileIntoBucket() {
	file := s.writeToTempFile()
	key := filepath.Join(s.uuid, file)

	s.job.remoteFile = s3.Key{Key: key}
	s.job.localPath = file

	// run this a couple of times to make sure
	for i := 0; i < 2; i++ {
		s.False(s.job.b.dryRun)
		s.NoError(s.job.Error())
		s.job.Run(context.TODO())
		s.NoError(s.job.Error())
	}

	exists, err := s.bucket.Exists(key)
	s.NoError(err)
	s.True(exists)

	s.Len(s.bucket.contents(s.uuid), 1)
}

func (s *SyncToSuite) TestSyncUploadsNewFileOverWrites() {
	var err error
	id := uuid.Must(uuid.NewV4())
	remoteFn := id.String() + "-repeated-upload"

	for i := 0; i < 5; i++ {
		// generates a new file each time
		file := s.writeToTempFile()
		targetFn := filepath.Join(s.uuid, remoteFn)

		s.job.remoteFile = s3.Key{Key: targetFn, ETag: "000"}
		s.job.localPath = file

		s.False(s.job.b.dryRun)
		err = s.job.Error()
		if !s.NoError(err) {
			fmt.Println(err)
		}
		s.job.Run(context.TODO())
		err = s.job.Error()
		if !s.NoError(err) {
			fmt.Println(err)
		}

		s.Len(s.bucket.contents(s.uuid), 1)
	}
}
