package greenbay

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

////////////////////////////////////////////////////////////////////////
//
// Public Interface for results.json output format
//
////////////////////////////////////////////////////////////////////////

// Results defines a ResultsProducer implementation for the Evergreen
// results.json output format.
type Results struct {
	skipPassing bool
	out         *resultsDocument
}

// SkipPassing causes the reporter to skip all passing tests in the report.
func (r *Results) SkipPassing() { r.skipPassing = true }

// Populate generates output, based on the content (via the Results()
// method) of an amboy.Queue instance. All jobs processed by that
// queue must also implement the greenbay.Checker interface.
func (r *Results) Populate(jobs <-chan amboy.Job) error {
	r.out = &resultsDocument{}

	if err := r.out.populate(jobsToCheck(r.skipPassing, jobs)); err != nil {
		return errors.Wrap(err, "constructing results document")
	}

	return nil
}

// ToFile writes results.json output output to the specified file.
func (r *Results) ToFile(fn string) error {
	if err := r.out.writeToFile(fn); err != nil {
		return errors.Wrap(err, "writing results to JSON")
	}

	if r.out.failed {
		return errors.New("tests failed")
	}

	return nil
}

// Print writes, to standard output, the results.json data.
func (r *Results) Print() error {
	if err := r.out.print(); err != nil {
		return errors.Wrap(err, "printing results")
	}

	if r.out.failed {
		return errors.New("tests failed")
	}

	return nil
}

////////////////////////////////////////////////////////////////////////
//
// Implementation for construction and generation of resultsDocument structure.
//
////////////////////////////////////////////////////////////////////////

// type definition and constructors

type resultsDocument struct {
	failed  bool
	Results []*resultsItem `bson:"results" json:"results" yaml:"results"`
}

type resultsItem struct {
	Status  string        `bson:"status" json:"status" yaml:"status"`
	Test    string        `bson:"test_file" json:"test_file" yaml:"test_file"`
	Code    int           `bson:"exit_code" json:"exit_code" yaml:"exit_code"`
	Elapsed time.Duration `bson:"elapsed" json:"elapsed" yaml:"elapsed"`
	Start   time.Time     `bson:"start" json:"start" yaml:"start"`
	End     time.Time     `bson:"end" json:"end" yaml:"end"`
}

// implementation of content generation.

func (r *resultsDocument) populate(checks <-chan workUnit) error {
	catcher := grip.NewCatcher()
	for wu := range checks {
		if wu.err != nil {
			catcher.Add(wu.err)
			continue
		}

		r.addItem(wu.output)
	}

	return catcher.Resolve()
}

func (r *resultsDocument) addItem(check CheckOutput) {
	item := &resultsItem{
		Test:    check.Name,
		Elapsed: check.Timing.Duration(),
		Start:   check.Timing.Start,
		End:     check.Timing.End,
	}
	r.Results = append(r.Results, item)

	item.Status = "pass"

	if !check.Passed {
		item.Status = "fail"
		item.Code = 1
		r.failed = true
	}
}

// output production

func (r *resultsDocument) write(w io.Writer) error {
	out, err := json.MarshalIndent(r, "   ", "   ")
	if err != nil {
		return errors.Wrap(err, "converting results to JSON")
	}

	if _, err = w.Write(out); err != nil {
		return errors.Wrapf(err, "writing results to file '%s' (%T)", w, w)
	}

	// adding a newline, but if it errors we shouldn't care.
	_, _ = w.Write([]byte("\n"))

	return nil
}

func (r *resultsDocument) print() error {
	return r.write(os.Stdout)
}

func (r *resultsDocument) writeToFile(fn string) error {
	buf := &bytes.Buffer{}

	if err := r.write(buf); err != nil {
		return errors.Wrap(err, "extracting JSON to buffer")
	}

	if err := ioutil.WriteFile(fn, buf.Bytes(), 0644); err != nil {
		return errors.Wrapf(err, "writing output to file '%s'", fn)
	}

	grip.Infoln("wrote results document to:", fn)
	return nil
}
