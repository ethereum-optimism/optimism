package engine

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestRecordBlockStatsRecordsGasAndBaseFee(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics("test", registry)

	metrics.RecordBlockStats(common.Hash{}, 1, 2, 3, 21, 42.5)

	require.Equal(t, float64(21), gaugeValue(t, metrics.BlockGas))
	require.Equal(t, 42.5, gaugeValue(t, metrics.BlockBaseFee))
}

func gaugeValue(t *testing.T, gauge prometheus.Gauge) float64 {
	t.Helper()

	metric := &dto.Metric{}
	require.NoError(t, gauge.Write(metric))
	require.NotNil(t, metric.Gauge)
	return metric.Gauge.GetValue()
}
