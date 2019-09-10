package check

import (
	"context"
	"fmt"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/pkg/errors"
)

// this would be an init function but is simply called from the init()
// in init.go to avoid ordering effects.
func registerPackageChecks() {
	packageCheckerFactoryFactory := func(name string, installed bool, checker packageChecker) func() amboy.Job {
		return func() amboy.Job {
			return &packageInstalled{
				checker:   checker,
				Base:      NewBase(name, 0),
				installed: installed,
			}
		}
	}

	var name string

	for pkg, checker := range packageCheckerRegistry {
		name = fmt.Sprintf("%s-installed", pkg)
		registry.AddJobType(name, packageCheckerFactoryFactory(name, true, checker))

		name = fmt.Sprintf("%s-not-installed", pkg)
		registry.AddJobType(name, packageCheckerFactoryFactory(name, false, checker))
	}
}

type packageInstalled struct {
	Package string `bson:"package" json:"package" yaml:"package"`
	*Base   `bson:"metadata" json:"metadata" yaml:"metadata"`

	installed bool
	checker   packageChecker
}

func (c *packageInstalled) Run(_ context.Context) {
	c.startTask()
	defer c.MarkComplete()

	exists, msg := c.checker(c.Package)

	if !c.installed {
		// this is the check for "package isn't installed" tasks
		c.setState(!exists)

		if exists {
			c.setMessage(msg)
			c.AddError(errors.Errorf("package '%s' exists (check=%s) and should not",
				c.Package, c.Name()))
		}
		return
	}

	// check for package is installed.
	c.setState(exists)

	if !exists {
		c.setMessage(msg)
		c.AddError(errors.Errorf("package %s does not exist and should", c.Package))
	}
}
