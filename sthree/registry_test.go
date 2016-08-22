package sthree

import (
	"os"
	"testing"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip"
)

// RegistrySuite collects tests of the bucketRegistry, which provides
// a factory for and pool of bucket tracking resources.
type RegistrySuite struct {
	registry *bucketRegistry
	require  *require.Assertions
	suite.Suite
}

func TestRegistrySuite(t *testing.T) {
	suite.Run(t, new(RegistrySuite))
}

func (s *RegistrySuite) SetupSuite() {
	grip.SetName("curator.sthree.registry.suite")
	grip.CatchError(grip.UseNativeLogger())
	s.registry = newBucketRegistry()
	s.require = s.Require()
}

func (s *RegistrySuite) SetupTest() {
	s.registry.m = make(map[string]*Bucket)
	s.registry.c = AWSConnectionConfiguration{}

	for _, b := range buckets.m {
		b.Close()
	}

	s.require.Len(buckets.m, 0)
}

func (s *RegistrySuite) TearDownSuite() {
	for _, b := range s.registry.m {
		b.Close()
	}

	s.require.Len(buckets.m, 0)
	s.require.Len(s.registry.m, 0)
}

func (s *RegistrySuite) TestInitialStateIsEmpty() {
	s.Len(s.registry.m, 0)
	s.Equal(s.registry.c.Region, aws.Region{})

	s.Equal(s.registry.c.Auth.AccessKey, "")
	s.Equal(s.registry.c.Auth.SecretKey, "")
}

func (s *RegistrySuite) TestImpactOfInitializationOperation() {
	s.NoError(os.Setenv("AWS_PROFILE", "default"))
	s.registry.init()

	s.Equal(s.registry.c.Region, aws.USEast)

	// these checks depend on having aws credentials in
	// ~/.aws/credentials or environment variables
	s.NotEqual("", s.registry.c.Auth.AccessKey)
	s.NotEqual("", s.registry.c.Auth.SecretKey)
}

func (s *RegistrySuite) TestSetCredentialChangesInternalValue() {
	s.Equal(s.registry.c.Region, aws.Region{})
	s.registry.init()

	s.Equal(s.registry.c.Region, aws.USEast)

	s.registry.setCredentials(AWSConnectionConfiguration{
		Region: aws.USWest,
		Auth: aws.Auth{
			AccessKey: "foo",
			SecretKey: "bar",
		},
	})

	s.Equal(s.registry.c.Region, aws.USWest)
	s.Equal(s.registry.c.Auth.AccessKey, "foo")
	s.Equal(s.registry.c.Auth.SecretKey, "bar")
}

func (s *RegistrySuite) TestSetCredentialDoesNotNulifyRegion() {
	s.registry.init()

	s.Equal(s.registry.c.Region, aws.USEast)

	new := AWSConnectionConfiguration{
		Auth: aws.Auth{
			AccessKey: "foo",
			SecretKey: "bar",
		},
	}

	s.Equal(new.Region, aws.Region{})
	s.registry.setCredentials(new)

	s.Equal(s.registry.c.Region, aws.USEast)
}

// Test standard bucket getter

func (s *RegistrySuite) TestRegistryShouldFunctionAsBucketFactory() {
	s.registry.init()

	b := s.registry.getBucket("test")
	s.Len(s.registry.m, 1)
	s.Equal(b.name, "test")
	s.Equal(b.credentials, s.registry.c)
	s.Equal(string(b.NewFilePermission), "bucket-owner-full-control")

	second := s.registry.getBucket("test")
	s.Len(s.registry.m, 1)
	s.Equal(b.name, "test")
	s.Exactly(b, second)
}

func (s *RegistrySuite) TestBucketCreationFromExistingBucket() {
	name := "test-recreation"
	s.Len(buckets.m, 0)
	b := buckets.getBucket(name)
	defer b.Close()

	s.Len(buckets.m, 1)
	s.Equal(b.name, name)
	s.Equal(b.NewFilePermission, s3.BucketOwnerFull)

	b.NewFilePermission = s3.PublicRead
	s.Equal(b.NewFilePermission, s3.PublicRead)

	second := b.NewBucket(name + "two")
	defer second.Close()

	s.NoError(second.Open())

	s.Len(buckets.m, 2)
	s.Equal(b.NewFilePermission, second.NewFilePermission)

	two := buckets.getBucket(name + "two")
	defer two.Close()

	s.Exactly(second, two)
	s.NotEqual(b, second)
	s.NotEqual(b, two)
}

