package operations

import (
	"os"
	"strings"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/logging"
	"github.com/mongodb/grip/send"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// Splunk provides a command interface to log the output of commands (or standard input.)
func Splunk() cli.Command {
	return cli.Command{
		Name:  "splunk",
		Usage: "tools to log operations directly to splunk",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name",
				Value: "curator",
			},
			cli.StringFlag{
				Name:   "url",
				Usage:  "",
				EnvVar: "GRIP_SPLUNK_SERVER_URL",
			},
			cli.StringFlag{
				Name:   "token",
				Usage:  "",
				EnvVar: "GRIP_SPLUNK_CLIENT_TOKEN",
			},
			cli.StringFlag{
				Name:   "channel",
				Usage:  "",
				EnvVar: "GRIP_SPLUNK_CHANNEL",
			},
			cli.StringFlag{
				Name:  "json",
				Usage: "when specified, all input is parsed as new-line seperated json",
			},
			cli.StringFlag{
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
			splunkLogCommand(),
			splunkLogPipe(),
			splunkLogFollowFile(),
		},
	}
}

func setupSplunkLogger(c *cli.Context) (*cmdLogger, error) {
	info := send.GetSplunkConnectionInfo()

	if url := c.Parent().String("url"); url != "" {
		info.ServerURL = url
	}

	if token := c.Parent().String("token"); token != "" {
		info.Token = token
	}

	if channel := c.Parent().String("channel"); channel != "" {
		info.Channel = channel
	}

	out := &cmdLogger{
		logJSON:     c.Parent().Bool("json"),
		annotations: getAnnotations(c.Parent().StringSlice("annotation")),
	}

	if !info.Populated() {
		return out, errors.New("splunk configuration is insufficient")
	}

	sender, err := send.NewSplunkLogger(c.Parent().String("name"), info, grip.GetSender().Level())
	if err != nil {
		return out, errors.Wrap(err, "problem constructing logger")
	}
	out.logger = logging.MakeGrip(sender)
	out.closer = func() {
		grip.CatchError(sender.Close())
	}

	return out, nil
}

func getAnnotations(data []string) map[string]string {
	if len(data) == 0 {
		return nil
	}

	out := make(map[string]string)
	for _, n := range data {
		parts := strings.SplitN(n, ":", 2)
		if len(parts) != 2 {
			continue
		}
		out[parts[0]] = parts[1]
	}

	return out
}

///////////////////////////////////
//
// Subcommands
//
///////////////////////////////////

func splunkLogCommand() cli.Command {
	return cli.Command{
		Name:  "command",
		Usage: "run a command and write all standard input and error to splunk",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "exec",
				Usage: "a single command, (e.g. quoted) to run in the splunk",
			},
		},
		Action: func(c *cli.Context) error {
			clogger, err := setupSplunkLogger(c)
			defer clogger.closer()
			if err != nil {
				return errors.Wrap(err, "problem configuring splunk connection")
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

func splunkLogPipe() cli.Command {
	return cli.Command{
		Name:  "pipe",
		Usage: "send standard input to splunk",
		Action: func(c *cli.Context) error {
			clogger, err := setupSplunkLogger(c)
			defer clogger.closer()
			if err != nil {
				return errors.Wrap(err, "problem configuring splunk connection")
			}

			if err := clogger.readPipe(os.Stdin); err != nil {
				return errors.Wrap(err, "problem reading from standard input")
			}

			return nil
		},
	}
}

func splunkLogFollowFile() cli.Command {
	return cli.Command{
		Name:  "follow",
		Usage: "tail a (single) file and log changes to splunk",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "file",
				Usage: "specify a file to watch for changes to log",
			},
		},
		Action: func(c *cli.Context) error {
			clogger, err := setupSplunkLogger(c)
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
