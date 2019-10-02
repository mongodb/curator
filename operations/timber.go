package operations

import (
	"context"
	"os"

	"github.com/evergreen-ci/timber"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/send"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const timberPathFlagName = "confPath"

// Timber command line function.
func Timber() cli.Command {
	return cli.Command{
		Name:  "timber",
		Usage: "tools for writing logs to a cedar backed buildlogger instance",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  timberPathFlagName,
				Usage: "specify the path of the input file",
			},
		},
		Subcommands: []cli.Command{
			timberCommand(),
			timberPipe(),
			timberFollowFile(),
		},
	}
}

func timberCommand() cli.Command {
	return cli.Command{
		Name:  "command",
		Usage: "run a command and write all standard output and error to the buildlogger via timber",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "exec",
				Usage: "a single command, (e.g. quoted) to run in the buildlogger",
			},
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			clogger, err := setupTimber(ctx, c.Parent().String(timberPathFlagName))
			defer clogger.closer() // should close before checking error.
			if err != nil {
				return errors.Wrap(err, "problem configuring logger")
			}
			//clogger.addMeta = c.Parent().Bool("addMeta")

			cmd, err := getCmd(c.String("exec"))
			if err != nil {
				return errors.Wrap(err, "problem creating command object")
			}

			return errors.Wrap(clogger.runCommand(cmd), "problem running command")
		},
	}
}

func timberPipe() cli.Command {
	return cli.Command{
		Name:  "pipe",
		Usage: "send standard input to the buildlogger via timber",
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			clogger, err := setupTimber(ctx, c.Parent().String(timberPathFlagName))
			defer clogger.closer()
			if err != nil {
				return errors.Wrap(err, "problem configuring logger")
			}

			if err := clogger.readPipe(os.Stdin); err != nil {
				return errors.Wrap(err, "problem reading from standard input")
			}

			return nil
		},
	}
}

func timberFollowFile() cli.Command {
	const fileFlagName = "filename"
	return cli.Command{
		Name:  "follow",
		Usage: "tail a (single) file and log changes to buildlogger via timber",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  fileFlagName,
				Usage: "specify a file to watch for changes to log",
			},
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			clogger, err := setupTimber(ctx, c.Parent().String(timberPathFlagName))
			defer clogger.closer()
			if err != nil {
				return errors.Wrap(err, "problem configuring logger")
			}

			fn := c.String(fileFlagName)
			if err := clogger.followFile(fn); err != nil {
				return errors.Wrapf(err, "problem following file %s", fn)
			}
			return nil
		},
	}
}

func setupTimber(ctx context.Context, path string) (*cmdLogger, error) {
	out := &cmdLogger{
		//annotations: data,
	}

	var sender send.Sender

	out.closer = func() {
		if sender != nil {
			grip.Warning(sender.Close())
		}
	}

	out.logger = grip.NewJournaler("buildlogger")

	opts, err := timber.LoadLoggerOptions(path)
	if err != nil {
		return out, errors.Wrapf(err, "problem loading logger options from %s", path)
	}
	sender, err = timber.MakeLogger(ctx, "curator", opts)
	if err != nil {
		return out, errors.Wrap(err, "problem creating logger")
	}
	if err := out.logger.SetSender(sender); err != nil {
		return out, errors.Wrap(err, "problem setting global sender")
	}

	switch opts.Format {
	case timber.LogFormatJSON:
		out.logLine = out.logJSONLine
	case timber.LogFormatBSON:
		out.logLine = out.logBSONLine
	default:
		out.logLine = out.logTextLine
	}

	return out, nil
}
