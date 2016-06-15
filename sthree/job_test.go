package sthree

import (
	"strings"
	"testing"

	"github.com/mongodb/amboy"
	"github.com/stretchr/testify/suite"
)

// BucketJobSuite collects tests of the amboy.Job implementations that
// support syncing files to and from S3.
type BucketJobSuite struct {
	fromJob *syncFromJob
	toJob   *syncToJob
	bucket  *Bucket
	jobs    []amboy.Job
	suite.Suite
}

func TestBucketJobSuite(t *testing.T) {
	suite.Run(t, new(BucketJobSuite))
}

func (s *BucketJobSuite) SetupSuite() {
	s.bucket = GetBucket("build-test-curator")
	s.NoError(s.bucket.Open())
}

func (s *BucketJobSuite) SetupTest() {
	s.toJob = newSyncToJob("local-file-name", &s3Item{Name: "remote-file-name"}, s.bucket)
	s.fromJob = newSyncFromJob("local-file-name", &s3Item{}, s.bucket)
	s.jobs = []amboy.Job{s.toJob, s.fromJob}
}

func (s *BucketJobSuite) TearDownSuite() {
	s.bucket.Close()
}

func (s *BucketJobSuite) TestSyncJobsImplementInterface() {
	job := (*amboy.Job)(nil)

	// test that the objects theme selves are correct
	s.Implements(job, new(syncFromJob))
	s.Implements(job, new(syncToJob))

	// test that the job constructors produce valid implementations
	for _, syncJob := range s.jobs {
		s.Implements(job, syncJob)
	}
}

func (s *BucketJobSuite) TestSyncJobCorrectlyStoresFileNames() {
	s.Equal("local-file-name", s.toJob.localPath)
	s.Equal("local-file-name", s.fromJob.localPath)
	s.Equal("remote-file-name", s.toJob.remoteFile.Name)
}

func (s *BucketJobSuite) TestSyncJobsHaveExpectedJobTypes() {
	s.Equal(0, s.fromJob.Type().Version)
	s.Equal(0, s.toJob.Type().Version)

	s.Equal("s3-sync-from", s.fromJob.Type().Name)
	s.Equal("s3-sync-to", s.toJob.Type().Name)
}

func (s *BucketJobSuite) TestSyncJobsHaveWellFormedName() {
	strings.HasSuffix(s.fromJob.ID(), "sync-from")
	strings.HasSuffix(s.toJob.ID(), "sync-to")
}

func (s *BucketJobSuite) TestSyncJobsAreIncompleteByDefault() {
	for _, job := range s.jobs {
		s.False(job.Completed())
	}
}

func (s *BucketJobSuite) TestMarkCompleteMethodChangesCompleteState() {
	s.False(s.fromJob.Completed())
	s.False(s.toJob.Completed())

	s.fromJob.markComplete()
	s.toJob.markComplete()

	s.True(s.fromJob.Completed())
	s.True(s.toJob.Completed())
}

// TODO write test for run method once we have a test bucket.
