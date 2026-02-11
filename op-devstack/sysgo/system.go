package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
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
	return defaultMinimalSystemOpts(&ids, dest)
}

func defaultMinimalSystemOpts(ids *DefaultMinimalSystemIDs, dest *DefaultMinimalSystemIDs) stack.CombinedOption[*Orchestrator] {
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

	// Level 1: L1 nodes (must be first)
	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	// Level 2: L2 EL (depends on L1)
	opt.Add(WithL2ELNode(ids.L2EL))

	// Level 3: L2 CL + Faucets in parallel (both only need L1 + L2 EL)
	opt.Add(stack.InParallel[*Orchestrator](
		WithL2CLNode(ids.L2CL, ids.L1CL, ids.L1EL, ids.L2EL, L2CLSequencer()),
		WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2EL}),
	))

	// Level 4: Services that need L2 CL, in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithBatcher(ids.L2Batcher, ids.L1EL, ids.L2CL, ids.L2EL),
		WithProposer(ids.L2Proposer, ids.L1EL, &ids.L2CL, nil),
		WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2CL, ids.L1EL, ids.L2EL),
		WithL2Challenger(ids.L2Challenger, ids.L1EL, ids.L1CL, nil, nil, &ids.L2CL, []stack.L2ELNodeID{
			ids.L2EL,
		}),
	))

	opt.Add(WithL2MetricsDashboard())

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = *ids
	}))

	return opt
}

// DefaultTwoL2System defines a minimal system with a single L1 and two L2 chains,
// without interop or supervisor: both L2s get their own ELs, and we attach L2CL nodes
// via the default L2CL selector (which can be set to supernode to share a single process).
type DefaultTwoL2SystemIDs struct {
	L1   stack.L1NetworkID
	L1EL stack.L1ELNodeID
	L1CL stack.L1CLNodeID

	L2A   stack.L2NetworkID
	L2ACL stack.L2CLNodeID
	L2AEL stack.L2ELNodeID

	L2B   stack.L2NetworkID
	L2BCL stack.L2CLNodeID
	L2BEL stack.L2ELNodeID

	Supernode   stack.SupernodeID
	L2ABatcher  stack.L2BatcherID
	L2AProposer stack.L2ProposerID
	L2BBatcher  stack.L2BatcherID
	L2BProposer stack.L2ProposerID
}

func NewDefaultTwoL2SystemIDs(l1ID, l2AID, l2BID eth.ChainID) DefaultTwoL2SystemIDs {
	return DefaultTwoL2SystemIDs{
		L1:          stack.L1NetworkID(l1ID),
		L1EL:        stack.NewL1ELNodeID("l1", l1ID),
		L1CL:        stack.NewL1CLNodeID("l1", l1ID),
		L2A:         stack.L2NetworkID(l2AID),
		L2ACL:       stack.NewL2CLNodeID("sequencer", l2AID),
		L2AEL:       stack.NewL2ELNodeID("sequencer", l2AID),
		L2B:         stack.L2NetworkID(l2BID),
		L2BCL:       stack.NewL2CLNodeID("sequencer", l2BID),
		L2BEL:       stack.NewL2ELNodeID("sequencer", l2BID),
		Supernode:   stack.NewSupernodeID("supernode-two-l2-system", l2AID, l2BID),
		L2ABatcher:  stack.NewL2BatcherID("main", l2AID),
		L2AProposer: stack.NewL2ProposerID("main", l2AID),
		L2BBatcher:  stack.NewL2BatcherID("main", l2BID),
		L2BProposer: stack.NewL2ProposerID("main", l2BID),
	}
}

func DefaultTwoL2System(dest *DefaultTwoL2SystemIDs) stack.Option[*Orchestrator] {
	ids := NewDefaultTwoL2SystemIDs(DefaultL1ID, DefaultL2AID, DefaultL2BID)
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
			WithPrefundedL2(ids.L1.ChainID(), ids.L2B.ChainID()),
		),
	)

	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	// Both L2 ELs in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithL2ELNode(ids.L2AEL),
		WithL2ELNode(ids.L2BEL),
	))

	// Both L2 CLs + Faucets in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithL2CLNode(ids.L2ACL, ids.L1CL, ids.L1EL, ids.L2AEL, L2CLSequencer()),
		WithL2CLNode(ids.L2BCL, ids.L1CL, ids.L1EL, ids.L2BEL, L2CLSequencer()),
		WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2AEL, ids.L2BEL}),
	))

	// All batchers and proposers in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithBatcher(ids.L2ABatcher, ids.L1EL, ids.L2ACL, ids.L2AEL),
		WithProposer(ids.L2AProposer, ids.L1EL, &ids.L2ACL, nil),
		WithBatcher(ids.L2BBatcher, ids.L1EL, ids.L2BCL, ids.L2BEL),
		WithProposer(ids.L2BProposer, ids.L1EL, &ids.L2BCL, nil),
	))

	opt.Add(WithL2MetricsDashboard())

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}

