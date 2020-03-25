package remote

import (
	"context"
	"io"

	"github.com/evergreen-ci/mrpc/mongowire"
)

const (
	LoggingCacheSizeCommand   = "logging_cache_size"
	LoggingCacheCreateCommand = "create_logging_cache"
	LoggingCacheDeleteCommand = "delete_logging_cache"
	LoggingCacheGetCommand    = "get_logging_cache"
	LoggingSendMessageCommand = "send_message"
)

func (s *mdbService) loggingSize(ctx context.Context, w io.Writer, msg mongowire.Message)        {}
func (s *mdbService) loggingCreate(ctx context.Context, w io.Writer, msg mongowire.Message)      {}
func (s *mdbService) loggingGet(ctx context.Context, w io.Writer, msg mongowire.Message)         {}
func (s *mdbService) loggingDelete(ctx context.Context, w io.Writer, msg mongowire.Message)      {}
func (s *mdbService) loggingSendMessage(ctx context.Context, w io.Writer, msg mongowire.Message) {}
