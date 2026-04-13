package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TwoL2SupernodeFollowL2 extends TwoL2SupernodeInterop with one follow-source
// follower per chain.
type TwoL2SupernodeFollowL2 struct {
	TwoL2SupernodeInterop

	L2AFollowEL *dsl.L2ELNode
	L2AFollowCL *dsl.L2CLNode
	L2BFollowEL *dsl.L2ELNode
	L2BFollowCL *dsl.L2CLNode
}

// NewTwoL2SupernodeFollowL2 creates a fresh follow-source variant of the two-L2
// supernode interop preset for the current test.
func NewTwoL2SupernodeFollowL2(t devtest.T, delaySeconds uint64, opts ...Option) *TwoL2SupernodeFollowL2 {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewTwoL2SupernodeFollowL2", opts, twoL2SupernodeInteropPresetSupportedOptionKinds)
	return twoL2SupernodeFollowL2FromRuntime(t, sysgo.NewTwoL2SupernodeFollowL2RuntimeWithConfig(t, delaySeconds, presetCfg))
}
