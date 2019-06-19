package cli

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/cheynewallace/tabby"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/mongodb/grip/recovery"
	"github.com/mongodb/jasper"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli"
)

// RunCMD provides a simple user-centered command-line interface for
// running commands on a remote instance.
func RunCMD() cli.Command {
	const (
		commandFlagName = "command"
		envFlagName     = "env"
		sudoFlagName    = "sudo"
		sudoAsFlagName  = "sudo_as"
		idFlagName      = "id"
		execFlagName    = "exec"
		tagFlagName     = "tag"
		waitFlagName    = "wait"

		logPollInterval = 100 * time.Millisecond
	)

	defaultID := uuid.Must(uuid.NewV4())

	return cli.Command{
		Name:  "run",
		Usage: "Run a command with Jasper",
		Flags: append(clientFlags(),
			cli.StringSliceFlag{
				Name:  joinFlagNames(commandFlagName, "c"),
				Usage: "specify a command to run on a remote jasper service. may specify more than once",
			},
			cli.StringSliceFlag{
				Name:  envFlagName,
				Usage: "specify environment variables, in '<key>=<val>' forms. may specify more than once",
			},
			cli.BoolFlag{
				Name:  sudoFlagName,
				Usage: "use this flag to run the command with sudo",
			},
			cli.StringFlag{
				Name:  sudoAsFlagName,
				Usage: "use this to run commands as another user as in 'sudo -u <user>'",
			},
			cli.StringFlag{
				Name:  idFlagName,
				Usage: "specify an id for this process (optional)",
				Value: defaultID.String(),
			},
			cli.BoolFlag{
				Name:  execFlagName,
				Usage: "when specified execute commands directly without shell. If multiple commands are specified they do not share a shell environment",
			},
			cli.StringSliceFlag{
				Name:  tagFlagName,
				Usage: "specify one or more tag names for the process",
			},
			cli.BoolFlag{
				Name:  waitFlagName,
				Usage: "specify to block until the process returns (subject to service timeouts), propagating the exit code from process",
			},
		),
		Before: mergeBeforeFuncs(clientBefore(),
			func(c *cli.Context) error {
				if len(c.StringSlice(commandFlagName)) == 0 {
					if c.NArg() == 0 {
						return errors.New("must specify a command")
					}
					return errors.Wrap(c.Set(commandFlagName, strings.Join(c.Args(), " ")), "problem setting command")
				}
				return nil
			}),
		Action: func(c *cli.Context) error {
			envvars := c.StringSlice(envFlagName)
			cmds := c.StringSlice(commandFlagName)
			useSudo := c.Bool(sudoFlagName)
			sudoAs := c.String(sudoAsFlagName)
			useExec := c.Bool(execFlagName)
			cmdID := c.String(idFlagName)
			tags := c.StringSlice(tagFlagName)
			wait := c.Bool(waitFlagName)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			logger := jasper.NewInMemoryLogger()

			return withConnection(ctx, c, func(client jasper.RemoteClient) error {
				cmd := client.CreateCommand(ctx).Sudo(useSudo).ID(cmdID).SetTags(tags)

				if wait {
					cmd.Background(true).AppendLoggers(logger).RedirectErrorToOutput(true)
				}

				for _, cmdStr := range cmds {
					if useExec {
						cmd.Append(cmdStr)
					} else {
						cmd.Bash(cmdStr)
					}
				}

				if sudoAs != "" {
					cmd.SudoAs(sudoAs)
				}

				for _, e := range envvars {
					parts := strings.SplitN(e, "=", 2)
					cmd.AddEnv(parts[0], parts[1])
				}

				if err := cmd.Run(ctx); err != nil {
					return errors.WithStack(err)
				}

				if wait {
					logDone := make(chan struct{})
					go func() {
						defer recovery.LogStackTraceAndContinue("log handling thread")
						defer close(logDone)
						timer := time.NewTimer(0)
						defer timer.Stop()

						for {
							select {
							case <-ctx.Done():
								grip.Notice("operation canceled")
								return
							case <-timer.C:
								logLines, err := client.GetLogStream(ctx, cmdID, logger.Options.InMemoryCap)
								if err != nil {
									grip.Error(message.WrapError(err, "problem polling for log lines, aborting log following"))
									return
								}

								for _, ln := range logLines.Logs {
									grip.Info(ln)
								}

								if !logLines.Done {
									timer.Reset(0)
									continue
								}

								timer.Reset(randDur(logPollInterval))
							}
						}
					}()

					exit, err := cmd.Wait(ctx)
					grip.Notice(err)
					<-logDone
					os.Exit(exit)
				}

				return nil
			})
		},
	}
}

