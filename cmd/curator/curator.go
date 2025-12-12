package main

import (
	"os"

	"github.com/mongodb/curator"
	"github.com/mongodb/curator/operations"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/level"
	"github.com/mongodb/grip/send"
	jaspercli "github.com/mongodb/jasper/cli"
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
	grip.EmergencyFatal(err)
}

// we build the app outside of main so that we can test the operation
func buildApp() *cli.App {
	app := cli.NewApp()
	app.Name = "curator"
	app.Usage = "build and release tool"
	app.Version = curator.BuildRevision

	// Register sub-commands here.
	app.Commands = []cli.Command{
		operations.HelloWorld(),
		operations.Version(),
		operations.Repo(),
		operations.S3(),
		operations.Archive(),
		operations.PruneCache(),
		operations.Artifacts(),
		operations.SystemInfo(),
		operations.BuildLogger(),
		operations.Splunk(),
		operations.Notify(),
		operations.Greenbay(),
		jaspercli.Jasper(),
		operations.FTDC(),
		operations.Backup(),
		operations.CalculateRollups(),
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

func loggingSetup(name, l string) error {
	if err := grip.SetSender(send.MakeErrorLogger()); err != nil {
		return err
	}
	grip.SetName(name)

	sender := grip.GetSender()
	info := sender.Level()
	info.Threshold = level.FromString(l)

	return sender.SetLevel(info)
}
