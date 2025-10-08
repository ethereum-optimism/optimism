package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type SingleChainMultiNodeWithSafeSourceL2 struct {
	Minimal

	L2ELB *dsl.L2ELNode
	L2CLB *dsl.L2CLNode
}

func WithSingleChainMultiNodeWithSafeSourceL2() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSingleChainMultiNodeWithSafeSourceL2System(&sysgo.DefaultSingleChainMultiNodeWithSafeSourceL2SystemIDs{}))
}

func NewSingleChainMultiNodeWithSafeSourceL2(t devtest.T) *SingleChainMultiNodeWithSafeSourceL2 {
	preset := NewSingleChainMultiNodeWithSafeSourceL2WithoutCheck(t)
	// Ensure the L2 sourced node is in sync with the sequencer before starting tests
	dsl.CheckAll(t,
		preset.L2CLB.MatchedFn(preset.L2CL, types.CrossSafe, 30),
		preset.L2CLB.MatchedFn(preset.L2CL, types.LocalUnsafe, 30),
	)
	return preset
}

func NewSingleChainMultiNodeWithSafeSourceL2WithoutCheck(t devtest.T) *SingleChainMultiNodeWithSafeSourceL2 {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)
	minimal := minimalFromSystem(t, system, orch)
	l2 := system.L2Network(match.Assume(t, match.L2ChainA))
	verifierCL := l2.L2CLNode(match.Assume(t,
		match.And(
			match.Not(match.WithSequencerActive(t.Ctx())),
			match.Not[stack.L2CLNodeID, stack.L2CLNode](minimal.L2CL.ID()),
		)))
	verifierEL := l2.L2ELNode(match.Assume(t,
		match.And(
			match.EngineFor(verifierCL),
			match.Not[stack.L2ELNodeID, stack.L2ELNode](minimal.L2EL.ID()))))
	preset := &SingleChainMultiNodeWithSafeSourceL2{
		Minimal: *minimal,
		L2ELB:   dsl.NewL2ELNode(verifierEL, orch.ControlPlane()),
		L2CLB:   dsl.NewL2CLNode(verifierCL, orch.ControlPlane()),
	}
	return preset
}