func (s *RegistrySuite) TestRemoveBucketFromRegistryOnClose() {
	b := buckets.getBucket("test")

	s.Len(buckets.m, 1)
	s.NoError(b.Open())
	b.Close()
	s.require.Len(s.registry.m, 0)
}

func (s *RegistrySuite) TestRegisterDuplicateNameShouldNotOverwriteExistingBucket() {
	s.Len(buckets.m, 0)
	b := buckets.getBucket("test")
	s.Len(buckets.m, 1)

	dup := buckets.getBucket("test")
	s.Exactly(b, dup)
	s.Equal(dup.credentials.Auth.AccessKey, b.credentials.Auth.AccessKey)

	buckets.registerBucket(&Bucket{name: "test"})
	new := buckets.getBucket("test")
	s.Equal(new, dup)
	s.Equal(new, b)
}

// test bucketGetterWithProfile no AWS_PROFILE env var set, so should
// fall back to the same behavior as bucket getting through the normal means.

func (s *RegistrySuite) TestRegistryShouldFunctionAsBucketFactoryWithProfileAndEnvUnset() {
	b := buckets.getBucketFromProfile("test", "extra")
	s.Len(buckets.m, 1)
	s.Equal(b.name, "test")
	s.NotEqual(b.credentials, buckets.c)
	s.Equal(string(b.NewFilePermission), "bucket-owner-full-control")

	second := buckets.getBucketFromProfile("test", "extra")
	s.Len(buckets.m, 1)
	s.Equal(b.name, "test")
	s.Exactly(b, second)
}

func (s *RegistrySuite) TestBucketCreationFromExistingBucketWithProfileAndEnvUnset() {
	b := buckets.getBucketFromProfile("test", "extra")
	s.Len(buckets.m, 1)
	s.Equal(b.name, "test")
	s.Equal(b.NewFilePermission, s3.BucketOwnerFull)

	b.NewFilePermission = s3.PublicRead
	second := b.NewBucket("two")
	s.Len(buckets.m, 2)
	s.Equal(b.NewFilePermission, second.NewFilePermission)

	two := buckets.getBucketFromProfile("two", "extra")
	s.Equal(second, two)
	s.NotEqual(b, second)
	s.NotEqual(b, two)
}

func (s *RegistrySuite) TestRemoveBucketFromRegistryOnCloseWithProfileAndEnvUnset() {
	s.Len(buckets.m, 0)
	b := buckets.getBucketFromProfile("test", "extra")

	s.Len(buckets.m, 1)
	b.Close()
	s.Len(buckets.m, 0)
}

func (s *RegistrySuite) TestRegisterDuplicateNameShouldNotOverwriteExistingBucketWithProfileAndEnvUnset() {
	s.Len(buckets.m, 0)
	b := buckets.getBucketFromProfile("test", "extra")
	s.Len(buckets.m, 1)

	dup := buckets.getBucketFromProfile("test", "extra")
	s.Exactly(b, dup)
	s.Equal(dup.credentials.Auth.AccessKey, b.credentials.Auth.AccessKey)

	buckets.registerBucket(&Bucket{name: "test"})
	new := buckets.getBucketFromProfile("test", "extra")
	s.Equal(new, dup)
	s.Equal(new, b)
}

// test bucketGetterWithProfile *with* AWS_PROFILE env var set, Use different crentials

func (s *RegistrySuite) TestGetBucketFromCacheWithProfileSet() {
	bucketName := "build-curator-testing"
	one := GetBucketWithProfile(bucketName, "foo")
	s.NoError(os.Setenv("AWS_PROFILE", "foo"))
	// defer s.NoError(os.Unsetenv("AWS_PROFILE"))

	two := GetBucketWithProfile(bucketName, "foo")
	s.Equal(one.credentials, two.credentials)
}
