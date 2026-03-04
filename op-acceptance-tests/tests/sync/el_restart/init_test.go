package el_restart

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		WithOpNodeRethVerifier(),
		presets.WithCompatibleTypes(compat.SysGo),
	)
}

// WithOpNodeRethVerifier creates a single-chain multi-node system where:
//   - Sequencer: op-geth EL + op-node CL
//   - Verifier: op-reth EL + op-node CL
//
// This allows testing EL restart recovery when op-reth loses in-memory state.
func WithOpNodeRethVerifier() stack.CommonOption {
	var ids sysgo.DefaultSingleChainMultiNodeSystemIDs
	return stack.MakeCommon(opNodeRethVerifierSystem(&ids))
}

func opNodeRethVerifierSystem(dest *sysgo.DefaultSingleChainMultiNodeSystemIDs) stack.Option[*sysgo.Orchestrator] {
	ids := sysgo.NewDefaultSingleChainMultiNodeSystemIDs(sysgo.DefaultL1ID, sysgo.DefaultL2AID)

	opt := stack.Combine[*sysgo.Orchestrator]()
	// Sequencer system: op-geth + op-node + batcher + proposer
	opt.Add(sysgo.DefaultMinimalSystem(&ids.DefaultMinimalSystemIDs))

	// Verifier EL: use op-reth explicitly (instead of WithL2ELNode which defaults to op-geth)
	opt.Add(sysgo.WithOpReth(ids.L2ELB))
	// Verifier CL: op-node (default for WithL2CLNode)
	opt.Add(sysgo.WithL2CLNode(ids.L2CLB, ids.L1CL, ids.L1EL, ids.L2ELB))

	// P2P connections between sequencer and verifier
	opt.Add(sysgo.WithL2CLP2PConnection(ids.L2CL, ids.L2CLB))
	opt.Add(sysgo.WithL2ELP2PConnection(ids.L2EL, ids.L2ELB, false))

	opt.Add(stack.Finally(func(orch *sysgo.Orchestrator) {
		*dest = ids
	}))

	return opt
}
