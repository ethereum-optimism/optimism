package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DefaultMinimalSystemIDs struct {
	L1   stack.L1NetworkID
	L1EL stack.L1ELNodeID
	L1CL stack.L1CLNodeID

	L2   stack.L2NetworkID
	L2CL stack.L2CLNodeID
	L2EL stack.L2ELNodeID

	L2Batcher  stack.L2BatcherID
	L2Proposer stack.L2ProposerID
}

func NewDefaultMinimalSystemIDs(l1ID, l2ID eth.ChainID) DefaultMinimalSystemIDs {
	ids := DefaultMinimalSystemIDs{
		L1:         stack.L1NetworkID(l1ID),
		L1EL:       stack.L1ELNodeID{Key: "l1", ChainID: l1ID},
		L1CL:       stack.L1CLNodeID{Key: "l1", ChainID: l1ID},
		L2:         stack.L2NetworkID(l2ID),
		L2CL:       stack.L2CLNodeID{Key: "sequencer", ChainID: l2ID},
		L2EL:       stack.L2ELNodeID{Key: "sequencer", ChainID: l2ID},
		L2Batcher:  stack.L2BatcherID{Key: "main", ChainID: l2ID},
		L2Proposer: stack.L2ProposerID{Key: "main", ChainID: l2ID},
	}
	return ids
}

func DefaultMinimalSystem(dest *DefaultMinimalSystemIDs) stack.Option[*Orchestrator] {
	l1ID := eth.ChainIDFromUInt64(900)
	l2ID := eth.ChainIDFromUInt64(901)
	ids := NewDefaultMinimalSystemIDs(l1ID, l2ID)

	opt := stack.Combine[*Orchestrator]()
	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		o.P().Logger().Info("Setting up")
	}))

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	opt.Add(WithDeployer(),
		WithDeployerOptions(
			WithLocalContractSources(),
			WithCommons(ids.L1.ChainID()),
			WithPrefundedL2(ids.L2.ChainID()),
		),
	)

	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	opt.Add(WithL2ELNode(ids.L2EL, nil))
	opt.Add(WithL2CLNode(ids.L2CL, true, ids.L1CL, ids.L1EL, ids.L2EL))

	opt.Add(WithBatcher(ids.L2Batcher, ids.L1EL, ids.L2CL, ids.L2EL))
	opt.Add(WithProposer(ids.L2Proposer, ids.L1EL, &ids.L2CL, nil))

	opt.Add(WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2EL}))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}

// struct of the services, so we can access them later and do not have to guess their IDs.
type DefaultInteropSystemIDs struct {
	L1   stack.L1NetworkID
	L1EL stack.L1ELNodeID
	L1CL stack.L1CLNodeID

	Superchain stack.SuperchainID
	Cluster    stack.ClusterID

	Supervisor stack.SupervisorID
	Sequencer  stack.SequencerID

	L2A   stack.L2NetworkID
	L2ACL stack.L2CLNodeID
	L2AEL stack.L2ELNodeID

	L2B   stack.L2NetworkID
	L2BCL stack.L2CLNodeID
	L2BEL stack.L2ELNodeID

	L2ABatcher stack.L2BatcherID
	L2BBatcher stack.L2BatcherID

	L2AProposer stack.L2ProposerID
	L2BProposer stack.L2ProposerID
}

