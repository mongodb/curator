package rpc

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/mongodb/jasper"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// startTestService creates a server for testing purposes that terminates when
// the context is done.
func startTestService(ctx context.Context, mngr jasper.Manager, addr net.Addr, creds *Credentials) error {
	closeService, err := StartService(ctx, mngr, addr, creds)
	if err != nil {
		return errors.Wrap(err, "could not start server")
	}

	go func() {
		<-ctx.Done()
		closeService()
	}()

	return nil
}

// newTestClient establishes a client for testing purposes that closes when
// the context is done.
func newTestClient(ctx context.Context, addr net.Addr, creds *Credentials) (jasper.RemoteClient, error) {
	client, err := NewClient(ctx, addr, creds)
	if err != nil {
		return nil, errors.Wrap(err, "could not get client")
	}

	go func() {
		<-ctx.Done()
		client.CloseConnection()
	}()

	return client, nil
}

// buildDir gets the Jasper build directory.
func buildDir(t *testing.T) string {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(filepath.Dir(cwd), "build")
}
