package check

import (
	"context"
	"fmt"
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/grip"
)

// MockCheck implements a Mock check which is useful for for testing,
// and for demonstrating the requirements for implementing a novel
// greenbay.Check instance.
type MockCheck struct {
	hasRun     bool
	shouldFail bool
	*Base
}

func init() {
	// this registers mock-check in the amboy job registry, the
	// name of which is used directly in the greenbay config file,
	// and allows us to construct an amboy Job (i.e. work unit,)
	// from YAML. Every check must do this.
	name := "mock-check"
	registry.AddJobType(name, func() amboy.Job {
		return &MockCheck{
			Base: NewBase(name, 0), // (name, version)
		}
	})
}

// Run implements the body of the check, and is the only component of
// the amboy.Job or greenbay.Check interfaces not implemented by
// Base. These functions must: mark the job complete with the
// c.markComplete() method (even if the job fails,) set the job state
// with c.setState(), to report success or failure, and optionally add
// errors with the c.addError(<error>) or messages with the
// c.setMessage(<message>).
func (c *MockCheck) Run(_ context.Context) {
	// starts/restarts the timer for the task.
	c.startTask()

	// mark the job complete tells greenbay (amboy) that work on
	// the check is complete. This should always run, hence the
	// defer, to prevent rerunning tests.
	defer c.MarkComplete()

	// tasks *must* set the state of the check. Unless you set
	// this to a "true" value, greenbay will report the test as
	// failed.
	c.setState(!c.shouldFail)

	m := fmt.Sprintf("ran task %s, at %s (shouldFail=%t)",
		c.ID(), time.Now(), c.shouldFail)

	grip.Info(m)

	// this operation sets "m" as the message, which is included
	// in the test output. Checks can, and should add this
	// information, but it is not a requirement.
	c.setMessage(m)

	c.hasRun = true
}