func NewDefaultInteropSystemIDs(l1ID, l2AID, l2BID eth.ChainID) DefaultInteropSystemIDs {
	ids := DefaultInteropSystemIDs{
		L1:          stack.L1NetworkID(l1ID),
		L1EL:        stack.L1ELNodeID{Key: "l1", ChainID: l1ID},
		L1CL:        stack.L1CLNodeID{Key: "l1", ChainID: l1ID},
		Superchain:  "main", // TODO(#15244): hardcoded to match the deployer default ID
		Cluster:     "main",
		Supervisor:  "1-primary", // prefix with number for ordering of supervisors
		Sequencer:   "dev",
		L2A:         stack.L2NetworkID(l2AID),
		L2ACL:       stack.L2CLNodeID{Key: "sequencer", ChainID: l2AID},
		L2AEL:       stack.L2ELNodeID{Key: "sequencer", ChainID: l2AID},
		L2B:         stack.L2NetworkID(l2BID),
		L2BCL:       stack.L2CLNodeID{Key: "sequencer", ChainID: l2BID},
		L2BEL:       stack.L2ELNodeID{Key: "sequencer", ChainID: l2BID},
		L2ABatcher:  stack.L2BatcherID{Key: "main", ChainID: l2AID},
		L2BBatcher:  stack.L2BatcherID{Key: "main", ChainID: l2BID},
		L2AProposer: stack.L2ProposerID{Key: "main", ChainID: l2AID},
		L2BProposer: stack.L2ProposerID{Key: "main", ChainID: l2BID},
	}
	return ids
}

func DefaultInteropSystem(dest *DefaultInteropSystemIDs) stack.Option[*Orchestrator] {
	l1ID := eth.ChainIDFromUInt64(900)
	l2AID := eth.ChainIDFromUInt64(901)
	l2BID := eth.ChainIDFromUInt64(902)
	ids := NewDefaultInteropSystemIDs(l1ID, l2AID, l2BID)

	opt := stack.Combine[*Orchestrator]()
	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		o.P().Logger().Info("Setting up")
	}))

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	opt.Add(WithDeployer(),
		WithDeployerOptions(
			WithLocalContractSources(),
			WithCommons(ids.L1.ChainID()),
			WithPrefundedL2(ids.L2A.ChainID()),
			WithPrefundedL2(ids.L2B.ChainID()),
			WithInteropAtGenesis(), // this can be overridden by later options
		),
	)

	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	opt.Add(WithSupervisor(ids.Supervisor, ids.Cluster, ids.L1EL))

	opt.Add(WithL2ELNode(ids.L2AEL, &ids.Supervisor))
	opt.Add(WithL2ELNode(ids.L2BEL, &ids.Supervisor))

	opt.Add(WithL2CLNode(ids.L2ACL, true, ids.L1CL, ids.L1EL, ids.L2AEL))
	opt.Add(WithL2CLNode(ids.L2BCL, true, ids.L1CL, ids.L1EL, ids.L2BEL))

	opt.Add(WithSequencer(ids.Sequencer, ids.L2ACL, ids.L1EL, ids.L2AEL))

	opt.Add(WithBatcher(ids.L2ABatcher, ids.L1EL, ids.L2ACL, ids.L2AEL))
	opt.Add(WithBatcher(ids.L2BBatcher, ids.L1EL, ids.L2BCL, ids.L2BEL))

	opt.Add(WithManagedBySupervisor(ids.L2ACL, ids.Supervisor))
	opt.Add(WithManagedBySupervisor(ids.L2BCL, ids.Supervisor))

	opt.Add(WithProposer(ids.L2AProposer, ids.L1EL, nil, &ids.Supervisor))
	opt.Add(WithProposer(ids.L2BProposer, ids.L1EL, nil, &ids.Supervisor))

	// TODO(#15057): maybe L2 challenger

	opt.Add(WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2AEL, ids.L2BEL}))

	// Upon evaluation of the option, export the contents we created.
	// Ids here are static, but other things may be exported too.
	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}

type RedundantInteropSystemIDs struct {
	DefaultInteropSystemIDs

	L2A2CL stack.L2CLNodeID
	L2A2EL stack.L2ELNodeID
}

