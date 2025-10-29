package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DefaultSingleChainThreeNodesSystemIDs struct {
	DefaultMinimalSystemIDs

	L2CLB stack.L2CLNodeID
	L2ELB stack.L2ELNodeID
	L2CLC stack.L2CLNodeID
	L2ELC stack.L2ELNodeID
}

func NewDefaultSingleChainThreeNodesSystemIDs(l1ID, l2ID eth.ChainID) DefaultSingleChainThreeNodesSystemIDs {
	minimal := NewDefaultMinimalSystemIDs(l1ID, l2ID)
	return DefaultSingleChainThreeNodesSystemIDs{
		DefaultMinimalSystemIDs: minimal,
		L2CLB:                   stack.NewL2CLNodeID("b", l2ID),
		L2ELB:                   stack.NewL2ELNodeID("b", l2ID),
		L2CLC:                   stack.NewL2CLNodeID("c", l2ID),
		L2ELC:                   stack.NewL2ELNodeID("c", l2ID),
	}
}

func DefaultSingleChainThreeNodesSystem(dest *DefaultSingleChainThreeNodesSystemIDs) stack.Option[*Orchestrator] {
	ids := NewDefaultSingleChainThreeNodesSystemIDs(DefaultL1ID, DefaultL2AID)

	opt := stack.Combine[*Orchestrator]()
	opt.Add(DefaultMinimalSystem(&dest.DefaultMinimalSystemIDs))

	// Add second node (B)
	opt.Add(WithL2ELNode(ids.L2ELB, L2ELWithListenAddr("127.0.0.2:0")))
	opt.Add(WithL2CLNode(ids.L2CLB, ids.L1CL, ids.L1EL, ids.L2ELB))

	// Add third node (C)
	opt.Add(WithL2ELNode(ids.L2ELC, L2ELWithListenAddr("127.0.0.3:0")))
	opt.Add(WithL2CLNode(ids.L2CLC, ids.L1CL, ids.L1EL, ids.L2ELC))

	// P2P connect L2CL nodes - connect all nodes to each other
	opt.Add(WithL2CLP2PConnection(ids.L2CL, ids.L2CLB))
	opt.Add(WithL2CLP2PConnection(ids.L2CL, ids.L2CLC))
	opt.Add(WithL2CLP2PConnection(ids.L2CLB, ids.L2CLC))

	// P2P connect L2EL nodes - connect all nodes to each other
	opt.Add(WithL2ELP2PConnection(ids.L2EL, ids.L2ELB))
	opt.Add(WithL2ELP2PConnection(ids.L2EL, ids.L2ELC))
	opt.Add(WithL2ELP2PConnection(ids.L2ELB, ids.L2ELC))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))
	return opt
}
