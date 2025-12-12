package operations

import (
	"fmt"
	"strconv"
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
					"specify output format, either 'email', 'slack' (default), 'xmpp', 'jira', 'github', or 'print'.",
					"define the 'GRIP_SLACK_CLIENT_TOKEN' env", "\tvariable for slack credentials.",
					"you can specify credentials for github and jira using arguments or envvars.",
					"email defaults to contacting the MTA on localhost:25, but is configurable via", "\toptions on this command.",
					"define the 'GRIP_XMPP_HOSTNAME', 'GRIP_XMPP_USERNAME',", "\tand 'GRIP_XMPP_PASSWORD' environment variables ",
					"\tto configure the xmpp output.",
				}, "\n\t"),
				Value: "slack",
			},
			cli.StringFlag{
				Name:  "message",
				Usage: "specify the message to send",
			},
			cli.StringFlag{
				Name:  "source",
				Usage: "set the logging source",
				Value: "curator",
			},
			cli.StringFlag{
				Name: "target",
				Usage: strings.Join([]string{
					"specify the recipient: the slack channel or the email/xmpp address.",
					"for jira, specify the issue key; for github specify <account>/<repo>",
				}, "\n\t"),
			},

			// options for creating alerts for specifc senders
			cli.StringFlag{
				Name:  "jiraURL",
				Usage: "for the jira sender, specify the url (e.g. https://jira.example.net/) of the instance",
			},
			cli.StringFlag{
				Name:  "issue",
				Usage: "specify a github or jira issue ID to create a comment on an existing issue rather than create a new issue.",
			},

			// options used to specify authentication credentials.
			cli.StringFlag{
				Name:   "githubToken",
				Usage:  "specify a github api auth token",
				EnvVar: "CURATOR_GITHUB_API_TOKEN",
			},
			cli.StringFlag{
				Name:   "username",
				Usage:  "use to specify username for jira and email methods. Optional for email.",
				EnvVar: "CURATOR_NOTIFY_USERNAME",
			},
			cli.StringFlag{
				Name:   "password",
				Usage:  "use to specify password for jira and email methods. Optional for email.",
				EnvVar: "CURATOR_NOTIFY_PASSWORD",
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
		},
		Action: func(c *cli.Context) error {
			var (
				sender send.Sender
				err    error
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
					return errors.Wrap(err, "building Slack logger")
				}
			case "email":
				opts := &send.SMTPOptions{
					Name:     c.String("source"),
					From:     c.String("emailFrom"),
					Server:   c.String("emailServer"),
					Port:     c.Int("emailPort"),
					UseSSL:   c.Bool("emailSSL"),
					Username: c.String("username"),
					Password: c.String("password"),
				}
				recips := c.StringSlice("emailRecipient")
				if err = opts.AddRecipients(recips...); err != nil {
					return errors.Wrapf(err, "adding email recipients %s", recips)
				}

				sender, err = send.MakeSMTPLogger(opts)
				if err != nil {
					return errors.Wrap(err, "building email logger")
				}
			case "xmpp":
				sender, err = send.MakeXMPP(c.String("target"))

				if err != nil {
					return errors.Wrap(err, "building Jabber/XMPP logger")
				}
			case "github":
				info := strings.SplitN(c.String("target"), "/", 2)
				if len(info) != 2 {
					return errors.Errorf("'%s' is not a valid <account>/<repo> specification",
						c.String("target"))
				}

				opts := &send.GithubOptions{
					Account: info[0],
					Repo:    info[1],
					Token:   c.String("ghToken"),
				}

				issue := c.String("issue")
				if issue == "" {
					sender, err = send.NewGithubIssuesLogger(c.String("source"), opts)
				} else {
					var id int
					id, err = strconv.Atoi(issue)
					if err != nil {
						return errors.Errorf("'%s' is not a valid issue ID", issue)
					}
					sender, err = send.NewGithubCommentLogger(c.String("source"),
						id, opts)
				}

				if err != nil {
					return errors.Wrap(err, "setting up GitHub logger")
				}
			case "print":
				sender = send.MakeNative()
				err = sender.SetFormatter(func(m message.Composer) (string, error) {
					return fmt.Sprintf("[notify=%s] [p=%s]: %s", c.String("target"), m.Priority(), m.String()), nil
				})
				if err != nil {
					return errors.Wrap(err, "setting up message formatting function")
				}
			default:
				return errors.Errorf("output '%s' is not supported", c.String("output"))
			}

			sender.SetName(c.String("source"))

			// we want to log errors sending messages to curator's process logging (e.g. standard output)
			if err = sender.SetErrorHandler(send.ErrorHandlerFromSender(grip.GetSender())); err != nil {
				return errors.Wrap(err, "setting error handler")
			}

			msg := message.NewString(c.String("message"))
			if err = msg.SetPriority(level.FromString(c.Parent().String("level"))); err != nil {
				return errors.Wrap(err, "setting log level")
			}

			sender.Send(msg)
			grip.Infof("message '%s' sent", msg)
			return nil
		},
	}
}
