package check

import (
	"context"
	"strings"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/pkg/errors"
)

func registerProgramChecks() {
	programCheckFactoryFactory := func(name string, c compiler) func() amboy.Job {
		return func() amboy.Job {
			return &programOutputCheck{
				Base:     NewBase(name, 0),
				compiler: c,
			}
		}
	}

	registrar := func(table map[string]compilerFactory) {
		for name, factory := range table {
			name = strings.Replace(name, "compile-", "run-program-", 1)
			registry.AddJobType(name, programCheckFactoryFactory(name, factory()))
		}
	}

	registrar(compilerInterfaceFactoryTable())
	registrar(goCompilerIterfaceFactoryTable())
	registrar(scriptCompilerInterfaceFactoryTable())
}

type programOutputCheck struct {
	Source         string `bson:"source" json:"source" yaml:"source"`
	ExpectedOutput string `bson:"output" json:"output" yaml:"output"`
	*Base          `bson:"metadata" json:"metadata" yaml:"metadata"`
	compiler       compiler
}

func (c *programOutputCheck) Run(_ context.Context) {
	c.startTask()
	defer c.MarkComplete()

	if err := c.compiler.Validate(); err != nil {
		c.setState(false)
		c.AddError(err)
		return
	}

	if c.ExpectedOutput == "" {
		c.setState(false)
		c.AddError(errors.Errorf("expected output for check '%s' can't be empty", c.ID()))
		return
	}

	c.ExpectedOutput = strings.Trim(c.ExpectedOutput, "\r\t\n ")

	output, err := c.compiler.CompileAndRun(c.Source)
	if err != nil {
		c.setState(false)
		c.AddError(err)
		c.setMessage(output)
		return
	}

	if c.ExpectedOutput != output {
		c.setState(false)
		c.AddError(errors.New("expected output does not match actual output"))
		c.setMessage([]string{
			"-------------------- EXPECTED --------------------",
			c.ExpectedOutput,
			"-------------------- ACTUAL --------------------",
			output,
		})
		return
	}
	c.setState(true)
}
