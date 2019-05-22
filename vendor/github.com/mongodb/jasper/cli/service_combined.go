package cli

import (
	"context"

	"github.com/mongodb/grip"
	"github.com/mongodb/jasper"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	restHostFlagName = "rest_host"
	restPortFlagName = "rest_port"

	rpcHostFlagName         = "rpc_host"
	rpcPortFlagName         = "rpc_port"
	rpcKeyFilePathFlagName  = "rpc_key_path"
	rpcCertFilePathFlagName = "rpc_cert_path"
)

func serviceCombined() cli.Command {
	return cli.Command{
		Name:  "combined",
		Usage: "start a combined multiprotocol service",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   restHostFlagName,
				EnvVar: envVarRESTHost,
				Usage:  "the host running the REST service ",
				Value:  defaultLocalHostName,
			},
			cli.IntFlag{
				Name:   restPortFlagName,
				EnvVar: envVarRPCPort,
				Usage:  "the port running the REST service ",
				Value:  defaultRESTPort,
			},
			cli.StringFlag{
				Name:   rpcHostFlagName,
				EnvVar: envVarRPCHost,
				Usage:  "the host running the RPC service ",
				Value:  defaultLocalHostName,
			},
			cli.IntFlag{
				Name:   rpcPortFlagName,
				EnvVar: envVarRPCPort,
				Usage:  "the port running the RPC service",
				Value:  defaultRPCPort,
			},
			cli.StringFlag{
				Name:  rpcCertFilePathFlagName,
				Usage: "the path to the RPC certificate file",
			},
			cli.StringFlag{
				Name:  rpcKeyFilePathFlagName,
				Usage: "the path to the RPC key file",
			},
		},
		Before: mergeBeforeFuncs(
			validatePort(restPortFlagName),
			validatePort(rpcPortFlagName),
		),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			go handleSignals(ctx, cancel)

			manager, err := jasper.NewLocalManager(false)
			if err != nil {
				return errors.Wrap(err, "failed to construct manager")
			}

			// Assemble the REST service.
			restHost := c.String(restHostFlagName)
			restPort := c.Int(restPortFlagName)
			grip.Infof("start REST service at '%s:%d'", restHost, restPort)
			closeRESTService, err := makeRESTService(ctx, restHost, restPort, manager)
			if err != nil {
				return errors.Wrap(err, "failed to create REST service")
			}
			defer func() {
				grip.Warning(errors.Wrap(closeRESTService(), "error stopping REST service"))
			}()

			// Assemble the RPC service.
			rpcHost := c.String(rpcHostFlagName)
			rpcPort := c.Int(rpcPortFlagName)
			grip.Infof("start RPC service at '%s:%d'", rpcHost, rpcPort)
			closeRPCService, err := makeRPCService(ctx, rpcHost, rpcPort, manager, c.String(rpcCertFilePathFlagName), c.String(rpcKeyFilePathFlagName))
			if err != nil {
				return errors.Wrap(err, "failed to create RPC service")
			}
			defer func() {
				grip.Warning(errors.Wrap(closeRPCService(), "error stopping RPC service"))
			}()

			// Wait for both services to shut down.
			<-ctx.Done()
			return nil
		},
	}
}
