package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TwoL2SupernodeInteropPeerEL is a two-L2 interop preset where, per chain,
// the sequencer is a sibling op-node + op-geth pair independent of the
// supernode, and the supernode runs a separate virtual-node-fronted op-geth
// that snap/full-syncs via devp2p directly from the sibling sequencer EL.
// The supernode VN is configured as a non-sequencer with Sync.SyncMode =
// ELSync, reqresp disabled, and discovery disabled — so block transport into
// the supernode-fronted EL flows exclusively through execution-layer devp2p.
//
// Tests use this topology to wipe the supernode data dir together with the
// supernode-fronted EL while block production continues uninterrupted on the
// sibling sequencer. After restart, the supernode-fronted EL must be
// explicitly re-peered with its sibling (admin_addPeer) — discovery is off
// and the wipe drops the peer table.
//
// Field convention:
//   - TwoL2.L2ACL / L2BCL are the supernode VN proxies (where cross-safe
//     etc. is observed).
//   - TwoL2SupernodeInterop.L2ELA / L2ELB are the **supernode-fronted** ELs
//     (the cold-start wipe targets).
//   - SequencerL2AEL / SequencerL2BEL and SequencerL2ACL / SequencerL2BCL
//     expose the sibling sequencer pair, which the test peers the wiped EL
//     against after restart.
type TwoL2SupernodeInteropPeerEL struct {
	TwoL2SupernodeInterop

	// Sibling sequencer pair per chain. The batcher and proposer are wired
	// to these; the supernode is not.
	SequencerL2AEL *dsl.L2ELNode
	SequencerL2BEL *dsl.L2ELNode
	SequencerL2ACL *dsl.L2CLNode
	SequencerL2BCL *dsl.L2CLNode
}

// NewTwoL2SupernodeInteropPeerEL creates a fresh peer-EL interop preset for
// the current test.
func NewTwoL2SupernodeInteropPeerEL(t devtest.T, delaySeconds uint64, opts ...Option) *TwoL2SupernodeInteropPeerEL {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewTwoL2SupernodeInteropPeerEL", opts, twoL2SupernodeInteropPresetSupportedOptionKinds)
	runtime := sysgo.NewTwoL2SupernodeInteropPeerELRuntimeWithConfig(t, delaySeconds, presetCfg)
	return twoL2SupernodeInteropPeerELFromRuntime(t, runtime)
}
