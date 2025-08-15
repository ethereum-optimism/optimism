package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var (
	DefaultL1ID  = eth.ChainIDFromUInt64(900)
	DefaultL2AID = eth.ChainIDFromUInt64(901)
	DefaultL2BID = eth.ChainIDFromUInt64(902)
)

type DefaultMinimalSystemIDs struct {
	L1   stack.L1NetworkID
	L1EL stack.L1ELNodeID
	L1CL stack.L1CLNodeID

	L2   stack.L2NetworkID
	L2CL stack.L2CLNodeID
	L2EL stack.L2ELNodeID

	L2Batcher    stack.L2BatcherID
	L2Proposer   stack.L2ProposerID
	L2Challenger stack.L2ChallengerID

	TestSequencer stack.TestSequencerID
}

func NewDefaultMinimalSystemIDs(l1ID, l2ID eth.ChainID) DefaultMinimalSystemIDs {
	ids := DefaultMinimalSystemIDs{
		L1:            stack.L1NetworkID(l1ID),
		L1EL:          stack.NewL1ELNodeID("l1", l1ID),
		L1CL:          stack.NewL1CLNodeID("l1", l1ID),
		L2:            stack.L2NetworkID(l2ID),
		L2CL:          stack.NewL2CLNodeID("sequencer", l2ID),
		L2EL:          stack.NewL2ELNodeID("sequencer", l2ID),
		L2Batcher:     stack.NewL2BatcherID("main", l2ID),
		L2Proposer:    stack.NewL2ProposerID("main", l2ID),
		L2Challenger:  stack.NewL2ChallengerID("main", l2ID),
		TestSequencer: "test-sequencer",
	}
	return ids
}

func DefaultMinimalSystem(dest *DefaultMinimalSystemIDs) stack.Option[*Orchestrator] {
	ids := NewDefaultMinimalSystemIDs(DefaultL1ID, DefaultL2AID)

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

	opt.Add(WithL2ELNode(ids.L2EL))
	opt.Add(WithL2CLNode(ids.L2CL, ids.L1CL, ids.L1EL, ids.L2EL, L2CLSequencer()))

	opt.Add(WithBatcher(ids.L2Batcher, ids.L1EL, ids.L2CL, ids.L2EL))
	opt.Add(WithProposer(ids.L2Proposer, ids.L1EL, &ids.L2CL, nil))

	opt.Add(WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2EL}))

	opt.Add(WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2CL, ids.L1EL, ids.L2EL))

	opt.Add(WithL2Challenger(ids.L2Challenger, ids.L1EL, ids.L1CL, nil, nil, &ids.L2CL, []stack.L2ELNodeID{
		ids.L2EL,
	}))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}

type DefaultMinimalSystemWithSyncTesterIDs struct {
	DefaultMinimalSystemIDs

	SyncTester stack.SyncTesterID
}

func NewDefaultMinimalSystemWithSyncTesterIDs(l1ID, l2ID eth.ChainID) DefaultMinimalSystemWithSyncTesterIDs {
	minimal := NewDefaultMinimalSystemIDs(l1ID, l2ID)
	return DefaultMinimalSystemWithSyncTesterIDs{
		DefaultMinimalSystemIDs: minimal,
		SyncTester:              stack.NewSyncTesterID("s", l2ID),
	}
}

func DefaultMinimalSystemWithSyncTester(dest *DefaultMinimalSystemWithSyncTesterIDs) stack.Option[*Orchestrator] {
	l1ID := eth.ChainIDFromUInt64(900)
	l2ID := eth.ChainIDFromUInt64(901)
	ids := NewDefaultMinimalSystemWithSyncTesterIDs(l1ID, l2ID)

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

	opt.Add(WithL2ELNode(ids.L2EL))
	opt.Add(WithL2CLNode(ids.L2CL, ids.L1CL, ids.L1EL, ids.L2EL, L2CLSequencer()))

	opt.Add(WithBatcher(ids.L2Batcher, ids.L1EL, ids.L2CL, ids.L2EL))
	opt.Add(WithProposer(ids.L2Proposer, ids.L1EL, &ids.L2CL, nil))

	opt.Add(WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2EL}))

	opt.Add(WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2CL, ids.L1EL, ids.L2EL))

	opt.Add(WithL2Challenger(ids.L2Challenger, ids.L1EL, ids.L1CL, nil, nil, &ids.L2CL, []stack.L2ELNodeID{
		ids.L2EL,
	}))

	opt.Add(WithSyncTesters([]stack.L2ELNodeID{ids.L2EL}))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}

