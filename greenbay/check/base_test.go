package check

import (
	"errors"
	"strings"
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/job"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BaseCheckSuite struct {
	base    *Base
	require *require.Assertions
	suite.Suite
}

func TestBaseCheckSuite(t *testing.T) {
	suite.Run(t, new(BaseCheckSuite))
}

func (s *BaseCheckSuite) SetupSuite() {
	s.require = s.Require()
}

func (s *BaseCheckSuite) SetupTest() {
	s.base = &Base{Base: &job.Base{}}
}

func (s *BaseCheckSuite) TestInitialValuesOfBaseObject() {
	s.False(s.base.Status().Completed)
	s.False(s.base.WasSuccessful)
	s.NoError(s.base.Error())
}

func (s *BaseCheckSuite) TestAddErrorWithNilObjectDoesNotChangeErrorState() {
	for i := 0; i < 100; i++ {
		s.base.AddError(nil)
		s.NoError(s.base.Error())
		s.False(s.base.HasErrors())
	}
}

func (s *BaseCheckSuite) TestAddErrorsPersistsErrorsInJob() {
	for i := 1; i <= 100; i++ {
		s.base.AddError(errors.New("foo"))
		s.Error(s.base.Error())
		s.True(s.base.HasErrors())
		s.Len(strings.Split(s.base.Error().Error(), "\n"), i)
	}
}

func (s *BaseCheckSuite) TestOutputStructGenertedReflectsStateOfBaseObject() {
	s.base = &Base{
		TestSuites:    []string{"foo", "bar"},
		WasSuccessful: true,
		Message:       "baz",
		Base: &job.Base{
			JobType: amboy.JobType{
				Name:    "base-greenbay-check",
				Version: 42,
			},
		},
	}
	s.base.SetID("foo")

	output := s.base.Output()
	s.Equal("foo", output.Name)
	s.Equal("base-greenbay-check", output.Check)
	s.Equal("foo", output.Suites[0])
	s.Equal("bar", output.Suites[1])
	s.False(output.Completed)
	s.True(output.Passed)
	s.Equal("", output.Error)
	s.Equal("baz", output.Message)
}

func (s *BaseCheckSuite) TestMutableIDMethod() {
	for _, name := range []string{"foo", "bar", "baz", "bot"} {
		s.base.SetID(name)
		s.NotEqual(s.base.Name(), s.base.ID())
		s.Equal(name, s.base.ID())
		s.Equal(s.base.ID(), s.base.Base.Name)
	}
}

func (s *BaseCheckSuite) TestStatMutability() {
	for _, state := range []bool{true, false, false, true, true} {
		s.base.setState(state)
		s.Equal(state, s.base.WasSuccessful)
	}
}

func (s *BaseCheckSuite) TestSetMessageConvertsTypesToString() {
	var mOne interface{} = "foo"

	s.base.setMessage(mOne)
	s.Equal("foo", s.base.Message)

	s.base.setMessage(true)
	s.Equal("true", s.base.Message)

	s.base.setMessage(nil)
	s.Equal("<nil>", s.base.Message)

	s.base.setMessage(100)
	s.Equal("100", s.base.Message)

	s.base.setMessage(112)
	s.Equal("112", s.base.Message)

	s.base.setMessage(errors.New("foo"))
	s.Equal("foo", s.base.Message)

	s.base.setMessage(errors.New("bar"))
	s.Equal("bar", s.base.Message)

	strs := []string{"foo", "bar", "baz"}
	s.base.setMessage([]string{"foo", "bar", "baz"})
	s.Equal(strings.Join(strs, "\n"), s.base.Message)
}

func (s *BaseCheckSuite) TestSetSuitesOverridesExistingSuites() {
	cases := [][]string{
		{},
		{"foo", "bar"},
		{"1", "false"},
		{"greenbay", "kenosha", "jainseville"},
	}

	for _, suites := range cases {
		s.base.SetSuites(suites)
		s.Equal(suites, s.base.Suites())
	}
}
