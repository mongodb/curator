package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/send"
)

// MainSuite is a collection of tests that exercise the main() of the
// program, and associated operations and top-level configuration.
type MainSuite struct {
	suite.Suite
}

func TestMainSuite(t *testing.T) {
	suite.Run(t, new(MainSuite))
}

func (s *MainSuite) TestLoggingSetupUsingDefaultSender() {
	err := loggingSetup("test", "info")
	s.NoError(err)
	s.Equal(grip.Name(), "test")

	nl, err := send.NewNativeLogger("test", level.Alert, level.Info)
	s.NoError(err)
	s.IsType(grip.Sender(), nl)
}

func (s *MainSuite) TestLogSetupWithInvalidLevelDoesNotChangeLevel() {
	// when you specify an invalid level, grip shouldn't change
	// the level.
	s.Equal(grip.ThresholdLevel(), level.Info)
	err := loggingSetup("test", "QUIET")
	s.NoError(err)
	s.Equal(grip.ThresholdLevel(), level.Info)

	// Following case is just to make sure that normal
	// setting still works as expected.
	err = loggingSetup("test", "debug")
	s.NoError(err)
	s.Equal(grip.ThresholdLevel(), level.Debug)
}

func (s *MainSuite) TestAppBuilderFunctionSetsCorrectProperties() {
	app := buildApp()

	s.Equal("curator", app.Name)

	// the exact number will change, but should be >0
	s.NotEqual(len(app.Commands), 0)

	// The app should have some top level flags, and the first
	// flag should be the logging-level configuration.
	s.NotZero(app.Flags)
	s.Equal(app.Flags[0].GetName(), "level")

	// we do logging set up here, so it needs to be set
	s.NotZero(app.Before)
}
