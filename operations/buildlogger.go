package operations

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	shlex "github.com/anmitsu/go-shlex"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/mongodb/grip/send"
	"github.com/papertrail/go-tail/follower"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// BuildLogger constructs the command object for all build logger client entry-points.
func BuildLogger() cli.Command {
	return cli.Command{
		Name:  "buildlogger",
		Usage: "tools for writing logs to a buildlogger (logkeeper) instance",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "url",
				Usage:  "url of buildlogger/logkeeper server",
				EnvVar: "BUILDLOGGER_URL",
			},
			cli.StringFlag{
				Name:   "phase",
				EnvVar: "MONGO_PHASE",
				Value:  "unknown",
			},
			cli.StringFlag{
				Name:   "builder",
				EnvVar: "MONGO_BUILDER_NAME",
				Value:  "unknown",
			},
			cli.StringFlag{
				Name:   "test",
				EnvVar: "MONGO_TEST_FILE_NAME",
				Value:  "unknown",
			},
			cli.IntFlag{
				Name:  "count",
				Usage: "number of messages to buffer before sending",
				Value: 1000,
			},
			cli.DurationFlag{
				Name: "interval",
				Usage: "number of seconds to wait before sending messages," +
					"if number of buffered messages does not reach the count threshold",
				Value: 20 * time.Second,
			},
			cli.StringFlag{
				Name:   "credentials",
				Usage:  "file name of json formated username and password document for buildlogger server",
				EnvVar: "BUILDLOGGER_CREDENTIALS",
			},
			cli.StringFlag{
				Name:   "username",
				Usage:  "alternate credential specification method. Must specify username and password",
				EnvVar: "BUILDLOGGER_USERNAME",
			},
			cli.StringFlag{
				Name:   "password",
				Usage:  "alternate credential specification method. Must specify username and password",
				EnvVar: "BUILDLOGGER_PASSWORD",
			},
			cli.StringFlag{
				Name:  "json",
				Usage: "when specified, all input is parsed as new-line seperated json",
			},
			cli.BoolFlag{
				Name:  "addMeta",
				Usage: "when sending json data, add logging meta data to each message",
			},
			cli.StringSliceFlag{
				Name: "annotation",
				Usage: "Optional. Specify key pairs in the form of <key>:<value>. " +
					"You may specify this command more than once. " +
					"Keys must not contain the : character.",
			},
		},
		Subcommands: []cli.Command{
			buildLogCommand(),
			buildLogPipe(),
			buildLogFollowFile(),
		},
	}
}

///////////////////////////////////
//
// Subcommands
//
///////////////////////////////////

func buildLogCommand() cli.Command {
	return cli.Command{
		Name:  "command",
		Usage: "run a command and write all standard input and error to the buildlogger",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "exec",
				Usage: "a single command, (e.g. quoted) to run in the buildlogger",
			},
		},
		Action: func(c *cli.Context) error {
			clogger, err := setupBuildLogger(
				getBuildloggerConfig(c),
				getAnnotations(c.Parent().StringSlice("annotation")),
				c.Parent().Bool("json"),
				c.Parent().Int("count"),
				c.Parent().Duration("interval"))
			defer clogger.closer() // should close before checking error.
			if err != nil {
				return errors.Wrap(err, "problem configuring buildlogger")
			}
			clogger.addMeta = c.Parent().Bool("addMeta")

			cmd, err := getCmd(c.String("exec"))
			if err != nil {
				return errors.Wrap(err, "problem creating command object")
			}

			return errors.Wrap(clogger.runCommand(cmd), "problem running command")
		},
	}
}

func buildLogPipe() cli.Command {
	return cli.Command{
		Name:  "pipe",
		Usage: "send standard input to the buildlogger",
		Action: func(c *cli.Context) error {
			clogger, err := setupBuildLogger(
				getBuildloggerConfig(c),
				getAnnotations(c.Parent().StringSlice("annotation")),
				c.Parent().Bool("json"),
				c.Parent().Int("count"),
				c.Parent().Duration("interval"))

			defer clogger.closer()
			if err != nil {
				return errors.Wrap(err, "problem configuring buildlogger")
			}

			if err := clogger.readPipe(os.Stdin); err != nil {
				return errors.Wrap(err, "problem reading from standard input")
			}

			return nil
		},
	}
}

