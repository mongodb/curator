package repobuilder

import (
	"os"
	"testing"

	"github.com/mongodb/amboy/dependency"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip"
)

type RepoJobSuite struct {
	j       *Job
	require *require.Assertions
	suite.Suite
}

func TestRepoJobSuite(t *testing.T) {
	suite.Run(t, new(RepoJobSuite))
}

func (s *RepoJobSuite) SetupSuite() {
	s.require = s.Require()
}

func (s *RepoJobSuite) SetupTest() {
	s.j = buildRepoJob()
	s.require.NotNil(s.j)
}

func (s *RepoJobSuite) TearDownTest() {
	for _, path := range s.j.workingDirs {
		grip.CatchError(os.RemoveAll(path))
	}
}

func (s *RepoJobSuite) TestIdIsAccessorForNameAttribute() {
	s.Equal(s.j.Name, s.j.ID())
	s.j.Name = "foo"
	s.Equal("foo", s.j.ID())
	s.Equal(s.j.Name, s.j.ID())
}

func (s *RepoJobSuite) TestDependencyAccessorIsCorrect() {
	s.Equal(dependency.AlwaysRun, s.j.Dependency().Type().Name)
}

func (s *RepoJobSuite) TestSetDependencyAcceptsDifferentAlwaysRunInstances() {
	originalDep := s.j.Dependency()
	newDep := dependency.NewAlways()
	s.True(originalDep != newDep)

	s.j.SetDependency(newDep)
	s.True(originalDep != s.j.Dependency())
	s.Exactly(newDep, s.j.Dependency())
}

func (s *RepoJobSuite) TestSetDependencyRejectsNonAlwaysRunDependencies() {
	s.Equal(dependency.AlwaysRun, s.j.Dependency().Type().Name)
	localDep := dependency.NewLocalFileInstance()
	s.NotEqual(localDep.Type().Name, dependency.AlwaysRun)
	s.j.SetDependency(localDep)
	s.Equal(dependency.AlwaysRun, s.j.Dependency().Type().Name)
}
