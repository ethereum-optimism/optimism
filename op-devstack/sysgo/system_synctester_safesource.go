package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DefaultSimpleSystemWithSyncTesterSafeSourceL2IDs struct {
	DefaultMinimalSystemIDs

	L2CL2          stack.L2CLNodeID
	SyncTesterL2EL stack.L2ELNodeID
	SyncTester     stack.SyncTesterID
}

func NewDefaultSimpleSystemWithSyncTesterSafeSourceL2IDs(l1ID, l2ID eth.ChainID) DefaultSimpleSystemWithSyncTesterSafeSourceL2IDs {
	minimal := NewDefaultMinimalSystemIDs(l1ID, l2ID)
	return DefaultSimpleSystemWithSyncTesterSafeSourceL2IDs{
		DefaultMinimalSystemIDs: minimal,
		L2CL2:                   stack.NewL2CLNodeID("verifier", l2ID),
		SyncTesterL2EL:          stack.NewL2ELNodeID("sync-tester-el", l2ID),
		SyncTester:              stack.NewSyncTesterID("sync-tester", l2ID),
	}
}

func DefaultSimpleSystemWithSyncTesterSafeSourceL2(dest *DefaultSimpleSystemWithSyncTesterSafeSourceL2IDs) stack.Option[*Orchestrator] {
	l1ID := eth.ChainIDFromUInt64(900)
	l2ID := eth.ChainIDFromUInt64(901)
	ids := NewDefaultSimpleSystemWithSyncTesterSafeSourceL2IDs(l1ID, l2ID)

	opt := stack.Combine[*Orchestrator]()
	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		o.P().Logger().Info("Setting up with SafeSource L2")
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

	opt.Add(WithSyncTester(ids.SyncTester, []stack.L2ELNodeID{ids.L2EL}))

	// Create a SyncTesterEL with the same chain ID as the EL node
	opt.Add(WithSyncTesterL2ELNode(ids.SyncTesterL2EL, ids.L2EL))

	// Create L2CL2 with safe-source=l2 pointing to L2CL
	// We need to set the RPC URL after L2CL is started
	opt.Add(stack.AfterDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()

		// Get L2CL to extract its RPC URL
		l2CL, ok := orch.l2CLs.Get(ids.L2CL)
		require.True(ok, "L2CL node must exist before creating L2CL2 with safe-source=l2")

		// Create L2CL2 with safe-source=l2 configuration
		safeSourceOpt := L2CLOptionFn(func(p devtest.P, id stack.L2CLNodeID, cfg *L2CLConfig) {
			cfg.SafeSource = nodeSync.SafeSourceL2
			cfg.SafeSourceL2RPC = l2CL.UserRPC()
		})

		WithL2CLNode(ids.L2CL2, ids.L1CL, ids.L1EL, ids.SyncTesterL2EL, safeSourceOpt).Deploy(orch)
	}))

	// P2P Connect CLs to signal unsafe heads
	opt.Add(WithL2CLP2PConnection(ids.L2CL, ids.L2CL2))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}
