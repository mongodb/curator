package repobuilder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mongodb/amboy/dependency"
	"github.com/mongodb/grip"
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
	s.j = buildRepoJob()
	s.require.NotNil(s.j)
}

func (s *RepoJobSuite) TearDownTest() {
	for _, path := range s.j.workingDirs {
		grip.CatchError(os.RemoveAll(filepath.Join(j.WorkSpace, path)))
	}
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
