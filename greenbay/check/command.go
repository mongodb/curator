package check

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

func init() {
	shellOperationFactoryFactory := func(name string, expectedFailure bool) func() amboy.Job {
		return func() amboy.Job {
			return &shellOperation{
				Environment: make(map[string]string),
				shouldFail:  expectedFailure,
				Base:        NewBase(name, 0), // (name, version)
			}
		}
	}

	checks := map[string]bool{
		"shell-operation":       false,
		"shell-operation-error": true,
	}

	for name, shouldFail := range checks {
		registry.AddJobType(name, shellOperationFactoryFactory(name, shouldFail))
	}
}

type shellOperation struct {
	Command          string            `bson:"command" json:"command" yaml:"command"`
	WorkingDirectory string            `bson:"working_directory" json:"working_directory" yaml:"working_directory"`
	Environment      map[string]string `bson:"environment" json:"environment" yaml:"environment"`
	*Base            `bson:"metadata" json:"metadata,omitempty" yaml:"metadata,omitempty"`

	shouldFail bool
}

func (c *shellOperation) Run(_ context.Context) {
	c.startTask()
	defer c.MarkComplete()

	logMsg := []string{fmt.Sprintf("command='%s'", c.Command)}

	// I don't like "sh -c" as a thing, but it parallels the way
	// that Evergreen runs tasks (for now,) and it gets us away
	// from needing to do special shlex parsing, though
	// (https://github.com/google/shlex) seems like a good start.
	cmd := exec.Command("sh", "-c", c.Command)
	if c.WorkingDirectory != "" {
		cmd.Dir = c.WorkingDirectory
		logMsg = append(logMsg, fmt.Sprintf("dir='%s'", c.WorkingDirectory))
	}

	if len(c.Environment) > 0 {
		env := []string{}
		for key, value := range c.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
		logMsg = append(logMsg, fmt.Sprintf("env='%s'", strings.Join(env, " ")))
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		logMsg = append(logMsg, fmt.Sprintf("err='%+v'", err))

		if !c.shouldFail {
			c.setState(false)
			c.AddError(errors.Wrapf(err, "command failed",
				c.ID(), c.Command))
		} else {
			c.setState(true)
		}
	} else if c.shouldFail {
		c.setState(false)
		c.AddError(errors.Errorf("command '%s' succeeded but test expects it to fail",
			c.Command))
	} else {
		c.setState(true)
	}

	grip.Debug(strings.Join(logMsg, ", "))

	if !c.getState() {
		c.setMessage(string(out))
	}
}
