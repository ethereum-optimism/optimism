package sysgo

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/stretchr/testify/require"
)

func TestWithL2ELKind(t *testing.T) {
	o := &Orchestrator{}
	stack.ApplyOptionLifecycle(WithL2ELKind("op-reth"), o)
	require.Equal(t, "op-reth", o.l2ELKind)
}

func TestWithL2ELKind_OverridesEnvVar(t *testing.T) {
	t.Setenv("DEVSTACK_L2EL_KIND", "op-geth")
	o := &Orchestrator{}
	stack.ApplyOptionLifecycle(WithL2ELKind("op-reth"), o)
	require.Equal(t, "op-reth", o.l2ELKind, "programmatic WithL2ELKind should override env var")
}
