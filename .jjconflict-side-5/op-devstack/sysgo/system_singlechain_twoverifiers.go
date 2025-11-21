package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DefaultSingleChainTwoVerifiersSystemIDs struct {
	DefaultSingleChainMultiNodeSystemIDs

	L2CLC stack.L2CLNodeID
	L2ELC stack.L2ELNodeID

	TestSequencer stack.TestSequencerID
}

func NewDefaultSingleChainTwoVerifiersSystemIDs(l1ID, l2ID eth.ChainID) DefaultSingleChainTwoVerifiersSystemIDs {
	return DefaultSingleChainTwoVerifiersSystemIDs{
		DefaultSingleChainMultiNodeSystemIDs: NewDefaultSingleChainMultiNodeSystemIDs(l1ID, l2ID),
		L2CLC:                                stack.NewL2CLNodeID("c", l2ID),
		L2ELC:                                stack.NewL2ELNodeID("c", l2ID),
	}
}

func DefaultSingleChainTwoVerifiersFollowL2System(dest *DefaultSingleChainTwoVerifiersSystemIDs) stack.Option[*Orchestrator] {
	ids := NewDefaultSingleChainTwoVerifiersSystemIDs(DefaultL1ID, DefaultL2AID)

	opt := stack.Combine[*Orchestrator]()
	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		o.P().Logger().Info("Setting up")
	}))

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	opt.Add(WithDeployer(),
		WithDeployerOptions(
			WithLocalContractSources(),
			WithCommons(ids.L1.ChainID()),
			WithPrefundedL2(ids.L1.ChainID(), ids.L2.ChainID()),
		),
	)

	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	opt.Add(WithL2ELNode(ids.L2ELB))
	opt.Add(WithL2CLNode(ids.L2CLB, ids.L1CL, ids.L1EL, ids.L2ELB))

	opt.Add(WithL2ELNode(ids.L2EL))
	opt.Add(WithL2CLNodeFollowL2(ids.L2CL, ids.L1CL, ids.L1EL, ids.L2EL, ids.L2CLB, L2CLSequencer()))

	opt.Add(WithL2ELNode(ids.L2ELC))
	opt.Add(WithL2CLNodeFollowL2(ids.L2CLC, ids.L1CL, ids.L1EL, ids.L2ELC, ids.L2CLB))

	opt.Add(WithL2CLP2PConnection(ids.L2CL, ids.L2CLB))
	opt.Add(WithL2ELP2PConnection(ids.L2EL, ids.L2ELB, false))
	opt.Add(WithL2CLP2PConnection(ids.L2CL, ids.L2CLC))
	opt.Add(WithL2ELP2PConnection(ids.L2EL, ids.L2ELC, false))
	opt.Add(WithL2CLP2PConnection(ids.L2CLB, ids.L2CLC))
	opt.Add(WithL2ELP2PConnection(ids.L2ELB, ids.L2ELC, false))

	opt.Add(WithBatcher(ids.L2Batcher, ids.L1EL, ids.L2CL, ids.L2EL))
	opt.Add(WithProposer(ids.L2Proposer, ids.L1EL, &ids.L2CL, nil))

	opt.Add(WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2EL}))

	opt.Add(WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2CLB, ids.L1EL, ids.L2ELB))

	opt.Add(WithL2Challenger(ids.L2Challenger, ids.L1EL, ids.L1CL, nil, nil, &ids.L2CL, []stack.L2ELNodeID{
		ids.L2EL,
	}))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}
