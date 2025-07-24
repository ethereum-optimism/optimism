package faucet

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/HashKeyChain/verse/op-faucet/config"
	fconf "github.com/HashKeyChain/verse/op-faucet/faucet/backend/config"
	opmetrics "github.com/HashKeyChain/verse/op-service/metrics"
	"github.com/HashKeyChain/verse/op-service/oppprof"
	oprpc "github.com/HashKeyChain/verse/op-service/rpc"
	"github.com/HashKeyChain/verse/op-service/testlog"
)

// TestService is a quick smoke-test to check the service is up and running
func TestService(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	cfg := &config.Config{
		Version: "v0.0.1",
		Faucets: &fconf.Config{},
		MetricsConfig: opmetrics.CLIConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1",
			ListenPort: 0,
		},
		RPC: oprpc.CLIConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0,
			EnableAdmin: true,
		},
		PprofConfig: oppprof.CLIConfig{
			ListenEnabled: true,
			ListenAddr:    "127.0.0.1",
			ListenPort:    0,
		},
	}
	srv, err := FromConfig(context.Background(), cfg, logger)
	require.NoError(t, err)
	require.NoError(t, srv.Start(context.Background()))
	require.NotEmpty(t, srv.RPC())
	require.Contains(t, srv.FaucetEndpoint("foobar"), "/faucet/foobar")
	require.Empty(t, srv.Faucets())
	require.Empty(t, srv.Defaults())
	require.False(t, srv.Stopped())
	require.NoError(t, srv.Stop(context.Background()))
	require.True(t, srv.Stopped())
}
