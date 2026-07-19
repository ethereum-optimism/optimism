package mon

import (
	"context"
	"net"
	"testing"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/stretchr/testify/require"
)

func TestServiceMetricsAddr(t *testing.T) {
	service := &Service{
		logger:  testlog.Logger(t, 0),
		metrics: metrics.NewMetrics(),
	}
	require.Nil(t, service.MetricsAddr())

	err := service.initMetricsServer(&opmetrics.CLIConfig{
		Enabled:    true,
		ListenAddr: "127.0.0.1",
		ListenPort: 0,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, service.Stop(context.Background()))
	})

	addr := service.MetricsAddr()
	require.IsType(t, &net.TCPAddr{}, addr)
	require.NotZero(t, addr.(*net.TCPAddr).Port)
}