type DefaultSingleChainInteropSystemIDs struct {
	L1   stack.L1NetworkID
	L1EL stack.L1ELNodeID
	L1CL stack.L1CLNodeID

	Superchain stack.SuperchainID
	Cluster    stack.ClusterID

	Supervisor    stack.SupervisorID
	TestSequencer stack.TestSequencerID

	L2A   stack.L2NetworkID
	L2ACL stack.L2CLNodeID
	L2AEL stack.L2ELNodeID

	L2ABatcher    stack.L2BatcherID
	L2AProposer   stack.L2ProposerID
	L2ChallengerA stack.L2ChallengerID
}

func NewDefaultSingleChainInteropSystemIDs(l1ID, l2AID eth.ChainID) DefaultSingleChainInteropSystemIDs {
	ids := DefaultSingleChainInteropSystemIDs{
		L1:            stack.L1NetworkID(l1ID),
		L1EL:          stack.NewL1ELNodeID("l1", l1ID),
		L1CL:          stack.NewL1CLNodeID("l1", l1ID),
		Superchain:    "main", // TODO(#15244): hardcoded to match the deployer default ID
		Cluster:       stack.ClusterID("main"),
		Supervisor:    "1-primary", // prefix with number for ordering of supervisors
		TestSequencer: "dev",
		L2A:           stack.L2NetworkID(l2AID),
		L2ACL:         stack.NewL2CLNodeID("sequencer", l2AID),
		L2AEL:         stack.NewL2ELNodeID("sequencer", l2AID),
		L2ABatcher:    stack.NewL2BatcherID("main", l2AID),
		L2AProposer:   stack.NewL2ProposerID("main", l2AID),
		L2ChallengerA: stack.NewL2ChallengerID("main", l2AID),
	}
	return ids
}

func DefaultSingleChainInteropSystem(dest *DefaultSingleChainInteropSystemIDs) stack.Option[*Orchestrator] {
	ids := NewDefaultSingleChainInteropSystemIDs(DefaultL1ID, DefaultL2AID)
	opt := stack.Combine[*Orchestrator]()
	opt.Add(baseInteropSystem(&ids))

	opt.Add(WithL2Challenger(ids.L2ChallengerA, ids.L1EL, ids.L1CL, &ids.Supervisor, &ids.Cluster, &ids.L2ACL, []stack.L2ELNodeID{
		ids.L2AEL,
	}))

	opt.Add(WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2AEL}))

	// Upon evaluation of the option, export the contents we created.
	// Ids here are static, but other things may be exported too.
	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}

// baseInteropSystem defines a system that supports interop with a single chain
// Components which are shared across multiple chains are not started, allowing them to be added later including
// any additional chains that have been added.
func baseInteropSystem(ids *DefaultSingleChainInteropSystemIDs) stack.Option[*Orchestrator] {
	opt := stack.Combine[*Orchestrator]()
	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		o.P().Logger().Info("Setting up")
	}))

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	opt.Add(WithDeployer(),
		WithDeployerOptions(
			WithLocalContractSources(),
			WithCommons(ids.L1.ChainID()),
			WithPrefundedL2(ids.L1.ChainID(), ids.L2A.ChainID()),
			WithInteropAtGenesis(), // this can be overridden by later options
		),
	)

	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	opt.Add(WithSupervisor(ids.Supervisor, ids.Cluster, ids.L1EL))

	opt.Add(WithL2ELNode(ids.L2AEL, L2ELWithSupervisor(ids.Supervisor)))
	opt.Add(WithL2CLNode(ids.L2ACL, ids.L1CL, ids.L1EL, ids.L2AEL, L2CLSequencer(), L2CLIndexing()))
	opt.Add(WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2ACL, ids.L1EL, ids.L2AEL))
	opt.Add(WithBatcher(ids.L2ABatcher, ids.L1EL, ids.L2ACL, ids.L2AEL))

	opt.Add(WithManagedBySupervisor(ids.L2ACL, ids.Supervisor))

	// Note: we provide L2 CL nodes still, even though they are not used post-interop.
	// Since we may create an interop infra-setup, before interop is even scheduled to run.
	opt.Add(WithProposer(ids.L2AProposer, ids.L1EL, &ids.L2ACL, &ids.Supervisor))
	return opt
}

