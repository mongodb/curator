/*
Package greenbay provides the core greenbay application
functionality, distinct from user interfaces.

The core greenbay test execution code is available here to support
better testing and alternate interfaces. Currently the only interface
is a command line interface, but we could wrap this functionality in a
web service to support easier integration with monitoring tools or
other health-check services.

The core functionality of the application is in the Application
structure which stores application and facilitates the integration of
output production, test running, and test configuration.
*/
package greenbay

import (
	"context"
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/queue"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

// Application encapsulates the execution of a greenbay run. You can
// construct the object, either with NewApplication(), or by building a
// Application structure yourself.
type Application struct {
	Output     *OutputOptions
	Conf       *Configuration
	NumWorkers int
	Tests      []string
	Suites     []string
}

// NewApplication configures the greenbay application and manages the
// construction of the main config object as well as the output
// configuration structure. Returns an error if there are problems
// constructing either the main config or the output
// configuration objects.
func NewApplication(confPath, outFn, format string, quiet bool, jobs int, suite, tests []string) (*Application, error) {
	out, err := NewOutputOptions(outFn, format, quiet)
	if err != nil {
		return nil, errors.Wrap(err, "generating output definition")
	}

	conf, err := ReadConfig(confPath)
	if err != nil {
		return nil, errors.Wrap(err, "parsing config file")
	}

	app := &Application{
		Conf:       conf,
		Output:     out,
		NumWorkers: jobs,
		Tests:      tests,
		Suites:     suite,
	}

	return app, nil
}

// Run executes all tasks defined in the application, and produces
// results as described by the output configuration. Returns an error
// if any test failed and/or if there were any problems with test
// execution.
func (a *Application) Run(ctx context.Context) error {
	if a.Conf == nil || a.Output == nil {
		return errors.New("application is not correctly constructed:" +
			"system and output configuration must be specified.")
	}

	// make sure we clean up after ourselves if we return early
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	q := queue.NewLocalLimitedSize(a.NumWorkers, 2048)

	if err := q.Start(ctx); err != nil {
		return errors.Wrap(err, "starting workers")
	}

	// begin "real" work
	start := time.Now()
	catcher := grip.NewCatcher()

	for check := range a.Conf.GetAllTests(a.Tests, a.Suites) {
		if check.Err != nil {
			catcher.Add(check.Err)
			continue
		}
		catcher.Add(q.Put(ctx, check.Job))
	}
	if catcher.HasErrors() {
		return errors.Wrap(catcher.Resolve(), "collecting and submitting jobs")
	}

	stats := q.Stats(ctx)
	grip.Noticef("registered %d jobs, running checks now", stats.Total)
	amboy.WaitInterval(ctx, q, 10*time.Millisecond)

	grip.Noticef("checks complete in [num=%d, runtime=%s] ", stats.Total, time.Since(start))
	if err := a.Output.ProduceResults(ctx, q); err != nil {
		return errors.Wrap(err, "producing job results")
	}

	return nil
}
