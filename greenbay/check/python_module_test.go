package check

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestPythonModuleCheckValidator(t *testing.T) {
	assert := assert.New(t)
	check := &pythonModuleVersion{
		Base: NewBase("foo", 0),
	}
	assert.Equal("", check.PythonInterpreter)
	assert.Equal("", check.Relationship)

	// this works, because the implementation uses validate() to
	// set defaults if they're unset.
	assert.NoError(check.validate())
	assert.Equal("python", check.PythonInterpreter)
	assert.Equal("gte", check.Relationship)

	// set everything to valid comparators, resulting in no errors:
	for _, comp := range []string{"gt", "lt", "gte", "gte", "eq"} {
		check.Relationship = ""
		assert.Equal("", check.Relationship)

		check.Relationship = comp
		assert.NoError(check.validate())
		assert.Equal(comp, check.Relationship)
	}

	// now set relationship to something invalid and make sure its invalid
	for _, invalid := range []string{"true", "false", "nil", "0", "1", "neq", "ne", "gth", "lth"} {
		check.Relationship = ""
		assert.Equal("", check.Relationship)

		check.Relationship = invalid
		assert.NoError(check.validate())
	}
}

// These test make sure that "failures" and "success" are reported
// correctly, but they take advantage of the implementation of the
// check itself, which executes a python one-liner via "python -c".

type PythonModuleSuite struct {
	name    string
	check   *pythonModuleVersion
	require *require.Assertions
	suite.Suite
}

func TestPythonModuleSuite(t *testing.T) {
	suite.Run(t, new(PythonModuleSuite))
}

func (s *PythonModuleSuite) SetupSuite() {
	s.name = "python-module"
	s.require = s.Require()
}

func (s *PythonModuleSuite) SetupTest() {
	s.check = &pythonModuleVersion{
		Base:         NewBase(s.name, 0),
		Relationship: "eq",
		Module:       "os",
		Statement:    "'1.0.0'",
		Version:      "1.0.0",
	}
}

func (s *PythonModuleSuite) TestDefaultFixtureProducesPassingResult() {
	// the default fixture as above should pass, this case just
	// confirms that.

	s.check.Run(context.Background())
	s.NoError(s.check.Error())
	output := s.check.Output()
	s.True(output.Passed, output.Error)
	s.True(output.Completed)
}

func (s *PythonModuleSuite) TestReturnsErrorWithInvalidComparator() {
	s.check.Relationship = "neq"
	// this is probably logically invalid but we removed this as an error as part of MAKE-330
	// the current test enforces the new behavior
	s.NoError(s.check.validate())

	s.check.Run(context.Background())
	s.Error(s.check.Error())
	s.False(s.check.Output().Passed)
}

func (s *PythonModuleSuite) TestReturnsErrorIfCommandFails() {
	// triggering a failure by messing up pythonInterpreter

	s.check.PythonInterpreter = "py\thon"
	s.check.Run(context.Background())
	s.Error(s.check.Error())
	s.False(s.check.Output().Passed)
}

func (s *PythonModuleSuite) TestInvalidVersionsReturnedByOutputTriggerErrors() {
	s.check.Statement = "100000"

	s.check.Run(context.Background())
	s.Error(s.check.Error())
	s.False(s.check.Output().Passed)
}

func (s *PythonModuleSuite) TestInvalidExpectedVersionTriggersError() {
	s.check.Version = "100000"

	s.check.Run(context.Background())
	s.Error(s.check.Error())
	s.False(s.check.Output().Passed)
}

func (s *PythonModuleSuite) TestReturnsErrorWhenExpectedValueDoesNotPassComparison() {
	s.check.Statement = "'1.1.1'"
	s.Equal("1.0.0", s.check.Version)
	s.NoError(s.check.validate())
	s.Equal("eq", s.check.Relationship)

	s.check.Run(context.Background())
	s.Error(s.check.Error())
	s.False(s.check.Output().Passed)
}
