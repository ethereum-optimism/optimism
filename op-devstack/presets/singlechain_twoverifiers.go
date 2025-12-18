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

func NewSingleChainTwoVerifiersWithoutCheck(t devtest.T) *SingleChainTwoVerifiers {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)
	singleChainMultiNode := NewSingleChainMultiNodeWithoutCheck(t)
	l2 := system.L2Network(match.Assume(t, match.L2ChainA))
	verifierCL := l2.L2CLNode(match.Assume(t,
		match.And(
			match.Not(match.WithSequencerActive(t.Ctx())),
			match.Not[stack.L2CLNodeID, stack.L2CLNode](singleChainMultiNode.L2CL.ID()),
			match.Not[stack.L2CLNodeID, stack.L2CLNode](singleChainMultiNode.L2CLB.ID()),
		)))
	verifierEL := l2.L2ELNode(match.Assume(t,
		match.And(
			match.Not[stack.L2ELNodeID, stack.L2ELNode](singleChainMultiNode.L2EL.ID()),
			match.Not[stack.L2ELNodeID, stack.L2ELNode](singleChainMultiNode.L2ELB.ID()),
		)))
	preset := &SingleChainTwoVerifiers{
		SingleChainMultiNode: *singleChainMultiNode,
		L2ELC:                dsl.NewL2ELNode(verifierEL, orch.ControlPlane()),
		L2CLC:                dsl.NewL2CLNode(verifierCL, orch.ControlPlane()),
		TestSequencer:        dsl.NewTestSequencer(system.TestSequencer(match.Assume(t, match.FirstTestSequencer))),
	}
	return preset
}

func WithSingleChainTwoVerifiersFollowL2() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSingleChainTwoVerifiersFollowL2System(&sysgo.DefaultSingleChainTwoVerifiersSystemIDs{}))
}