func RedundantInteropSystem(dest *RedundantInteropSystemIDs) stack.Option[*Orchestrator] {
	l1ID := eth.ChainIDFromUInt64(900)
	l2AID := eth.ChainIDFromUInt64(901)
	l2BID := eth.ChainIDFromUInt64(902)
	ids := RedundantInteropSystemIDs{
		DefaultInteropSystemIDs: NewDefaultInteropSystemIDs(l1ID, l2AID, l2BID),
		L2A2CL:                  stack.L2CLNodeID{Key: "verifier", ChainID: l2AID},
		L2A2EL:                  stack.L2ELNodeID{Key: "verifier", ChainID: l2AID},
	}

	// start with default interop system
	var parentIds DefaultInteropSystemIDs
	opt := stack.Combine[*Orchestrator]()
	opt.Add(DefaultInteropSystem(&parentIds))

	opt.Add(WithL2ELNode(ids.L2A2EL, &ids.Supervisor))
	opt.Add(WithL2CLNode(ids.L2A2CL, false, ids.L1CL, ids.L1EL, ids.L2A2EL))

	// verifier must be also managed or it cannot advance
	opt.Add(WithManagedBySupervisor(ids.L2A2CL, ids.Supervisor))

	// P2P connect L2CL nodes
	opt.Add(WithL2CLP2PConnection(ids.L2ACL, ids.L2A2CL))

	// Upon evaluation of the option, export the contents we created.
	// Ids here are static, but other things may be exported too.
	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}

type MultiSupervisorInteropSystemIDs struct {
	RedundantInteropSystemIDs

	SupervisorSecondary stack.SupervisorID

	L2B2CL stack.L2CLNodeID
	L2B2EL stack.L2ELNodeID
}

func MultiSupervisorInteropSystem(dest *MultiSupervisorInteropSystemIDs) stack.Option[*Orchestrator] {
	l1ID := eth.ChainIDFromUInt64(900)
	l2AID := eth.ChainIDFromUInt64(901)
	l2BID := eth.ChainIDFromUInt64(902)
	ids := MultiSupervisorInteropSystemIDs{
		RedundantInteropSystemIDs: RedundantInteropSystemIDs{
			DefaultInteropSystemIDs: NewDefaultInteropSystemIDs(l1ID, l2AID, l2BID),
			L2A2CL:                  stack.L2CLNodeID{Key: "verifier", ChainID: l2AID},
			L2A2EL:                  stack.L2ELNodeID{Key: "verifier", ChainID: l2AID},
		},
		SupervisorSecondary: "2-secondary", // prefix with number for ordering of supervisors
		L2B2CL:              stack.L2CLNodeID{Key: "verifier", ChainID: l2BID},
		L2B2EL:              stack.L2ELNodeID{Key: "verifier", ChainID: l2BID},
	}

	// start with default interop system
	var parentIds DefaultInteropSystemIDs
	opt := stack.Combine[*Orchestrator]()
	opt.Add(DefaultInteropSystem(&parentIds))

	// add backup supervisor
	opt.Add(WithSupervisor(ids.SupervisorSecondary, ids.Cluster, ids.L1EL))

	opt.Add(WithL2ELNode(ids.L2A2EL, &ids.SupervisorSecondary))
	opt.Add(WithL2CLNode(ids.L2A2CL, false, ids.L1CL, ids.L1EL, ids.L2A2EL))

	opt.Add(WithL2ELNode(ids.L2B2EL, &ids.SupervisorSecondary))
	opt.Add(WithL2CLNode(ids.L2B2CL, false, ids.L1CL, ids.L1EL, ids.L2B2EL))

	// verifier must be also managed or it cannot advance
	// we attach verifer L2CL with backup supervisor
	opt.Add(WithManagedBySupervisor(ids.L2A2CL, ids.SupervisorSecondary))
	opt.Add(WithManagedBySupervisor(ids.L2B2CL, ids.SupervisorSecondary))

	// P2P connect L2CL nodes
	opt.Add(WithL2CLP2PConnection(ids.L2ACL, ids.L2A2CL))
	opt.Add(WithL2CLP2PConnection(ids.L2BCL, ids.L2B2CL))

	// Upon evaluation of the option, export the contents we created.
	// Ids here are static, but other things may be exported too.
	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}
