package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/recovery"
	"github.com/urfave/cli"
)

// Service encapsulates the functionality to set up Jasper services.
func Service() cli.Command {
	return cli.Command{
		Name:  "service",
		Usage: "tools for running Jasper services",
		Flags: []cli.Flag{},
		Subcommands: []cli.Command{
			serviceRPC(),
			serviceREST(),
			serviceCombined(),
		},
	}
}

func handleSignals(ctx context.Context, cancel context.CancelFunc) {
	defer recovery.LogStackTraceAndContinue("graceful shutdown")
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	select {
	case <-sigChan:
		grip.Debug("received signal")
	case <-ctx.Done():
		grip.Debug("context canceled")
	}
}