// DefaultSupernodeTwoL2System runs two L2 chains that share a single supernode instance for their CL,
// wiring thin L2CL wrappers that route via the supernode RPC router.
func DefaultSupernodeTwoL2System(dest *DefaultTwoL2SystemIDs) stack.Option[*Orchestrator] {
	ids := NewDefaultTwoL2SystemIDs(DefaultL1ID, DefaultL2AID, DefaultL2BID)
	opt := stack.Combine[*Orchestrator]()
	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		o.P().Logger().Info("Setting up (supernode)")
	}))

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	opt.Add(WithDeployer(),
		WithDeployerOptions(
			WithLocalContractSources(),
			WithCommons(ids.L1.ChainID()),
			WithPrefundedL2(ids.L1.ChainID(), ids.L2A.ChainID()),
			WithPrefundedL2(ids.L1.ChainID(), ids.L2B.ChainID()),
		),
	)

	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	// Both L2 ELs in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithL2ELNode(ids.L2AEL),
		WithL2ELNode(ids.L2BEL),
	))

	// Shared supernode for both L2 chains
	opt.Add(WithSharedSupernodeCLs(ids.Supernode, []L2CLs{{CLID: ids.L2ACL, ELID: ids.L2AEL}, {CLID: ids.L2BCL, ELID: ids.L2BEL}}, ids.L1CL, ids.L1EL))

	// All batchers, proposers, and faucets in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithBatcher(ids.L2ABatcher, ids.L1EL, ids.L2ACL, ids.L2AEL),
		WithProposer(ids.L2AProposer, ids.L1EL, &ids.L2ACL, nil),
		WithBatcher(ids.L2BBatcher, ids.L1EL, ids.L2BCL, ids.L2BEL),
		WithProposer(ids.L2BProposer, ids.L1EL, &ids.L2BCL, nil),
		WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2AEL, ids.L2BEL}),
	))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}

// DefaultSupernodeInteropTwoL2System runs two L2 chains with a shared supernode that has
// interop verification enabled. Use delaySeconds=0 for interop at genesis, or a positive value
// to test the transition from normal safety to interop-verified safety.
func DefaultSupernodeInteropTwoL2System(dest *DefaultTwoL2SystemIDs, delaySeconds uint64) stack.Option[*Orchestrator] {
	ids := NewDefaultTwoL2SystemIDs(DefaultL1ID, DefaultL2AID, DefaultL2BID)
	opt := stack.Combine[*Orchestrator]()
	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		if delaySeconds == 0 {
			o.P().Logger().Info("Setting up (supernode with interop)")
		} else {
			o.P().Logger().Info("Setting up (supernode with delayed interop)", "delay_seconds", delaySeconds)
		}
	}))

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	opt.Add(WithDeployer(),
		WithDeployerOptions(
			WithLocalContractSources(),
			WithCommons(ids.L1.ChainID()),
			WithPrefundedL2(ids.L1.ChainID(), ids.L2A.ChainID()),
			WithPrefundedL2(ids.L1.ChainID(), ids.L2B.ChainID()),
			WithInteropAtGenesis(), // Enable interop contracts
		),
	)

	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	// Both L2 ELs in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithL2ELNode(ids.L2AEL),
		WithL2ELNode(ids.L2BEL),
	))

	// Shared supernode for both L2 chains with interop enabled
	cls := []L2CLs{{CLID: ids.L2ACL, ELID: ids.L2AEL}, {CLID: ids.L2BCL, ELID: ids.L2BEL}}
	if delaySeconds == 0 {
		opt.Add(WithSharedSupernodeCLsInterop(ids.Supernode, cls, ids.L1CL, ids.L1EL))
	} else {
		opt.Add(WithSharedSupernodeCLsInteropDelayed(ids.Supernode, cls, ids.L1CL, ids.L1EL, delaySeconds))
	}

	// All batchers, proposers, and faucets in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithBatcher(ids.L2ABatcher, ids.L1EL, ids.L2ACL, ids.L2AEL),
		WithProposer(ids.L2AProposer, ids.L1EL, &ids.L2ACL, nil),
		WithBatcher(ids.L2BBatcher, ids.L1EL, ids.L2BCL, ids.L2BEL),
		WithProposer(ids.L2BProposer, ids.L1EL, &ids.L2BCL, nil),
		WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2AEL, ids.L2BEL}),
	))

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
		SyncTester:              stack.NewSyncTesterID("sync-tester", l2ID),
	}
}

