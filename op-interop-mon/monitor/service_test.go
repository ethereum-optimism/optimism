package monitor

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestInteropMonitorStopWithMetricsDisabled(t *testing.T) {
	cfg := serviceTestConfig()
	cfg.MetricsConfig.Enabled = false

	service, err := InteropMonitorServiceFromClients(context.Background(), "test", cfg, nil, log.New())
	require.NoError(t, err)
	require.NoError(t, service.Stop(context.Background()))
	require.True(t, service.Stopped())
}

func TestInteropMonitorInitFailureCleansUpServers(t *testing.T) {
	metricsPort := reserveLocalPort(t)
	pprofPort := reserveLocalPort(t)
	rpcListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rpcListener.Close()) })

	cfg := serviceTestConfig()
	cfg.MetricsConfig = opmetrics.CLIConfig{
		Enabled:    true,
		ListenAddr: "127.0.0.1",
		ListenPort: metricsPort,
	}
	cfg.PprofConfig = oppprof.CLIConfig{
		ListenEnabled: true,
		ListenAddr:    "127.0.0.1",
		ListenPort:    pprofPort,
	}
	cfg.RPCConfig.ListenPort = rpcListener.Addr().(*net.TCPAddr).Port

	service, err := InteropMonitorServiceFromClients(context.Background(), "test", cfg, nil, log.New())
	require.Nil(t, service)
	require.ErrorContains(t, err, "failed to start rpc server")
	requirePortAvailable(t, metricsPort)
	requirePortAvailable(t, pprofPort)
}

func TestInteropMonitorPartialInitCleanupIsNilSafe(t *testing.T) {
	service := &InteropMonitorService{Log: log.New()}

	require.NoError(t, service.Stop(context.Background()))
	require.True(t, service.Stopped())
}

func TestInteropMonitorCLIInitFailureCleanupIsNilSafe(t *testing.T) {
	cfg := serviceTestConfig()
	cfg.DependencySetPath = t.TempDir() + "/missing-dependency-set.json"

	service, err := InteropMonitorServiceFromCLIConfig(context.Background(), "test", cfg, log.New())
	require.Nil(t, service)
	require.ErrorContains(t, err, "failed to load dependency set")
}

func serviceTestConfig() *CLIConfig {
	return &CLIConfig{
		PollInterval: time.Second,
		RPCConfig: oprpc.CLIConfig{
			ListenAddr: "127.0.0.1",
			ListenPort: 0,
		},
		PprofConfig: oppprof.CLIConfig{
			ListenAddr: "127.0.0.1",
			ListenPort: 0,
		},
	}
}

func reserveLocalPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	return port
}

func requirePortAvailable(t *testing.T, port int) {
	t.Helper()
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	require.NoError(t, listener.Close())
}
