package greenbay

import (
	"context"

	"github.com/mongodb/amboy"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

// OutputOptions represents all operations for output generation, and
// provides methods for accessing and producing results using that
// configuration regardless of underlying output format.
type OutputOptions struct {
	writeFile       bool
	suppressPassing bool
	fn              string
	format          string
}

// NewOutputOptions provides a constructor to generate a valid OutputOptions
// structure. Returns an error if the specified format is not valid or
// registered.
func NewOutputOptions(fn, format string, quiet bool) (*OutputOptions, error) {
	_, exists := GetResultsFactory(format)
	if !exists {
		return nil, errors.Errorf("no results format named '%s' exists", format)
	}

	o := &OutputOptions{}
	o.format = format
	o.suppressPassing = quiet

	if fn != "" {
		o.writeFile = true
		o.fn = fn
	}

	return o, nil
}

// GetResultsProducer returns the ResultsProducer implementation
// specified in the OutputOptions structure, and returns an error if the
// format specified in the structure does not refer to a registered
// type.
func (o *OutputOptions) GetResultsProducer() (ResultsProducer, error) {
	factory, ok := GetResultsFactory(o.format)
	if !ok {
		return nil, errors.Errorf("no results format named '%s' exists", o.format)
	}

	rp := factory()
	if o.suppressPassing {
		rp.SkipPassing()
	}

	return rp, nil
}

// ProduceResults takes an amboy.Queue object and produces results
// according to the options specified in the OutputOptions
// structure. ProduceResults returns an error if any of the tests
// failed in the operation.
func (o *OutputOptions) ProduceResults(ctx context.Context, q amboy.Queue) error {
	if q == nil {
		return errors.New("cannot populate results with a nil queue")
	}

	return o.CollectResults(q.Results(ctx))
}

// CollectResults takes a channel that produces jobs and produces results
// according to the options specified in the OutputOptions
// structure. ProduceResults returns an error if any of the tests
// failed in the operation.
func (o *OutputOptions) CollectResults(jobs <-chan amboy.Job) error {
	rp, err := o.GetResultsProducer()
	if err != nil {
		return errors.Wrap(err, "fetching results producer")
	}

	if err := rp.Populate(jobs); err != nil {
		return errors.Wrap(err, "generating results content")
	}

	// Actually write output to respective streems
	catcher := grip.NewCatcher()

	catcher.Add(rp.Print())

	if o.writeFile {
		catcher.Add(rp.ToFile(o.fn))
	}

	return catcher.Resolve()
}

// Report produces the results of a test run in a parseable map
// structure for programmatic use.
func (o *OutputOptions) Report(jobs <-chan amboy.Job) (map[string]*CheckOutput, error) {
	rp := &Report{}
	output := make(map[string]*CheckOutput)

	if err := rp.Populate(jobs); err != nil {
		return output, errors.Wrap(err, "generating results content")
	}

	for k, v := range rp.results {
		output[k] = v
	}

	return output, nil
}
