package operations

import (
	"time"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/level"
	"github.com/mongodb/grip/message"
	"github.com/mongodb/grip/send"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// SystemInfo returns a command object with subcommands for specific
// stats gathering operations.
func SystemInfo() cli.Command {
	return cli.Command{
		Name:    "stat",
		Aliases: []string{"stats"},
		Usage:   "collectors for system and process information",
		Subcommands: []cli.Command{
			systemInfo(),
			processInfo(),
			processTree(),
			processAll(),
		},
	}
}

func addSysInfoFlags(flags ...cli.Flag) []cli.Flag {
	return append(flags,
		cli.DurationFlag{
			Name:  "interval, i",
			Usage: "specify an interval for stats collection",
			Value: 10 * time.Second,
		},
		cli.IntFlag{
			Name:  "count",
			Usage: "specify maximum number of times to collect stats. Defaults to infinite",
			Value: 0,
		},
		cli.StringFlag{
			Name:  "file",
			Usage: "when specified, write output to a file, otherwise log to standard output",
		})
}

func systemInfo() cli.Command {
	return cli.Command{
		Name:  "system",
		Usage: "collects system level statistics",
		Flags: addSysInfoFlags(),
		Action: func(c *cli.Context) error {
			logger, closer, err := getLogger(c.String("file"))
			defer closer()
			if err != nil {
				return errors.Wrap(err, "problem building logger")
			}

			return doCollection(c.Int("count"), c.Duration("interval"), func() error {
				logger.Info(message.CollectSystemInfo())
				return nil
			})
		},
	}
}

func processAll() cli.Command {
	return cli.Command{
		Name:  "process-all",
		Usage: "collect process information for all processes on the system",
		Flags: addSysInfoFlags(),
		Action: func(c *cli.Context) error {
			logger, closer, err := getLogger(c.String("file"))
			if err != nil {
				return errors.Wrap(err, "problem building logger")
			}
			defer closer()

			return doCollection(c.Int("count"), c.Duration("interval"), func() error {
				logger.Info(message.CollectAllProcesses())
				return nil
			})
		},
	}

}

func processInfo() cli.Command {
	return cli.Command{
		Name:  "process",
		Usage: "collect process information about a single specific pid",
		Flags: addSysInfoFlags(
			cli.IntFlag{
				Name:  "pid",
				Usage: "specify a pid to collect data for",
			}),
		Action: func(c *cli.Context) error {
			pid := int32(c.Int("pid"))
			if pid == 0 {
				return errors.New("must specify a pid")
			}

			logger, closer, err := getLogger(c.String("file"))
			defer closer()
			if err != nil {
				return errors.Wrap(err, "problem building logger")
			}

			return doCollection(c.Int("count"), c.Duration("interval"), func() error {
				logger.Info(message.CollectProcessInfo(pid))
				return nil
			})
		},
	}
}

func processTree() cli.Command {
	return cli.Command{
		Name:  "process-tree",
		Usage: "collect process information for the current process and all children processes",
		Flags: addSysInfoFlags(
			cli.IntFlag{
				Name:  "pid",
				Usage: "specify the pid of a parent process",
			}),
		Action: func(c *cli.Context) error {
			pid := int32(c.Int("pid"))
			if pid == 0 {
				return errors.New("must specify a pid")
			}

			logger, closer, err := getLogger(c.String("file"))
			defer closer()
			if err != nil {
				return errors.Wrap(err, "problem building logger")
			}

			return doCollection(c.Int("count"), c.Duration("interval"), func() error {
				logger.Info(message.CollectProcessInfoWithChildren(pid))
				return nil
			})
		},
	}
}

///////////////////////////////////////////////////////////////////////////
//
// functions to handle logging set up and looping
//
///////////////////////////////////////////////////////////////////////////

type closer func()

func getLogger(fn string) (grip.Journaler, closer, error) {
	closer := func() {}
	logger := grip.NewJournaler("curator.stats")
	lvl := send.LevelInfo{
		Threshold: level.Debug,
		Default:   level.Info,
	}

	var sender send.Sender
	var err error

	if fn != "" {
		sender, err = send.MakeJSONFileLogger(fn)
		if err != nil {
			return nil, closer, errors.Wrap(err, "problem building logger")
		}
		closer = func() { grip.CatchCritical(sender.Close()) }

		if err = logger.SetSender(sender); err != nil {
			return nil, closer, errors.Wrap(err, "problem configuring logger")
		}
	} else {
		sender = send.MakeJSONConsoleLogger()
		closer = func() { grip.CatchCritical(sender.Close()) }

		if err = logger.SetSender(sender); err != nil {
			return nil, closer, errors.Wrap(err, "problem configuring logger")
		}
	}

	if err := sender.SetLevel(lvl); err != nil {
		return nil, closer, errors.Wrap(err, "problem setting logging threshold")
	}

	return logger, closer, nil
}

func doCollection(count int, interval time.Duration, op func() error) error {
	for {
		if err := op(); err != nil {
			return err
		}

		count--
		if count == 0 {
			break
		}

		time.Sleep(interval)
	}

	return nil
}
