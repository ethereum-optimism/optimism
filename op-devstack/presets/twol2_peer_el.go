package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TwoL2SupernodeInteropPeerEL extends TwoL2SupernodeInterop with a sibling
// sequencer EL+CL pair per chain. The embedded fields (L2ELA/B, L2ACL/B)
// point at the supernode-fronted nodes — the wipe targets. The supernode VN
// runs in ELSync so the wiped EL recovers via devp2p from the sibling.
type TwoL2SupernodeInteropPeerEL struct {
	TwoL2SupernodeInterop

	SequencerL2AEL *dsl.L2ELNode
	SequencerL2BEL *dsl.L2ELNode
	SequencerL2ACL *dsl.L2CLNode
	SequencerL2BCL *dsl.L2CLNode
}

func NewTwoL2SupernodeInteropPeerEL(t devtest.T, delaySeconds uint64, opts ...Option) *TwoL2SupernodeInteropPeerEL {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewTwoL2SupernodeInteropPeerEL", opts, twoL2SupernodeInteropPresetSupportedOptionKinds)
	return twoL2SupernodeInteropPeerELFromRuntime(t, sysgo.NewTwoL2SupernodeInteropPeerELRuntimeWithConfig(t, delaySeconds, presetCfg))
}
