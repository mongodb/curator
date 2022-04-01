package check

import (
	"context"
	"runtime"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/pkg/errors"
)

func registerSystemLimitChecks() {
	limitCheckFactoryFactory := func(name string, cfunc limitValueCheck) func() amboy.Job {
		return func() amboy.Job {
			return &limitCheck{
				Base:      NewBase(name, 0),
				limitTest: cfunc,
			}
		}
	}

	for name, checkFunc := range limitValueCheckTable() {
		registry.AddJobType(name, limitCheckFactoryFactory(name, checkFunc))
	}
}

type limitValueCheck func(int) (bool, error)

type limitCheck struct {
	Value     int `bson:"value" json:"value" yaml:"value"`
	*Base     `bson:"metadata" json:"metadata" yaml:"metadata"`
	limitTest limitValueCheck
}

func (c *limitCheck) Run(_ context.Context) {
	c.startTask()
	defer c.MarkComplete()

	result, err := c.limitTest(c.Value)
	if err != nil {
		c.setState(false)
		c.AddError(err)
		return
	}

	if !result {
		c.setState(false)
		c.AddError(errors.Errorf("limit in check '%s' is incorrect", c.ID()))
		return
	}

	c.setState(true)
}

func undefinedLimitCheckFactory(name string) limitValueCheck {
	return func(_ int) (bool, error) {
		return false, errors.Errorf("limit check '%s' is not defined on this platform (%s)",
			name, runtime.GOOS)
	}
}
