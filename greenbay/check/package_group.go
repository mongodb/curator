package check

import (
	"context"
	"fmt"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/pkg/errors"
)

func registerPackageGroupChecks() {
	packageGroupFactoryFactory := func(name string, gr GroupRequirements, checker packageChecker) func() amboy.Job {
		return func() amboy.Job {
			gr.Name = name
			return &packageGroup{
				Base:         NewBase(name, 0),
				Requirements: gr,
				checker:      checker,
			}
		}
	}

	for pkg, checker := range packageCheckerRegistry {
		for group, requirements := range groupRequirementRegistry {
			name := fmt.Sprintf("%s-group-%s", pkg, group)
			registry.AddJobType(name, packageGroupFactoryFactory(name, requirements, checker))
		}
	}
}

type packageGroup struct {
	Packages     []string          `bson:"packages" json:"packages" yaml:"packages"`
	Requirements GroupRequirements `bson:"requirements" json:"requirements" yaml:"requirements"`
	*Base        `bson:"metadata" json:"metadata" yaml:"metadata"`
	checker      packageChecker
}

func (c *packageGroup) Run(_ context.Context) {
	c.startTask()
	defer c.MarkComplete()

	if err := c.Requirements.Validate(); err != nil {
		c.setState(false)
		c.AddError(err)
		return
	}

	if len(c.Packages) == 0 {
		c.setState(false)
		c.AddError(errors.Errorf("no packages for '%s' (%s) check",
			c.ID(), c.Name()))
		return
	}

	var installed []string
	var missing []string
	var messages []string

	for _, pkg := range c.Packages {
		exists, msg := c.checker(pkg)
		if exists {
			installed = append(installed, pkg)
		} else {
			missing = append(missing, pkg)
		}
		messages = append(messages, msg)
	}

	result, err := c.Requirements.GetResults(len(installed), len(missing))
	c.setState(result)
	c.AddError(err)

	if !result {
		c.setMessage(messages)
		c.AddError(errors.New("group of packages does not satisfy check requirements"))
	}
}
