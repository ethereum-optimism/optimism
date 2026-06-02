package interop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
	"github.com/stretchr/testify/require"
)

// TestNewPreInitializesDecisionMetrics guards the alerting contract: every
// decision label series must exist at 0 from process startup. Prometheus
// increase()/rate() cannot see a counter's first increment if the series only
// springs into existence already non-zero, so an invalidate alert built on
// increase(...{decision="invalidate"}[w]) > 0 would miss the very event it
// guards unless the series starts at 0.
func TestNewPreInitializesDecisionMetrics(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, nil).Build()
	require.NotNil(t, h.interop)

	for _, d := range roundDecisions {
		v, found := gatheredCounter(t, h.interop.metrics, "supernode_interop_round_decisions_total", "decision", d.String())
		require.Truef(t, found, "decision series %q must exist from startup for alerting", d)
		require.Zerof(t, v, "decision series %q must start at 0", d)
	}
}

// TestNewPreInitializesInvalidationMetrics guards the same alerting contract for
// the per-chain invalidation counter: every configured chain's series must exist
// at 0 from startup so a per-chain invalidate alert has a 0 baseline.
func TestNewPreInitializesInvalidationMetrics(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, nil).WithChain(8453, nil).Build()
	require.NotNil(t, h.interop)

	for _, id := range []uint64{10, 8453} {
		chainID := eth.ChainIDFromUInt64(id).String()
		v, found := gatheredCounter(t, h.interop.metrics, "supernode_interop_invalidations_total", "chain_id", chainID)
		require.Truef(t, found, "invalidation series for chain %q must exist from startup for alerting", chainID)
		require.Zerof(t, v, "invalidation series for chain %q must start at 0", chainID)
	}
}

// gatheredCounter reads the value of a counter series for a given single-label
// value directly from the registry, without touching the metric (WithLabelValues
// would itself create the series and mask a missing one).
func gatheredCounter(t *testing.T, m *resources.SupernodeMetrics, metricName, labelName, labelValue string) (float64, bool) {
	t.Helper()
	families, err := m.Registry().Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() != metricName {
			continue
		}
		for _, metric := range mf.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == labelName && label.GetValue() == labelValue {
					return metric.GetCounter().GetValue(), true
				}
			}
		}
	}
	return 0, false
}
