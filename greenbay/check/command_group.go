package check

import (
	"context"
	"fmt"
	"strings"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/curator/greenbay"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

func registerCommandGroupChecks() {
	commandGroupFactoryFactory := func(name string, gr GroupRequirements) func() amboy.Job {
		gr.Name = name
		return func() amboy.Job {
			return &shellGroup{
				Base:         NewBase(name, 0),
				Requirements: gr,
			}
		}
	}

	for group, requirements := range groupRequirementRegistry {
		name := fmt.Sprintf("command-group-%s", group)
		registry.AddJobType(name, commandGroupFactoryFactory(name, requirements))
	}
}

type shellGroup struct {
	Commands     []*shellOperation `bson:"commands" json:"commands" yaml:"commands"`
	Requirements GroupRequirements `bson:"requirements" json:"requirements" yaml:"requirements"`
	*Base        `bson:"metadata" json:"metadata" yaml:"metadata"`
}

func (c *shellGroup) Run(ctx context.Context) {
	c.startTask()
	defer c.MarkComplete()

	if err := c.Requirements.Validate(); err != nil {
		c.setState(false)
		c.AddError(err)
		return
	}

	if len(c.Commands) == 0 {
		c.setState(false)
		c.AddError(errors.Errorf("no files specified for '%s' (%s) check",
			c.ID(), c.Name()))
		return
	}

	var success []*greenbay.CheckOutput
	var failure []*greenbay.CheckOutput

	for idx, cmd := range c.Commands {
		if cmd.Base == nil {
			cmd.Base = NewBase(fmt.Sprintf("%s-%d", c.ID(), idx), c.Type().Version)
		}

		cmd.Run(ctx)

		result := cmd.Output()
		if result.Passed {
			success = append(success, &result)
		} else {
			failure = append(failure, &result)
		}
	}

	result, err := c.Requirements.GetResults(len(success), len(failure))
	c.setState(result)
	c.AddError(err)
	grip.Debugf("task '%s' received result %t, with %d successes and %d failures",
		c.ID(), result, len(success), len(failure))

	if !result {
		var output []string
		var errs []string
		var printableResults []*greenbay.CheckOutput

		if c.Requirements.None {
			printableResults = success
		} else if c.Requirements.Any || c.Requirements.One {
			printableResults = success
			printableResults = append(printableResults, failure...)
		} else {
			printableResults = failure
		}

		for _, cmd := range printableResults {
			if cmd.Message != "" {
				output = append(output, cmd.Message)
			}

			if cmd.Error != "" {
				errs = append(errs, cmd.Error)
			}
		}

		c.setMessage(output)
		c.AddError(errors.New(strings.Join(errs, "\n")))
	}
}
