package check

import (
	"runtime"

	"github.com/pkg/errors"
)

type undefinedCompileCheck struct {
	name string
}

func (c *undefinedCompileCheck) Compile(_ string, _ ...string) error { return c.Validate() }
func (c *undefinedCompileCheck) CompileAndRun(_ string, _ ...string) (string, error) {
	err := c.Validate()
	return err.Error(), err
}
func (c *undefinedCompileCheck) Validate() error {
	return errors.Errorf("compiler check '%s' is not defined on this platform (%s)",
		c.name, runtime.GOOS)
}

func undefinedCompileCheckFactory(name string) func() compiler {
	return func() compiler {
		return &undefinedCompileCheck{name}
	}
}
