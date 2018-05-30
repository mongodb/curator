package greenbay

import "github.com/mongodb/amboy"

// ResultsProducer defines a common interface for generating results
// in different formats.
type ResultsProducer interface {
	// Populate takes an amboy.Queue instance that contains
	// completed greenbay.Checker instances to produce
	// output. Returns an error if the queue contained Job
	// instances that do not implement
	// greenbay.Checker. Implementations are not required to
	// de-duplicate tasks in the case that the Populate() method
	// is called multiple times on
	Populate(<-chan amboy.Job) error

	// ToFile takes a string, for a file name, and writes the
	// results to a file with that name. Returns an error if any
	// of the tasks did not pass. You may call this method
	// multiple times.
	ToFile(string) error

	// Print prints, to standard output, the results in a given
	// format. Returns an error if the results in the format have
	// any failed checks.
	Print() error

	// SkipPassing causes the reporter to skip all passing tests
	// in the report.
	SkipPassing()
}
