package repobuilder

import (
	"os"
	"testing"

	"github.com/mongodb/amboy/registry"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip"
)

type RpmRepoSuite struct {
	j       *BuildRPMRepoJob
	require *require.Assertions
	suite.Suite
}

func TestRpmRepoSuite(t *testing.T) {
	suite.Run(t, new(RpmRepoSuite))
}

func (s *RpmRepoSuite) SetupSuite() {
	s.require = s.Require()
}

func (s *RpmRepoSuite) SetupTest() {
	s.j = &BuildRPMRepoJob{buildRepoJob()}
	s.require.NotNil(s.j)
}

func (s *RpmRepoSuite) TearDownTest() {
	for _, path := range s.j.workingDirs {
		grip.CatchError(os.RemoveAll(path))
	}
}

func (s *RpmRepoSuite) TestRpmBuilderImplementsRequiredInternalMethods() {
	s.Implements((*jobImpl)(nil), s.j)
}

func (s *RpmRepoSuite) TestRegisteredFactoryProducesEqualValues() {
	factory, err := registry.GetJobFactory("build-rpm-repo")
	if s.NoError(err) {
		s.Equal(s.j, factory())
	}
}

func (s *RpmRepoSuite) TestConstructedObjectHasExpectedValues() {
	conf, err := GetConfig("config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("rhel5", "enterprise")
	s.True(ok)
	s.j, err = NewBuildRPMRepo(conf, repo, "2.8.8", "x86_64", "default", "foo", "bar", "baz")
	s.NoError(err)

	// basic checks to make sure we create the instance
	s.Equal(s.j.Version, s.j.release.String())
	s.Equal([]string{"foo", "bar", "baz"}, s.j.PackagePaths)
	s.False(s.j.DryRun)
}

func (s *RpmRepoSuite) TestConstructorReturnsErrorForInvalidVersion() {
	conf, err := GetConfig("config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("rhel5", "enterprise")
	s.True(ok)
	_, err = NewBuildRPMRepo(conf, repo, "2.8.8.8", "x86_64", "default", "foo", "bar", "baz")

	s.Error(err)
}

func (s *RpmRepoSuite) TestCompletedSetter() {
	conf, err := GetConfig("config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("rhel5", "enterprise")
	repo.Bucket = "build-curator-testing"
	s.True(ok)
	s.j, err = NewBuildRPMRepo(conf, repo, "2.8.8", "x86_64", "default")
	s.NoError(err)

	s.False(s.j.DryRun)
	s.j.DryRun = true

	s.False(s.j.Completed())
	s.Equal(s.j.IsComplete, s.j.Completed())

	s.j.Run()
	s.True(s.j.Completed())
	s.Equal(s.j.IsComplete, s.j.Completed())
}
