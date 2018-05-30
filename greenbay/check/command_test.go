package check

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandCheck(t *testing.T) {
	assert := assert.New(t)

	checks := map[string]bool{
		"true":                   true,
		"false":                  false,
		"echo foo":               true,
		"command-does-not-exist": false,
		"sleep 0":                true,
		"sh -c \"true\"":         true,
		"python --version":       true,
		"pythong --version":      false,
		"echo $VALUE":            true,
	}

	envMaps := []map[string]string{
		map[string]string{},
		map[string]string{
			"VALUE": "env var",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, env := range envMaps {
		for _, wd := range []string{"", "../", "./", "../../"} {
			for cmd, expected := range checks {
				check := &shellOperation{
					WorkingDirectory: wd,
					Command:          cmd,
					shouldFail:       false,
					Environment:      env,
					Base:             NewBase("cmd", 0),
				}
				check.Run(ctx)
				output := check.Output()
				assert.True(output.Completed)
				if expected {
					assert.True(output.Passed, wd)
					assert.NoError(check.Error())
				} else {
					if assert.False(output.Passed, cmd) {
						assert.Error(check.Error())
					}

				}
			}

		}
	}
}
