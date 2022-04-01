package operations

import (
	"context"
	"os"

	"github.com/evergreen-ci/timber/buildlogger"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/send"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	timberPathFlagName       = "confPath"
	timberMetaFlagName       = "addMeta"
	timberAnnotationFlagName = "annotation"
)

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
			cli.BoolFlag{
				Name:  timberMetaFlagName,
				Usage: "when sending json or bson data, add logging meta data to each message",
			},
			cli.StringSliceFlag{
				Name: timberAnnotationFlagName,
				Usage: "Optional. Specify key pairs in the form of <key>:<value>. " +
					"You may specify this command more than once. " +
					"Keys must not contain the : character.",
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

			clogger, err := setupTimber(
				ctx,
				c.Parent().String(timberPathFlagName),
				c.Parent().Bool(timberMetaFlagName),
				getAnnotations(c.Parent().StringSlice(timberAnnotationFlagName)),
			)
			defer clogger.closer() // should close before checking error.
			if err != nil {
				return errors.Wrap(err, "configuring logger")
			}

			cmd, err := getCmd(c.String("exec"))
			if err != nil {
				return errors.Wrap(err, "creating command object")
			}

			return errors.Wrap(clogger.runCommand(cmd), "running command")
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

			clogger, err := setupTimber(
				ctx,
				c.Parent().String(timberPathFlagName),
				c.Parent().Bool(timberMetaFlagName),
				getAnnotations(c.Parent().StringSlice(timberAnnotationFlagName)),
			)
			defer clogger.closer()
			if err != nil {
				return errors.Wrap(err, "configuring logger")
			}

			if err := clogger.readPipe(os.Stdin); err != nil {
				return errors.Wrap(err, "reading from standard input")
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

			clogger, err := setupTimber(
				ctx,
				c.Parent().String(timberPathFlagName),
				c.Parent().Bool(timberMetaFlagName),
				getAnnotations(c.Parent().StringSlice(timberAnnotationFlagName)),
			)
			defer clogger.closer()
			if err != nil {
				return errors.Wrap(err, "configuring logger")
			}

			fn := c.String(fileFlagName)
			if err := clogger.followFile(fn); err != nil {
				return errors.Wrapf(err, "following file '%s'", fn)
			}
			return nil
		},
	}
}

func setupTimber(ctx context.Context, path string, meta bool, data map[string]string) (*cmdLogger, error) {
	out := &cmdLogger{
		annotations: data,
		addMeta:     meta,
	}

	var sender send.Sender
	out.closer = func() {
		if sender != nil {
			grip.Warning(sender.Close())
		}
	}

	out.logger = grip.NewJournaler("timber")

	opts, err := buildlogger.LoadLoggerOptions(path)
	if err != nil {
		return out, errors.Wrapf(err, "loading logger options from file '%s'", path)
	}
	sender, err = buildlogger.MakeLoggerWithContext(ctx, "curator", opts)
	if err != nil {
		return out, errors.Wrap(err, "creating logger")
	}
	if err := out.logger.SetSender(sender); err != nil {
		return out, errors.Wrap(err, "setting global sender")
	}

	switch opts.Format {
	case buildlogger.LogFormatJSON:
		out.logLine = out.logJSONLine
	case buildlogger.LogFormatBSON:
		out.logLine = out.logBSONLine
	default:
		out.logLine = out.logTextLine
	}

	return out, nil
}
