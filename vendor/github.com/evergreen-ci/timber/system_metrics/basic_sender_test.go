package systemmetrics

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/evergreen-ci/timber"
	"github.com/evergreen-ci/timber/internal"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type mockClient struct {
	createErr bool
	addErr    bool
	closeErr  bool
	info      *internal.SystemMetrics
	data      *internal.SystemMetricsData
	close     *internal.SystemMetricsSeriesEnd
}

func (mc *mockClient) CreateSystemMetricRecord(_ context.Context, in *internal.SystemMetrics, opts ...grpc.CallOption) (*internal.SystemMetricsResponse, error) {
	if mc.createErr {
		return nil, errors.New("create error")
	}
	mc.info = in
	return &internal.SystemMetricsResponse{
		Id: "ID",
	}, nil
}

func (mc *mockClient) AddSystemMetrics(_ context.Context, in *internal.SystemMetricsData, opts ...grpc.CallOption) (*internal.SystemMetricsResponse, error) {
	if mc.addErr {
		return nil, errors.New("add error")
	}
	mc.data = in
	return &internal.SystemMetricsResponse{
		Id: "ID",
	}, nil
}

func (mc *mockClient) StreamSystemMetrics(_ context.Context, opts ...grpc.CallOption) (internal.CedarSystemMetrics_StreamSystemMetricsClient, error) {
	return nil, errors.New("Not implemented")
}

func (mc *mockClient) CloseMetrics(_ context.Context, in *internal.SystemMetricsSeriesEnd, opts ...grpc.CallOption) (*internal.SystemMetricsResponse, error) {
	if mc.closeErr {
		return nil, errors.New("close error")
	}
	mc.close = in
	return &internal.SystemMetricsResponse{
		Id: "ID",
	}, nil
}

type mockServer struct {
	createErr bool
	addErr    bool
	closeErr  bool
	info      bool
	data      bool
	close     bool
}

func (mc *mockServer) CreateSystemMetricRecord(_ context.Context, in *internal.SystemMetrics) (*internal.SystemMetricsResponse, error) {
	if mc.createErr {
		return nil, errors.New("create error")
	}
	mc.info = true
	return &internal.SystemMetricsResponse{
		Id: "ID",
	}, nil
}

func (mc *mockServer) AddSystemMetrics(_ context.Context, in *internal.SystemMetricsData) (*internal.SystemMetricsResponse, error) {
	if mc.addErr {
		return nil, errors.New("add error")
	}
	mc.data = true
	return &internal.SystemMetricsResponse{
		Id: "ID",
	}, nil
}

func (mc *mockServer) StreamSystemMetrics(internal.CedarSystemMetrics_StreamSystemMetricsServer) error {
	return nil
}

func (mc *mockServer) CloseMetrics(_ context.Context, in *internal.SystemMetricsSeriesEnd) (*internal.SystemMetricsResponse, error) {
	if mc.closeErr {
		return nil, errors.New("close error")
	}
	mc.close = true
	return &internal.SystemMetricsResponse{
		Id: "ID",
	}, nil
}

func TestNewSystemMetricsClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := &mockServer{}
	require.NoError(t, startRPCService(ctx, srv, 5000))
	t.Run("ValidOptions", func(t *testing.T) {
		connOpts := ConnectionOptions{
			Client: http.Client{},
			DialOpts: timber.DialCedarOptions{
				BaseAddress: "localhost",
				RPCPort:     "5000",
			},
		}
		client, err := NewSystemMetricsClient(ctx, connOpts)
		require.NoError(t, err)
		require.NotNil(t, client)
		require.NoError(t, client.CloseSystemMetrics(ctx, "ID"))
		assert.True(t, srv.close)
	})
	t.Run("InvalidOptions", func(t *testing.T) {
		connOpts := ConnectionOptions{}
		client, err := NewSystemMetricsClient(ctx, connOpts)
		require.Error(t, err)
		require.Nil(t, client)
	})
}

func TestNewSystemMetricsClientWithExistingClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := &mockServer{}
	require.NoError(t, startRPCService(ctx, srv, 6000))
	addr := fmt.Sprintf("localhost:%d", 6000)
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	require.NoError(t, err)

	t.Run("ValidOptions", func(t *testing.T) {
		client, err := NewSystemMetricsClientWithExistingConnection(ctx, conn)
		require.NoError(t, err)
		require.NotNil(t, client)
		require.NoError(t, client.CloseSystemMetrics(ctx, "ID"))
		assert.True(t, srv.close)
	})
	t.Run("InvalidOptions", func(t *testing.T) {
		client, err := NewSystemMetricsClientWithExistingConnection(ctx, nil)
		require.Error(t, err)
		require.Nil(t, client)
	})
}

func TestCloseClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := &mockServer{}
	require.NoError(t, startRPCService(ctx, srv, 7000))
	t.Run("WithoutExistingConnection", func(t *testing.T) {
		connOpts := ConnectionOptions{
			Client: http.Client{},
			DialOpts: timber.DialCedarOptions{
				BaseAddress: "localhost",
				RPCPort:     "7000",
			},
		}
		client, err := NewSystemMetricsClient(ctx, connOpts)
		require.NoError(t, err)
		require.NotNil(t, client)
		require.NoError(t, client.CloseSystemMetrics(ctx, "ID"))
		assert.True(t, srv.close)

		require.NoError(t, client.CloseClient())
		require.Error(t, client.CloseSystemMetrics(ctx, "ID"))
	})
	t.Run("WithExistingConnection", func(t *testing.T) {
		addr := fmt.Sprintf("localhost:%d", 7000)
		conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
		require.NoError(t, err)
		client, err := NewSystemMetricsClientWithExistingConnection(ctx, conn)
		require.NoError(t, err)
		require.NoError(t, client.CloseSystemMetrics(ctx, "ID"))
		assert.True(t, srv.close)

		require.NoError(t, client.CloseClient())
		require.NoError(t, client.CloseSystemMetrics(ctx, "ID"))
		require.NoError(t, conn.Close())
	})
}

func TestCreateSystemMetricsRecord(t *testing.T) {
	ctx := context.Background()
	t.Run("ValidOptions", func(t *testing.T) {
		mc := &mockClient{}
		s := SystemMetricsClient{
			client: mc,
		}
		id, err := s.CreateSystemMetricRecord(ctx, SystemMetricsOptions{
			Project:     "project",
			Version:     "version",
			Variant:     "variant",
			TaskName:    "taskname",
			TaskId:      "taskid",
			Execution:   1,
			Mainline:    true,
			Compression: CompressionTypeNone,
			Schema:      SchemaTypeRawEvents,
		})
		require.NoError(t, err)
		assert.Equal(t, id, "ID")
		assert.Equal(t, &internal.SystemMetrics{
			Info: &internal.SystemMetricsInfo{
				Project:   "project",
				Version:   "version",
				Variant:   "variant",
				TaskName:  "taskname",
				TaskId:    "taskid",
				Execution: 1,
				Mainline:  true,
			},
			Artifact: &internal.SystemMetricsArtifactInfo{
				Compression: internal.CompressionType(CompressionTypeNone),
				Schema:      internal.SchemaType(SchemaTypeRawEvents),
			},
		}, mc.info)
	})
	t.Run("InvalidOptions", func(t *testing.T) {
		mc := &mockClient{}
		s := SystemMetricsClient{
			client: mc,
		}
		id, err := s.CreateSystemMetricRecord(ctx, SystemMetricsOptions{
			Project:     "project",
			Version:     "version",
			Variant:     "variant",
			TaskName:    "taskname",
			TaskId:      "taskid",
			Execution:   1,
			Mainline:    true,
			Compression: 6,
			Schema:      SchemaTypeRawEvents,
		})
		require.Error(t, err)
		assert.Equal(t, id, "")
		assert.Nil(t, mc.data)
		id, err = s.CreateSystemMetricRecord(ctx, SystemMetricsOptions{
			Project:     "project",
			Version:     "version",
			Variant:     "variant",
			TaskName:    "taskname",
			TaskId:      "taskid",
			Execution:   1,
			Mainline:    true,
			Compression: CompressionTypeNone,
			Schema:      6,
		})
		require.Error(t, err)
		assert.Equal(t, id, "")
		assert.Nil(t, mc.data)
	})
	t.Run("RPCError", func(t *testing.T) {
		mc := &mockClient{
			createErr: true,
		}
		s := SystemMetricsClient{
			client: mc,
		}
		id, err := s.CreateSystemMetricRecord(ctx, SystemMetricsOptions{
			Project:     "project",
			Version:     "version",
			Variant:     "variant",
			TaskName:    "taskname",
			TaskId:      "taskid",
			Execution:   1,
			Mainline:    true,
			Compression: CompressionTypeNone,
			Schema:      SchemaTypeRawEvents,
		})
		require.Error(t, err)
		assert.Equal(t, id, "")
		assert.Nil(t, mc.data)
	})
}

