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

type SingleChainThreeNodes struct {
	Minimal

	L2ELB *dsl.L2ELNode
	L2CLB *dsl.L2CLNode
	L2ELC *dsl.L2ELNode
	L2CLC *dsl.L2CLNode
}

func WithSingleChainThreeNodes() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSingleChainThreeNodesSystem(&sysgo.DefaultSingleChainThreeNodesSystemIDs{}))
}

func NewSingleChainThreeNodes(t devtest.T) *SingleChainThreeNodes {
	preset := NewSingleChainThreeNodesWithoutCheck(t)
	// Ensure the follower nodes are in sync with the sequencer before starting tests
	dsl.CheckAll(t,
		preset.L2CLB.MatchedFn(preset.L2CL, types.CrossSafe, 30),
		preset.L2CLB.MatchedFn(preset.L2CL, types.LocalUnsafe, 30),
		preset.L2CLC.MatchedFn(preset.L2CL, types.CrossSafe, 30),
		preset.L2CLC.MatchedFn(preset.L2CL, types.LocalUnsafe, 30),
	)
	return preset
}

func NewSingleChainThreeNodesWithoutCheck(t devtest.T) *SingleChainThreeNodes {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)
	minimal := minimalFromSystem(t, system, orch)
	l2 := system.L2Network(match.Assume(t, match.L2ChainA))

	// Find verifier B (second node)
	verifierCLB := l2.L2CLNode(match.Assume(t,
		match.And(
			match.Not(match.WithSequencerActive(t.Ctx())),
			match.Not[stack.L2CLNodeID, stack.L2CLNode](minimal.L2CL.ID()),
		)))
	verifierELB := l2.L2ELNode(match.Assume(t,
		match.And(
			match.EngineFor(verifierCLB),
			match.Not[stack.L2ELNodeID, stack.L2ELNode](minimal.L2EL.ID()))))

	// Find verifier C (third node)
	verifierCLC := l2.L2CLNode(match.Assume(t,
		match.And(
			match.Not(match.WithSequencerActive(t.Ctx())),
			match.Not[stack.L2CLNodeID, stack.L2CLNode](minimal.L2CL.ID()),
			match.Not[stack.L2CLNodeID, stack.L2CLNode](verifierCLB.ID()),
		)))
	verifierELC := l2.L2ELNode(match.Assume(t,
		match.And(
			match.EngineFor(verifierCLC),
			match.Not[stack.L2ELNodeID, stack.L2ELNode](minimal.L2EL.ID()),
			match.Not[stack.L2ELNodeID, stack.L2ELNode](verifierELB.ID()))))

	preset := &SingleChainThreeNodes{
		Minimal: *minimal,
		L2ELB:   dsl.NewL2ELNode(verifierELB, orch.ControlPlane()),
		L2CLB:   dsl.NewL2CLNode(verifierCLB, orch.ControlPlane()),
		L2ELC:   dsl.NewL2ELNode(verifierELC, orch.ControlPlane()),
		L2CLC:   dsl.NewL2CLNode(verifierCLC, orch.ControlPlane()),
	}
	return preset
}
