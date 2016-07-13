package sthree

import (
	"testing"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip"
)

// RegistrySuite collects tests of the bucketRegistry, which provides
// a factory for and pool of bucket tracking resources.
type RegistrySuite struct {
	registry *bucketRegistry
	suite.Suite
}

func TestVersionSuite(t *testing.T) {
	suite.Run(t, new(RegistrySuite))
}

func (s *RegistrySuite) SetupSuite() {
	grip.SetName("curator.sthree.registry.suite")
	grip.CatchError(grip.UseNativeLogger())
	s.registry = newBucketRegistry()
}

func (s *RegistrySuite) SetupTest() {
	s.registry.m = make(map[string]*Bucket)
	s.registry.c = AWSConnectionConfiguration{}
}

func (s *RegistrySuite) TestInitialStateIsEmpty() {
	s.Len(s.registry.m, 0)
	s.Equal(s.registry.c.Region, aws.Region{})
	s.Equal(s.registry.c.Auth.AccessKey, "")
	s.Equal(s.registry.c.Auth.SecretKey, "")
}

func (s *RegistrySuite) TestImpactOfInitializationOperation() {
	s.registry.init()

	s.Equal(s.registry.c.Region, aws.USEast)

	// these checks depend on having aws credentials in
	// ~/.aws/credentials or environment variables
	s.NotEqual(s.registry.c.Auth.AccessKey, "")
	s.NotEqual(s.registry.c.Auth.SecretKey, "")
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
	s.registry.init()

	b := s.registry.getBucket("test")
	s.Len(s.registry.m, 1)
	s.Equal(b.name, "test")
	s.Equal(b.NewFilePermission, s3.BucketOwnerFull)

	b.NewFilePermission = s3.PublicRead
	second := b.NewBucket("two")
	s.Len(s.registry.m, 2)
	s.Equal(b.NewFilePermission, second.NewFilePermission)

	two := s.registry.getBucket("two")
	s.Exactly(second, two)
	s.NotEqual(b, second)
	s.NotEqual(b, two)
}

func (s *RegistrySuite) TestRemoveBucketFromRegistryOnClose() {
	b := s.registry.getBucket("test")

	s.Len(s.registry.m, 1)
	b.Close()
	s.Len(s.registry.m, 0)
}

func (s *RegistrySuite) TestRegisterDuplicateNameShouldOverwriteExistingBucket() {
	s.Len(s.registry.m, 0)
	b := s.registry.getBucket("test")
	s.Len(s.registry.m, 1)

	b.credentials.Auth.AccessKey = "foo"

	dup := s.registry.getBucket("test")
	s.Exactly(b, dup)
	s.Equal(dup.credentials.Auth.AccessKey, b.credentials.Auth.AccessKey)

	s.registry.registerBucket(&Bucket{name: "test"})
	new := s.registry.getBucket("test")
	s.NotEqual(new, dup)
	s.NotEqual(new, b)
}
