package operations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/amboy/rest"
	"github.com/mongodb/curator/greenbay"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
	"github.com/urfave/cli"

	// we need the init() methods in the check package to fire
	_ "github.com/mongodb/curator/greenbay/check"
)

// Greenbay returns the sub-command object for Greenbay, a system
// integration and acceptance testing platform.
func Greenbay() cli.Command {
	return cli.Command{
		Name:  "greenbay",
		Usage: "provides system configuration integration and acceptance testing",
		Subcommands: []cli.Command{
			list(),
			checks(),
			service(),
			client(),
		},
	}
}

func addConfArg(a ...cli.Flag) []cli.Flag {
	cwd, _ := os.Getwd()
	configPath := filepath.Join(cwd, "greenbay.yaml")

	return append(a,
		cli.StringFlag{
			Name: "conf",
			Usage: fmt.Sprintln("path to config file. '.json', '.yaml', and '.yml' extensions ",
				"supported.", "Default path:", configPath),
			Value: configPath,
		})

}

func addArgs(a ...cli.Flag) []cli.Flag {
	args := append(a,
		cli.StringSliceFlag{
			Name:  "test",
			Usage: "specify a check, by name. may specify multiple times",
		},
		cli.StringSliceFlag{
			Name:  "suite",
			Usage: "specify a suite or suites, by name. if not specified, runs the 'all' suite",
		},
		cli.StringFlag{
			Name:  "output",
			Usage: "path of file to write output too. Defaults to *not* writing output to a file",
			Value: "",
		},
		cli.BoolFlag{
			Name:  "quiet",
			Usage: "specify to only print failing tests",
		},
		cli.StringFlag{
			Name: "format",
			Usage: fmt.Sprintln("Selects the output format, defaults to a format that mirrors gotest,",
				"but also supports evergreen's results format.",
				"Use 'gotest' (default), 'json', 'result', or 'log'."),
			Value: "gotest",
		})

	return addConfArg(args...)
}

////////////////////////////////////////////////////////////////////////
//
// Define SubCommands
//
////////////////////////////////////////////////////////////////////////

func list() cli.Command {
	return cli.Command{
		Name:  "list",
		Usage: "list all available checks",
		Action: func(c *cli.Context) error {
			var list []string
			for name := range registry.JobTypeNames() {
				list = append(list, name)
			}

			if len(list) == 0 {
				return errors.New("no jobs registered")
			}

			sort.Strings(list)
			fmt.Printf("Registered Greenbay Checks:\n\t%s\n",
				strings.Join(list, "\n\t"))

			grip.Infof("%d checks registered", len(list))
			return nil
		},
	}
}

func checks() cli.Command {
	defaultNumJobs := runtime.NumCPU()

	return cli.Command{
		Name:  "run",
		Usage: "run greenbay suites",
		Flags: addArgs(
			cli.IntFlag{
				Name: "jobs",
				Usage: fmt.Sprintf("specify the number of parallel tests to run. (Default %d)",
					defaultNumJobs),
				Value: defaultNumJobs,
			}),
		Action: func(c *cli.Context) error {
			// Note: in the future in may make sense to
			// use this context to timeout the work of the
			// underlying processes.
			ctx := context.Background()

			suites := c.StringSlice("suite")
			tests := c.StringSlice("test")
			if len(suites) == 0 && len(tests) == 0 {
				suites = append(suites, "all")
			}

			app, err := greenbay.NewApplication(
				c.String("conf"),
				c.String("output"),
				c.String("format"),
				c.Bool("quiet"),
				c.Int("jobs"),
				suites,
				tests)

			if err != nil {
				return errors.Wrap(err, "problem prepping to run tests")
			}

			return errors.Wrap(app.Run(ctx), "problem running tests")
		},
	}
}

func service() cli.Command {
	return cli.Command{
		Name:  "service",
		Usage: "run a amboy service with greenbay checks loaded.",
		Flags: addConfArg(
			cli.IntFlag{
				Name:  "port",
				Usage: "http port to run service on",
				Value: 3000,
			},
			cli.StringFlag{
				Name: "host",
				Usage: fmt.Sprintln("host for the remote greenbay instance. ",
					"Defaults to '' which listens on all ports."),
				Value: "",
			},
			cli.IntFlag{
				Name:  "cache",
				Usage: "number of jobs to store",
				Value: 1000,
			},
			cli.IntFlag{
				Name:  "jobs",
				Usage: "specify the number of parallel tests to run.",
				Value: 2,
			},
			cli.StringFlag{
				Name: "logOutput, o",
				Usage: fmt.Sprintln("specify the logging format, choices are:",
					"[stdout, file, json-stdout, json-file, systemd, syslog]"),
			},
			cli.StringFlag{
				Name:  "file, f",
				Usage: "specify the file to write the log to, for file-based output methods",
			},
			cli.BoolFlag{
				Name:  "disableStats",
				Usage: "disable the sysinfo and process tree stats endpoints",
			}),
		Action: func(c *cli.Context) error {
			grip.CatchEmergencyFatal(greenbay.SetupLogging(c.String("logOutput"), c.String("file")))

			ctx := context.Background()
			info := rest.ServiceInfo{QueueSize: c.Int("cache"), NumWorkers: c.Int("jobs")}

			s, err := greenbay.NewService(c.String("conf"), c.String("host"), c.Int("port"))
			grip.CatchEmergencyFatal(err)

			s.DisableStats = c.Bool("disableStats")

			grip.Info("starting greenbay workers")
			grip.EmergencyFatal(s.Open(ctx, info))
			defer s.Close()

			grip.Infof("starting service on port %d", c.Int("port"))
			s.Run()
			grip.Info("service shutting down")

			return nil
		},
	}
}

func client() cli.Command {
	return cli.Command{
		Name:  "client",
		Usage: "run a check or checks on a remote greenbay service",
		Flags: addArgs(
			cli.StringFlag{
				Name:  "host",
				Usage: "host for the remote greenbay instance.",
				Value: "http://localhost",
			},
			cli.IntFlag{
				Name:  "port",
				Usage: "port for the remote greenbay service.",
				Value: 80,
			}),
		Action: func(c *cli.Context) error {
			// Note: in the future in may make sense to
			// use this context to timeout the work of the
			// underlying processes.
			ctx := context.Background()

			suites := c.StringSlice("suite")
			tests := c.StringSlice("test")

			if len(suites) == 0 && len(tests) == 0 {
				suites = append(suites, "all")
			}

			app, err := greenbay.NewClient(
				c.String("conf"),
				c.String("host"),
				c.Int("port"),
				c.String("output"),
				c.String("format"),
				c.Bool("quiet"),
				suites,
				tests)

			if err != nil {
				return errors.Wrap(err, "problem constructing client to run tasks")
			}

			return errors.Wrap(app.Run(ctx), "problem running tests remotely")
		},
	}

}
