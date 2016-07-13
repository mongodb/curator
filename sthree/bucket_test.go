package sthree

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/goamz/goamz/aws"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip"
)

// BucketSuite contains tests of the base bucket interface for
// interacting with files and s3 objects, with a simple
// interface. These tests include both unit tests of our
// implementation and some integration tests. As a result, users
// without access to the "build-test-curator" bucket won't be able to
// run this suite.
type BucketSuite struct {
	b          *Bucket
	bucketName string
	require    *require.Assertions
	uuid       string
	tempDir    string
	suite.Suite
}

func TestBucketSuite(t *testing.T) {
	suite.Run(t, new(BucketSuite))
}

func (s *BucketSuite) SetupSuite() {
	s.require = s.Require()

	// TODO make this overrideable by environment variable.
	s.bucketName = "build-test-curator"

	// save a uuid for this suite to use as a prefix
	id := uuid.NewV4()
	s.uuid = id.String()

	grip.SetName("curator.sthree.bucket.suite")
	grip.CatchError(grip.UseNativeLogger())
	grip.Noticef("running s3 bucket tests, using %s (%s)", s.bucketName, s.uuid)

	tempDir, err := ioutil.TempDir("", s.uuid)
	s.require.NoError(err)
	s.tempDir = tempDir
}

func (s *BucketSuite) SetupTest() {
	s.b = GetBucket(s.bucketName)
}

func (s *BucketSuite) TearDownTest() {
	grip.CatchError(s.b.DeletePrefix(s.uuid))
	buckets.removeBucket(s.b)
}

func (s *BucketSuite) TearDownSuite() {
	b := GetBucket(s.bucketName)
	s.NoError(b.DeletePrefix(s.uuid))
	s.NoError(os.RemoveAll(s.tempDir))
	buckets.removeBucket(b)
}

func (s *BucketSuite) TestAdditionalMimeTypesAndMimeTypeDiscovery() {
	// map of file names to expected types:
	cases := map[string]string{
		"foo.deb":               "application/octet-stream",
		"foo/bar.deb":           "application/octet-stream",
		"foo/bar.tar.deb":       "application/octet-stream",
		"foo.rpm":               "application/x-redhat-package-manager",
		"foo/bar.rpm":           "application/x-redhat-package-manager",
		"foo/bar.tar.rpm":       "application/x-redhat-package-manager",
		"tarball.gz":            "application/x-gzip",
		"foo.tar.gz":            "application/x-gzip",
		"foo.tgz":               "application/x-gzip",
		"prefix/tarball.gz":     "application/x-gzip",
		"prefix/foo.tar.gz":     "application/x-gzip",
		"prefix/foo.tgz":        "application/x-gzip",
		"data.json":             "application/json",
		"prefix/data.feed.json": "application/json",
		"data.yaml":             "text/x-yaml; charset=utf-8",
		"prefix/data.feed.yaml": "text/x-yaml; charset=utf-8",
		"data.yml":              "text/x-yaml; charset=utf-8",
		"prefix/data.feed.yml":  "text/x-yaml; charset=utf-8",
		"foo.txt":               "text/plain; charset=utf-8",
		"prefix/foo.txt":        "text/plain; charset=utf-8",
	}

	for fileName, mimeType := range cases {
		s.Equal(mimeType, getMimeType(fileName))
	}

	// test the default fallback behavior to text/plain
	s.Equal("text/plain", getMimeType("unrecognized.foo"))
	// if there's no ".", then there's no extension
	s.Equal("text/plain", getMimeType("unrecognizedtargz"))
}

func (s *BucketSuite) TestCredentialSetterOverridesCredentialsForASingleBucket() {
	newCreds := AWSConnectionConfiguration{
		Auth: aws.Auth{
			AccessKey: "foo",
			SecretKey: "bar"},
		Region: aws.USWest,
	}

	s.Equal(s.b.credentials.Region, aws.USEast)
	s.b.SetCredentials(newCreds)
	s.Equal(s.b.credentials.Region, aws.USWest)
	s.NotNil(s.b.bucket)

	// having changed the credentials for the bucket named test,
	// means if we get another pointer to this variable it's set
	// (i.e. buckets are shared. )
	copyOfOne := GetBucket(s.bucketName)
	second := GetBucket("test-two")
	s.NotEqual(s.b.credentials.Region, second.credentials.Region)
	s.Equal(s.b.credentials.Region, copyOfOne.credentials.Region)
}