// ListCMD provides a user interface to inspect processes managed by a
// jasper instance and their state. The output of the command is a
// human-readable table.
func ListCMD() cli.Command {
	return cli.Command{
		Name:  "list",
		Usage: "list jasper managed commands with human readable output",
		Flags: append(clientFlags(),
			cli.BoolFlag{
				Name:  "running",
				Usage: "show only running processes",
			},
			cli.BoolFlag{
				Name:  "terminated",
				Usage: "show only terminated (complete) processes",
			},
			cli.BoolFlag{
				Name:  "failed",
				Usage: "show only failed processes",
			},
			cli.BoolFlag{
				Name:  "successful",
				Usage: "show only successful processes",
			},
			cli.StringFlag{
				Name:  "group",
				Usage: "specify a tag to return a list of processes, overrides other options",
			},
		),
		Before: clientBefore(),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			filter := jasper.All

			switch {
			case c.Bool("running"):
				filter = jasper.Running
			case c.Bool("terminated"):
				filter = jasper.Terminated
			case c.Bool("failed"):
				filter = jasper.Failed
			case c.Bool("successful"):
				filter = jasper.Successful
			}

			group := c.String("group")

			return withConnection(ctx, c, func(client jasper.RemoteClient) error {
				var (
					procs []jasper.Process
					err   error
				)

				if group == "" {
					procs, err = client.List(ctx, filter)
				} else {
					procs, err = client.Group(ctx, group)

				}

				if err != nil {
					return errors.Wrap(err, "problem getting list")
				}

				t := tabby.New()
				t.AddHeader("ID", "PID", "Running", "Complete", "Tags", "Command")
				for _, p := range procs {
					info := p.Info(ctx)
					t.AddLine(p.ID(), info.PID, p.Running(ctx), p.Complete(ctx), p.GetTags(), strings.Join(info.Options.Args, " "))
				}
				t.Print()
				return nil
			})
		},
	}
}

// KillCMD terminates a single process by id, sending either TERM or KILL.
func KillCMD() cli.Command {
	const (
		idFlagName   = "id"
		killFlagName = "kill"
	)
	return cli.Command{
		Name:  "kill",
		Usage: "terminate processes",
		Flags: append(clientFlags(),
			cli.StringFlag{
				Name:  joinFlagNames(idFlagName, "i"),
				Usage: "specify the id of the process to kill",
			},
			cli.BoolFlag{
				Name:  killFlagName,
				Usage: "send KILL (9) rather than term (15)",
			},
		),
		Before: mergeBeforeFuncs(
			clientBefore(),
			func(c *cli.Context) error {
				if len(c.StringSlice(idFlagName)) == 0 {
					if c.NArg() != 1 {
						return errors.New("must specify a command ID")
					}
					return errors.Wrap(c.Set(idFlagName, c.Args().First()), "problem setting id from positional flags")
				}
				return nil
			}),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sendKill := c.Bool(killFlagName)
			procID := c.String(idFlagName)
			return withConnection(ctx, c, func(client jasper.RemoteClient) error {
				proc, err := client.Get(ctx, procID)
				if err != nil {
					return errors.WithStack(err)
				}

				if sendKill {
					return errors.WithStack(jasper.Kill(ctx, proc))
				}
				return errors.WithStack(jasper.Terminate(ctx, proc))
			})
		},
	}
}

// ClearCMD removes all terminated/exited tracked processes from
// jasper Manger.
func ClearCMD() cli.Command {
	return cli.Command{
		Name:   "clear",
		Usage:  "terminate processes",
		Flags:  clientFlags(),
		Before: clientBefore(),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			return withConnection(ctx, c, func(client jasper.RemoteClient) error {
				client.Clear(ctx)
				return nil
			})
		},
	}
}

// KillAllCMD terminates all processes with a given tag, sending either TERM or KILL.
func KillAllCMD() cli.Command {
	const (
		groupFlagName = "group"
		killFlagName  = "kill"
	)

	return cli.Command{
		Name:  "kill",
		Usage: "terminate processes",
		Flags: append(clientFlags(),
			cli.StringFlag{
				Name:  groupFlagName,
				Usage: "specify the group of process to kill",
			},
			cli.BoolFlag{
				Name:  "kill",
				Usage: "send KILL (9) rather than term (15)",
			},
		),
		Before: mergeBeforeFuncs(
			clientBefore(),
			func(c *cli.Context) error {
				if c.String(groupFlagName) == "" {
					return errors.Errorf("flag '--%s' was not specified", groupFlagName)
				}
				return nil
			}),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sendKill := c.Bool(killFlagName)
			group := c.String(groupFlagName)
			return withConnection(ctx, c, func(client jasper.RemoteClient) error {
				procs, err := client.Group(ctx, group)
				if err != nil {
					return errors.WithStack(err)
				}

				if sendKill {
					return errors.WithStack(jasper.KillAll(ctx, procs))
				}
				return errors.WithStack(jasper.TerminateAll(ctx, procs))
			})
		},
	}
}