// struct of the services, so we can access them later and do not have to guess their IDs.
type DefaultInteropSystemIDs struct {
	DefaultSingleChainInteropSystemIDs

	L2B   stack.L2NetworkID
	L2BCL stack.L2CLNodeID
	L2BEL stack.L2ELNodeID

	L2BBatcher    stack.L2BatcherID
	L2BProposer   stack.L2ProposerID
	L2ChallengerB stack.L2ChallengerID
}

func NewDefaultInteropSystemIDs(l1ID, l2AID, l2BID eth.ChainID) DefaultInteropSystemIDs {
	ids := DefaultInteropSystemIDs{
		DefaultSingleChainInteropSystemIDs: NewDefaultSingleChainInteropSystemIDs(l1ID, l2AID),
		L2B:                                stack.L2NetworkID(l2BID),
		L2BCL:                              stack.NewL2CLNodeID("sequencer", l2BID),
		L2BEL:                              stack.NewL2ELNodeID("sequencer", l2BID),
		L2BBatcher:                         stack.NewL2BatcherID("main", l2BID),
		L2BProposer:                        stack.NewL2ProposerID("main", l2BID),
		L2ChallengerB:                      stack.NewL2ChallengerID("main", l2BID),
	}
	return ids
}

func DefaultInteropSystem(dest *DefaultInteropSystemIDs) stack.Option[*Orchestrator] {
	ids := NewDefaultInteropSystemIDs(DefaultL1ID, DefaultL2AID, DefaultL2BID)
	opt := stack.Combine[*Orchestrator]()

	// start with single chain interop system
	opt.Add(baseInteropSystem(&ids.DefaultSingleChainInteropSystemIDs))

	opt.Add(WithDeployerOptions(
		WithPrefundedL2(ids.L1.ChainID(), ids.L2B.ChainID()),
		WithInteropAtGenesis(), // this can be overridden by later options
	))
	opt.Add(WithL2ELNode(ids.L2BEL, L2ELWithSupervisor(ids.Supervisor)))
	opt.Add(WithL2CLNode(ids.L2BCL, ids.L1CL, ids.L1EL, ids.L2BEL, L2CLSequencer(), L2CLIndexing()))
	opt.Add(WithBatcher(ids.L2BBatcher, ids.L1EL, ids.L2BCL, ids.L2BEL))

	opt.Add(WithManagedBySupervisor(ids.L2BCL, ids.Supervisor))

	// Note: we provide L2 CL nodes still, even though they are not used post-interop.
	// Since we may create an interop infra-setup, before interop is even scheduled to run.
	opt.Add(WithProposer(ids.L2BProposer, ids.L1EL, &ids.L2BCL, &ids.Supervisor))

	// Deploy separate challengers for each chain.  Can be reduced to a single challenger when the DisputeGameFactory
	// is actually shared.
	opt.Add(WithL2Challenger(ids.L2ChallengerA, ids.L1EL, ids.L1CL, &ids.Supervisor, &ids.Cluster, &ids.L2ACL, []stack.L2ELNodeID{
		ids.L2AEL, ids.L2BEL,
	}))
	opt.Add(WithL2Challenger(ids.L2ChallengerB, ids.L1EL, ids.L1CL, &ids.Supervisor, &ids.Cluster, &ids.L2BCL, []stack.L2ELNodeID{
		ids.L2BEL, ids.L2AEL,
	}))

	opt.Add(WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2AEL, ids.L2BEL}))

	// Upon evaluation of the option, export the contents we created.
	// Ids here are static, but other things may be exported too.
	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}

func DefaultIsthmusSuperProofsSystem(dest *DefaultInteropSystemIDs) stack.Option[*Orchestrator] {
	return defaultSuperProofsSystem(dest)
}

func DefaultInteropProofsSystem(dest *DefaultInteropSystemIDs) stack.Option[*Orchestrator] {
	return defaultSuperProofsSystem(dest, WithInteropAtGenesis())
}

