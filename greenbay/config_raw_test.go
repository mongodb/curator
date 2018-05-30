package greenbay

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/job"
	"github.com/mongodb/amboy/registry"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const mockShellCheckName string = "mock-shell-check"

func init() {
	registry.AddJobType(mockShellCheckName, func() amboy.Job {
		return &mockShellCheck{
			mockCheckBase: *newMockCheckBase(mockShellCheckName, 0),
		}
	})
}

// Suite Definition

type RawCheckSuite struct {
	check   *rawTest
	require *require.Assertions
	suite.Suite
}

func TestRawCheckSuite(t *testing.T) {
	suite.Run(t, new(RawCheckSuite))
}

// Mock/Minimal Implementation

type mockShellCheck struct {
	mockCheckBase
	shell *job.ShellJob
}

func (c *mockShellCheck) Run(ctx context.Context) {
	c.shell.Run(ctx)
}

// Fixtures

func (s *RawCheckSuite) SetupSuite() {
	s.require = s.Require()
}

func (s *RawCheckSuite) SetupTest() {
	check := &mockShellCheck{
		shell:         job.NewShellJob("echo foo", ""),
		mockCheckBase: *newMockCheckBase("check-working-shell", 0),
	}

	jsonJob, err := json.Marshal(check)
	s.NoError(err)

	s.check = &rawTest{
		Name:      "check-working-shell",
		Suites:    []string{"one", "two"},
		RawArgs:   jsonJob,
		Operation: mockShellCheckName,
	}
}

// Test Cases

func (s *RawCheckSuite) TestGetCheckFailsWhenCheckDoesNotExist() {
	s.check.Operation = "DOES-NOT-EXIST"

	// fails because no check with this name exists
	c, err := s.check.getChecker()
	s.Error(err)
	s.Nil(c)
}

func (s *RawCheckSuite) TestGetCheckFailsWithJobThatIsNotAlsoChecker() {
	s.check.Operation = "shell"

	// fails because shell job doesn't implement the checker addition.
	c, err := s.check.getChecker()
	s.Error(err)
	s.Nil(c)
}

func (s *RawCheckSuite) TestGetCheckerReturnsValidButNilJobWithCorrectType() {
	c, err := s.check.getChecker()
	s.NoError(err)
	s.NotNil(c)
	s.Equal("", c.ID())
}

func (s *RawCheckSuite) TestResolveCheckPropogatesErrorsFromGetChecker() {
	for _, op := range []string{"DOES-NOT-EXIST", "shell", "group"} {
		s.check.Operation = op
		c, err := s.check.resolveCheck()
		s.Nil(c)
		s.Error(err)
	}
}

func (s *RawCheckSuite) TestResolveCheckErrorsWithMalformedJson() {
	s.check.RawArgs = s.check.RawArgs[4:]

	c, err := s.check.resolveCheck()
	s.Nil(c)
	s.Error(err)
}

func (s *RawCheckSuite) TestResolveCheckReturnsPopulatedChecker() {
	c, err := s.check.resolveCheck()
	s.NoError(err)
	s.NotNil(c)

	s.Equal(s.check.Name, c.Name())
	s.Equal(s.check.Suites, c.Suites())
}
