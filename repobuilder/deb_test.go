package repobuilder

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip"
)

type DebRepoSuite struct {
	j       *BuildDEBRepoJob
	require *require.Assertions
	suite.Suite
}

func TestDebRepoSuite(t *testing.T) {
	suite.Run(t, new(DebRepoSuite))
}

func (s *DebRepoSuite) SetupSuite() {
	s.require = s.Require()
}

func (s *DebRepoSuite) SetupTest() {
	s.j = &BuildDEBRepoJob{buildRepoJob()}
	s.require.NotNil(s.j)
}

func (s *DebRepoSuite) TearDownTest() {
	for _, path := range s.j.workingDirs {
		grip.CatchError(os.RemoveAll(path))
	}
}

func (s *DebRepoSuite) TestRpmBuilderImplementsRequiredInternalMethods() {
	s.Implements((*jobImpl)(nil), s.j)
}

func (s *DebRepoSuite) TestConstructedObjectHasExpectedValues() {
	conf, err := GetConfig("config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("debian8", "enterprise")
	s.True(ok)
	s.j, err = NewBuildDEBRepo(conf, repo, "2.8.8", "x86_64", "default", "foo", "bar", "baz")
	s.NoError(err)

	// basic checks to make sure we create the instance
	s.Equal(s.j.Version, s.j.release.String())
	s.Equal([]string{"foo", "bar", "baz"}, s.j.PackagePaths)
	s.False(s.j.DryRun)
}

func (s *DebRepoSuite) TestConstructorReturnsErrorForInvalidVersion() {
	conf, err := GetConfig("config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("debian8", "enterprise")
	s.True(ok)
	_, err = NewBuildDEBRepo(conf, repo, "2.8.8.8", "x86_64", "default", "foo", "bar", "baz")

	s.Error(err)
}

func (s *DebRepoSuite) TestCompletedSetter() {
	conf, err := GetConfig("config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("debian8", "enterprise")
	repo.Bucket = "build-curator-testing"
	s.True(ok)
	s.j, err = NewBuildDEBRepo(conf, repo, "2.8.8", "x86_64", "default")
	s.NoError(err)

	s.False(s.j.DryRun)
	s.j.DryRun = true

	s.False(s.j.Completed())
	s.Equal(s.j.IsComplete, s.j.Completed())

	s.j.Run()
	s.True(s.j.Completed())
	s.Equal(s.j.IsComplete, s.j.Completed())
}
