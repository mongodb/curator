package jasper

import (
	"io"
	"io/ioutil"
	"time"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/level"
	"github.com/mongodb/grip/send"
	"github.com/pkg/errors"
)

// OutputOptions provides a common way to define and represent the
// output behavior of a evergreen/subprocess.Command operation.
type OutputOptions struct {
	Output            io.Writer `json:"-"`
	Error             io.Writer `json:"-"`
	SuppressOutput    bool      `json:"suppress_output"`
	SuppressError     bool      `json:"suppress_error"`
	SendOutputToError bool      `json:"redirect_output_to_error"`
	SendErrorToOutput bool      `json:"redirect_error_to_output"`
	Loggers           []Logger  `json:"loggers"`
	outputSender      *send.WriterSender
	errorSender       *send.WriterSender
	outputMulti       io.Writer
	errorMulti        io.Writer
}

type LogType string

const (
	LogBuildloggerV2 = "buildloggerv2"
	LogBuildloggerV3 = "buildloggerv3"
	LogDefault       = "default"
	LogFile          = "file"
	LogInherit       = "inherit"
	LogSplunk        = "splunk"
	LogSumologic     = "sumologic"
	LogInMemory      = "inmemory"
)

const (
	DefaultLogName = "jasper"
)

type LogFormat string

const (
	LogFormatPlain   = "plain"
	LogFormatDefault = "default"
	LogFormatJSON    = "json"
	LogFormatInvalid = "invalid"
)

// By default, logger reads from both standard output and standard error.
type LogOptions struct {
	BufferOptions      BufferOptions             `json:"buffer_options"`
	BuildloggerOptions send.BuildloggerConfig    `json:"buildlogger_options"`
	DefaultPrefix      string                    `json:"default_prefix"`
	FileName           string                    `json:"file_name"`
	Format             LogFormat                 `json:"format"`
	InMemoryCap        int                       `json:"in_memory_cap"`
	SplunkOptions      send.SplunkConnectionInfo `json:"splunk_options"`
	SumoEndpoint       string                    `json:"sumo_endpoint"`
}

func (opts BufferOptions) Validate() error {
	if opts.Buffered && opts.Duration < 0 || opts.MaxSize < 0 {
		return errors.New("cannot have negative buffer duration or size")
	}
	return nil
}

func (opts LogOptions) Validate() error {
	return opts.BufferOptions.Validate()
}

type Logger struct {
	Type    LogType    `json:"log_type"`
	Options LogOptions `json:"log_options"`
	sender  send.Sender
}

func (l Logger) Validate() error {
	catcher := grip.NewBasicCatcher()
	catcher.Add(l.Type.Validate())
	catcher.Add(l.Options.Validate())
	return catcher.Resolve()
}

func (l LogType) Validate() error {
	switch l {
	case LogBuildloggerV2, LogBuildloggerV3, LogDefault, LogFile, LogInherit, LogSplunk, LogSumologic, LogInMemory:
		return nil
	default:
		return errors.New("unknown log type")
	}
}

func (f LogFormat) Validate() error {
	switch f {
	case LogFormatDefault, LogFormatJSON, LogFormatPlain:
		return nil
	default:
		return errors.New("unknown log format")
	}
}

func (f LogFormat) MakeFormatter() (send.MessageFormatter, error) {
	switch f {
	case LogFormatDefault:
		return send.MakeDefaultFormatter(), nil
	case LogFormatPlain:
		return send.MakePlainFormatter(), nil
	case LogFormatJSON:
		return send.MakeJSONFormatter(), nil
	default:
		return nil, errors.New("unknown log format")
	}
}

func (l *Logger) Configure() (send.Sender, error) {
	if l.sender != nil {
		return l.sender, nil
	}

	var sender send.Sender
	var err error

	switch l.Type {
	case LogBuildloggerV2, LogBuildloggerV3:
		if l.Options.BuildloggerOptions.Local == nil {
			l.Options.BuildloggerOptions.Local = send.MakeNative()
		}
		if l.Options.BuildloggerOptions.Local.Name() == "" {
			l.Options.BuildloggerOptions.Local.SetName(DefaultLogName)
		}
		sender, err = send.MakeBuildlogger(DefaultLogName, &l.Options.BuildloggerOptions)
		if err != nil {
			return nil, err
		}
	case LogDefault:
		if l.Options.DefaultPrefix == "" {
			l.Options.DefaultPrefix = DefaultLogName
		}
		sender, err = send.NewNativeLogger(l.Options.DefaultPrefix, send.LevelInfo{Default: level.Trace, Threshold: level.Trace})
		if err != nil {
			return nil, err
		}
	case LogFile:
		sender, err = send.MakePlainFileLogger(l.Options.FileName)
		if err != nil {
			return nil, err
		}
		sender.SetName(DefaultLogName)
	case LogInherit:
		sender = grip.GetSender()
	case LogSplunk:
		if !l.Options.SplunkOptions.Populated() {
			return nil, errors.New("missing connection info for output type splunk")
		}
		sender, err = send.NewSplunkLogger(DefaultLogName, l.Options.SplunkOptions, send.LevelInfo{Default: level.Trace, Threshold: level.Trace})
		if err != nil {
			return nil, err
		}
	case LogSumologic:
		if l.Options.SumoEndpoint == "" {
			return nil, errors.New("missing endpoint for output type sumologic")
		}
		sender, err = send.NewSumo(DefaultLogName, l.Options.SumoEndpoint)
		if err != nil {
			return nil, err
		}
	case LogInMemory:
		if l.Options.InMemoryCap <= 0 {
			return nil, errors.New("invalid inmemory capacity")
		}
		sender, err = send.NewInMemorySender(DefaultLogName, send.LevelInfo{Default: level.Trace, Threshold: level.Trace}, l.Options.InMemoryCap)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown log type")
	}

	formatter, err := l.Options.Format.MakeFormatter()
	if err != nil {
		return nil, err
	}
	if err := sender.SetFormatter(formatter); err != nil {
		return nil, errors.New("failed to set log format")
	}

	if l.Options.BufferOptions.Buffered {
		if l.Options.BufferOptions.Duration < 0 || l.Options.BufferOptions.MaxSize < 0 {
			return nil, errors.New("buffer options cannot be negative")
		}
		sender = send.NewBufferedSender(sender, l.Options.BufferOptions.Duration, l.Options.BufferOptions.MaxSize)
	}

	l.sender = sender

	return l.sender, nil
}

