package greenbay

import (
	"github.com/mongodb/amboy"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/level"
	"github.com/mongodb/grip/message"
	"github.com/mongodb/grip/send"
	"github.com/pkg/errors"
)

// GripOutput provides a ResultsProducer implementation that writes
// the results of a greenbay run to logging using the grip logging
// package.
type GripOutput struct {
	gripOutputData
}

// Populate generates output messages based on a sequence of
// amboy.Jobs. All jobs must also implement the greenbay.Checker
// interface. Returns an error if there are any invalid jobs.
func (r *GripOutput) Populate(jobs <-chan amboy.Job) error {
	catcher := grip.NewCatcher()

	r.useJSONLoggers = false

	for wu := range jobsToCheck(r.skipPassing, jobs) {
		if wu.err != nil {
			catcher.Add(wu.err)
			continue
		}

		dur := wu.output.Timing.End.Sub(wu.output.Timing.Start)
		if wu.output.Passed {
			r.passedMsgs = append(r.passedMsgs,
				message.NewFormatted("PASSED: '%s' [time='%s', msg='%s', error='%s']",
					wu.output.Name, dur, wu.output.Message, wu.output.Error))
		} else {
			r.failedMsgs = append(r.failedMsgs,
				message.NewFormatted("FAILED: '%s' [time='%s', msg='%s', error='%s']",
					wu.output.Name, dur, wu.output.Message, wu.output.Error))
		}
	}

	return catcher.Resolve()
}

// JSONResults provides a structured output JSON format.
type JSONResults struct {
	gripOutputData
}

// Populate generates output messages based on a sequence of
// amboy.Jobs. All jobs must also implement the greenbay.Checker
// interface. Returns an error if there are any invalid jobs.
func (r *JSONResults) Populate(jobs <-chan amboy.Job) error {
	catcher := grip.NewCatcher()
	r.useJSONLoggers = true

	for wu := range jobsToCheck(r.skipPassing, jobs) {
		if wu.err != nil {
			catcher.Add(wu.err)
			continue
		}
		if wu.output.Passed {
			r.passedMsgs = append(r.passedMsgs, &jsonOutput{output: wu.output})
		} else {
			r.failedMsgs = append(r.failedMsgs, &jsonOutput{output: wu.output})
		}
	}
	return catcher.Resolve()
}

type gripOutputData struct {
	useJSONLoggers bool
	skipPassing    bool
	passedMsgs     []message.Composer
	failedMsgs     []message.Composer
}

// SkipPassing causes the reporter to skip all passing tests in the report.
func (r *gripOutputData) SkipPassing() { r.skipPassing = true }

// ToFile logs, to the specified file, the results of the greenbay
// operation. If any tasks failed, this operation returns an error.
func (r *gripOutputData) ToFile(fn string) error {
	var sender send.Sender
	var err error
	logger := grip.NewJournaler("greenbay")

	if r.useJSONLoggers {
		sender, err = send.NewJSONFileLogger("greenbay", fn, send.LevelInfo{Default: level.Info, Threshold: level.Info})
	} else {
		sender, err = send.NewFileLogger("greenbay", fn, send.LevelInfo{Default: level.Info, Threshold: level.Info})
	}

	if err != nil {
		return errors.Wrapf(err, "problem setting up output logger to file '%s'", fn)
	}

	if err := logger.SetSender(sender); err != nil {
		return errors.Wrap(err, "problem configuring logger")
	}

	r.logResults(logger)

	numFailed := len(r.failedMsgs)
	if numFailed > 0 {
		return errors.Errorf("%d test(s) failed", numFailed)
	}

	return nil
}

// Print logs, to standard output, the results of the greenbay
// operation. If any tasks failed, this operation returns an error.
func (r *gripOutputData) Print() error {
	logger := grip.NewJournaler("greenbay")
	var sender send.Sender
	var err error

	if r.useJSONLoggers {
		sender, err = send.NewJSONConsoleLogger("greenbay", send.LevelInfo{Default: level.Info, Threshold: level.Info})
	} else {
		sender, err = send.NewNativeLogger("greenbay", send.LevelInfo{Default: level.Info, Threshold: level.Info})
	}

	if err != nil {
		return errors.Wrap(err, "problem setting up logger")
	}

	if err := logger.SetSender(sender); err != nil {
		return errors.Wrap(err, "problem configuring logger")
	}

	r.logResults(logger)

	numFailed := len(r.failedMsgs)
	if numFailed > 0 {
		return errors.Errorf("%d test(s) failed", numFailed)
	}

	return nil
}

func (r *gripOutputData) logResults(logger grip.Journaler) {
	for _, msg := range r.passedMsgs {
		logger.Notice(msg)
	}

	for _, msg := range r.failedMsgs {
		logger.Alert(msg)
	}
}
