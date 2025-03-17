package systemgo

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// struct of the services, so we can access them later and do not have to guess their IDs.
type DefaultInteropSystemIDs struct {
	L1   system2.L1NetworkID
	L1EL system2.L1ELNodeID
	L1CL system2.L1CLNodeID

	Superchain system2.SuperchainID
	Cluster    system2.ClusterID

	Supervisor system2.SupervisorID

	L2A   system2.L2NetworkID
	L2ACL system2.L2CLNodeID
	L2AEL system2.L2ELNodeID

	L2B   system2.L2NetworkID
	L2BCL system2.L2CLNodeID
	L2BEL system2.L2ELNodeID

	L2ABatcher system2.L2BatcherID
	L2BBatcher system2.L2BatcherID

	L2AProposer system2.L2ProposerID
	L2BProposer system2.L2ProposerID
}

func DefaultInteropSystem(contractPaths ContractPaths) (DefaultInteropSystemIDs, system2.Option) {
	l1ID := eth.ChainIDFromUInt64(900)
	l2AID := eth.ChainIDFromUInt64(901)
	l2BID := eth.ChainIDFromUInt64(902)
	ids := DefaultInteropSystemIDs{
		L1:          system2.L1NetworkID{Key: "l1", ChainID: l1ID},
		L1EL:        system2.L1ELNodeID{Key: "l1", ChainID: l1ID},
		L1CL:        system2.L1CLNodeID{Key: "l1", ChainID: l1ID},
		Superchain:  "dev",
		Cluster:     "dev",
		Supervisor:  "dev",
		L2A:         system2.L2NetworkID{Key: "l2A", ChainID: l2AID},
		L2ACL:       system2.L2CLNodeID{Key: "sequencer", ChainID: l2AID},
		L2AEL:       system2.L2ELNodeID{Key: "sequencer", ChainID: l2AID},
		L2B:         system2.L2NetworkID{Key: "l2B", ChainID: l2BID},
		L2BCL:       system2.L2CLNodeID{Key: "sequencer", ChainID: l2BID},
		L2BEL:       system2.L2ELNodeID{Key: "sequencer", ChainID: l2BID},
		L2ABatcher:  system2.L2BatcherID{Key: "main", ChainID: l2AID},
		L2BBatcher:  system2.L2BatcherID{Key: "main", ChainID: l2BID},
		L2AProposer: system2.L2ProposerID{Key: "main", ChainID: l2AID},
		L2BProposer: system2.L2ProposerID{Key: "main", ChainID: l2BID},
	}

	opt := system2.Option(func(setup *system2.Setup) {
		setup.Log.Info("Setting up")
	})

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	opt.Add(WithInteropGen(ids.L1, ids.Superchain, ids.Cluster,
		[]system2.L2NetworkID{ids.L2A, ids.L2B}, contractPaths))

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

	return ids, opt
}
