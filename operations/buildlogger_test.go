package operations

import (
	"os"
	"os/exec"
	"testing"

	"github.com/mongodb/grip"
	"github.com/stretchr/testify/assert"
)

func TestBuildLoggerRunCommand(t *testing.T) {
	var err error
	var cmd *exec.Cmd

	assert := assert.New(t)
	clogger := cmdLogger{
		logger: grip.NewJournaler("buildlogger.test"),
	}

	// error and non-error cases should both work as expected
	err = clogger.runCommand(exec.Command("ls"))
	grip.Info(err)
	assert.NoError(err)

	err = clogger.runCommand(exec.Command("dfkjdexit", "0"))
	grip.Info(err)
	assert.Error(err)

	// want to make sure that we exercise the path with too-small buffers.
	err = clogger.runCommand(exec.Command("ls"))
	grip.Info(err)
	assert.NoError(err)

	err = clogger.runCommand(exec.Command("dfkjdexit", "0"))
	grip.Info(err)
	assert.Error(err)

	// runCommand should error if the output streams are pre set.
	cmd = &exec.Cmd{}
	cmd.Stderr = os.Stderr
	err = clogger.runCommand(cmd)
	grip.Info(err)
	assert.Error(err)

	cmd = &exec.Cmd{}
	cmd.Stdout = os.Stdout
	err = clogger.runCommand(cmd)
	grip.Info(err)
	assert.Error(err)
}
