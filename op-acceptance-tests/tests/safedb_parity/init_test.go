package safedb_parity

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

const (
	opNodeVerifierKey = "verifier-op-node"
	konaVerifierKey   = "verifier-kona"
)

// paritySystem runs one chain with an op-node verifier and a kona-node verifier deriving from the
// same L1. op-node is the reference: every safe head answer kona-node gives must match it.
type paritySystem struct {
	*presets.MixedSingleChainFrontends
	SequencerEL *dsl.L2ELNode
	OpNode      *dsl.L2CLNode
	OpNodeEL    *dsl.L2ELNode
	Kona        *dsl.L2CLNode
	KonaEL      *dsl.L2ELNode
}

// newParitySystem starts both verifiers with a safe head database.
func newParitySystem(t devtest.T) *paritySystem {
	return newParitySystemWithSafeDB(t, true)
}

// newParitySystemWithoutSafeDB starts both verifiers without one, to compare how they refuse.
func newParitySystemWithoutSafeDB(t devtest.T) *paritySystem {
	return newParitySystemWithSafeDB(t, false)
}

func newParitySystemWithSafeDB(t devtest.T, enabled bool) *paritySystem {
	// Every node's CL kind is pinned, so the kona-node CI variant would only repeat this run.
	sysgo.SkipOnKonaNode(t, "pins op-node and kona-node explicitly; covered by the op-node variant")
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       "sequencer-op-reth",
				CLKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      sysgo.MixedL2CLOpNode,
				IsSequencer: true,
			},
			{
				ELKey:        opNodeVerifierKey + "-op-reth",
				CLKey:        opNodeVerifierKey,
				ELKind:       sysgo.MixedL2ELOpReth,
				CLKind:       sysgo.MixedL2CLOpNode,
				EnableSafeDB: enabled,
			},
			{
				ELKey:        konaVerifierKey + "-op-reth",
				CLKey:        konaVerifierKey,
				ELKind:       sysgo.MixedL2ELOpReth,
				CLKind:       sysgo.MixedL2CLKona,
				EnableSafeDB: enabled,
			},
		},
		WithTestSequencer: true,
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)

	sys := &paritySystem{MixedSingleChainFrontends: frontends}
	for _, node := range frontends.Nodes {
		switch {
		case node.Spec.IsSequencer:
			sys.SequencerEL = node.EL
		case node.Spec.CLKey == opNodeVerifierKey:
			sys.OpNode, sys.OpNodeEL = node.CL, node.EL
		case node.Spec.CLKey == konaVerifierKey:
			sys.Kona, sys.KonaEL = node.CL, node.EL
		}
	}
	require := t.Require()
	require.NotNil(sys.SequencerEL, "mixed runtime must expose the sequencer EL")
	require.NotNil(sys.OpNode, "mixed runtime must expose the op-node verifier")
	require.NotNil(sys.Kona, "mixed runtime must expose the kona-node verifier")
	return sys
}
