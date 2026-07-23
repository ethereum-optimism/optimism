package metrics

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestRecordHardforkActivationTimes(t *testing.T) {
	canyon := uint64(1_708_534_401)
	pectraBlobSchedule := uint64(1_748_666_000)
	jovian := uint64(1_900_000_000)

	m := NewMetrics("test", nil)
	m.RecordHardforkActivationTimes(&rollup.Config{
		L2ChainID:              big.NewInt(10),
		CanyonTime:             &canyon,
		PectraBlobScheduleTime: &pectraBlobSchedule,
		JovianTime:             &jovian,
	})

	got, err := gatherHardforkActivationTimes(m)
	require.NoError(t, err)
	require.Equal(t, map[string]float64{
		"10/canyon/l2_timestamp":                      float64(canyon),
		"10/pectra_blob_schedule/l1_origin_timestamp": float64(pectraBlobSchedule),
		"10/jovian/l2_timestamp":                      float64(jovian),
	}, got)
}

func TestRecordHardforkActivationTimesKeepKarstUpgradeGas(t *testing.T) {
	karst := uint64(1_950_000_000)

	m := NewMetrics("test", nil)
	m.RecordHardforkActivationTimes(&rollup.Config{
		L2ChainID:           big.NewInt(10),
		KarstTime:           &karst,
		KeepKarstUpgradeGas: true,
	})

	got, err := gatherHardforkActivationTimes(m)
	require.NoError(t, err)
	require.Equal(t, map[string]float64{
		"10/karst/l2_timestamp":                  float64(karst),
		"10/keep_karst_upgrade_gas/l2_timestamp": float64(karst),
	}, got)

	// Flag unset: no keep_karst_upgrade_gas series is emitted.
	m2 := NewMetrics("test", nil)
	m2.RecordHardforkActivationTimes(&rollup.Config{
		L2ChainID: big.NewInt(10),
		KarstTime: &karst,
	})
	got2, err := gatherHardforkActivationTimes(m2)
	require.NoError(t, err)
	require.Equal(t, map[string]float64{
		"10/karst/l2_timestamp": float64(karst),
	}, got2)
}

func gatherHardforkActivationTimes(m *Metrics) (map[string]float64, error) {
	mfs, err := m.registry.Gather()
	if err != nil {
		return nil, err
	}
	out := make(map[string]float64)
	for _, mf := range mfs {
		if mf.GetName() != "op_node_test_hardfork_activation_timestamp" {
			continue
		}
		for _, metric := range mf.GetMetric() {
			out[hardforkActivationMetricKey(metric)] = metric.GetGauge().GetValue()
		}
	}
	return out, nil
}

func hardforkActivationMetricKey(metric *io_prometheus_client.Metric) string {
	labels := make(map[string]string)
	for _, label := range metric.GetLabel() {
		labels[label.GetName()] = label.GetValue()
	}
	return labels["chain_id"] + "/" + labels["fork"] + "/" + labels["activation_basis"]
}
