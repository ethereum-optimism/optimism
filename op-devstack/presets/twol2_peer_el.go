package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TwoL2SupernodeInteropPeerEL extends TwoL2SupernodeInterop with a second
// supernode that follows both chains as a verifier. The embedded fields
// (Supernode, L2ELA, L2ELB, …) describe the sequencer side and continue
// to drive block production. VerifierSupernode + VerifierL2ELA/B are the
// wipe targets: the verifier VNs run in NonSequencer/ELSync mode and
// resync over EL devp2p from the sequencer ELs after a wipe.
type TwoL2SupernodeInteropPeerEL struct {
	TwoL2SupernodeInterop

	VerifierSupernode *dsl.Supernode
	VerifierL2ELA     *dsl.L2ELNode
	VerifierL2ELB     *dsl.L2ELNode
	VerifierL2ACL     *dsl.L2CLNode
	VerifierL2BCL     *dsl.L2CLNode
}

func NewTwoL2SupernodeInteropPeerEL(t devtest.T, delaySeconds uint64, opts ...Option) *TwoL2SupernodeInteropPeerEL {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewTwoL2SupernodeInteropPeerEL", opts, twoL2SupernodeInteropPresetSupportedOptionKinds)
	return twoL2SupernodeInteropPeerELFromRuntime(t, sysgo.NewTwoL2SupernodeInteropPeerELRuntimeWithConfig(t, delaySeconds, presetCfg))
}
