package node

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestConductorClientOverrideLeader(t *testing.T) {
	m := metrics.NewMetrics("", nil)
	c := NewConductorClient(&config.Config{}, testlog.Logger(t, log.LevelInfo), m)
	require.Equal(t, 0.0, gaugeValue(t, m.ConductorOverride))

	require.NoError(t, c.OverrideLeader(context.Background()))
	require.Equal(t, 1.0, gaugeValue(t, m.ConductorOverride))

	// With the override set, commits are skipped without dialing the conductor.
	payload := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{BlockNumber: 1}}
	require.NoError(t, c.CommitUnsafePayload(context.Background(), payload))
}

func gaugeValue(t *testing.T, g prometheus.Gauge) float64 {
	var out dto.Metric
	require.NoError(t, g.Write(&out))
	return out.GetGauge().GetValue()
}