func DefaultMinimalSystemWithSyncTester(dest *DefaultMinimalSystemWithSyncTesterIDs, fcu eth.FCUState) stack.Option[*Orchestrator] {
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

	// Level 1: L1 nodes
	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	// Level 2: L2 EL
	opt.Add(WithL2ELNode(ids.L2EL))

	// Level 3: L2 CL + Faucets + SyncTester in parallel (all only need L1 + L2 EL)
	opt.Add(stack.InParallel[*Orchestrator](
		WithL2CLNode(ids.L2CL, ids.L1CL, ids.L1EL, ids.L2EL, L2CLSequencer()),
		WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2EL}),
		WithSyncTester(ids.SyncTester, []stack.L2ELNodeID{ids.L2EL}),
	))

	// Level 4: Services that need L2 CL, in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithBatcher(ids.L2Batcher, ids.L1EL, ids.L2CL, ids.L2EL),
		WithProposer(ids.L2Proposer, ids.L1EL, &ids.L2CL, nil),
		WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2CL, ids.L1EL, ids.L2EL),
		WithL2Challenger(ids.L2Challenger, ids.L1EL, ids.L1CL, nil, nil, &ids.L2CL, []stack.L2ELNodeID{
			ids.L2EL,
		}),
	))

	opt.Add(WithL2MetricsDashboard())

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

	// Challenger and faucets in parallel (both depend on services started by baseInteropSystem)
	opt.Add(stack.InParallel[*Orchestrator](
		WithL2Challenger(ids.L2ChallengerA, ids.L1EL, ids.L1CL, &ids.Supervisor, &ids.Cluster, &ids.L2ACL, []stack.L2ELNodeID{
			ids.L2AEL,
		}),
		WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2AEL}),
	))

	// Upon evaluation of the option, export the contents we created.
	// Ids here are static, but other things may be exported too.
	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}

// DefaultMinimalInteropSystem creates a minimal system with interop contracts but no supervisor.
// This tests interop contract deployment with local finality (SupervisorEnabled=false in op-node).
func DefaultMinimalInteropSystem(dest *DefaultMinimalSystemIDs) stack.Option[*Orchestrator] {
	ids := NewDefaultMinimalSystemIDs(DefaultL1ID, DefaultL2AID)
	opt := stack.Combine[*Orchestrator]()

	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		o.P().Logger().Info("Setting up minimal interop (no supervisor)")
	}))

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	opt.Add(WithDeployer(),
		WithDeployerOptions(
			WithLocalContractSources(),
			WithCommons(ids.L1.ChainID()),
			WithPrefundedL2(ids.L1.ChainID(), ids.L2.ChainID()),
			WithInteropAtGenesis(),
		),
	)

	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	// No supervisor - interop with local finality only
	opt.Add(WithL2ELNode(ids.L2EL))

	// L2 CL + Faucets in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithL2CLNode(ids.L2CL, ids.L1CL, ids.L1EL, ids.L2EL, L2CLSequencer()),
		WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2EL}),
	))

	// Services that need L2 CL, in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2CL, ids.L1EL, ids.L2EL),
		WithBatcher(ids.L2Batcher, ids.L1EL, ids.L2CL, ids.L2EL),
		WithProposer(ids.L2Proposer, ids.L1EL, &ids.L2CL, nil),
	))

	opt.Add(WithL2MetricsDashboard())

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

	// Services that need L2 CL, in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2ACL, ids.L1EL, ids.L2AEL),
		WithBatcher(ids.L2ABatcher, ids.L1EL, ids.L2ACL, ids.L2AEL),
		WithManagedBySupervisor(ids.L2ACL, ids.Supervisor),
		// Note: we provide L2 CL nodes still, even though they are not used post-interop.
		// Since we may create an interop infra-setup, before interop is even scheduled to run.
		WithProposer(ids.L2AProposer, ids.L1EL, &ids.L2ACL, &ids.Supervisor),
	))

	opt.Add(WithL2MetricsDashboard())

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

	// Chain B post-CL services, challengers, and faucets in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithBatcher(ids.L2BBatcher, ids.L1EL, ids.L2BCL, ids.L2BEL),
		WithManagedBySupervisor(ids.L2BCL, ids.Supervisor),
		// Note: we provide L2 CL nodes still, even though they are not used post-interop.
		// Since we may create an interop infra-setup, before interop is even scheduled to run.
		WithProposer(ids.L2BProposer, ids.L1EL, &ids.L2BCL, &ids.Supervisor),
		// Deploy separate challengers for each chain.  Can be reduced to a single challenger when the DisputeGameFactory
		// is actually shared.
		WithL2Challenger(ids.L2ChallengerA, ids.L1EL, ids.L1CL, &ids.Supervisor, &ids.Cluster, &ids.L2ACL, []stack.L2ELNodeID{
			ids.L2AEL, ids.L2BEL,
		}),
		WithL2Challenger(ids.L2ChallengerB, ids.L1EL, ids.L1CL, &ids.Supervisor, &ids.Cluster, &ids.L2BCL, []stack.L2ELNodeID{
			ids.L2BEL, ids.L2AEL,
		}),
		WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2AEL, ids.L2BEL}),
	))

	opt.Add(WithL2MetricsDashboard())

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