func defaultSuperProofsSystem(dest *DefaultInteropSystemIDs, deployerOpts ...DeployerOption) stack.CombinedOption[*Orchestrator] {
	ids := NewDefaultInteropSystemIDs(DefaultL1ID, DefaultL2AID, DefaultL2BID)
	opt := stack.Combine[*Orchestrator]()

	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		o.P().Logger().Info("Setting up")
	}))

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	opt.Add(WithDeployer(), WithDeployerOptions(
		append([]DeployerOption{
			WithLocalContractSources(),
			WithCommons(ids.L1.ChainID()),
			WithPrefundedL2(ids.L1.ChainID(), ids.L2A.ChainID()),
			WithPrefundedL2(ids.L1.ChainID(), ids.L2B.ChainID()),
		}, deployerOpts...)...))

	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	opt.Add(WithSupervisor(ids.Supervisor, ids.Cluster, ids.L1EL))

	opt.Add(WithL2ELNode(ids.L2AEL, L2ELWithSupervisor(ids.Supervisor)))
	opt.Add(WithL2CLNode(ids.L2ACL, ids.L1CL, ids.L1EL, ids.L2AEL, L2CLSequencer(), L2CLIndexing()))

	opt.Add(WithL2ELNode(ids.L2BEL, L2ELWithSupervisor(ids.Supervisor)))
	opt.Add(WithL2CLNode(ids.L2BCL, ids.L1CL, ids.L1EL, ids.L2BEL, L2CLSequencer(), L2CLIndexing()))

	opt.Add(WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2ACL, ids.L1EL, ids.L2AEL))

	opt.Add(WithBatcher(ids.L2ABatcher, ids.L1EL, ids.L2ACL, ids.L2AEL))
	opt.Add(WithBatcher(ids.L2BBatcher, ids.L1EL, ids.L2BCL, ids.L2BEL))

	opt.Add(WithManagedBySupervisor(ids.L2ACL, ids.Supervisor))
	opt.Add(WithManagedBySupervisor(ids.L2BCL, ids.Supervisor))

	opt.Add(WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2AEL, ids.L2BEL}))

	opt.Add(WithSuperRoots(ids.L1.ChainID(), ids.L1EL, ids.L2ACL, ids.Supervisor, ids.L2A.ChainID()))

	opt.Add(WithSuperProposer(ids.L2AProposer, ids.L1EL, &ids.Supervisor))

	opt.Add(WithSuperL2Challenger(ids.L2ChallengerA, ids.L1EL, ids.L1CL, &ids.Supervisor, &ids.Cluster, []stack.L2ELNodeID{
		ids.L2BEL, ids.L2AEL,
	}))

	// Upon evaluation of the option, export the contents we created.
	// Ids here are static, but other things may be exported too.
	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}

type MultiSupervisorInteropSystemIDs struct {
	DefaultInteropSystemIDs

	// Supervisor does not support multinode so need a additional supervisor for verifier nodes
	SupervisorSecondary stack.SupervisorID

	L2A2CL stack.L2CLNodeID
	L2A2EL stack.L2ELNodeID
	L2B2CL stack.L2CLNodeID
	L2B2EL stack.L2ELNodeID
}

func MultiSupervisorInteropSystem(dest *MultiSupervisorInteropSystemIDs) stack.Option[*Orchestrator] {
	ids := MultiSupervisorInteropSystemIDs{
		DefaultInteropSystemIDs: NewDefaultInteropSystemIDs(DefaultL1ID, DefaultL2AID, DefaultL2BID),
		SupervisorSecondary:     "2-secondary", // prefix with number for ordering of supervisors
		L2A2CL:                  stack.NewL2CLNodeID("verifier", DefaultL2AID),
		L2A2EL:                  stack.NewL2ELNodeID("verifier", DefaultL2AID),
		L2B2CL:                  stack.NewL2CLNodeID("verifier", DefaultL2BID),
		L2B2EL:                  stack.NewL2ELNodeID("verifier", DefaultL2BID),
	}

	// start with default interop system
	var parentIds DefaultInteropSystemIDs
	opt := stack.Combine[*Orchestrator]()
	opt.Add(DefaultInteropSystem(&parentIds))

	// add backup supervisor
	opt.Add(WithSupervisor(ids.SupervisorSecondary, ids.Cluster, ids.L1EL))

	opt.Add(WithL2ELNode(ids.L2A2EL, L2ELWithSupervisor(ids.SupervisorSecondary)))
	opt.Add(WithL2CLNode(ids.L2A2CL, ids.L1CL, ids.L1EL, ids.L2A2EL, L2CLIndexing()))

	opt.Add(WithL2ELNode(ids.L2B2EL, L2ELWithSupervisor(ids.SupervisorSecondary)))
	opt.Add(WithL2CLNode(ids.L2B2CL, ids.L1CL, ids.L1EL, ids.L2B2EL, L2CLIndexing()))

	// verifier must be also managed or it cannot advance
	// we attach verifier L2CL with backup supervisor
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