type BufferOptions struct {
	Buffered bool          `json:"buffered"`
	Duration time.Duration `json:"duration"`
	MaxSize  int           `json:"max_size"`
}

func (o OutputOptions) outputIsNull() bool {
	if o.Output == nil {
		return true
	}

	if o.Output == ioutil.Discard {
		return true
	}

	return false
}

func (o OutputOptions) outputLogging() bool {
	return len(o.Loggers) > 0 && !o.SuppressOutput
}

func (o OutputOptions) errorLogging() bool {
	return len(o.Loggers) > 0 && !o.SuppressError
}

func (o OutputOptions) errorIsNull() bool {
	if o.Error == nil {
		return true
	}

	if o.Error == ioutil.Discard {
		return true
	}

	return false
}

func (o OutputOptions) Validate() error {
	catcher := grip.NewBasicCatcher()

	if o.SuppressOutput && (!o.outputIsNull() || o.outputLogging()) {
		catcher.Add(errors.New("cannot suppress output if output is defined"))
	}

	if o.SuppressError && (!o.errorIsNull() || o.errorLogging()) {
		catcher.Add(errors.New("cannot suppress error if error is defined"))
	}

	if o.Error == o.Output && !o.errorIsNull() {
		catcher.Add(errors.New("cannot specify the same value for error and output"))
	}

	if o.SuppressOutput && o.SendOutputToError {
		catcher.Add(errors.New("cannot suppress output and redirect it to error"))
	}

	if o.SuppressError && o.SendErrorToOutput {
		catcher.Add(errors.New("cannot suppress error and redirect it to output"))
	}

	if o.SendOutputToError && o.Error == nil && !o.errorLogging() {
		catcher.Add(errors.New("cannot redirect output to error without a defined error writer"))
	}

	if o.SendErrorToOutput && o.Output == nil && !o.outputLogging() {
		catcher.Add(errors.New("cannot redirect error to output without a defined output writer"))
	}

	if o.SendOutputToError && o.SendErrorToOutput {
		catcher.Add(errors.New("cannot create redirect cycle between output and error"))
	}

	for _, logger := range o.Loggers {
		if err := logger.Validate(); err != nil {
			catcher.Add(err)
		}
		if err := logger.Options.Format.Validate(); err != nil {
			catcher.Add(err)
		}
	}

	return catcher.Resolve()
}

func (o *OutputOptions) GetOutput() (io.Writer, error) {
	if o.SendOutputToError {
		return o.GetError()
	}

	if o.outputIsNull() && !o.outputLogging() {
		return ioutil.Discard, nil
	}

	if o.outputMulti != nil {
		return o.outputMulti, nil
	}

	if o.outputLogging() {
		outSenders := []send.Sender{}

		for i := range o.Loggers {
			sender, err := o.Loggers[i].Configure()
			if err != nil {
				return ioutil.Discard, err
			}
			outSenders = append(outSenders, sender)
		}

		var outMulti send.Sender
		if len(outSenders) == 1 {
			outMulti = outSenders[0]
		} else {
			var err error
			outMulti, err = send.NewMultiSender(DefaultLogName, send.LevelInfo{Default: level.Info, Threshold: level.Trace}, outSenders)
			if err != nil {
				return ioutil.Discard, err
			}
		}
		o.outputSender = send.NewWriterSender(outMulti)
	}

	if !o.outputIsNull() && o.outputLogging() {
		o.outputMulti = io.MultiWriter(o.Output, o.outputSender)
	} else if !o.outputIsNull() {
		o.outputMulti = o.Output
	} else {
		o.outputMulti = o.outputSender
	}

	return o.outputMulti, nil
}

func (o *OutputOptions) GetError() (io.Writer, error) {
	if o.SendErrorToOutput {
		return o.GetOutput()
	}

	if o.errorIsNull() && !o.errorLogging() {
		return ioutil.Discard, nil
	}

	if o.errorMulti != nil {
		return o.errorMulti, nil
	}

	if o.errorLogging() {
		errSenders := []send.Sender{}

		for i := range o.Loggers {
			sender, err := o.Loggers[i].Configure()
			if err != nil {
				return ioutil.Discard, err
			}
			errSenders = append(errSenders, sender)
		}

		errMulti, err := send.NewMultiSender(DefaultLogName, send.LevelInfo{Default: level.Error, Threshold: level.Trace}, errSenders)
		if err != nil {
			return ioutil.Discard, err
		}
		o.errorSender = send.NewWriterSender(errMulti)
	}

	if !o.errorIsNull() && o.errorLogging() {
		o.errorMulti = io.MultiWriter(o.Error, o.errorSender)
	} else if !o.errorIsNull() {
		o.errorMulti = o.Error
	} else {
		o.errorMulti = o.errorSender
	}

	return o.errorMulti, nil
}
