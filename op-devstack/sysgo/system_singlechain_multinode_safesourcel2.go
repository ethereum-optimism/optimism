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

	// Create L2CLB with safe-source=l2 pointing to L2CL and setup P2P connections
	// We need to set the RPC URL after L2CL is started
	opt.Add(stack.AfterDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()

		// Get L2EL to extract its RPC URL for safe-source=l2
		l2EL, ok := orch.l2ELs.Get(ids.L2EL)
		require.True(ok, "L2EL node must exist before creating L2CLB with safe-source=l2")

		// Create L2CLB with safe-source=l2 configuration
		safeSourceOpt := L2CLOptionFn(func(p devtest.P, id stack.L2CLNodeID, cfg *L2CLConfig) {
			cfg.SafeSource = nodeSync.SafeSourceL2
			cfg.SafeSourceL2RPC = l2EL.UserRPC()
		})

		// Create the node by calling both Deploy and AfterDeploy
		nodeOpt := WithL2CLNode(ids.L2CLB, ids.L1CL, ids.L1EL, ids.L2ELB, safeSourceOpt)
		nodeOpt.Deploy(orch)
		nodeOpt.AfterDeploy(orch)

		// P2P connect L2CL nodes - done in same AfterDeploy after node creation
		p2pOpt1 := WithL2CLP2PConnection(ids.L2CL, ids.L2CLB)
		p2pOpt1.Deploy(orch)
		p2pOpt1.AfterDeploy(orch)

		p2pOpt2 := WithL2ELP2PConnection(ids.L2EL, ids.L2ELB)
		p2pOpt2.Deploy(orch)
		p2pOpt2.AfterDeploy(orch)
	}))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))
	return opt
}
