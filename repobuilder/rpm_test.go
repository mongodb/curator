package repobuilder

import (
	"os"
	"testing"

	"github.com/mongodb/amboy/dependency"
	"github.com/mongodb/amboy/registry"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
	s.j = buildRPMRepoJob()
	s.require.NotNil(s.j)
}

func (s *RpmRepoSuite) TearDownTest() {
	for _, path := range s.j.workingDirs {
		s.j.grip.CatchError(os.RemoveAll(path))
	}
}

func (s *RpmRepoSuite) TestRegisteredFactoryProducesEqualValues() {
	factory, err := registry.GetJobFactory(s.j.Type().Name)
	s.NoError(err)
	s.Equal(s.j, factory())
}

func (s *RpmRepoSuite) TestConstructedObjectHasExpectedValues() {
	conf, err := GetConfig("config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("rhel5", "enterprise")
	s.True(ok)
	s.j, err = NewBuildRPMRepo(repo, "2.8.8", "x86_64", "default", "foo", "bar", "baz")
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
	s.j, err = NewBuildRPMRepo(repo, "2.8.8.8", "x86_64", "default", "foo", "bar", "baz")

	s.Error(err)
}

func (s *RpmRepoSuite) TestIdIsAccessorForNameAttribute() {
	s.Equal(s.j.Name, s.j.ID())
	s.j.Name = "foo"
	s.Equal("foo", s.j.ID())
	s.Equal(s.j.Name, s.j.ID())
}

func (s *RpmRepoSuite) TestCompleetSetter() {
	conf, err := GetConfig("config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("rhel5", "enterprise")
	s.True(ok)
	s.j, err = NewBuildRPMRepo(repo, "2.8.8", "x86_64", "default")
	s.NoError(err)

	s.False(s.j.DryRun)
	s.j.DryRun = true

	s.False(s.j.Completed())
	s.Equal(s.j.IsComplete, s.j.Completed())

	// ignoring the error here because it depends on local
	// createrepo and other factors which are not actually what
	// we're testing here.
	_ = s.j.Run()
	s.True(s.j.Completed())
	s.Equal(s.j.IsComplete, s.j.Completed())
}

func (s *RpmRepoSuite) TestDependencyAccessorIsCorrect() {
	s.Equal(s.j.D, s.j.Dependency())

	s.Equal(dependency.AlwaysRun, s.j.D.Type().Name)
}

func (s *RpmRepoSuite) TestSetDependencyRejectsNonAlwaysRunDependencies() {
	s.Equal(dependency.AlwaysRun, s.j.D.Type().Name)
	localDep := dependency.NewLocalFileInstance()
	s.NotEqual(localDep.Type().Name, dependency.AlwaysRun)
	s.j.SetDependency(localDep)
	s.Equal(dependency.AlwaysRun, s.j.D.Type().Name)
}

func (s *RpmRepoSuite) TestSetDependencyAcceptsDifferentAlwaysRunInstances() {
	originalDep := s.j.Dependency()
	newDep := dependency.NewAlways()
	s.True(originalDep != newDep)

	s.j.SetDependency(newDep)
	s.True(originalDep != s.j.Dependency())
	s.Exactly(newDep, s.j.Dependency())
}
