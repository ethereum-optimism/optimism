package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DefaultSingleChainTwoVerifiersSystemIDs struct {
	DefaultSingleChainMultiNodeSystemIDs

	L2CLC stack.L2CLNodeID
	L2ELC stack.L2ELNodeID
}

func NewDefaultSingleChainTwoVerifiersSystemIDs(l1ID, l2ID eth.ChainID) DefaultSingleChainTwoVerifiersSystemIDs {
	return DefaultSingleChainTwoVerifiersSystemIDs{
		DefaultSingleChainMultiNodeSystemIDs: NewDefaultSingleChainMultiNodeSystemIDs(l1ID, l2ID),
		L2CLC:                                stack.NewL2CLNodeID("c", l2ID),
		L2ELC:                                stack.NewL2ELNodeID("c", l2ID),
	}
}

func DefaultSingleChainTwoVerifiersSystem(dest *DefaultSingleChainTwoVerifiersSystemIDs) stack.Option[*Orchestrator] {
	ids := NewDefaultSingleChainTwoVerifiersSystemIDs(DefaultL1ID, DefaultL2AID)

	opt := stack.Combine[*Orchestrator]()
	opt.Add(DefaultSingleChainMultiNodeSystem(&dest.DefaultSingleChainMultiNodeSystemIDs))

	opt.Add(WithL2ELNode(ids.L2ELC))
	// Specific options are applied after global options
	// this means unsafeOnly is always disabled for the second verifier
	opt.Add(WithL2CLNode(ids.L2CLC, ids.L1CL, ids.L1EL, ids.L2ELC, L2CLVerifierDisableUnsafeOnly()))

	opt.Add(WithL2CLP2PConnection(ids.L2CL, ids.L2CLC))
	opt.Add(WithL2ELP2PConnection(ids.L2EL, ids.L2ELC))
	opt.Add(WithL2CLP2PConnection(ids.L2CLB, ids.L2CLC))
	opt.Add(WithL2ELP2PConnection(ids.L2ELB, ids.L2ELC))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))
	return opt
}
