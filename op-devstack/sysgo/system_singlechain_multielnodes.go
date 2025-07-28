package sysgo

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DefaultSingleChainMultiELNodesSystemIDs struct {
	DefaultMinimalSystemIDs

	L2ELs []stack.L2ELNodeID
}

func NewDefaultSingleChainMultiELNodesSystemIDs(l1ID, l2ID eth.ChainID, n int) DefaultSingleChainMultiELNodesSystemIDs {
	minimal := NewDefaultMinimalSystemIDs(l1ID, l2ID)

	ids := make([]stack.L2ELNodeID, n)
	for i := 0; i < n; i++ {
		ids[i] = stack.NewL2ELNodeID(fmt.Sprintf("verifier_%d", i), l2ID)
	}

	return DefaultSingleChainMultiELNodesSystemIDs{
		DefaultMinimalSystemIDs: minimal,
		L2ELs:                   ids,
	}
}

func DefaultSingleChainMultiELNodesSystem(dest *DefaultSingleChainMultiELNodesSystemIDs, n int) stack.Option[*Orchestrator] {
	ids := NewDefaultSingleChainMultiELNodesSystemIDs(DefaultL1ID, DefaultL2AID, n)

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

	opt.Add(WithL2ELNode(ids.L2EL, nil))
	for _, l2ELID := range ids.L2ELs {
		opt.Add(WithL2ELNode(l2ELID, nil))
	}

	opt.Add(WithL2CLNode(ids.L2CL, false, false, ids.L1CL, ids.L1EL, append([]stack.L2ELNodeID{ids.L2EL}, ids.L2ELs...)))

	opt.Add(WithBatcher(ids.L2Batcher, ids.L1EL, ids.L2CL, ids.L2EL))
	opt.Add(WithProposer(ids.L2Proposer, ids.L1EL, &ids.L2CL, nil))

	opt.Add(WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2EL}))

	opt.Add(WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2CL, ids.L1EL, ids.L2EL))

	opt.Add(WithL2Challenger(ids.L2Challenger, ids.L1EL, ids.L1CL, nil, nil, &ids.L2CL, []stack.L2ELNodeID{
		ids.L2EL,
	}))

	// P2P connect L2CL nodes
	for i := 0; i < len(ids.L2ELs); i++ {
		opt.Add(WithL2ELP2PConnection(ids.L2EL, ids.L2ELs[i])) // sequencer to other verifiers
		for j := i + 1; j < len(ids.L2ELs); j++ {
			opt.Add(WithL2ELP2PConnection(ids.L2ELs[i], ids.L2ELs[j]))
		}
	}

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))
	return opt
}
