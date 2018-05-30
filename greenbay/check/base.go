/*
Package check provides implementation of check functions or jobs that
are used in system validation.

Base

The base job implements all components of the amboy.Job interface and
all common components of the greenbay.Check interface, including error
handling and job processing. All checks should, typically, compose a
pointer to Base.

For an example of a check that uses Base, see the test job in the
"mock_check_for_test.go" file.
*/
package check

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/dependency"
	"github.com/mongodb/amboy/job"
	"github.com/mongodb/curator/greenbay"
)

// Base is a type that all new checks should compose, and provides an
// implementation of most common amboy.Job and greenbay.Check methods.
type Base struct {
	WasSuccessful bool                `bson:"passed" json:"passed" yaml:"passed"`
	Message       string              `bson:"message" json:"message" yaml:"message"`
	TestSuites    []string            `bson:"suites" json:"suites" yaml:"suites"`
	Timing        greenbay.TimingInfo `bson:"timing" json:"timing" yaml:"timing"`
	*job.Base     `bson:"metadata" json:"metadata" yaml:"metadata"`

	mutex sync.RWMutex
}

// NewBase exists for use in the constructors of individual checks.
func NewBase(checkName string, version int) *Base {
	b := &Base{
		Timing: greenbay.TimingInfo{
			Start: time.Now(),
		},
		Base: &job.Base{
			JobType: amboy.JobType{
				Name:    checkName,
				Version: version,
			},
		},
	}

	b.SetDependency(dependency.NewAlways())

	return b
}

//////////////////////////////////////////////////////////////////////
//
// greenbay.Checker base methods implementation
//
//////////////////////////////////////////////////////////////////////

// Output returns a consistent output format for greenbay.Checks,
// which may be useful for generating common output formats.
func (b *Base) Output() greenbay.CheckOutput {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	out := greenbay.CheckOutput{
		Name:      b.ID(),
		Check:     b.Type().Name,
		Suites:    b.Suites(),
		Completed: b.Status().Completed,
		Passed:    b.WasSuccessful,
		Message:   b.Message,
		Timing: greenbay.TimingInfo{
			Start: b.Timing.Start,
			End:   b.Timing.End,
		},
	}

	if err := b.Error(); err != nil {
		out.Error = err.Error()
	}

	return out
}

func (b *Base) setState(result bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.Timing.End = time.Now()
	b.WasSuccessful = result
}

func (b *Base) getState() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.WasSuccessful
}

func (b *Base) setMessage(m interface{}) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	switch msg := m.(type) {
	case string:
		b.Message = msg
	case []string:
		b.Message = strings.Join(msg, "\n")
	case error:
		b.Message = msg.Error()
	case int:
		b.Message = strconv.Itoa(msg)
	default:
		b.Message = fmt.Sprintf("%+v", msg)
	}
}

// Suites reports which suites the current check belongs to.
func (b *Base) Suites() []string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.TestSuites
}

// SetSuites allows callers, typically the configuration parser, to
// set the suites.
func (b *Base) SetSuites(suites []string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.TestSuites = suites
}

// Name returns the name of the *check* rather than the name of the
// task.
func (b *Base) Name() string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.JobType.Name
}

func (b *Base) startTask() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.Timing.Start = time.Now()
}
