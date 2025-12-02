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
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
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
