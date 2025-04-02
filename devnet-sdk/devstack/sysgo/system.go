package sysgo

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// struct of the services, so we can access them later and do not have to guess their IDs.
type DefaultInteropSystemIDs struct {
	L1   stack.L1NetworkID
	L1EL stack.L1ELNodeID
	L1CL stack.L1CLNodeID

	Superchain stack.SuperchainID
	Cluster    stack.ClusterID

	Supervisor stack.SupervisorID

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

func DefaultInteropSystem(contractPaths ContractPaths, dest *DefaultInteropSystemIDs) stack.Option {
	l1ID := eth.ChainIDFromUInt64(900)
	l2AID := eth.ChainIDFromUInt64(901)
	l2BID := eth.ChainIDFromUInt64(902)
	ids := DefaultInteropSystemIDs{
		L1:          stack.L1NetworkID{Key: "l1", ChainID: l1ID},
		L1EL:        stack.L1ELNodeID{Key: "l1", ChainID: l1ID},
		L1CL:        stack.L1CLNodeID{Key: "l1", ChainID: l1ID},
		Superchain:  "dev",
		Cluster:     "dev",
		Supervisor:  "dev",
		L2A:         stack.L2NetworkID{Key: "l2A", ChainID: l2AID},
		L2ACL:       stack.L2CLNodeID{Key: "sequencer", ChainID: l2AID},
		L2AEL:       stack.L2ELNodeID{Key: "sequencer", ChainID: l2AID},
		L2B:         stack.L2NetworkID{Key: "l2B", ChainID: l2BID},
		L2BCL:       stack.L2CLNodeID{Key: "sequencer", ChainID: l2BID},
		L2BEL:       stack.L2ELNodeID{Key: "sequencer", ChainID: l2BID},
		L2ABatcher:  stack.L2BatcherID{Key: "main", ChainID: l2AID},
		L2BBatcher:  stack.L2BatcherID{Key: "main", ChainID: l2BID},
		L2AProposer: stack.L2ProposerID{Key: "main", ChainID: l2AID},
		L2BProposer: stack.L2ProposerID{Key: "main", ChainID: l2BID},
	}

	opt := stack.Option(func(o stack.Orchestrator) {
		o.P().Logger().Info("Setting up")
	})

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	opt.Add(WithInteropGen(ids.L1, ids.Superchain, ids.Cluster,
		[]stack.L2NetworkID{ids.L2A, ids.L2B}, contractPaths))

	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	opt.Add(WithSupervisor(ids.Supervisor, ids.Cluster, ids.L1EL))

	// TODO(#15027): create L1 Faucet

	opt.Add(WithL2ELNode(ids.L2AEL, &ids.Supervisor))
	opt.Add(WithL2ELNode(ids.L2BEL, &ids.Supervisor))

	// TODO(#15027): create L2 faucet

	opt.Add(WithL2CLNode(ids.L2ACL, true, ids.L1CL, ids.L1EL, ids.L2AEL))
	opt.Add(WithL2CLNode(ids.L2BCL, true, ids.L1CL, ids.L1EL, ids.L2BEL))

	opt.Add(WithBatcher(ids.L2ABatcher, ids.L1EL, ids.L2ACL, ids.L2AEL))
	opt.Add(WithBatcher(ids.L2BBatcher, ids.L1EL, ids.L2BCL, ids.L2BEL))

	opt.Add(WithManagedBySupervisor(ids.L2ACL, ids.Supervisor))
	opt.Add(WithManagedBySupervisor(ids.L2BCL, ids.Supervisor))

	opt.Add(WithProposer(ids.L2AProposer, ids.L1EL, nil, &ids.Supervisor))
	opt.Add(WithProposer(ids.L2BProposer, ids.L1EL, nil, &ids.Supervisor))

	// TODO(#15057): maybe L2 challenger

	// Upon evaluation of the option, export the contents we created.
	// Ids here are static, but other things may be exported too.
	opt.Add(func(orch stack.Orchestrator) {
		*dest = ids
	})

	return opt
}
