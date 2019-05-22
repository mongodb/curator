package cli

import (
	"context"
	"fmt"
	"net"

	"github.com/mongodb/grip"
	"github.com/mongodb/jasper"
	"github.com/mongodb/jasper/rpc"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	keyFilePathFlagName = "key_path"

	envVarRPCHost  = "JASPER_RPC_HOST"
	envVarRPCPort  = "JASPER_RPC_PORT"
	defaultRPCPort = 2286
)

func serviceRPC() cli.Command {
	return cli.Command{
		Name:  "rpc",
		Usage: "run an RPC service",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   hostFlagName,
				EnvVar: envVarRPCHost,
				Usage:  "the host running the RPC service",
				Value:  defaultLocalHostName,
			},
			cli.IntFlag{
				Name:   portFlagName,
				EnvVar: envVarRPCPort,
				Usage:  "the port running the RPC service",
				Value:  defaultRPCPort,
			},
			cli.StringFlag{
				Name:  keyFilePathFlagName,
				Usage: "the path to the certificate file",
			},
			cli.StringFlag{
				Name:  certFilePathFlagName,
				Usage: "the path to the key file",
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
			grip.Infof("starting RPC service at '%s:%d'", host, port)
			closeService, err := makeRPCService(ctx, host, port, manager, c.String(certFilePathFlagName), c.String(keyFilePathFlagName))
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

// makeRPCService creates an RPC service around the manager serving requests on
// the host and port.
func makeRPCService(ctx context.Context, host string, port int, manager jasper.Manager, certFilePath, keyFilePath string) (jasper.CloseFunc, error) {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve RPC address")
	}

	closeService, err := rpc.StartService(ctx, manager, addr, certFilePath, keyFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "error starting RPC service")
	}

	return closeService, nil
}
