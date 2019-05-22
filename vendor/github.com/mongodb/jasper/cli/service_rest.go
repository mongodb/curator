package cli

import (
	"context"

	"github.com/mongodb/grip"
	"github.com/mongodb/jasper"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	envVarRESTHost  = "JASPER_REST_HOST"
	envVarRESTPort  = "JASPER_REST_PORT"
	defaultRESTPort = 2287
)

func serviceREST() cli.Command {
	return cli.Command{
		Name:  "rest",
		Usage: "run a REST service",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   hostFlagName,
				EnvVar: envVarRESTHost,
				Usage:  "the host running the REST service",
				Value:  defaultLocalHostName,
			},
			cli.IntFlag{
				Name:   portFlagName,
				EnvVar: envVarRESTPort,
				Usage:  "the port running the REST service",
				Value:  defaultRESTPort,
			},
		},
		Before: validatePort(portFlagName),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			go handleSignals(ctx, cancel)

			manager, err := jasper.NewLocalManager(false)
			if err != nil {
				return errors.Wrap(err, "failed to construct manager")
			}

			host := c.String(hostFlagName)
			port := c.Int(portFlagName)
			grip.Infof("starting REST service at '%s:%d'", host, port)
			closeService, err := makeRESTService(ctx, host, port, manager)
			if err != nil {
				return errors.Wrap(err, "failed to create service")
			}
			defer func() {
				grip.Warning(errors.Wrap(closeService(), "error stopping service"))
			}()

			// Wait for service to shut down.
			<-ctx.Done()
			return nil
		},
	}
}

// makeRESTService creates a REST service around the manager serving requests on
// the host and port.
func makeRESTService(ctx context.Context, host string, port int, manager jasper.Manager) (jasper.CloseFunc, error) {
	service := jasper.NewManagerService(manager)
	app := service.App(ctx)
	app.SetPrefix("jasper")
	if err := app.SetHost(host); err != nil {
		return nil, errors.Wrap(err, "error setting REST host")
	}
	if err := app.SetPort(port); err != nil {
		return nil, errors.Wrap(err, "error setting REST port")
	}
	go func() {
		grip.Warning(errors.Wrap(app.Run(ctx), "error running REST app"))
	}()
	return func() error { return nil }, nil
}
