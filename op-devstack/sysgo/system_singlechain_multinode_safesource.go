package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DefaultSingleChainMultiNodeWithSafeSourceL2SystemIDs struct {
	DefaultMinimalSystemIDs

	L2CLB stack.L2CLNodeID
	L2ELB stack.L2ELNodeID
}

func NewDefaultSingleChainMultiNodeWithSafeSourceL2SystemIDs(l1ID, l2ID eth.ChainID) DefaultSingleChainMultiNodeWithSafeSourceL2SystemIDs {
	minimal := NewDefaultMinimalSystemIDs(l1ID, l2ID)
	return DefaultSingleChainMultiNodeWithSafeSourceL2SystemIDs{
		DefaultMinimalSystemIDs: minimal,
		L2CLB:                   stack.NewL2CLNodeID("b", l2ID),
		L2ELB:                   stack.NewL2ELNodeID("b", l2ID),
	}
}

func DefaultSingleChainMultiNodeWithSafeSourceL2System(dest *DefaultSingleChainMultiNodeWithSafeSourceL2SystemIDs) stack.Option[*Orchestrator] {
	ids := NewDefaultSingleChainMultiNodeWithSafeSourceL2SystemIDs(DefaultL1ID, DefaultL2AID)

	opt := stack.Combine[*Orchestrator]()
	opt.Add(DefaultMinimalSystem(&dest.DefaultMinimalSystemIDs))

	opt.Add(WithL2ELNode(ids.L2ELB))

	// Create L2CLB with safe-source=l2 pointing to L2CL
	// We need to set the RPC URL after L2CL is started
	opt.Add(stack.AfterDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()

		// Get L2CL to extract its RPC URL
		l2CL, ok := orch.l2CLs.Get(ids.L2CL)
		require.True(ok, "L2CL node must exist before creating L2CLB with safe-source=l2")

		// Create L2CLB with safe-source=l2 configuration
		safeSourceOpt := L2CLOptionFn(func(p devtest.P, id stack.L2CLNodeID, cfg *L2CLConfig) {
			cfg.SafeSource = nodeSync.SafeSourceL2
			cfg.SafeSourceL2RPC = l2CL.UserRPC()
		})

		WithL2CLNode(ids.L2CLB, ids.L1CL, ids.L1EL, ids.L2ELB, safeSourceOpt).Deploy(orch)
	}))

	// P2P connect L2CL nodes
	opt.Add(WithL2CLP2PConnection(ids.L2CL, ids.L2CLB))
	opt.Add(WithL2ELP2PConnection(ids.L2EL, ids.L2ELB))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))
	return opt
}
