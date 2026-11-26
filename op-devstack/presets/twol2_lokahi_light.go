package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TwoL2LokahiLight is two interop chains, each run by a source node that derives the chain and
// follows one lokahi process, and by a light sequencer and a light follower. A light node derives
// nothing and takes its safe and finalized labels from the source. Its test sequencer controls L1
// and builds blocks on each chain's sequencer.
type TwoL2LokahiLight struct {
	TwoL2Lokahi

	ChainA LokahiChainNodes
	ChainB LokahiChainNodes
}

// NewTwoL2LokahiLight starts a [TwoL2LokahiLight] system with interop active at genesis for the
// current test. It skips the test under op-node.
func NewTwoL2LokahiLight(t devtest.T) *TwoL2LokahiLight {
	skipKonaOnlyPresetOnOpNode(t)
	runtime := sysgo.NewTwoL2LokahiLightRuntime(t)
	base := twoL2LokahiFromRuntime(t, runtime, 0)
	return &TwoL2LokahiLight{
		TwoL2Lokahi: *base,
		ChainA:      lokahiChainNodes(t, base.L2A, base.L2ACL, base.L2ELA, runtime.Chains["l2a"]),
		ChainB:      lokahiChainNodes(t, base.L2B, base.L2BCL, base.L2ELB, runtime.Chains["l2b"]),
	}
}
