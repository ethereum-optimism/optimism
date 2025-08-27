package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DefaultSingleChainMultiNodeSystemIDs struct {
	DefaultMinimalSystemIDs

	L2CLB stack.L2CLNodeID
	L2ELB stack.L2ELNodeID
}

func NewDefaultSingleChainMultiNodeSystemIDs(l1ID, l2ID eth.ChainID) DefaultSingleChainMultiNodeSystemIDs {
	minimal := NewDefaultMinimalSystemIDs(l1ID, l2ID)
	return DefaultSingleChainMultiNodeSystemIDs{
		DefaultMinimalSystemIDs: minimal,
		L2CLB:                   stack.NewL2CLNodeID("b", l2ID),
		L2ELB:                   stack.NewL2ELNodeID("b", l2ID),
	}
}

func DefaultSingleChainMultiNodeSystem(dest *DefaultSingleChainMultiNodeSystemIDs) stack.Option[*Orchestrator] {
	ids := NewDefaultSingleChainMultiNodeSystemIDs(DefaultL1ID, DefaultL2AID)

	opt := stack.Combine[*Orchestrator]()
	opt.Add(DefaultMinimalSystem(&dest.DefaultMinimalSystemIDs))

	opt.Add(WithL2ELNode(ids.L2ELB))
	opt.Add(WithL2CLNode(ids.L2CLB, ids.L1CL, ids.L1EL, ids.L2ELB))

	// P2P connect L2CL nodes
	opt.Add(WithL2CLP2PConnection(ids.L2CL, ids.L2CLB))
	opt.Add(WithL2ELP2PConnection(ids.L2EL, ids.L2ELB))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))
	return opt
}