type DefaultSupernodeInteropProofsSystemIDs struct {
	DefaultInteropSystemIDs
	Supernode stack.SupernodeID
}

func NewDefaultSupernodeInteropProofsSystemIDs(l1ID, l2AID, l2BID eth.ChainID) DefaultSupernodeInteropProofsSystemIDs {
	return DefaultSupernodeInteropProofsSystemIDs{
		DefaultInteropSystemIDs: NewDefaultInteropSystemIDs(l1ID, l2AID, l2BID),
		Supernode:               stack.NewSupernodeID("supernode-two-system-proofs", l2AID, l2BID),
	}
}

func DefaultSupernodeIsthmusSuperProofsSystem(dest *DefaultSupernodeInteropProofsSystemIDs) stack.Option[*Orchestrator] {
	return defaultSupernodeSuperProofsSystem(dest)
}

// DefaultSupernodeInteropProofsSystem creates a super-roots proofs system that sources super-roots via op-supernode
// (instead of op-supervisor). Interop is enabled at genesis.
func DefaultSupernodeInteropProofsSystem(dest *DefaultSupernodeInteropProofsSystemIDs) stack.Option[*Orchestrator] {
	return defaultSupernodeSuperProofsSystem(dest, WithInteropAtGenesis())
}

func defaultSupernodeSuperProofsSystem(dest *DefaultSupernodeInteropProofsSystemIDs, deployerOpts ...DeployerOption) stack.CombinedOption[*Orchestrator] {
	ids := NewDefaultSupernodeInteropProofsSystemIDs(DefaultL1ID, DefaultL2AID, DefaultL2BID)
	opt := stack.Combine[*Orchestrator]()

	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		o.P().Logger().Info("Setting up (supernode)")
	}))

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	opt.Add(WithDeployer(), WithDeployerOptions(
		append([]DeployerOption{
			WithLocalContractSources(),
			WithCommons(ids.L1.ChainID()),
			WithPrefundedL2(ids.L1.ChainID(), ids.L2A.ChainID()),
			WithPrefundedL2(ids.L1.ChainID(), ids.L2B.ChainID()),
			WithDevFeatureEnabled(deployer.OptimismPortalInteropDevFlag),
		}, deployerOpts...)...,
	))

	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	// Both L2 ELs in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithL2ELNode(ids.L2AEL),
		WithL2ELNode(ids.L2BEL),
	))

	// Shared supernode for both L2 chains (registers per-chain L2CL proxies)
	opt.Add(WithSharedSupernodeCLs(ids.Supernode, []L2CLs{{CLID: ids.L2ACL, ELID: ids.L2AEL}, {CLID: ids.L2BCL, ELID: ids.L2BEL}}, ids.L1CL, ids.L1EL))

	// TestSequencer and batchers in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2ACL, ids.L1EL, ids.L2AEL),
		WithBatcher(ids.L2ABatcher, ids.L1EL, ids.L2ACL, ids.L2AEL),
		WithBatcher(ids.L2BBatcher, ids.L1EL, ids.L2BCL, ids.L2BEL),
	))

	// Run super roots migration using supernode as super root source
	opt.Add(WithSuperRootsFromSupernode(ids.L1.ChainID(), ids.L1EL, []stack.L2CLNodeID{ids.L2ACL, ids.L2BCL}, ids.Supernode, ids.L2A.ChainID()))

	// Post-migration services in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		// Start challenger after migration; use supernode RPCs as super-roots source.
		WithSupernodeL2Challenger(ids.L2ChallengerA, ids.L1EL, ids.L1CL, &ids.Supernode, &ids.Cluster, []stack.L2ELNodeID{
			ids.L2BEL, ids.L2AEL,
		}),
		// Start proposer after migration; use supernode RPCs as proposal source.
		WithSupernodeProposer(ids.L2AProposer, ids.L1EL, &ids.Supernode),
		WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2AEL, ids.L2BEL}),
	))

	opt.Add(WithL2MetricsDashboard())

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
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
			WithDevFeatureEnabled(deployer.OptimismPortalInteropDevFlag),
		}, deployerOpts...)...))

	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	opt.Add(WithSupervisor(ids.Supervisor, ids.Cluster, ids.L1EL))

	// Both L2 ELs in parallel (both need supervisor)
	opt.Add(stack.InParallel[*Orchestrator](
		WithL2ELNode(ids.L2AEL, L2ELWithSupervisor(ids.Supervisor)),
		WithL2ELNode(ids.L2BEL, L2ELWithSupervisor(ids.Supervisor)),
	))

	// Both L2 CLs in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithL2CLNode(ids.L2ACL, ids.L1CL, ids.L1EL, ids.L2AEL, L2CLSequencer(), L2CLIndexing()),
		WithL2CLNode(ids.L2BCL, ids.L1CL, ids.L1EL, ids.L2BEL, L2CLSequencer(), L2CLIndexing()),
	))

	// Post-CL services in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2ACL, ids.L1EL, ids.L2AEL),
		WithBatcher(ids.L2ABatcher, ids.L1EL, ids.L2ACL, ids.L2AEL),
		WithBatcher(ids.L2BBatcher, ids.L1EL, ids.L2BCL, ids.L2BEL),
		WithManagedBySupervisor(ids.L2ACL, ids.Supervisor),
		WithManagedBySupervisor(ids.L2BCL, ids.Supervisor),
		WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2AEL, ids.L2BEL}),
	))

	opt.Add(WithSuperRoots(ids.L1.ChainID(), ids.L1EL, []stack.L2CLNodeID{ids.L2ACL, ids.L2BCL}, ids.Supervisor, ids.L2A.ChainID()))

	// Post-migration services in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithSuperProposer(ids.L2AProposer, ids.L1EL, &ids.Supervisor),
		WithSuperL2Challenger(ids.L2ChallengerA, ids.L1EL, ids.L1CL, &ids.Supervisor, &ids.Cluster, []stack.L2ELNodeID{
			ids.L2BEL, ids.L2AEL,
		}),
	))

	opt.Add(WithL2MetricsDashboard())

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

	// Both verifier L2 ELs in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithL2ELNode(ids.L2A2EL, L2ELWithSupervisor(ids.SupervisorSecondary)),
		WithL2ELNode(ids.L2B2EL, L2ELWithSupervisor(ids.SupervisorSecondary)),
	))

	// Both verifier L2 CLs in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithL2CLNode(ids.L2A2CL, ids.L1CL, ids.L1EL, ids.L2A2EL, L2CLIndexing()),
		WithL2CLNode(ids.L2B2CL, ids.L1CL, ids.L1EL, ids.L2B2EL, L2CLIndexing()),
	))

	// Post-CL services in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		// verifier must be also managed or it cannot advance
		// we attach verifier L2CL with backup supervisor
		WithManagedBySupervisor(ids.L2A2CL, ids.SupervisorSecondary),
		WithManagedBySupervisor(ids.L2B2CL, ids.SupervisorSecondary),
		// P2P connect L2CL nodes
		WithL2CLP2PConnection(ids.L2ACL, ids.L2A2CL),
		WithL2CLP2PConnection(ids.L2BCL, ids.L2B2CL),
	))

	opt.Add(WithL2MetricsDashboard())

	// Upon evaluation of the option, export the contents we created.
	// Ids here are static, but other things may be exported too.
	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}

