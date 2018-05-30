package check

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type GroupRequirementsSuite struct {
	name    string
	req     GroupRequirements
	require *require.Assertions
	suite.Suite
}

func TestGroupRequirementsSuite(t *testing.T) {
	suite.Run(t, new(GroupRequirementsSuite))
}

func (s *GroupRequirementsSuite) SetupSuite() {
	s.require = s.Require()
}

func (s *GroupRequirementsSuite) SetupTest() {
	s.name = "foo"
	s.req = GroupRequirements{Name: s.name}
}

func (s *GroupRequirementsSuite) TestInitializedObjectHasExpectedState() {
	s.Equal(s.req, GroupRequirements{Name: s.name})
	s.False(s.req.Any)
	s.False(s.req.All)
	s.False(s.req.One)
	s.False(s.req.None)
	s.Equal(s.name, s.req.Name)
}

func (s *GroupRequirementsSuite) TestObjectWithEmptyNameIsNotValid() {
	r := GroupRequirements{}
	s.Equal("", r.Name)
	s.Error(r.Validate())
}

func (s *GroupRequirementsSuite) TestValidationOfEmptyObjectIsFalse() {
	s.Equal(s.req, GroupRequirements{Name: s.name})
	s.Error(s.req.Validate())
}

func (s *GroupRequirementsSuite) TestGetResultsOfInvalidObjectReturnsError() {
	s.Error(s.req.Validate())
	result, err := s.req.GetResults(0, 0)
	s.Error(err)
	s.False(result)
}

func (s *GroupRequirementsSuite) TestReqObjectValidatesIfOnlyOneOptionIsSet() {
	s.Error(s.req.Validate())
	s.req.Any = true
	s.NoError(s.req.Validate())
	s.req = GroupRequirements{Name: s.name}

	s.Error(s.req.Validate())
	s.req.All = true
	s.NoError(s.req.Validate())
	s.req = GroupRequirements{Name: s.name}

	s.Error(s.req.Validate())
	s.req.One = true
	s.NoError(s.req.Validate())
	s.req = GroupRequirements{Name: s.name}

	s.Error(s.req.Validate())
	s.req.None = true
	s.NoError(s.req.Validate())
	s.req = GroupRequirements{Name: s.name}
}

func (s *GroupRequirementsSuite) TestReqObjectIsNotValidIfMultipleOptionsAreSet() {
	s.req = GroupRequirements{Name: s.name, One: true}
	s.NoError(s.req.Validate())
	s.req.Any = true
	s.Error(s.req.Validate())

	s.req = GroupRequirements{Name: s.name, One: true}
	s.NoError(s.req.Validate())
	s.req.All = true
	s.Error(s.req.Validate())

	s.req = GroupRequirements{Name: s.name, One: true}
	s.NoError(s.req.Validate())
	s.req.None = true
	s.Error(s.req.Validate())
}

func (s *GroupRequirementsSuite) TestResultsWithNoPassesAndOneFailure() {
	var result bool
	var err error

	passes := 0
	failures := 1

	s.req = GroupRequirements{Name: s.name, All: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)

	s.req = GroupRequirements{Name: s.name, One: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)

	s.req = GroupRequirements{Name: s.name, Any: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)

	s.req = GroupRequirements{Name: s.name, None: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.True(result)
}

func (s *GroupRequirementsSuite) TestResultsWithNoPassesAndManyFailures() {
	var result bool
	var err error

	passes := 0
	failures := 5

	s.req = GroupRequirements{Name: s.name, All: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)

	s.req = GroupRequirements{Name: s.name, One: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)

	s.req = GroupRequirements{Name: s.name, Any: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)

	s.req = GroupRequirements{Name: s.name, None: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.True(result)
}

func (s *GroupRequirementsSuite) TestResultsWithOnePassAndNoFailures() {
	var result bool
	var err error

	passes := 1
	failures := 0

	s.req = GroupRequirements{Name: s.name, All: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.True(result)

	s.req = GroupRequirements{Name: s.name, One: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.True(result)

	s.req = GroupRequirements{Name: s.name, Any: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.True(result)

	s.req = GroupRequirements{Name: s.name, None: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)
}

func (s *GroupRequirementsSuite) TestResultsWithManyPassesAndNoFailures() {
	var result bool
	var err error

	passes := 5
	failures := 0

	s.req = GroupRequirements{Name: s.name, All: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.True(result)

	s.req = GroupRequirements{Name: s.name, One: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)

	s.req = GroupRequirements{Name: s.name, Any: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.True(result)

	s.req = GroupRequirements{Name: s.name, None: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)
}

func (s *GroupRequirementsSuite) TestResultsWithManyPassesAndFailures() {
	var result bool
	var err error

	passes := 5
	failures := 5

	s.req = GroupRequirements{Name: s.name, All: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)

	s.req = GroupRequirements{Name: s.name, One: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)

	s.req = GroupRequirements{Name: s.name, Any: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.True(result)

	s.req = GroupRequirements{Name: s.name, None: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)
}

func (s *GroupRequirementsSuite) TestResultsWithOnePassAndFailure() {
	var result bool
	var err error

	passes := 1
	failures := 1

	s.req = GroupRequirements{Name: s.name, All: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)

	s.req = GroupRequirements{Name: s.name, One: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.True(result)

	s.req = GroupRequirements{Name: s.name, Any: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.True(result)

	s.req = GroupRequirements{Name: s.name, None: true}
	result, err = s.req.GetResults(passes, failures)
	s.NoError(err)
	s.False(result)
}
