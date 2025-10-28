package sysgo

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/stretchr/testify/require"
)

func TestWithL2MetricsDashboard_DefaultDisabled(t *testing.T) {
	// This should run without error if orch.l2MetricsEndpoints is unset
	stack.ApplyOptionLifecycle(WithL2MetricsDashboard(), &Orchestrator{})
}

func TestWithL2MetricsDashboard_DisabledIfEndpointsRegisteredButNotExplicitlyEnabled(t *testing.T) {

	o := &Orchestrator{}
	o.RegisterL2MetricsEndpoints("test", PrometheusMetricsEndpoint{
		host:              "localhost",
		port:              "9090",
		isLocal:           true,
		isRunningInDocker: false,
	})

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
