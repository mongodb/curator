package check

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func makeShellJob(cmd, wd string, shouldSucceed bool, env map[string]string) *shellOperation {
	return &shellOperation{
		Command:          cmd,
		WorkingDirectory: wd,
		Environment:      env,
		shouldFail:       !shouldSucceed,
		Base:             NewBase("cmd", 0),
	}
}

type CmdGroupSuite struct {
	wd      string
	name    string
	env     map[string]string
	cmds    map[string]bool
	check   *shellGroup
	require *require.Assertions
	suite.Suite
}

func TestCmdGroupEmptyEnvAndCWDSuite(t *testing.T) {
	s := new(CmdGroupSuite)
	s.env = map[string]string{}
	s.wd = "./"
	suite.Run(t, s)
}

func TestCmdGroupWithEnvAndCWDSuite(t *testing.T) {
	s := new(CmdGroupSuite)
	s.env = map[string]string{
		"VALUE": "env var",
	}
	s.wd = "./"
	suite.Run(t, s)
}

func TestCmdGroupEmptyEnvAndOptWDSuite(t *testing.T) {
	s := new(CmdGroupSuite)
	s.env = map[string]string{}
	s.wd = "/opt"
	suite.Run(t, s)
}

func TestCmdGroupWithEnvAndOptWDSuite(t *testing.T) {
	s := new(CmdGroupSuite)
	s.env = map[string]string{
		"VALUE": "env var",
	}
	s.wd = "/opt"
	suite.Run(t, s)
}

func (s *CmdGroupSuite) SetupSuite() {
	s.name = "cmd-group"
	s.require = s.Require()
	s.cmds = map[string]bool{
		"true":              true,
		"false":             false,
		"exit 0":            true,
		"exit 1":            false,
		"python --version":  true,
		"pythong --version": false,
		"echo $VALUE":       true,
	}
}

func (s *CmdGroupSuite) SetupTest() {
	s.check = &shellGroup{
		Base: NewBase("cmd-group", 0),
	}
}

func (s *CmdGroupSuite) TestWithInvalidGroupDefinition() {
	s.check.Requirements = GroupRequirements{Name: s.name, All: true, None: true}

	s.check.Run(context.Background())
	s.Error(s.check.Error())
	output := s.check.Output()
	s.False(output.Passed)
}

func (s *CmdGroupSuite) TestWithAllRequiredSucceed() {
	s.check.Requirements = GroupRequirements{Name: s.name, All: true}
	for cmd, willSucceed := range s.cmds {
		s.check.Commands = append(s.check.Commands,
			makeShellJob(cmd, s.wd, willSucceed, s.env))
	}

	s.check.Run(context.Background())
	s.NoError(s.check.Error())
	output := s.check.Output()
	s.True(output.Passed)
}

func (s *CmdGroupSuite) TestWithAnyRequiredSucceed() {
	s.check.Requirements = GroupRequirements{Name: s.name, Any: true}

	for cmd, willSucceed := range s.cmds {
		s.check.Commands = append(s.check.Commands,
			makeShellJob(cmd, s.wd, willSucceed, s.env))
	}

	s.check.Run(context.Background())
	s.NoError(s.check.Error())
	output := s.check.Output()
	s.True(output.Passed)
}

func (s *CmdGroupSuite) TestWithOneRequiredSucceed() {
	s.check.Requirements = GroupRequirements{Name: s.name, One: true}
	var successfulTasks int

	for cmd, willSucceed := range s.cmds {
		if willSucceed {
			successfulTasks++
			if successfulTasks > 1 {
				continue
			}
		}

		s.check.Commands = append(s.check.Commands,
			makeShellJob(cmd, s.wd, true, s.env))
	}

	s.check.Run(context.Background())
	s.NoError(s.check.Error())
	output := s.check.Output()
	s.True(output.Passed)
}

func (s *CmdGroupSuite) TestWithNoneRequiredSucceed() {
	s.check.Requirements = GroupRequirements{Name: s.name, None: true}

	for cmd, willSucceed := range s.cmds {
		s.check.Commands = append(s.check.Commands,
			makeShellJob(cmd, s.wd, !willSucceed, s.env))
	}

	s.check.Run(context.Background())
	s.NoError(s.check.Error())
	output := s.check.Output()
	s.True(output.Passed)
}

func (s *CmdGroupSuite) TestWithAllRequiredSucceedToFail() {
	s.check.Requirements = GroupRequirements{Name: s.name, All: true}
	for cmd := range s.cmds {
		s.check.Commands = append(s.check.Commands,
			makeShellJob(cmd, s.wd, true, s.env))
	}

	s.check.Run(context.Background())
	s.Error(s.check.Error())
	output := s.check.Output()
	s.False(output.Passed)
}

func (s *CmdGroupSuite) TestWithAnyRequiredSucceedToFail() {
	s.check.Requirements = GroupRequirements{Name: s.name, Any: true}
	for cmd, willSucceed := range s.cmds {
		if willSucceed {
			continue
		}
		s.check.Commands = append(s.check.Commands,
			makeShellJob(cmd, s.wd, true, s.env))
	}

	s.check.Run(context.Background())
	s.Error(s.check.Error())
	output := s.check.Output()
	s.False(output.Passed)
}

func (s *CmdGroupSuite) TestWithOneRequiredSucceedToFail() {
	s.check.Requirements = GroupRequirements{Name: s.name, One: true}

	for cmd, willSucceed := range s.cmds {
		s.check.Commands = append(s.check.Commands,
			makeShellJob(cmd, s.wd, willSucceed, s.env))
	}

	s.check.Run(context.Background())
	s.Error(s.check.Error())
	output := s.check.Output()
	s.False(output.Passed)
}

func (s *CmdGroupSuite) TestWithNoneRequiredSucceedToFail() {
	s.check.Requirements = GroupRequirements{Name: s.name, None: true}

	for cmd, willSucceed := range s.cmds {
		s.check.Commands = append(s.check.Commands,
			makeShellJob(cmd, s.wd, willSucceed, s.env))
	}

	s.check.Run(context.Background())
	s.Error(s.check.Error())
	output := s.check.Output()
	s.False(output.Passed)
}