func (s *BucketSuite) TestOpenMethodStartsQueueAndConnections() {
	// make sure that we start out with a non-opened host.
	s.False(s.b.open)

	// abort if opening causes an error
	s.require.NoError(s.b.Open())
	s.True(s.b.queue.Started())

	// confirm that the bucket is open
	s.True(s.b.open)
	s.NotNil(s.b.bucket)
	s.True(s.b.queue.Started())

	// calling open a second time should be a noop and not change
	// any of the properties
	bucketFirst := s.b.bucket
	s.NoError(s.b.Open())
	s.Equal(*bucketFirst, *s.b.bucket)

	// cleanup at the end
	s.b.Close()
	s.False(s.b.open)
}

func (s *BucketSuite) TestContentsAndListProduceIdenticalData() {
	// right now this noops in the code because the "test" bucket
	// doesn't have any data, and we might want to have some
	// additional fixtures that adds data to the bucket.

	s.require.NoError(s.b.Open())
	var prefix string
	content := s.b.contents(prefix)

	var count int

	for bucketItem := range s.b.list(prefix) {
		item, ok := content[bucketItem.Name]

		if s.True(ok, fmt.Sprintf("item %s should exist in bucket %s",
			bucketItem.Name, s.bucketName)) {

			s.Equal(bucketItem.Name, item.Name)
			s.Equal(bucketItem.MD5, item.MD5)
		}

		count++
	}

	s.Len(content, count)
}

func (s *BucketSuite) TestJobNumberIsConfigurableBeforeBucketOpens() {
	s.Equal(s.b.numJobs, runtime.NumCPU()*4)

	for i := 1; i < 25; i++ {
		s.False(s.b.open)
		s.NoError(s.b.SetNumJobs(i))
		s.NoError(s.b.Open())
		s.True(s.b.open)
		s.Equal(i, s.b.queue.Runner().Size())
		s.b.Close()
	}
}

func (s *BucketSuite) TestJobNumberIsNotConfigurableAfterBucketOpens() {
	s.NoError(s.b.Open())
	s.True(s.b.open)

	s.Equal(s.b.numJobs, runtime.NumCPU()*4)
	s.Equal(s.b.numJobs, s.b.queue.Runner().Size())
	s.Error(s.b.SetNumJobs(100))
	s.Equal(s.b.numJobs, s.b.queue.Runner().Size())
}

func (s *BucketSuite) TestPutOptionUploadsFile() {
	local := "bucket.go"
	remote := filepath.Join(s.uuid, local+".one")

	s.NoError(s.b.Open())

	s.NoError(s.b.Put(local, remote))

	contents := s.b.contents(s.uuid)
	_, ok := contents[remote]
	s.True(ok)
}

func (s *BucketSuite) TestGetRetrievesFileIsTheSameAsSourceData() {
	local := "bucket.go"
	remote := filepath.Join(s.uuid, local+".two")

	s.NoError(s.b.Open())

	// get the hash of the files' contents
	originalData, err := ioutil.ReadFile(local)
	s.NoError(err)
	originalHash := md5.Sum(originalData)

	// upload the file to s3
	s.NoError(s.b.Put(local, remote))

	// download the file to a temp location
	copy := filepath.Join(s.tempDir, local)
	s.NoError(s.b.Get(remote, copy))

	// hash the copy
	copyData, err := ioutil.ReadFile(copy)
	s.NoError(err)
	newHash := md5.Sum(copyData)

	// make sure the hashes are equal
	s.Equal(newHash, originalHash)
}

func (s *BucketSuite) TestGetMakesEnclosingDirectories() {
	local := "bucket.go"
	remote := filepath.Join(s.uuid, local+".three")

	s.NoError(s.b.Open())

	// upload the file to s3
	s.NoError(s.b.Put(local, remote))

	// download the file to a temp location, in a directory that doesn't exist
	copy := filepath.Join(s.tempDir, "newDir", local)

	// directroy doesn't exist
	_, err := os.Stat(filepath.Dir(copy))
	s.True(os.IsNotExist(err))

	s.NoError(s.b.Get(remote, copy))

	_, err = os.Stat(copy)
	s.False(os.IsNotExist(err))
}

