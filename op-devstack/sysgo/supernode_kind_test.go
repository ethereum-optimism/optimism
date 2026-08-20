package sysgo

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

func TestSupernodeKindFromEnv(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  string
		want SupernodeKind
	}{
		{name: "unset defaults to the in-process Go supernode", env: "", want: SupernodeOpSupernode},
		{name: "explicit op-supernode", env: "op-supernode", want: SupernodeOpSupernode},
		{name: "lokahi", env: "lokahi", want: SupernodeLokahi},
	} {
		t.Run(tc.name, func(t *testing.T) {
			kind, err := supernodeKindFromEnv(tc.env)
			require.NoError(t, err)
			require.Equal(t, tc.want, kind)
		})
	}
}

// An unknown value must be rejected rather than defaulted: falling back to op-supernode
// would report a green run for an implementation that never started.
func TestSupernodeKindFromEnvRejectsUnknown(t *testing.T) {
	for _, env := range []string{"kona-node", "op-node", "Lokahi", "supernode"} {
		t.Run(env, func(t *testing.T) {
			kind, err := supernodeKindFromEnv(env)
			require.ErrorContains(t, err, devstackSupernodeKindEnv)
			require.ErrorContains(t, err, env)
			require.Empty(t, kind)
		})
	}
}

// The exported resolver must read the documented variable name.
func TestResolveSupernodeKindReadsEnv(t *testing.T) {
	t.Setenv(devstackSupernodeKindEnv, string(SupernodeLokahi))
	require.Equal(t, SupernodeLokahi, ResolveSupernodeKind(devtest.SerialT(t)))
}

// The supernode kind is independent of the single-chain CL kind: the acceptance CI variant
// sets both, asking for kona-node per chain and lokahi for the multi-chain presets.
func TestResolveSupernodeKindIgnoresL2CLKind(t *testing.T) {
	t.Setenv(devstackL2CLKindEnv, string(MixedL2CLKona))
	require.Equal(t, SupernodeOpSupernode, ResolveSupernodeKind(devtest.SerialT(t)))
}
