package check

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PackageGroupSuite struct {
	name    string
	check   *packageGroup
	require *require.Assertions
	suite.Suite
}

func TestPackageGroupSuite(t *testing.T) {
	suite.Run(t, new(PackageGroupSuite))
}

func (s *PackageGroupSuite) SetupSuite() {
	s.require = s.Require()
	s.name = "package-group"
}

func (s *PackageGroupSuite) SetupTest() {
	s.check = &packageGroup{
		Base:         NewBase(s.name, 0),
		Requirements: GroupRequirements{Name: s.name, All: true},
		checker:      packageCheckerFactory([]string{"echo", "PASS"}),
	}
}

func (s *PackageGroupSuite) TestInvalidRequirementsLeadsToFailure() {
	s.check.Requirements.All = true
	s.check.Requirements.None = true
	s.Error(s.check.Requirements.Validate())
	s.check.Run(context.Background())
	s.Error(s.check.Error())
}

func (s *PackageGroupSuite) TestPassingTestWithListOfPackages() {
	// this passes because the check is rigged to always pass
	s.check.Packages = []string{"foo", "bar", "baz"}

	s.check.Run(context.Background())
	s.NoError(s.check.Error())
	s.True(s.check.Output().Passed)
}

func (s *PackageGroupSuite) TestFailsWithNoneRequirements() {
	// this takes advantage of "none", and the rigged checker
	s.check.Requirements = GroupRequirements{Name: s.check.Name(), None: true}
	s.check.Packages = []string{"foo", "bar", "baz"}

	s.check.Run(context.Background())
	s.Error(s.check.Error())
	s.False(s.check.Output().Passed)
}

func (s *PackageGroupSuite) TestFailedCheckerIdentifiesMissingFunctions() {
	s.check.checker = packageCheckerFactory([]string{"python", "-c", "exit(1)"})

	s.check.Packages = []string{"foo", "bar", "baz"}

	s.check.Run(context.Background())
	s.Error(s.check.Error())
	s.False(s.check.Output().Passed)
}