func (s *BucketSuite) TestPutReturnsErrorForFilesThatDoNotExist() {
	s.Error(s.b.Put("foo/bar.go", filepath.Join(s.uuid, "foo/baz.go")))
}

func (s *BucketSuite) TestDeleteOperationRemovesPathFromBucket() {
	local := "bucket.go"
	remote := filepath.Join(s.uuid, local+".four")

	s.NoError(s.b.Open())

	// upload the file to s3
	s.NoError(s.b.Put(local, remote))

	contents := s.b.contents(s.uuid)
	_, ok := contents[remote]
	s.True(ok)

	s.NoError(s.b.Delete(remote))

	contents = s.b.contents(s.uuid)
	_, ok = contents[remote]
	s.False(ok)
}

func (s *BucketSuite) TestDeleteManyOperationRemovesManyPathsFromBucket() {
	local := "bucket.go"
	s.NoError(s.b.Open())
	prefix := uuid.NewV4().String()

	s.Len(s.b.contents(filepath.Join(s.uuid, prefix)), 0)

	var toDelete []string

	for i := 0; i < 20; i++ {
		name := filepath.Join(s.uuid, prefix, local+".five."+strconv.Itoa(i))
		s.NoError(s.b.Put(local, name))
		toDelete = append(toDelete, name)
	}

	s.Len(s.b.contents(filepath.Join(s.uuid, prefix)), 20)

	s.NoError(s.b.DeleteMany(toDelete...))

	s.Len(s.b.contents(filepath.Join(s.uuid, prefix)), 0)
}

func numFilesInPath(path string, includeDirs bool) (int, error) {
	numFiles := 0
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if includeDirs {
			numFiles++
		} else if !info.IsDir() {
			numFiles++
		}

		return nil
	})

	return numFiles, err
}

func (s *BucketSuite) TestSyncToUploadsNewFilesWithoutError() {
	pwd, err := os.Getwd()
	s.NoError(err)

	remotePrefix := filepath.Join(s.uuid, "sync-to-one")

	s.NoError(s.b.Open())

	s.Len(s.b.contents(remotePrefix), 0)

	err = s.b.SyncTo(pwd, remotePrefix)
	s.NoError(err)

	num, err := numFilesInPath(pwd, false)
	s.NoError(err)
	s.Len(s.b.contents(remotePrefix), num)
}

func (s *BucketSuite) TestSyncFromDownloadsFiles() {
	pwd, err := os.Getwd()
	s.NoError(err)
	s.NoError(s.b.Open())

	remotePrefix := filepath.Join(s.uuid, "sync-from-one")

	s.Len(s.b.contents(remotePrefix), 0)

	// populate bucket.
	err = s.b.SyncTo(pwd, remotePrefix)
	s.NoError(err)
	numFiles, err := numFilesInPath(pwd, false)
	s.NoError(err)

	// make sure we uploaded files
	s.Len(s.b.contents(remotePrefix), numFiles)
	s.True(numFiles > 0)

	local := filepath.Join(s.tempDir, "sync-from-one")
	err = s.b.SyncFrom(local, remotePrefix)
	s.NoError(err)

	// make sure we pulled the right number of files out of the
	// bucket.
	num, err := numFilesInPath(local, false)
	s.NoError(err)
	s.Equal(numFiles, num)
}

func (s *BucketSuite) TestSyncFromTestWhenFilesHaveChanged() {
	pwd, err := os.Getwd()
	s.NoError(err)
	s.NoError(s.b.Open())

	remotePrefix := filepath.Join(s.uuid, "sync-round-trip")
	err = s.b.SyncTo(pwd, remotePrefix)
	s.NoError(err)

	local := filepath.Join(s.tempDir, "sync-round-trip")
	err = s.b.SyncFrom(local, remotePrefix)
	s.NoError(err)

	err = filepath.Walk(local, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		s.NoError(os.Truncate(p, 4))

		return nil
	})
	s.NoError(err)

	err = s.b.SyncFrom(local, remotePrefix)
	s.NoError(err)

	err = filepath.Walk(local, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// above we truncate all the files to 4 bytes,to force
		// files to sync. We then check the size of all the
		// files, and if they're still 4 bytes, then the sync
		// failed.
		if info.Size() == 4 {
			s.Fail(fmt.Sprintf("file=%s was not synced", p))
		}

		return nil
	})

	s.NoError(err)
}
