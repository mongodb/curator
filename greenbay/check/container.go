package check

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/pkg/errors"
)

func init() {
	name := "lxc-containers-configured"
	registry.AddJobType(name, func() amboy.Job {
		return &containerCheck{
			Base:      NewBase(name, 0),
			container: lxcCheck{},
		}
	})
}

// Internal interface for checking if a container is running and if it
// has the right programs installed. Separate interface so that we can
// inject fake methods for testing, and easily add support for docker/chroots
// or other container systems.

type containerChecker interface {
	hostIsAccessible(string) error
	hostHasPrograms(string, []string) []string
}

// This implementation is copied almost directly from a legacy
// implementation of greenbay.
type lxcCheck struct{}

func (l lxcCheck) hostIsAccessible(host string) error {
	out, err := exec.Command("sudo", "lxc-wait", "-n", host, "-s", "RUNNING", "-t", "0").CombinedOutput()
	if err != nil {
		return errors.Errorf("lxc host is not running. [host='%s', error='%+v', output='%s']",
			host, string(out), err.Error())
	}

	start := time.Now()
	for i := 0; i < 20; i++ {
		err := exec.Command("ssh", "-o", "ConnectTimeout=20", "-o", "ConnectionAttempts=20", host, "hostname").Run()
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		return nil
	}

	return errors.Errorf("lxc host %s were not reachable after %s",
		host, time.Since(start).String())
}

func (l lxcCheck) hostHasPrograms(host string, programs []string) []string {
	var msgs []string

	for _, program := range programs {
		err := exec.Command("ssh", host, "which", program).Run()
		if err != nil {
			msgs = append(msgs,
				fmt.Sprintf("lxc host is missing program [host='%s', program='%s', error='%+v']",
					host, program, err))
			continue
		}
	}

	return msgs
}

////////////////////////////////////////////////////////////////////////
//
// Implementation of Check for Running Containers With Programs Running
//
////////////////////////////////////////////////////////////////////////

type containerCheck struct {
	Hostnames []string `bson:"hostnnames" json:"hostnnames" yaml:"hostnnames"`
	Programs  []string `bson:"programs" json:"programs" yaml:"programs"`
	*Base     `bson:"metadata" json:"metadata" yaml:"metadata"`
	container containerChecker
}

func (c *containerCheck) validate() error {
	if len(c.Hostnames) == 0 {
		return errors.Errorf("no hostnames configured for %s (%s)",
			c.ID(), c.Name())
	}

	return nil
}

func (c *containerCheck) Run(_ context.Context) {
	var failed bool
	c.startTask()
	defer c.MarkComplete()

	if err := c.validate(); err != nil {
		c.setState(false)
		c.AddError(err)
		return
	}

	var activeHosts int
	var messages []string
	for _, host := range c.Hostnames {
		if err := c.container.hostIsAccessible(host); err != nil {
			c.AddError(err)
			c.setState(false)

			failed = true
			continue
		}
		activeHosts++

		if msg := c.container.hostHasPrograms(host, c.Programs); len(msg) > 0 {
			c.AddError(errors.Errorf("host %s is missing %d programs", host, len(msg)))
			messages = append(messages, msg...)
			c.setState(false)
			failed = true
		}
	}

	if activeHosts != len(c.Hostnames) || len(messages) != 0 {
		c.setMessage(messages)
	}
	if !failed {
		c.setState(true)
	}
}
