package engine

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func gaugeValue(t *testing.T, g prometheus.Gauge) float64 {
	t.Helper()
	var m dto.Metric
	require.NoError(t, g.Write(&m))
	return m.GetGauge().GetValue()
}

// TestRecordBlockStats ensures gas and base fee are recorded on their own
// gauges. Distinct values are used so a regression that records the base fee
// onto the gas gauge (or leaves the base-fee gauge unset) is caught.
func TestRecordBlockStats(t *testing.T) {
	m := NewMetrics("test", prometheus.NewRegistry())

	const gas = uint64(21000)
	const baseFee = float64(1234)

	m.RecordBlockStats(common.Hash{0x01}, 5, 100, 3, gas, baseFee)

	require.Equal(t, float64(gas), gaugeValue(t, m.BlockGas), "BlockGas should record gas used")
	require.Equal(t, baseFee, gaugeValue(t, m.BlockBaseFee), "BlockBaseFee should record the block base fee")
}