func ProofSystem(dest *DefaultMinimalSystemIDs) stack.Option[*Orchestrator] {
	ids := NewDefaultMinimalSystemIDs(DefaultL1ID, DefaultL2AID)
	opt := defaultMinimalSystemOpts(&ids, dest)
	opt.Add(WithCannonGameTypeAdded(ids.L1EL, ids.L2.ChainID()))
	opt.Add(WithCannonKonaGameTypeAdded())
	return opt
}

type SingleChainSystemWithFlashblocksIDs struct {
	L1   stack.L1NetworkID
	L1EL stack.L1ELNodeID
	L1CL stack.L1CLNodeID

	L2            stack.L2NetworkID
	L2CL          stack.L2CLNodeID
	L2EL          stack.L2ELNodeID
	L2Builder     stack.OPRBuilderNodeID
	L2RollupBoost stack.RollupBoostNodeID

	L2Batcher    stack.L2BatcherID
	L2Proposer   stack.L2ProposerID
	L2Challenger stack.L2ChallengerID

	TestSequencer stack.TestSequencerID
}

func NewDefaultSingleChainSystemWithFlashblocksIDs(l1ID, l2ID eth.ChainID) SingleChainSystemWithFlashblocksIDs {
	ids := SingleChainSystemWithFlashblocksIDs{
		L1:            stack.L1NetworkID(l1ID),
		L1EL:          stack.NewL1ELNodeID("l1", l1ID),
		L1CL:          stack.NewL1CLNodeID("l1", l1ID),
		L2:            stack.L2NetworkID(l2ID),
		L2CL:          stack.NewL2CLNodeID("sequencer", l2ID),
		L2EL:          stack.NewL2ELNodeID("sequencer", l2ID),
		L2Builder:     stack.NewOPRBuilderNodeID("sequencer-builder", l2ID),
		L2RollupBoost: stack.NewRollupBoostNodeID("rollup-boost", l2ID),
		L2Batcher:     stack.NewL2BatcherID("main", l2ID),
		L2Proposer:    stack.NewL2ProposerID("main", l2ID),
		L2Challenger:  stack.NewL2ChallengerID("main", l2ID),
		TestSequencer: "test-sequencer",
	}
	return ids
}

