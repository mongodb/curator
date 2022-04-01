package greenbay

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/rest"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

// Client provides all of the core greenbay operations by
// doing requests against a remote service, provided by GreenbayService.
type Client struct {
	Conf   *Configuration
	Output *OutputOptions
	client *rest.QueueClient
	Tests  []string
	Suites []string
}

// NewClient constructs a greenbay client, with a signature that is
// roughly analogous to the NewApp constructor used for local greenbay
// operations.
func NewClient(confPath, host string, port int, outFn, format string, quiet bool, suite, tests []string) (*Client, error) {
	out, err := NewOutputOptions(outFn, format, quiet)
	if err != nil {
		return nil, errors.Wrap(err, "generating output definition")
	}

	conf, err := ReadConfig(confPath)
	if err != nil {
		return nil, errors.Wrap(err, "parsing config file")
	}

	c, err := rest.NewQueueClient(host, port, "")
	if err != nil {
		return nil, errors.Wrap(err, "constructing amboy rest client")
	}

	client := &Client{
		client: c,
		Conf:   conf,
		Output: out,
		Tests:  tests,
		Suites: suite,
	}

	return client, nil
}

// Run executes all greenbay checks on the remote system. Currently
// waits for all checks to complete for the default (20 seconds,) or
// until the context expires. Control the timeout using the context.
func (c *Client) Run(ctx context.Context) error {
	if c.Conf == nil || c.Output == nil {
		return errors.New("GreenbayApp is not correctly constructed:" +
			"system and output configuration must be specified.")
	}

	// make sure we clean up after ourselves if we return early
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// begin "real" work
	start := time.Now()
	catcher := grip.NewCatcher()
	ids := []string{}

	for check := range c.Conf.GetAllTests(c.Tests, c.Suites) {
		if check.Err != nil {
			catcher.Add(check.Err)
			continue
		}
		j := check.Job.(Checker)
		j.SetID(fmt.Sprintf("%s-%d-%s", j.ID(), time.Now().Unix(), uuid.New().String()))
		id, err := c.client.SubmitJob(ctx, j)
		if err != nil {
			catcher.Add(err)
			continue
		}
		ids = append(ids, id)
	}

	if catcher.HasErrors() {
		return errors.Wrap(catcher.Resolve(), "collecting and submitting jobs")
	}

	// TODO: make the Ids avalible in the app to retry waiting.

	// wait all will block for 20 seconds (by default, we could
	// timeout the context if needed to have control over that);
	// our main risk is that another client will submit jobs at
	// the same time, and we'll end up waiting for each other's
	// jobs. We could become much more clever here.
	//
	// However, the assumption is that 20 seconds will be enough
	// given that these jobs should complete faster than that.
	if !c.client.WaitAll(ctx) {
		return errors.Errorf("timed out waiting for %d jobs to complete", len(ids))
	}

	jobs, errs := c.getJobFromIds(ctx, ids)

	catcher.Add(c.Output.CollectResults(jobs))
	catcher.Add(<-errs)

	grip.Noticef("checks complete in [num=%d, runtime=%s] ", len(ids), time.Since(start))
	return catcher.Resolve()
}

func (c *Client) getJobFromIds(ctx context.Context, ids []string) (<-chan amboy.Job, <-chan error) {
	errs := make(chan error)
	jobs := make(chan amboy.Job, len(ids))
	catcher := grip.NewCatcher()

	for _, id := range ids {
		j, err := c.client.FetchJob(ctx, id)
		if err != nil {
			catcher.Add(err)
		}
		jobs <- j
	}
	close(jobs)
	if catcher.HasErrors() {
		errs <- catcher.Resolve()
	}
	close(errs)
	return jobs, errs
}
