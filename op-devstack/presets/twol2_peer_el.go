package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TwoL2SupernodeInteropPeerEL is a two-L2 interop preset where, per chain,
// the canonical sequencer (L2A/L2B and L2ACL/L2BCL, exposed via the embedded
// TwoL2SupernodeInterop fields) is a sibling op-node + op-geth pair, and the
// supernode runs a separate virtual-node-fronted op-geth that snap/full-syncs
// via devp2p from the sibling sequencer EL.
//
// Tests use this topology to wipe the supernode data dir together with the
// supernode-fronted EL while block production continues uninterrupted on the
// sibling sequencer. On restart, the supernode-fronted EL recovers via
// execution-layer peer-to-peer sync from its sibling, and the supernode VN
// cold-starts on top of the synced engine.
type TwoL2SupernodeInteropPeerEL struct {
	TwoL2SupernodeInterop

	// SupernodeL2AEL and SupernodeL2BEL are the supernode-fronted ELs —
	// the cold-start wipe targets. They are distinct from TwoL2.L2A /
	// TwoL2SupernodeInterop.L2ELA, which front the sibling sequencer EL.
	SupernodeL2AEL *dsl.L2ELNode
	SupernodeL2BEL *dsl.L2ELNode
}

// NewTwoL2SupernodeInteropPeerEL creates a fresh peer-EL interop preset for
// the current test.
func NewTwoL2SupernodeInteropPeerEL(t devtest.T, delaySeconds uint64, opts ...Option) *TwoL2SupernodeInteropPeerEL {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewTwoL2SupernodeInteropPeerEL", opts, twoL2SupernodeInteropPresetSupportedOptionKinds)
	runtime := sysgo.NewTwoL2SupernodeInteropPeerELRuntimeWithConfig(t, delaySeconds, presetCfg)
	return twoL2SupernodeInteropPeerELFromRuntime(t, runtime)
}