func DefaultSingleChainSystemWithFlashblocks(dest *SingleChainSystemWithFlashblocksIDs) stack.Option[*Orchestrator] {
	ids := NewDefaultSingleChainSystemWithFlashblocksIDs(DefaultL1ID, DefaultL2AID)
	return singleChainSystemWithFlashblocksOpts(&ids, dest)
}

func singleChainSystemWithFlashblocksOpts(ids *SingleChainSystemWithFlashblocksIDs, dest *SingleChainSystemWithFlashblocksIDs) stack.CombinedOption[*Orchestrator] {
	opt := stack.Combine[*Orchestrator]()
	// Precompute deterministic P2P identity and peering between sequencer EL and op-rbuilder EL.
	seqID := NewELNodeIdentity(0)
	builderID := NewELNodeIdentity(0) // allocate dynamic port for builder

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

	opt.Add(WithL2ELNode(ids.L2EL, L2ELWithP2PConfig("127.0.0.1", seqID.Port, seqID.KeyHex(), nil, nil)))
	opt.Add(WithOPRBuilderNode(ids.L2Builder, OPRBuilderWithNodeIdentity(builderID, "127.0.0.1", nil, nil)))
	// Sequencer adds builder as regular static peer (not trusted)
	opt.Add(WithL2ELP2PConnection(ids.L2EL, stack.L2ELNodeID(ids.L2Builder), false))
	// Builder adds sequencer as trusted peer
	opt.Add(WithL2ELP2PConnection(stack.L2ELNodeID(ids.L2Builder), ids.L2EL, true))
	opt.Add(WithRollupBoost(ids.L2RollupBoost, ids.L2EL, RollupBoostWithBuilderNode(ids.L2Builder)))

	opt.Add(WithL2CLNode(ids.L2CL, ids.L1CL, ids.L1EL, stack.L2ELNodeID(ids.L2RollupBoost), L2CLSequencer()))

	// Services that need L2 CL, in parallel
	opt.Add(stack.InParallel[*Orchestrator](
		WithBatcher(ids.L2Batcher, ids.L1EL, ids.L2CL, ids.L2EL),
		WithProposer(ids.L2Proposer, ids.L1EL, &ids.L2CL, nil),
		WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2EL}),
		WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2CL, ids.L1EL, ids.L2EL),
		WithL2Challenger(ids.L2Challenger, ids.L1EL, ids.L1CL, nil, nil, &ids.L2CL, []stack.L2ELNodeID{
			ids.L2EL,
		}),
	))

	opt.Add(WithL2MetricsDashboard())

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = *ids
	}))

	return opt
}
