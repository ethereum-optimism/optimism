package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type SingleChainTwoVerifiers struct {
	SingleChainMultiNode

	L2ELC *dsl.L2ELNode
	L2CLC *dsl.L2CLNode

	TestSequencer *dsl.TestSequencer
}

func NewSingleChainTwoVerifiersWithoutCheck(t devtest.T, opts ...stack.CommonOption) *SingleChainTwoVerifiers {
	system := shim.NewSystem(t)
	orch := NewTestOrchestrator(t, append([]stack.CommonOption{WithSingleChainTwoVerifiersFollowL2()}, opts...)...)
	orch.Hydrate(system)
	minimal := minimalFromSystem(t, system, orch)
	l2 := system.L2Network(match.Assume(t, match.L2ChainA))
	verifierCLB := l2.L2CLNode(match.Assume(t,
		match.And(
			match.Not(match.WithSequencerActive(t.Ctx())),
			match.Not(stack.ByID[stack.L2CLNode](minimal.L2CL.ID())),
		)))
	verifierELB := l2.L2ELNode(match.Assume(t,
		match.And(
			match.EngineFor(verifierCLB),
			match.Not(stack.ByID[stack.L2ELNode](minimal.L2EL.ID())))))
	singleChainMultiNode := &SingleChainMultiNode{
		Minimal: *minimal,
		L2ELB:   dsl.NewL2ELNode(verifierELB, orch.ControlPlane()),
		L2CLB:   dsl.NewL2CLNode(verifierCLB, orch.ControlPlane()),
	}
	verifierCLC := l2.L2CLNode(match.Assume(t,
		match.And(
			match.Not(match.WithSequencerActive(t.Ctx())),
			match.Not(stack.ByID[stack.L2CLNode](minimal.L2CL.ID())),
			match.Not(stack.ByID[stack.L2CLNode](singleChainMultiNode.L2CLB.ID())),
		)))
	verifierELC := l2.L2ELNode(match.Assume(t,
		match.And(
			match.Not(stack.ByID[stack.L2ELNode](minimal.L2EL.ID())),
			match.Not(stack.ByID[stack.L2ELNode](singleChainMultiNode.L2ELB.ID())),
		)))
	preset := &SingleChainTwoVerifiers{
		SingleChainMultiNode: *singleChainMultiNode,
		L2ELC:                dsl.NewL2ELNode(verifierELC, orch.ControlPlane()),
		L2CLC:                dsl.NewL2CLNode(verifierCLC, orch.ControlPlane()),
		TestSequencer:        dsl.NewTestSequencer(system.TestSequencer(match.Assume(t, match.FirstTestSequencer))),
	}
	return preset
}

func WithSingleChainTwoVerifiersFollowL2() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSingleChainTwoVerifiersFollowL2System(&sysgo.DefaultSingleChainTwoVerifiersSystemIDs{}))
}
