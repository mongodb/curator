package operations

import (
	"testing"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/level"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"
)

func init() {
	grip.SetName("curator.operations.test")

	sender := grip.GetSender()
	lvl := sender.Level()
	lvl.Threshold = level.Warning
	grip.Alert(sender.SetLevel(lvl))
}

// CommandsSuite provide a group of tests of the entry points and
// command wrappers for command-line interface to curator.
type CommandsSuite struct {
	suite.Suite
}

func TestCommandSuite(t *testing.T) {
	suite.Run(t, new(CommandsSuite))
}

func (s *CommandsSuite) TestHelloWorldOperationViaDirectCall() {
	ctx := cli.NewContext(nil, nil, nil)
	s.Require().NoError(cli.HandleAction(HelloWorld().Action, ctx))
}

func (s *CommandsSuite) TestHelloCommandObjectAttributes() {
	cmd := HelloWorld()
	s.Equal(cmd.Name, "hello")
	s.Contains(cmd.Aliases, "hi")
	s.Contains(cmd.Aliases, "hello-world")
	s.Len(cmd.Flags, 0)
}

func (s *CommandsSuite) TestHelloWorldFunctionReturnsHelloWorldString() {
	s.Equal(helloWorld(), "hello world!")
}
