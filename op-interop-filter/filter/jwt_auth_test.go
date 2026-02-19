package filter

import (
	"context"
	"crypto/rand"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum/log"
)

// testSupervisorAPI is a mock supervisor API for testing
type testSupervisorAPI struct{}

func (t *testSupervisorAPI) Ping(_ context.Context) string {
	return "pong"
}

// testAdminAPI is a mock admin API for testing
type testAdminAPI struct{}

func (t *testAdminAPI) GetFailsafeEnabled(_ context.Context) (bool, error) {
	return false, nil
}

func TestDedicatedAdminRPCServer(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	// Generate JWT secret
	var jwtSecret eth.Bytes32
	_, err := io.ReadFull(rand.Reader, jwtSecret[:])
	require.NoError(t, err)

	// Create filter server (public, no JWT)
	filterServer := oprpc.NewServer(
		"127.0.0.1",
		0,
		"test",
		oprpc.WithLogger(logger),
	)
	filterServer.AddAPI(rpc.API{
		Namespace: "supervisor",
		Service:   new(testSupervisorAPI),
	})

	// Create admin server (JWT-protected)
	adminServer := oprpc.NewServer(
		"127.0.0.1",
		0,
		"test",
		oprpc.WithLogger(logger),
		oprpc.WithJWTSecret(jwtSecret[:]),
	)
	adminServer.AddAPI(rpc.API{
		Namespace:     "admin",
		Service:       new(testAdminAPI),
		Authenticated: true,
	})

	require.NoError(t, filterServer.Start())
	t.Cleanup(func() {
		_ = filterServer.Stop()
	})

	require.NoError(t, adminServer.Start())
	t.Cleanup(func() {
		_ = adminServer.Stop()
	})

	filterEndpoint := "http://" + filterServer.Endpoint()
	adminEndpoint := "http://" + adminServer.Endpoint()

	// Create clients
	filterClient, err := rpc.Dial(filterEndpoint)
	require.NoError(t, err)
	t.Cleanup(filterClient.Close)

	adminUnauthClient, err := rpc.Dial(adminEndpoint)
	require.NoError(t, err)
	t.Cleanup(adminUnauthClient.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	adminAuthClient, err := client.NewRPC(
		ctx,
		logger,
		adminEndpoint,
		client.WithGethRPCOptions(rpc.WithHTTPAuth(gn.NewJWTAuth(jwtSecret))),
	)
	require.NoError(t, err)
	t.Cleanup(adminAuthClient.Close)

	t.Run("filter API works without JWT on its dedicated port", func(t *testing.T) {
		var res string
		err := filterClient.Call(&res, "supervisor_ping")
		require.NoError(t, err)
		require.Equal(t, "pong", res)
	})

	t.Run("admin API requires JWT on dedicated port - fails without auth", func(t *testing.T) {
		var res bool
		err := adminUnauthClient.Call(&res, "admin_getFailsafeEnabled")
		require.ErrorContains(t, err, "missing token")
	})

	t.Run("admin API works with valid JWT on dedicated port", func(t *testing.T) {
		var res bool
		err := adminAuthClient.CallContext(ctx, &res, "admin_getFailsafeEnabled")
		require.NoError(t, err)
		require.Equal(t, false, res)
	})
}

func TestFilterAPIWithoutAdminServer(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	// Create filter server WITHOUT admin server (simulates admin.rpc.addr not set)
	filterServer := oprpc.NewServer(
		"127.0.0.1",
		0,
		"test",
		oprpc.WithLogger(logger),
	)
	filterServer.AddAPI(rpc.API{
		Namespace: "supervisor",
		Service:   new(testSupervisorAPI),
	})

	require.NoError(t, filterServer.Start())
	t.Cleanup(func() {
		_ = filterServer.Stop()
	})

	endpoint := "http://" + filterServer.Endpoint()
	filterClient, err := rpc.Dial(endpoint)
	require.NoError(t, err)
	t.Cleanup(filterClient.Close)

	t.Run("filter API works without admin server configured", func(t *testing.T) {
		var res string
		err := filterClient.Call(&res, "supervisor_ping")
		require.NoError(t, err)
		require.Equal(t, "pong", res)
	})
}
