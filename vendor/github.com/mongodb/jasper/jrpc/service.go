package jrpc

import (
	"github.com/mongodb/jasper"
	"github.com/mongodb/jasper/jrpc/internal"
	"github.com/pkg/errors"
	grpc "google.golang.org/grpc"
)

func AttachService(manager jasper.Manager, s *grpc.Server) error {
	return errors.WithStack(internal.AttachService(manager, s))
}
