package sysgo

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func TestWithL2MetricsDashboard_DefaultDisabled(t *testing.T) {
	// This should run without error if orch.l2MetricsEndpoints is unset
	stack.ApplyOptionLifecycle(WithL2MetricsDashboard(), &Orchestrator{})
}

func TestWithL2MetricsDashboard_DisabledIfEndpointsRegisteredButNotExplicitlyEnabled(t *testing.T) {
	id := stack.NewL2ELNodeID("test", eth.ChainIDFromUInt64(11111111111))
	metricsTarget := NewPrometheusMetricsTarget("localhost", "9090", false)

	o := &Orchestrator{}
	o.RegisterL2MetricsTargets(id, metricsTarget)

	// This should run without error if disabled
	stack.ApplyOptionLifecycle(WithL2MetricsDashboard(), o)
}

func TestWithL2MetricsDashboard_DisabledIfNoEndpointsRegisteredButExplicitlyEnabled(t *testing.T) {

	o := &Orchestrator{}
	err := os.Setenv(sysgoMetricsEnabledEnvVar, "true")
	require.NoError(t, err, "error setting metrics enabled")

	// This should run without error if disabled
	stack.ApplyOptionLifecycle(WithL2MetricsDashboard(), o)
}
