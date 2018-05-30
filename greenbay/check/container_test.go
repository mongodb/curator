package check

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type passingContainer struct{}

func (c passingContainer) hostIsAccessible(h string) error               { return nil }
func (c passingContainer) hostHasPrograms(h string, p []string) []string { return []string{} }

type failingContainer struct{}

func (c failingContainer) hostIsAccessible(h string) error               { return errors.New("e") }
func (c failingContainer) hostHasPrograms(h string, p []string) []string { return []string{} }

type missingPrograms struct{}

func (c missingPrograms) hostIsAccessible(h string) error               { return nil }
func (c missingPrograms) hostHasPrograms(h string, p []string) []string { return []string{"e"} }

type ContainerCheckSuite struct {
	name    string
	check   *containerCheck
	require *require.Assertions
	suite.Suite
}

func TestContainerCheckSuite(t *testing.T) {
	suite.Run(t, new(ContainerCheckSuite))
}

func (s *ContainerCheckSuite) SetupSuite() {
	s.name = "host-check"
	s.require = s.Require()
}

func (s *ContainerCheckSuite) SetupTest() {
	s.check = &containerCheck{
		Hostnames: []string{"localhost"},
		Base:      NewBase(s.name, 0),
		container: passingContainer{},
	}
}

func (s *ContainerCheckSuite) TestWithOutHostsDefinedCheckFails() {
	s.check.Hostnames = []string{}
	s.Len(s.check.Hostnames, 0)
	s.Error(s.check.validate())
	s.check.Run(context.Background())
	s.Error(s.check.Error())
	s.False(s.check.Output().Passed)
}

func (s *ContainerCheckSuite) TestDefaultStateOfFixtureValidates() {
	s.Len(s.check.Hostnames, 1)
	s.NoError(s.check.validate())

}

func (s *ContainerCheckSuite) TestWithPassingHostMockTestsSucceed() {
	// this passes because we've mocked out the host end.
	s.IsType(passingContainer{}, s.check.container)
	s.check.Run(context.Background())
	s.NoError(s.check.Error())
	s.True(s.check.Output().Passed)
}

func (s *ContainerCheckSuite) TestWithFailingHostMockTestsFail() {
	// this passes because we've mocked out the host end.
	s.check.container = failingContainer{}
	s.check.Run(context.Background())
	s.Error(s.check.Error())
	s.False(s.check.Output().Passed)
}

func (s *ContainerCheckSuite) TestWithMissingProgramMockTestsFail() {
	// this passes because we've mocked out the host end.
	s.check.container = missingPrograms{}
	s.check.Run(context.Background())
	s.Error(s.check.Error())
	s.False(s.check.Output().Passed)
}
