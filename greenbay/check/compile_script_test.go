package check

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScriptCompilerImplementation(t *testing.T) {
	assert := assert.New(t)
	check := &compileScript{}

	// make sure that the validation and default script
	// interpreter is correct
	assert.Equal("", check.bin)
	assert.Error(check.Validate())

	check.bin = "/usr/bin/python"
	assert.NoError(check.Validate())

	// check that a basic hello world operation succeeds
	err := check.Compile("print('hello world')")
	assert.NoError(err)

	out, err := check.CompileAndRun("print('hello world!')")
	assert.NoError(err)
	assert.Equal("hello world!", out)

	// check that we detect errors that fail
	assert.Error(check.Compile("print('hi'); exit(1)"))

	out, err = check.CompileAndRun("print('hi'); exit(1)")
	assert.Error(err)
	assert.Equal("hi\n", out)
}
