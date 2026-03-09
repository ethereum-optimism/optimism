package node_utils

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/stretchr/testify/require"
)

func TestWithRequiredOpSequencerForTestSequencerAddsDefaultOpGethSequencer(t *testing.T) {
	cfg := withRequiredOpSequencerForTestSequencer(L2NodeConfig{
		KonaSequencerNodesWithReth: 1,
	})

	require.Equal(t, 1, cfg.OpSequencerNodesWithGeth)
	require.Equal(t, 1, cfg.KonaSequencerNodesWithReth)

	specs := mixedOpKonaNodeSpecs(cfg)
	require.True(t, hasNodeSpec(specs, sysgo.MixedL2ELOpGeth, sysgo.MixedL2CLOpNode, true))
}

func TestWithRequiredOpSequencerForTestSequencerPreservesExistingOpSequencer(t *testing.T) {
	cfg := withRequiredOpSequencerForTestSequencer(L2NodeConfig{
		OpSequencerNodesWithReth: 2,
	})

	require.Equal(t, 0, cfg.OpSequencerNodesWithGeth)
	require.Equal(t, 2, cfg.OpSequencerNodesWithReth)
}

func hasNodeSpec(specs []sysgo.MixedSingleChainNodeSpec, elKind sysgo.MixedL2ELKind, clKind sysgo.MixedL2CLKind, isSequencer bool) bool {
	for _, spec := range specs {
		if spec.ELKind == elKind && spec.CLKind == clKind && spec.IsSequencer == isSequencer {
			return true
		}
	}
	return false
}
