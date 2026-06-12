package monitor

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestStop_MetricsDisabledDoesNotPanic guards the nil-collector path: when
// metrics are disabled (the default), ms.collector is never constructed, so
// Stop() must not dereference it. Before the nil guard this panicked on every
// clean shutdown with metrics off — a path made reachable by running the
// divergence collector (replica endpoints set) without metrics.
func TestStop_MetricsDisabledDoesNotPanic(t *testing.T) {
	ms := &InteropMonitorService{
		Log: log.NewLogger(log.DiscardHandler()),
		// collector, divergenceCollector, rpcServer, pprofService, metricsSrv
		// all nil; finders/updaters/replicaClients all empty — the metrics-off,
		// no-replica default shape.
	}
	require.NotPanics(t, func() {
		require.NoError(t, ms.Stop(context.Background()))
	})
	require.True(t, ms.Stopped())
}
