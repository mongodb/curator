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
	grip.CatchEmergencyFatal(err)
}

// we build the app outside of main so that we can test the operation
func buildApp() *cli.App {
	app := cli.NewApp()
	app.Name = "curator"
	app.Usage = "build and release tool"
	app.Version = "0.0.1-pre"

	// Register sub-commands here.
	app.Commands = []cli.Command{
		operations.HelloWorld(),
		operations.S3(),
		operations.Repo(),
		operations.Index(),
		operations.PruneCache(),
		operations.Artifacts(),
		operations.SystemInfo(),
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
		loggingSetup(app.Name, c.String("level"))
		return nil
	}

	return app
}

func loggingSetup(name, level string) {
	grip.SetName(name)
	grip.SetThreshold(level)
}
