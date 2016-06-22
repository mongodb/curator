package repobuilder

import (
	"testing"

	"github.com/mongodb/amboy/dependency"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
	job := buildRepoJob()
	s.j = &job
	s.require.NotNil(s.j)
}

func (s *RepoJobSuite) TestIdIsAccessorForNameAttribute() {
	s.Equal(s.j.Name, s.j.ID())
	s.j.Name = "foo"
	s.Equal("foo", s.j.ID())
	s.Equal(s.j.Name, s.j.ID())
}

func (s *RepoJobSuite) TestDependencyAccessorIsCorrect() {
	s.Equal(s.j.D, s.j.Dependency())
	s.Equal(dependency.AlwaysRun, s.j.D.Type().Name)
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
	s.Equal(dependency.AlwaysRun, s.j.D.Type().Name)
	localDep := dependency.NewLocalFileInstance()
	s.NotEqual(localDep.Type().Name, dependency.AlwaysRun)
	s.j.SetDependency(localDep)
	s.Equal(dependency.AlwaysRun, s.j.D.Type().Name)
}
