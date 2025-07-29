package faucet

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-faucet/config"
	fconf "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/config"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
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
