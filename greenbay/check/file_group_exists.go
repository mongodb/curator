package check

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

func registerFileGroupChecks() {
	fileGroupFactoryFactory := func(name string, gr GroupRequirements) func() amboy.Job {
		gr.Name = name
		return func() amboy.Job {
			return &fileGroup{
				Base:         NewBase(name, 0),
				Requirements: gr,
			}
		}
	}

	for group, requirements := range groupRequirementRegistry {
		name := fmt.Sprintf("file-group-%s", group)
		registry.AddJobType(name, fileGroupFactoryFactory(name, requirements))
	}
}

type fileGroup struct {
	FileNames    []string          `bson:"file_names" json:"file_names" yaml:"file_names"`
	Requirements GroupRequirements `bson:"requirements" json:"requirements" yaml:"requirements"`
	*Base        `bson:"metadata" json:"metadata" yaml:"metadata"`
}

func (c *fileGroup) Run(_ context.Context) {
	c.startTask()
	defer c.MarkComplete()

	if err := c.Requirements.Validate(); err != nil {
		c.setState(false)
		c.AddError(err)
		return
	}

	if len(c.FileNames) == 0 {
		c.setState(false)
		c.AddError(errors.Errorf("no files specified for '%s' (%s) check",
			c.ID(), c.Name()))
		return
	}

	var extantFiles []string
	var missingFiles []string

	for _, fn := range c.FileNames {
		stat, err := os.Stat(fn)
		grip.Debugf("file '%s' stat: %+v", fn, stat)

		if os.IsNotExist(err) {
			missingFiles = append(missingFiles, fn)
			continue
		}

		extantFiles = append(extantFiles, fn)
	}

	msg := fmt.Sprintf("'%s' check. %d files exist, %d do not exist. "+
		"[existing=(%s), missing=(%s)]", c.Name(), len(extantFiles), len(missingFiles),
		strings.Join(extantFiles, ", "), strings.Join(missingFiles, ", "))
	grip.Debug(msg)

	result, err := c.Requirements.GetResults(len(extantFiles), len(missingFiles))
	c.setState(result)
	c.AddError(err)

	if !result {
		c.setMessage(msg)
		c.AddError(errors.New("group of files do not satisfy check requirements"))
	}
}
