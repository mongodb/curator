package greenbay

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/mongodb/amboy"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

// GoTest defines a ResultsProducer implementation that generates
// output in the format of "go test -v"
type GoTest struct {
	skipPassing bool
	numFailed   int
	buf         *bytes.Buffer
}

// SkipPassing causes the reporter to skip all passing tests in the report.
func (r *GoTest) SkipPassing() { r.skipPassing = true }

// Populate generates output, based on the content (via the Results()
// method) of an amboy.Queue instance. All jobs processed by that
// queue must also implement the greenbay.Checker interface.
func (r *GoTest) Populate(jobs <-chan amboy.Job) error {
	numFailed, err := produceResults(r.buf, jobsToCheck(r.skipPassing, jobs))

	if err != nil {
		return errors.Wrap(err, "problem generating gotest results")
	}

	r.numFailed = numFailed

	return nil
}

// ToFile writes the "go test -v" output to a file.
func (r *GoTest) ToFile(fn string) error {
	if err := ioutil.WriteFile(fn, r.buf.Bytes(), 0644); err != nil {
		return errors.Wrapf(err, "problem writing output to %s", fn)
	}

	if r.numFailed > 0 {
		return errors.Errorf("%d test(s) failed", r.numFailed)
	}

	return nil
}

// Print writes the "go test -v" output to standard output.
func (r *GoTest) Print() error {
	fmt.Println(strings.TrimRight(r.buf.String(), "\n"))

	if r.numFailed > 0 {
		return errors.Errorf("%d test(s) failed", r.numFailed)
	}

	return nil
}

////////////////////////////////////////////////////////////////////////
//
// Implementation of go test output generation
//
////////////////////////////////////////////////////////////////////////

func produceResults(w io.Writer, checks <-chan workUnit) (int, error) {
	catcher := grip.NewCatcher()

	var failedCount int

	for wu := range checks {
		if wu.err != nil {
			catcher.Add(wu.err)
			continue
		}

		if !printTestResult(w, wu.output) {
			failedCount++
		}
	}

	return failedCount, catcher.Resolve()
}

func printTestResult(w io.Writer, check CheckOutput) bool {
	fmt.Fprintln(w, "=== RUN", check.Name)
	if check.Message != "" {
		fmt.Fprintln(w, "    message:", check.Message)
	}

	if check.Error != "" {
		fmt.Fprintln(w, "    error:", check.Error)
	}

	dur := check.Timing.End.Sub(check.Timing.Start)

	if check.Passed {
		fmt.Fprintf(w, "--- PASS: %s (%s)\n", check.Name, dur)
	} else {
		fmt.Fprintf(w, "--- FAIL: %s (%s)\n", check.Name, dur)
	}

	return check.Passed
}
