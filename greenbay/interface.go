/*
Package greenbay contains the basic type definition used in the
greenbay application.

Overview

The Greenbay application is a system integration testing and
validation tool. It contains definitions of some generic test
functions, such as "does this file exist" and "do these commands
succeed." Specific tests are defined using these functions in a
configuration file, and tests are run on hosts to ensure that the
system is correctly configured.

The Checker interface is a superset of the amboy.Job interface. In
most cases, specific check implementations inculde the check.Base
type, which contains implementations of most methods (except for
Run()) required by this interface.
*/
package greenbay

import (
	"time"

	"github.com/mongodb/amboy"
)

// Checker is a superset of amboy.Job that includes several other
// features unique to Greenbay checks. These methods, in addition to
// all methods in the amboy.Job interface, except for Run(), are
// implemented by the check.Base type, which specific jobs can
// compose.
type Checker interface {
	// SetID modifies the ID reported by the ID() method in the
	// amboy.Job interface.
	SetID(string)

	// Output returns a common output format for all greenbay checks.
	Output() CheckOutput

	// Suites are a list of test suites associated with this check.
	SetSuites([]string)
	Suites() []string

	// Name returns the name of the checker. Use ID(), in the
	// amboy.Job interface to get a unique identifier for the
	// task. This is typically the same as the
	// amboy.Job.Type().Name value.
	Name() string

	// Checker includes the amboy.Job interface.
	amboy.Job
}

// CheckOutput provides a standard report format for tests that
// includes their result status and other metadata that may be useful
// in reporting data to users.
type CheckOutput struct {
	Completed bool       `bson:"completed" json:"completed" yaml:"completed"`
	Passed    bool       `bson:"passed" json:"passed" yaml:"passed"`
	Check     string     `bson:"check_type" json:"check_type" yaml:"check_type"`
	Name      string     `bson:"name" json:"name" yaml:"name"`
	Message   string     `bson:"message,omitempty" json:"message,omitempty" yaml:"message,omitempty"`
	Error     string     `bson:"error,omitempty" json:"error,omitempty" yaml:"error,omitempty"`
	Suites    []string   `bson:"suites" json:"suites" yaml:"suites"`
	Timing    TimingInfo `bson:"timing" json:"timing" yaml:"timing"`
}

// TimingInfo tracks the start and end time for a task.
type TimingInfo struct {
	Start time.Time `bson:"start_time" json:"start_time" yaml:"start_time"`
	End   time.Time `bson:"end_time" json:"end_time" yaml:"end_time"`
}

// Duration returns a time.Duration for the timing information stored
// in the TimingInfo object.
func (t TimingInfo) Duration() time.Duration {
	return t.End.Sub(t.Start)
}
