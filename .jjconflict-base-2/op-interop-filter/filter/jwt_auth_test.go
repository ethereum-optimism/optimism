package filter

import (
	"context"
	"crypto/rand"
	"io"
	"net/http"
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

// newTestJWTProtectedAdminHandler creates a JWT-protected handler for testing
func newTestJWTProtectedAdminHandler(jwtSecret []byte) http.Handler {
	srv := rpc.NewServer()
	if err := srv.RegisterName("admin", new(testAdminAPI)); err != nil {
		panic(err)
	}
	return gn.NewHTTPHandlerStack(srv, []string{"*"}, []string{"*"}, jwtSecret)
}

func TestJWTAuthentication(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	// Generate JWT secret
	var jwtSecret eth.Bytes32
	_, err := io.ReadFull(rand.Reader, jwtSecret[:])
	require.NoError(t, err)

	// Create server WITHOUT JWT on root - root stays public for supervisor API
	server := oprpc.ServerFromConfig(&oprpc.ServerConfig{
		RpcOptions: []oprpc.Option{
			oprpc.WithLogger(logger),
			// No WithJWTSecret - root stays public
		},
		Host:       "127.0.0.1",
		Port:       0,
		AppVersion: "test",
	})

	// Register supervisor API on root (public)
	server.AddAPI(rpc.API{
		Namespace: "supervisor",
		Service:   new(testSupervisorAPI),
	})

	// Register admin API with JWT protection using AddHandler
	server.AddHandler("/admin", newTestJWTProtectedAdminHandler(jwtSecret[:]))

	require.NoError(t, server.Start())
	t.Cleanup(func() {
		_ = server.Stop()
	})

	endpoint := "http://" + server.Endpoint()

	// Create clients
	supervisorClient, err := rpc.Dial(endpoint)
	require.NoError(t, err)
	t.Cleanup(supervisorClient.Close)

	adminUnauthClient, err := rpc.Dial(endpoint + "/admin")
	require.NoError(t, err)
	t.Cleanup(adminUnauthClient.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	adminAuthClient, err := client.NewRPC(
		ctx,
		logger,
		endpoint+"/admin",
		client.WithGethRPCOptions(rpc.WithHTTPAuth(gn.NewJWTAuth(jwtSecret))),
	)
	require.NoError(t, err)
	t.Cleanup(adminAuthClient.Close)

	t.Run("supervisor API works without JWT", func(t *testing.T) {
		var res string
		err := supervisorClient.Call(&res, "supervisor_ping")
		require.NoError(t, err)
		require.Equal(t, "pong", res)
	})

	t.Run("admin API requires JWT - fails without auth", func(t *testing.T) {
		var res bool
		err := adminUnauthClient.Call(&res, "admin_getFailsafeEnabled")
		require.ErrorContains(t, err, "missing token")
	})

	t.Run("admin API works with valid JWT", func(t *testing.T) {
		var res bool
		err := adminAuthClient.CallContext(ctx, &res, "admin_getFailsafeEnabled")
		require.NoError(t, err)
		require.Equal(t, false, res)
	})

	t.Run("supervisor API not accidentally gated when JWT configured", func(t *testing.T) {
		// This is a regression test - supervisor API must remain public
		// even when JWT is configured for admin routes
		var res string
		err := supervisorClient.Call(&res, "supervisor_ping")
		require.NoError(t, err, "supervisor API must not require JWT")
		require.Equal(t, "pong", res)
	})
}

func TestSupervisorAPIWithoutJWT(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	// Create server WITHOUT JWT secret - supervisor API works, no admin API
	server := oprpc.ServerFromConfig(&oprpc.ServerConfig{
		RpcOptions: []oprpc.Option{
			oprpc.WithLogger(logger),
		},
		Host:       "127.0.0.1",
		Port:       0,
		AppVersion: "test",
	})

	// Register supervisor API on root (no admin without JWT)
	server.AddAPI(rpc.API{
		Namespace: "supervisor",
		Service:   new(testSupervisorAPI),
	})

	require.NoError(t, server.Start())
	t.Cleanup(func() {
		_ = server.Stop()
	})

	endpoint := "http://" + server.Endpoint()
	client, err := rpc.Dial(endpoint)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	t.Run("supervisor API works without JWT configured", func(t *testing.T) {
		var res string
		err := client.Call(&res, "supervisor_ping")
		require.NoError(t, err)
		require.Equal(t, "pong", res)
	})
}
