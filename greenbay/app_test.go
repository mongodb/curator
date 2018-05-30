package greenbay

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

// Test cases:

type AppSuite struct {
	app *Application
	suite.Suite
}

func TestAppSuite(t *testing.T) {
	suite.Run(t, new(AppSuite))
}

func (s *AppSuite) SetupTest() {
	s.app = &Application{}
}

func (s *AppSuite) TestRunFailsWithUninitailizedConfAndOrOutput() {
	ctx := context.Background()
	s.Nil(s.app.Conf)
	s.Nil(s.app.Output)
	s.Error(s.app.Run(ctx))

	conf := &Configuration{}
	s.NotNil(conf)
	s.app.Conf = conf
	s.NotNil(s.app.Conf)
	s.Nil(s.app.Output)
	s.Error(s.app.Run(ctx))

	s.app.Conf = nil

	out := &OutputOptions{}
	s.NotNil(out)
	s.app.Output = out
	s.NotNil(s.app.Output)
	s.Nil(s.app.Conf)
	s.Error(s.app.Run(ctx))
}

func (s *AppSuite) TestConsturctorFailsIfConfPathDoesNotExist() {
	app, err := NewApplication("DOES-NOT-EXIST", "", "gotest", true, 3, []string{}, []string{})
	s.Error(err)
	s.Nil(app)
}

func (s *AppSuite) TestConsturctorFailsWithEmptyConfPath() {
	app, err := NewApplication("", "", "gotest", true, 3, []string{}, []string{})
	s.Error(err)
	s.Nil(app)
}

func (s *AppSuite) TestConstructorFailsWithInvalidOutputConfigurations() {
	app, err := NewApplication("", "", "DOES-NOT-EXIST", true, 3, []string{}, []string{})
	s.Error(err)
	s.Nil(app)
}

// TODO: add tests that exercise successful runs and dispatch actual
// tests and suites,but to do this we'll want to have better mock
// tests and configs, so holding off on that until MAKE-101
