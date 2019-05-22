package operations

import (
	"context"
	"fmt"
	"net"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/recovery"
	"github.com/mongodb/jasper"
	"github.com/mongodb/jasper/rpc"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

const (
	envVarJasperRPCPort   = "JASPER_RPC_PORT"
	envVarJasperRPCHost   = "JASPER_RPC_HOST"
	envVarJasperRESTPort  = "JASPER_REST_PORT"
	envVarJasperRESTHost  = "JASPER_REST_HOST"
	defaultJasperRPCPort  = 2286
	defaultJasperRESTPort = 2287
	defaultLocalHostName  = "localhost"
)

// Jasper is a process-management service provided as a component of
// curator.
func Jasper() cli.Command {
	return cli.Command{
		Name:  "jasper",
		Usage: "tools for running jasper process management services",
		Flags: []cli.Flag{},
		Subcommands: []cli.Command{
			jasperRPC(),
			jasperREST(),
			jasperCombined(),
		},
	}
}

func jasperCombined() cli.Command {
	return cli.Command{
		Name:  "service",
		Usage: "starts a combined multiprotocol jasper service",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:   "rpcPort",
				EnvVar: envVarJasperRPCPort,
				Value:  defaultJasperRPCPort,
			},
			cli.StringFlag{
				Name:   "rpcHost",
				EnvVar: envVarJasperRPCHost,
				Value:  defaultLocalHostName,
			},
			cli.IntFlag{
				Name:   "restPort",
				EnvVar: envVarJasperRPCPort,
				Value:  defaultJasperRESTPort,
			},
			cli.StringFlag{
				Name:   "restHost",
				EnvVar: envVarJasperRESTHost,
				Value:  defaultLocalHostName,
			},
		},
		Action: func(c *cli.Context) error {
			restHost := c.String("restHost")
			restPort := c.Int("restPort")
			rpcAddr := fmt.Sprintf("%s:%d", c.String("rpcHost"), c.Int("rpcPort"))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mngr, err := jasper.NewLocalManager(false)
			if err != nil {
				return errors.Wrap(err, "problem constructing manager")
			}

			// assemble the rest service
			rest := jasper.NewManagerService(mngr).App(ctx)
			rest.SetPrefix("jasper")
			if err = rest.SetPort(restPort); err != nil {
				return errors.WithStack(err)
			}

			if err = rest.SetHost(restHost); err != nil {
				return errors.WithStack(err)
			}

			// assemble the rpc service
			rpcSrv := grpc.NewServer()
			if err = rpc.AttachService(mngr, rpcSrv); err != nil {
				return errors.WithStack(err)
			}

			lis, err := net.Listen("tcp", rpcAddr)
			if err != nil {
				return errors.WithStack(err)
			}

			// start threads to handle services
			go signalListener(ctx, cancel)
			grip.Infof("starting jasper gRPC service on %s", rpcAddr)
			go func() { grip.Warning(rpcSrv.Serve(lis)) }()

			rpcWait := make(chan struct{})
			go func() {
				defer close(rpcWait)
				defer recovery.LogStackTraceAndContinue("waiting for rpc service")
				<-ctx.Done()
				rpcSrv.Stop()
				grip.Info("jasper rpc service terminated")
			}()

			// the rest application's Run method handle's
			// its own graceful shutdown.
			grip.Infof("starting jasper REST service on %s:%d", restHost, restPort)
			if err = rest.Run(ctx); err != nil {
				return errors.Wrap(err, "problem with rest service")
			}

			// wait for servers to shutdown
			<-rpcWait
			return nil
		},
	}

}

func jasperRPC() cli.Command {
	return cli.Command{
		Name:  "grpc",
		Usage: "run jasper service accessible with gRPC",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:   "port",
				EnvVar: envVarJasperRPCPort,
				Value:  defaultJasperRPCPort,
			},
			cli.StringFlag{
				Name:   "host",
				EnvVar: envVarJasperRPCHost,
				Value:  defaultLocalHostName,
			},
		},
		Action: func(c *cli.Context) error {
			port := c.Int("port")
			host := c.String("host")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mngr, err := jasper.NewLocalManager(false)
			if err != nil {
				return errors.Wrap(err, "problem constructing manager")
			}

			addr := fmt.Sprintf("%s:%d", host, port)
			lis, err := net.Listen("tcp", addr)
			if err != nil {
				return errors.WithStack(err)
			}

			grip.Infof("starting jasper gRPC service at '%s'", addr)
			rpcSrv := grpc.NewServer()

			if err = rpc.AttachService(mngr, rpcSrv); err != nil {
				return errors.WithStack(err)
			}

			go signalListener(ctx, cancel)
			go func() { grip.Warning(rpcSrv.Serve(lis)) }()

			wait := make(chan struct{})

			go func() {
				defer close(wait)
				defer recovery.LogStackTraceAndContinue("waiting for rpc service")
				<-ctx.Done()
				rpcSrv.Stop()
				grip.Info("jasper rpc service terminated")
			}()

			<-wait
			return nil
		},
	}
}

func jasperREST() cli.Command {
	return cli.Command{
		Name:  "rest",
		Usage: "run jasper service accessible with a REST interface",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:   "port",
				EnvVar: envVarJasperRESTPort,
				Value:  defaultJasperRESTPort,
			},
			cli.StringFlag{
				Name:   "host",
				EnvVar: envVarJasperRESTHost,
				Value:  defaultLocalHostName,
			},
		},
		Action: func(c *cli.Context) error {
			port := c.Int("port")
			host := c.String("host")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m, err := jasper.NewLocalManager(false)
			if err != nil {
				return errors.Wrap(err, "problem constructing base manager")
			}

			mngr := jasper.NewManagerService(m)
			app := mngr.App(ctx)
			app.SetPrefix("jasper")

			if err := app.SetPort(port); err != nil {
				return errors.WithStack(err)
			}

			if err := app.SetHost(host); err != nil {
				return errors.WithStack(err)
			}

			grip.Infof("starting jasper REST service at '%s:%d'", host, port)

			if err := app.Run(ctx); err != nil {
				return errors.WithStack(err)
			}

			grip.Infof("jasper service completed")

			return nil
		},
	}
}
