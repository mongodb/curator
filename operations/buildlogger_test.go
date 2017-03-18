package operations

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/mongodb/grip"
)

func TestBuildLoggerRunCommand(t *testing.T) {
	var err error
	var cmd *exec.Cmd

	assert := assert.New(t)
	logger := grip.NewJournaler("buildlogger.test")

	// error and non-error cases should both work as expected
	err = runCommand(logger, exec.Command("ls"), 100)
	grip.Info(err)
	assert.NoError(err)

	err = runCommand(logger, exec.Command("dfkjdexit", "0"), 100)
	grip.Info(err)
	assert.Error(err)

	// want to make sure that we exercise the path with too-small buffers.
	err = runCommand(logger, exec.Command("ls"), 1)
	grip.Info(err)
	assert.NoError(err)

	err = runCommand(logger, exec.Command("dfkjdexit", "0"), 1)
	grip.Info(err)
	assert.Error(err)

	// runCommand should error if the output streams are pre set.
	cmd = &exec.Cmd{}
	cmd.Stderr = os.Stderr
	err = runCommand(logger, cmd, 1)
	grip.Info(err)
	assert.Error(err)

	cmd = &exec.Cmd{}
	cmd.Stdout = os.Stdout
	err = runCommand(logger, cmd, 1)
	grip.Info(err)
	assert.Error(err)
}
