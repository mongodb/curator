//go:build linux || freebsd || solaris || darwin
// +build linux freebsd solaris darwin

package check

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimitCheckFactoryNegativeValue(t *testing.T) {
	assert := assert.New(t)
	checks := []limitValueCheck{
		limitCheckFactory("a", 0, 0),
		limitCheckFactory("b", 0, 10),
		limitCheckFactory("c", 0, 100),
		limitCheckFactory("d", 0, 1000),
		limitCheckFactory("e", 0, 999999),
	}
	for _, c := range checks {
		result, err := c(-1)
		assert.NoError(err)
		assert.True(result)
	}
}
