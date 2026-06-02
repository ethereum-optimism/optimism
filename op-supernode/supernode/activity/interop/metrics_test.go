package interop

import (
	"testing"

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
		v, found := gatheredDecisionCount(t, h.interop.metrics, d.String())
		require.Truef(t, found, "decision series %q must exist from startup for alerting", d)
		require.Zerof(t, v, "decision series %q must start at 0", d)
	}
}

// gatheredDecisionCount reads the value of the round-decisions counter for a
// given decision label directly from the registry, without touching the metric
// (WithLabelValues would itself create the series and mask a missing one).
func gatheredDecisionCount(t *testing.T, m *resources.SupernodeMetrics, decision string) (float64, bool) {
	t.Helper()
	families, err := m.Registry().Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() != "supernode_interop_round_decisions_total" {
			continue
		}
		for _, metric := range mf.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "decision" && label.GetValue() == decision {
					return metric.GetCounter().GetValue(), true
				}
			}
		}
	}
	return 0, false
}
