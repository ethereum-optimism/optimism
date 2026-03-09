package node_restart

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	node_utils "github.com/ethereum-optimism/optimism/rust/kona/tests/node/utils"
)

func newRestartPreset(t devtest.T) *node_utils.MixedOpKonaPreset {
	if sharedRestartRuntime != nil {
		return node_utils.NewMixedOpKonaFromRuntime(t, sharedRestartRuntime)
	}

	// Restart tests currently target a minimal kona-only topology.
	return node_utils.NewMixedOpKonaForConfig(t, node_utils.L2NodeConfig{
		KonaSequencerNodesWithGeth: 1,
		KonaNodesWithGeth:          1,
	})
}
