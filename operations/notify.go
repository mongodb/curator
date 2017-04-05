package operations

import (
	"fmt"
	"strings"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/level"
	"github.com/mongodb/grip/message"
	"github.com/mongodb/grip/send"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// Notify provides a front end to sesveral external notification
// services provided by grip/curator.
func Notify() cli.Command {
	return cli.Command{
		Name:  "notify",
		Usage: "send a notification to a target",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name: "output",
				Usage: strings.Join([]string{
					"specify output format, either 'email', 'slack' (default), 'xmpp', or 'print'.",
					"define the 'GRIP_SLACK_CLIENT_TOKEN' env", "\tvariable for slack credentials.",
					"email defaults to contacting the MTA on localhost:25, but is configurable via", "\toptions on this command.",
					"define the 'GRIP_XMPP_HOSTNAME', 'GRIP_XMPP_USERNAME',", "\tand 'GRIP_XMPP_PASSWORD' environment variables ", "\tto configure the xmpp output.",
				}, "\n\t"),
				Value: "slack",
			},
			cli.StringFlag{
				Name:  "message",
				Usage: "define the message to send",
			},
			cli.StringFlag{
				Name:  "source",
				Usage: "set the logging source",
				Value: "curator",
			},
			cli.StringFlag{
				Name:  "target",
				Usage: "specify the recipient: the slack channel or the email/xmpp address.",
			},

			// the remaining options are email only.
			cli.StringFlag{
				Name:  "emailFrom",
				Usage: "specify the from address for the email",
			},
			cli.StringSliceFlag{
				Name:  "emailRecipient",
				Usage: "specify one or more email recipient",
			},
			cli.StringFlag{
				Name:  "emailServer",
				Usage: "specify an smtp server.",
				Value: "localhost",
			},
			cli.IntFlag{
				Name:  "emailPort",
				Usage: "specify the port of the smtp server",
				Value: 25,
			},
			cli.BoolFlag{
				Name:  "emailSSL",
				Usage: "if specified, connect to the smtp server via ssl",
			},
			cli.StringFlag{
				Name:   "emailUsername",
				Usage:  "if specified authenticate to the smtp server using this \n\tusername.",
				EnvVar: "CURATOR_NOTIFY_SMTP_USERNAME",
			},
			cli.StringFlag{
				Name:   "emailPassword",
				Usage:  "set the password for authentication to the smtp \n\tserver.",
				EnvVar: "CURATOR_NOTIFY_SMTP_PASSWORD",
			},
		},
		Action: func(c *cli.Context) error {
			var (
				err    error
				sender send.Sender
			)
			switch c.String("output") {
			case "slack":
				opts := &send.SlackOptions{
					Channel: c.String("target"),
					Name:    c.String("source"),
					Fields:  true,
				}
				sender, err = send.MakeSlackLogger(opts)
				if err != nil {
					return errors.Wrap(err, "problem building slack logger")
				}
			case "email":
				opts := &send.SMTPOptions{
					Name:   c.String("source"),
					From:   c.String("emailFrom"),
					Server: c.String("emailServer"),
					Port:   c.Int("emailPort"),
					UseSSL: c.Bool("emailSSL"),
				}
				recips := c.StringSlice("emailRecipient")
				if err = opts.AddRecipients(recips...); err != nil {
					return errors.Wrapf(err, "problem adding addresses [%v]", recips)
				}

				sender, err = send.MakeSMTPLogger(opts)
				if err != nil {
					return errors.Wrap(err, "problem building email logger")
				}
			case "xmpp":
				sender, err = send.MakeXMPP(c.String("target"))

				if err != nil {
					return errors.Wrap(err, "problem building jabber/xmpp logger")
				}
			case "print":
				sender = send.MakeNative()
				err = sender.SetFormatter(func(m message.Composer) (string, error) {
					return fmt.Sprintf("[notify=%s] [p=%s]: %s", c.String("target"), m.Priority(), m.String()), nil
				})
				if err != nil {
					return errors.Wrap(err, "problem setting up message formatting function")
				}
			default:
				return errors.Errorf("output '%s' is not supported", c.String("output"))
			}

			sender.SetName(c.String("source"))

			// we want to log errors sending messages to curator's process logging (e.g. standard output)
			if err = sender.SetErrorHandler(send.ErrorHandlerFromSender(grip.GetSender())); err != nil {
				return errors.Wrap(err, "problem setting error handler")
			}

			msg := message.NewString(c.String("message"))
			if err = msg.SetPriority(level.FromString(c.Parent().String("level"))); err != nil {
				return errors.Wrap(err, "problem setting log level")
			}

			sender.Send(msg)
			grip.Infof("message '%s' sent", msg)
			return nil
		},
	}
}
