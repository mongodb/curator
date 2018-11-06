package operations

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/mongodb/grip"
	"github.com/mongodb/jasper"
	"github.com/mongodb/jasper/jrpc"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

// Jasper is a process-management service provided as a component of
// curator.
func Jasper() cli.Command {
	return cli.Command{
		Name:  "jasper",
		Usage: "tools for running jasper process management services",
		Flags: []cli.Flag{},
		Subcommands: []cli.Command{
			jasperGRPC(),
			jasperREST(),
		},
	}
}

func jasperGRPC() cli.Command {
	return cli.Command{
		Name:  "grpc",
		Usage: "run jasper service accessible with gRPC",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:   "port",
				EnvVar: "JASPER_GRPC_PORT",
				Value:  2286,
			},
			cli.StringFlag{
				Name:   "host",
				EnvVar: "JASPER_GRPC_HOST",
				Value:  "localhost",
			},
		},
		Action: func(c *cli.Context) error {
			port := c.Int("port")
			host := c.String("host")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mngr := jasper.NewLocalManagerBlockingProcesses()

			addr := fmt.Sprintf("%s:%d", host, port)
			lis, err := net.Listen("tcp", addr)
			if err != nil {
				return errors.WithStack(err)
			}

			grip.Infof("starting jasper gRPC service at '%s'", addr)
			rpcSrv := grpc.NewServer()

			if err = jrpc.AttachService(mngr, rpcSrv); err != nil {
				return errors.WithStack(err)
			}

			go grip.Debug(rpcSrv.Serve(lis))

			wait := make(chan struct{})

			go func() {
				defer close(wait)
				sigChan := make(chan os.Signal, 2)
				signal.Notify(sigChan, syscall.SIGTERM)

				select {
				case <-ctx.Done():
					grip.Debug("context canceled")
				case <-sigChan:
					grip.Info("got signal, terminating")
				}

				rpcSrv.Stop()
				grip.Info("jasper service terminated")
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
				EnvVar: "JASPER_REST_PORT",
				Value:  2287,
			},
			cli.StringFlag{
				Name:   "host",
				EnvVar: "JASPER_REST_HOST",
				Value:  "localhost",
			},
		},
		Action: func(c *cli.Context) error {
			port := c.Int("port")
			host := c.String("host")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mngr := jasper.NewManagerService(jasper.NewLocalManagerBlockingProcesses())
			app := mngr.App()
			app.SetPrefix("jasper")

			if err := app.SetPort(port); err != nil {
				return errors.WithStack(err)
			}

			if err := app.SetHost(host); err != nil {
				return errors.WithStack(err)
			}

			grip.Infof("starting jasper gRPC service at 'localhost:%s/japser/v1'", port)

			if err := app.Run(ctx); err != nil {
				return errors.WithStack(err)
			}

			grip.Infof("jasper service completed")

			return nil
		},
	}
}