func buildLogFollowFile() cli.Command {
	return cli.Command{
		Name:  "follow",
		Usage: "tail a (single) file and log changes to buildlogger",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "file",
				Usage: "specify a file to watch for changes to log",
			},
		},
		Action: func(c *cli.Context) error {
			clogger, err := setupBuildLogger(
				getBuildloggerConfig(c),
				getAnnotations(c.Parent().StringSlice("annotation")),
				c.Parent().Bool("json"),
				c.Parent().Int("count"),
				c.Parent().Duration("interval"))

			defer clogger.closer()
			if err != nil {
				return errors.Wrap(err, "problem configuring buildlogger")
			}

			fn := c.String("file")

			if err := clogger.followFile(fn); err != nil {
				return errors.Wrapf(err, "problem following file %s", fn)
			}
			return nil
		},
	}
}

////////////////////////////////////////////////////////////////////////
//
// Internal operations
//
////////////////////////////////////////////////////////////////////////

func getBuildloggerConfig(c *cli.Context) *send.BuildloggerConfig {
	return &send.BuildloggerConfig{
		URL:     c.Parent().String("url"),
		Phase:   c.Parent().String("phase"),
		Builder: c.Parent().String("builder"),
		Test:    c.Parent().String("test"),
		Local:   grip.GetSender(),
	}
}

func getCmd(command string) (*exec.Cmd, error) {
	args, err := shlex.Split(command, true)
	if err != nil {
		return nil, errors.Wrapf(err, "problem parsing command: %s", command)
	}

	var cmd *exec.Cmd
	if len(args) == 0 {
		return nil, errors.New("no command specified")
	} else if len(args) == 1 {
		cmd = exec.Command(args[0])
	} else {
		cmd = exec.Command(args[0], args[1:]...)
	}

	return cmd, nil
}

type cmdLogger struct {
	logger      grip.Journaler
	annotations map[string]string
	logJSON     bool
	addMeta     bool
	closer      func()
}

func setupBuildLogger(conf *send.BuildloggerConfig, data map[string]string, logJSON bool, count int, interval time.Duration) (*cmdLogger, error) {
	out := &cmdLogger{
		logJSON:     logJSON,
		annotations: data,
	}

	var toClose []send.Sender

	out.closer = func() {
		for _, sender := range toClose {
			if sender == nil {
				continue
			}

			grip.Warning(sender.Close())
		}
	}

	out.logger = grip.NewJournaler("buildlogger")

	globalSender, err := send.MakeBuildlogger("curator", conf)
	toClose = append(toClose, globalSender)
	if err != nil {
		return out, errors.Wrap(err, "problem configuring global sender")
	}

	globalBuffered := send.NewBufferedSender(globalSender, interval, count)
	toClose = append(toClose, globalBuffered)
	if err := out.logger.SetSender(globalBuffered); err != nil {
		return out, errors.Wrap(err, "problem setting global sender")
	}

	if conf.Test != "" {
		conf.CreateTest = true

		testSender, err := send.MakeBuildlogger(conf.Test, conf)
		toClose = append(toClose, testSender)
		if err != nil {
			return out, errors.Wrap(err, "problem constructing test logger")
		}

		testBuffered := send.NewBufferedSender(globalSender, interval, count)
		toClose = append(toClose, testBuffered)
		if err := out.logger.SetSender(testBuffered); err != nil {
			return out, errors.Wrap(err, "problem setting test logger")
		}
	}

	return out, nil
}

func (l *cmdLogger) runCommand(cmd *exec.Cmd) error {
	command := strings.Join(cmd.Args, " ")
	grip.Debugf("prepping command %s, in %s, with %s", command, cmd.Dir,
		strings.Join(cmd.Env, " "))

	// setup the output
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "problem getting standard output")
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "problem getting standard error")
	}

	// Now actually run the command
	startedAt := time.Now()
	if err = cmd.Start(); err != nil {
		return errors.Wrap(err, "problem starting command")
	}

	grip.Infoln("running command:", command)

	// collect and merge lines into a single output stream in the logger
	lines := make(chan []byte)
	loggerDone := make(chan struct{})
	stdOutDone := make(chan struct{})
	stdErrDone := make(chan struct{})

	go collectStream(lines, stdOut, stdOutDone)
	go collectStream(lines, stdErr, stdErrDone)
	if l.logJSON {
		go l.logJSONLines(lines, loggerDone)
	} else {
		go l.logLines(lines, loggerDone)
	}

	<-stdOutDone
	<-stdErrDone
	err = cmd.Wait()

	close(lines)
	<-loggerDone

	grip.Infof("completed command in %s", time.Since(startedAt))
	return errors.Wrap(err, "command returned an error")
}

