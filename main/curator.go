package main

import (
	"os"

	"github.com/mongodb/curator/operations"
	"github.com/tychoish/grip"
	"github.com/urfave/cli"
)

func main() {
	// this is where the main action of the program starts. The
	// command line interface is managed by the cli package and
	// its objects/structures. This, plus the basic configuration
	// in buildApp(), is all that's necessary for bootstrapping the
	// environment.
	app := buildApp()
	err := app.Run(os.Args)
	grip.CatchErrorFatal(err)
}

// we build the app outside of main so that we can test the operation
func buildApp() *cli.App {
	app := cli.NewApp()
	app.Name = "curator"
	app.Usage = "a package repository generation tool."
	app.Version = "0.0.1-pre"

	// Register sub-commands here.
	app.Commands = []cli.Command{
		operations.HelloWorld(),
		operations.S3(),
		operations.Repo(),
		operations.Index(),
		operations.PruneCache(),
		operations.Artifacts(),
	}

	// These are global options. Use this to configure logging or
	// other options independent from specific sub commands.
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "level",
			Value: "info",
			Usage: "Specify lowest visible loglevel as string: 'emergency|alert|critical|error|warning|notice|info|debug'",
		},
	}

	app.Before = func(c *cli.Context) error {
		return loggingSetup(app.Name, c.String("level"))
	}

	return app
}

// logging setup is separate to make it unit testable
func loggingSetup(name, level string) error {
	// grip is a systemd/standard logging wrapper.
	grip.SetName(name)
	grip.SetThreshold(level)

	// This set's the logging system to write logging messages to
	// standard output.
	//
	// Could also call "grip.UseSystemdLogger()" to write log
	// messages directly to systemd's journald logger,
	// grip.UseFileLogger(<filename>), to write log messages to a
	// file, among other possible logging backends.
	return grip.UseNativeLogger()
}
