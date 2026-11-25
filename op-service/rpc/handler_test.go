package rpc

import (
	"context"
	"crypto/rand"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestHandler(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	h := NewHandler("v1.2.3", WithLogger(logger))
	t.Cleanup(h.Stop)

	rpcEntry := rpc.API{
		Namespace: "foo",
		Service:   new(testAPI),
	}

	require.ErrorContains(t, h.AddRPC("/"), "suffix")
	require.ErrorContains(t, h.AddRPC(""), "already exists")
	require.ErrorContains(t, h.AddAPIToRPC("/extra", rpcEntry), "not found")
	require.NoError(t, h.AddRPC("/extra"))
	require.NoError(t, h.AddAPIToRPC("/extra", rpcEntry))

	// WS-RPC / HTTP-RPC / health are tested in server_test.go
}

func TestHandlerAuthentication(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	// generate JWT Secret
	var jwtSecret eth.Bytes32
	_, err := io.ReadFull(rand.Reader, jwtSecret[:])
	require.NoError(t, err)

	server := ServerFromConfig(&ServerConfig{
		RpcOptions: []Option{
			WithLogger(logger),
			WithWebsocketEnabled(),
			WithJWTSecret(jwtSecret[:]),
		},
		Host:       "127.0.0.1",
		Port:       0,
		AppVersion: "test",
	})

	namespace := "test"
	server.AddAPI(rpc.API{
		Namespace: namespace,
		Service:   new(testAPI),
	})

	isAuthenticated := false
	require.NoError(t, server.Handler.AddRPCWithAuthentication("/public", &isAuthenticated))
	require.NoError(t, server.AddAPIToRPC("/public", rpc.API{
		Namespace: namespace,
		Service:   new(testAPI),
	}))
	require.NoError(t, server.Start(), "must start")

	t.Cleanup(func() {
		err := server.Stop()
		if err != nil {
			panic(err)
		}
	})

	endpoint := "http://" + server.Endpoint()
	publicClient, err := rpc.Dial(endpoint + "/public")
	require.NoError(t, err)
	t.Cleanup(publicClient.Close)

	defaultUnauthenticatedClient, err := rpc.Dial(endpoint)
	require.NoError(t, err)
	t.Cleanup(defaultUnauthenticatedClient.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	defaultAuthenticatedClient, err := client.NewRPC(ctx, logger, endpoint, client.WithGethRPCOptions(rpc.WithHTTPAuth(gn.NewJWTAuth(jwtSecret))))
	require.NoError(t, err)
	t.Cleanup(defaultAuthenticatedClient.Close)

	t.Run("public RPC", func(t *testing.T) {
		var res int
		require.NoError(t, publicClient.Call(&res, namespace+"_frobnicate", 2))
		require.Equal(t, 4, res)
	})

	t.Run("default RPC - unauthenticated", func(t *testing.T) {
		var res int
		require.ErrorContains(t, defaultUnauthenticatedClient.Call(&res, namespace+"_frobnicate", 2), "missing token")
	})

	t.Run("default RPC - authenticated", func(t *testing.T) {
		var res int
		require.NoError(t, defaultAuthenticatedClient.CallContext(ctx, &res, namespace+"_frobnicate", 6))
		require.Equal(t, 12, res)
	})
}

// TestHandlerWithHTTPBodyLimit tests that the HTTP body limit option is correctly applied
func TestHandlerWithHTTPBodyLimit(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	tests := []struct {
		name              string
		bodyLimit         int
		expectedBodyLimit int
	}{
		{
			name:              "with 10MB limit",
			bodyLimit:         10 * 1024 * 1024,
			expectedBodyLimit: 10 * 1024 * 1024,
		},
		{
			name:              "with 64MB limit",
			bodyLimit:         64 * 1024 * 1024,
			expectedBodyLimit: 64 * 1024 * 1024,
		},
		{
			name:              "with 5MB limit (default)",
			bodyLimit:         5 * 1024 * 1024,
			expectedBodyLimit: 5 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler("v1.2.3", WithLogger(logger), WithHTTPBodyLimit(tt.bodyLimit))
			t.Cleanup(h.Stop)

			// Verify the HTTP body limit is set correctly
			require.Equal(t, tt.expectedBodyLimit, h.httpBodyLimit,
				"HTTP body limit should be %d bytes", tt.expectedBodyLimit)
		})
	}
}

// TestHandlerWithoutHTTPBodyLimit tests that handler works without body limit configured
func TestHandlerWithoutHTTPBodyLimit(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	h := NewHandler("v1.2.3", WithLogger(logger))
	t.Cleanup(h.Stop)

	// Verify the HTTP body limit is not set (0 means not configured)
	require.Equal(t, 0, h.httpBodyLimit,
		"HTTP body limit should be 0 when not configured")
}
