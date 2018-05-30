package check

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProgramCheckImplementation(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	// the passing case where output matches.
	check := &programOutputCheck{
		Base:           NewBase("test", 0),
		compiler:       &compileScript{bin: "/usr/bin/python"},
		Source:         "print('hello world')",
		ExpectedOutput: "hello world",
	}

	check.Run(ctx)
	err := check.Error()
	assert.NoError(err, fmt.Sprintf("%+v", err))
	output := check.Output()
	assert.True(output.Passed)
	assert.True(output.Completed)

	// the failing case where output is mismatched
	check = &programOutputCheck{
		Base:           NewBase("test", 0),
		compiler:       &compileScript{bin: "/usr/bin/python"},
		Source:         "print('hello world')",
		ExpectedOutput: "hello world???!!!",
	}

	check.Run(ctx)
	assert.Error(check.Error())
	output = check.Output()
	assert.False(output.Passed)
	assert.True(output.Completed)

	// the failing case where the command fails
	check = &programOutputCheck{
		Base:           NewBase("test", 0),
		compiler:       &compileScript{bin: "/usr/bin/python"},
		Source:         "exit(1)",
		ExpectedOutput: "hello world???!!!",
	}

	check.Run(ctx)
	assert.Error(check.Error())
	output = check.Output()
	assert.False(output.Passed)
	assert.True(output.Completed)

}
