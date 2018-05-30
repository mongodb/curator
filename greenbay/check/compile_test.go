package check

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompileCheckFailsForInvalidCompilerInterfaces(t *testing.T) {
	assert := assert.New(t) // nolint
	check := &compileCheck{Base: NewBase("foo", 0)}
	comp := &compileScript{}

	assert.Error(comp.Validate())
	check.compiler = comp

	check.Run(context.Background())
	output := check.Output()

	assert.Error(check.Error())
	assert.False(output.Passed)
	assert.True(output.Completed)
}

func TestCompilerImplementationsAreNotValidInDefaultState(t *testing.T) {
	assert := assert.New(t) // nolint

	comps := []compiler{
		&compileGCC{},
		&compileGolang{},
		&compileScript{},
	}

	for _, c := range comps {
		assert.Error(c.Validate(), fmt.Sprintf("%T: %+v", c, c))
	}

	// the auto constructed variants should be:
	assert.NoError(goCompilerAuto().Validate())
	assert.NoError(gccCompilerAuto().Validate())
}