func (l *cmdLogger) readPipe(pipe io.Reader) error {
	lvl := l.logger.GetSender().Level().Default
	input := bufio.NewScanner(pipe)
	for input.Scan() {
		switch {
		case l.logJSON:
			out := message.Fields{}
			line := input.Bytes()
			if err := json.Unmarshal(line, &out); err != nil {
				grip.Error(err)
				continue
			}
			switch {
			case l.addMeta:
				m := message.MakeFields(out)
				if err := l.addAnnotations(m); err != nil {
					grip.Error(err)
				}

				l.logger.Log(lvl, m)
			default:
				l.logger.Log(lvl, message.MakeSimpleFields(out))
			}
		default:
			m := message.NewDefaultMessage(lvl, input.Text())

			if l.addMeta {
				if err := l.addAnnotations(m); err != nil {
					grip.Error(err)
				}
			}

			l.logger.Log(lvl, m)
		}
	}

	return errors.Wrap(input.Err(), "problem reading from pipe")
}

func (l *cmdLogger) followFile(fn string) error {
	lvl := l.logger.GetSender().Level().Default
	tail, err := follower.New(fn, follower.Config{
		Reopen: true,
	})
	if err != nil {
		return errors.Wrapf(err, "problem setting up file follower of '%s'", fn)
	}
	defer tail.Close()

	if err = tail.Err(); err != nil {
		return errors.Wrapf(err, "problem starting up file follower of '%s'", fn)
	}

	for line := range tail.Lines() {
		switch {
		case l.logJSON:
			out := message.Fields{}
			if err := json.Unmarshal(line.Bytes(), &out); err != nil {
				grip.Error(err)
				continue
			}
			switch {
			case l.addMeta:
				m := message.MakeFields(out)
				if err := l.addAnnotations(m); err != nil {
					grip.Error(err)
				}

				l.logger.Log(lvl, message.MakeFields(out))
			default:
				l.logger.Log(lvl, message.MakeSimpleFields(out))
			}
		default:
			m := message.NewDefaultMessage(lvl, line.String())

			if l.addMeta {
				if err := l.addAnnotations(m); err != nil {
					grip.Error(err)
				}
			}

			l.logger.Log(lvl, m)
		}
	}

	if err = tail.Err(); err != nil {
		return errors.Wrapf(err, "problem finishing file following of '%s'", fn)
	}
	return nil
}

func collectStream(out chan<- []byte, input io.Reader, signal chan struct{}) {
	stream := bufio.NewScanner(input)

	for stream.Scan() {
		cp := []byte{}
		copy(cp, stream.Bytes())
		out <- cp
	}

	close(signal)
}

func (l *cmdLogger) addAnnotations(m message.Composer) error {
	if len(l.annotations) == 0 {
		return nil
	}
	catcher := grip.NewBasicCatcher()
	for k, v := range l.annotations {
		catcher.Add(m.Annotate(k, v))
	}
	return catcher.Resolve()
}

func (l *cmdLogger) logLines(lines <-chan []byte, signal chan struct{}) {
	logLevel := l.logger.GetSender().Level().Threshold

	for line := range lines {
		grip.Notice(line)
		m := message.NewBytesMessage(logLevel, line)
		grip.Error(l.addAnnotations(m))

		l.logger.Log(logLevel, m)
	}

	close(signal)
}

func (l *cmdLogger) logJSONLines(lines <-chan []byte, signal chan struct{}) {
	logLevel := l.logger.GetSender().Level().Threshold

	for line := range lines {
		grip.Notice(line)
		out := message.Fields{}
		if err := json.Unmarshal(line, &out); err != nil {
			grip.Error(err)
			continue
		}
		var m message.Composer

		switch {
		case l.addMeta:
			m = message.MakeFields(out)
			grip.Error(l.addAnnotations(m))
		default:
			m = message.MakeSimpleFields(out)
			grip.Error(l.addAnnotations(m))
		}
		l.logger.Log(logLevel, m)
	}

	close(signal)
}
