package sysgo

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

func TestWithL2MetricsDashboard_DefaultDisabled(t *testing.T) {
	// This should run without error if orch.l2MetricsEndpoints is unset
	stack.ApplyOptionLifecycle(WithL2MetricsDashboard(), &Orchestrator{})
}