func TestAddSystemMetrics(t *testing.T) {
	ctx := context.Background()
	t.Run("ValidOptions", func(t *testing.T) {
		mc := &mockClient{}
		s := SystemMetricsClient{
			client: mc,
		}
		require.NoError(t, s.AddSystemMetrics(ctx, "ID", []byte("Test byte string")))
		assert.Equal(t, &internal.SystemMetricsData{
			Id:   "ID",
			Data: []byte("Test byte string"),
		}, mc.data)
	})
	t.Run("InvalidOptions", func(t *testing.T) {
		mc := &mockClient{}
		s := SystemMetricsClient{
			client: mc,
		}
		require.Error(t, s.AddSystemMetrics(ctx, "", []byte("Test byte string")))
		assert.Nil(t, mc.data)
		require.Error(t, s.AddSystemMetrics(ctx, "ID", []byte{}))
		assert.Nil(t, mc.data)
	})
	t.Run("RPCError", func(t *testing.T) {
		mc := &mockClient{
			addErr: true,
		}
		s := SystemMetricsClient{
			client: mc,
		}
		require.Error(t, s.AddSystemMetrics(ctx, "ID", []byte("Test byte string")))
		assert.Nil(t, mc.data)
	})
}

func TestCloseSystemMetrics(t *testing.T) {
	ctx := context.Background()
	t.Run("ValidOptions", func(t *testing.T) {
		mc := &mockClient{}
		s := SystemMetricsClient{
			client: mc,
		}
		require.NoError(t, s.CloseSystemMetrics(ctx, "ID"))
		assert.Equal(t, &internal.SystemMetricsSeriesEnd{
			Id: "ID",
		}, mc.close)
	})
	t.Run("InvalidOptions", func(t *testing.T) {
		mc := &mockClient{}
		s := SystemMetricsClient{
			client: mc,
		}
		require.Error(t, s.CloseSystemMetrics(ctx, ""))
		assert.Nil(t, mc.data)
	})
	t.Run("RPCError", func(t *testing.T) {
		mc := &mockClient{
			closeErr: true,
		}
		s := SystemMetricsClient{
			client: mc,
		}
		require.Error(t, s.CloseSystemMetrics(ctx, "ID"))
		assert.Nil(t, mc.data)
	})
}

func startRPCService(ctx context.Context, service internal.CedarSystemMetricsServer, port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return errors.WithStack(err)
	}

	s := grpc.NewServer()
	internal.RegisterCedarSystemMetricsServer(s, service)

	go func() {
		_ = s.Serve(lis)
	}()
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	return nil
}
