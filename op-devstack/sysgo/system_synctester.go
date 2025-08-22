package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
)

type DefaultSimpleSystemWithSyncTesterIDs struct {
	DefaultMinimalSystemIDs

	L2CL2      stack.L2CLNodeID
	SyncTester stack.SyncTesterID
}

func NewDefaultSimpleSystemWithSyncTesterIDs(l1ID, l2ID eth.ChainID) DefaultSimpleSystemWithSyncTesterIDs {
	minimal := NewDefaultMinimalSystemIDs(l1ID, l2ID)
	return DefaultSimpleSystemWithSyncTesterIDs{
		DefaultMinimalSystemIDs: minimal,
		L2CL2:                   stack.NewL2CLNodeID("verifier", l2ID),
		SyncTester:              stack.NewSyncTesterID("s", l2ID),
	}
}

func DefaultSimpleSystemWithSyncTester(dest *DefaultSimpleSystemWithSyncTesterIDs, fcus sttypes.FCUState) stack.Option[*Orchestrator] {
	l1ID := eth.ChainIDFromUInt64(900)
	l2ID := eth.ChainIDFromUInt64(901)
	ids := NewDefaultSimpleSystemWithSyncTesterIDs(l1ID, l2ID)

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

	opt.Add(WithSyncTester([]stack.L2ELNodeID{ids.L2EL}, fcus))
	opt.Add(WithL2CLNodeWithSyncTester(ids.L2CL2, ids.L1CL, ids.L1EL))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}
