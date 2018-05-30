package check

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/grip"
)

func init() {
	fileExistsFactoryFactory := func(name string, shouldExist bool) func() amboy.Job {
		return func() amboy.Job {
			return &fileExistance{
				ShouldExist: shouldExist,
				Base:        NewBase(name, 0), // (name, version)
			}
		}

	}

	name := "file-exists"
	registry.AddJobType(name, fileExistsFactoryFactory(name, true))

	name = "file-does-not-exist"
	registry.AddJobType(name, fileExistsFactoryFactory(name, false))
}

type fileExistance struct {
	FileName    string `bson:"name" json:"name" yaml:"name"`
	ShouldExist bool   `bson:"should_exist" json:"should_exist" yaml:"should_exist"`
	*Base
}

func (c *fileExistance) Run(_ context.Context) {
	c.startTask()
	defer c.MarkComplete()

	var fileExists bool
	var verb string

	stat, err := os.Stat(c.FileName)
	fileExists = !os.IsNotExist(err)

	c.setState(fileExists == c.ShouldExist)
	if fileExists != c.ShouldExist {
		c.AddError(errors.New("file existence check did not detect expected state"))
	}

	if c.ShouldExist {
		verb = "should"
	} else {
		verb = "should not"
	}

	m := fmt.Sprintf("file '%s' %s exist. stats=%+v", c.FileName, verb, stat)
	grip.Debug(m)
	c.setMessage(m)
}
