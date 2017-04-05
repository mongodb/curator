package operations

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	shlex "github.com/anmitsu/go-shlex"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/level"
	"github.com/mongodb/grip/message"
	"github.com/mongodb/grip/send"
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
		},
		Subcommands: []cli.Command{
			logCommand(),
			logPipe(),
		},
	}
}

///////////////////////////////////
//
// Subcommands
//
///////////////////////////////////

func logCommand() cli.Command {
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
			conf := getBuildloggerConfig(c)
			logger, closer, err := setupBuildLogger(conf)
			defer closer()
			if err != nil {
				return errors.Wrap(err, "problem configuring buildlogger")
			}

			command := c.String("exec")
			args, err := shlex.Split(command, true)
			if err != nil {
				return errors.Wrapf(err, "problem parsing command: %s", command)
			}

			var cmd *exec.Cmd
			if len(args) == 0 {
				return errors.New("no command specified")
			} else if len(args) == 1 {
				cmd = exec.Command(args[0])
			} else {
				cmd = exec.Command(args[0], args[1:]...)
			}

			return errors.Wrap(runCommand(logger, cmd, c.Int("count")/2), "problem running command")
		},
	}
}

func logPipe() cli.Command {
	return cli.Command{
		Name:  "pipe",
		Usage: "send standard input to the buildlogger",
		Action: func(c *cli.Context) error {
			conf := getBuildloggerConfig(c)

			logger, closer, err := setupBuildLogger(conf)
			defer closer()
			if err != nil {
				return errors.Wrap(err, "problem configuring buildlogger")
			}

			input := bufio.NewScanner(os.Stdin)
			for input.Scan() {
				logger.Log(logger.DefaultLevel(),
					message.NewDefaultMessage(level.Info, input.Text()))
			}

			return errors.Wrap(input.Err(), "problem reading from standard input")
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
		URL:            c.Parent().String("url"),
		Phase:          c.Parent().String("phase"),
		Builder:        c.Parent().String("builder"),
		Test:           c.Parent().String("test"),
		BufferCount:    c.Parent().Int("count"),
		BufferInterval: c.Parent().Duration("interval"),
		Local:          grip.GetSender(),
	}
}

func setupBuildLogger(conf *send.BuildloggerConfig) (grip.Journaler, func(), error) {
	var toClose []send.Sender

	closer := func() {
		for _, sender := range toClose {
			grip.Warning(sender.Close())
		}
	}

	logger := grip.NewJournaler("buildlogger")
	globalSender, err := send.MakeBuildlogger("curator", conf)
	if err != nil {
		return nil, closer, errors.Wrap(err, "problem configuring global sender")
	}
	toClose = append(toClose, globalSender)
	if err := logger.SetSender(globalSender); err != nil {
		return nil, closer, errors.Wrap(err, "problem setting global sender")
	}

	if conf.Test != "" {
		conf.CreateTest = true
		testSender, err := send.MakeBuildlogger(conf.Test, conf)
		if err != nil {
			return nil, closer, errors.Wrap(err, "problem constructing test logger")
		}
		toClose = append(toClose, testSender)
		if err := logger.SetSender(testSender); err != nil {
			return nil, closer, errors.Wrap(err, "problem setting test logger")
		}
	}

	return logger, closer, nil
}

func runCommand(logger grip.Journaler, cmd *exec.Cmd, bufferLen int) error {
	if bufferLen < 100 {
		grip.Warningf("reset buffer to %d lines, from %d", 100, bufferLen)
		bufferLen = 100
	}

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
	lines := make(chan string, bufferLen)
	loggerDone := make(chan struct{})
	stdOutDone := make(chan struct{})
	stdErrDone := make(chan struct{})

	go collectStream(lines, stdOut, stdOutDone)
	go collectStream(lines, stdErr, stdErrDone)
	go logLines(logger, lines, loggerDone)

	<-stdOutDone
	<-stdErrDone
	err = cmd.Wait()

	close(lines)
	<-loggerDone

	grip.Infof("completed command in %s", time.Since(startedAt))

	return errors.Wrap(err, "command returned an error")
}

func collectStream(out chan<- string, input io.Reader, signal chan struct{}) {
	stream := bufio.NewScanner(input)

	for stream.Scan() {
		out <- stream.Text()
	}

	signal <- struct{}{}
}

func logLines(logger grip.Journaler, lines <-chan string, signal chan struct{}) {
	l := logger.ThresholdLevel()

	for line := range lines {
		logger.Log(l, message.NewDefaultMessage(l, line))
	}

	signal <- struct{}{}
}
