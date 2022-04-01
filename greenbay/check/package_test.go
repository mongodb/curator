package check

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageCheckImplementation(t *testing.T) {
	// this case checks the greenbay.Checker "packageInstalled"
	// implementation itself. The determination of weather a
	// package is installed is handled by the packageChecker implementation.

	assert := assert.New(t)
	failer := packageCheckerFactory([]string{"python", "-c", "exit(1)"})
	passer := packageCheckerFactory([]string{"echo", "foo"})
	ctx := context.Background()

	// if the check passes, as expected
	check := &packageInstalled{
		checker:   passer,
		Base:      NewBase("test", 0),
		installed: true,
	}
	check.Run(ctx)
	assert.NoError(check.Error())
	assert.True(check.Output().Passed)

	// when the check passes but we don't expect it to be installed
	check = &packageInstalled{
		checker:   passer,
		Base:      NewBase("test", 0),
		installed: false,
	}
	check.Run(ctx)
	assert.Error(check.Error())
	assert.False(check.Output().Passed)

	// if the check fails and we expect it to.
	check = &packageInstalled{
		checker:   failer,
		Base:      NewBase("test", 0),
		installed: false,
	}

	check.Run(ctx)
	assert.NoError(check.Error())
	assert.True(check.Output().Passed)

	// when the check fails and we don't expect it to be installed
	check = &packageInstalled{
		checker:   failer,
		Base:      NewBase("test", 0),
		installed: true,
	}
	check.Run(ctx)
	assert.Error(check.Error())
	assert.False(check.Output().Passed)
}
