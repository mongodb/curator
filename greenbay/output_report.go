package greenbay

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/mongodb/amboy"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

// Report implements a single machine-parsable json format for results, for use in the rest API
type Report struct {
	hasErrors   bool
	skipPassing bool
	results     map[string]*CheckOutput
}

// SkipPassing causes the reporter to skip all passing tests in the report.
func (r *Report) SkipPassing() { r.skipPassing = true }

// Populate generates output, based on the content (via the Results()
// method) of an amboy.Queue instance. All jobs processed by that
// queue must also implement the greenbay.Checker interface.
func (r *Report) Populate(jobs <-chan amboy.Job) error {
	r.results = make(map[string]*CheckOutput)
	catcher := grip.NewCatcher()

	for check := range jobsToCheck(r.skipPassing, jobs) {
		if check.err != nil {
			r.hasErrors = true
			catcher.Add(check.err)
			continue
		}
		r.results[check.output.Name] = &check.output
		if !check.output.Passed {
			r.hasErrors = true
		}
	}

	return catcher.Resolve()
}

// ToFile writes a JSON report to the specified file.
func (r *Report) ToFile(fn string) error {
	data, err := r.getJSON()
	if err != nil {
		return errors.WithStack(err)
	}

	if err := ioutil.WriteFile(fn, data, 0644); err != nil {
		return errors.Wrapf(err, "problem writing output to %s", fn)
	}

	if r.hasErrors {
		return errors.New("tests failed")
	}

	return nil
}

// Print generates the JSON report and writes the output via a
// fmt.Println operation.
func (r *Report) Print() error {
	data, err := r.getJSON()
	if err != nil {
		return errors.WithStack(err)
	}

	fmt.Println(string(data))

	if r.hasErrors {
		return errors.New("tests failed")
	}

	return nil
}

func (r *Report) getJSON() ([]byte, error) {
	data, err := json.MarshalIndent(r.results, "", "   ")
	if err != nil {
		return []byte{}, errors.Wrap(err, "problem marhsaling results")
	}
	return data, nil
}
