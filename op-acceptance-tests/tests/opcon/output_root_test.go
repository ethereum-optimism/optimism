package opcon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// These tests pin op-con-node's optimism_outputAtBlock RPC — the one rollup RPC
// that reaches out to the execution layer and computes a value locally. For each
// L2 block it fetches the block (eth_getBlockByNumber), reads the
// L2ToL1MessagePasser storage root (eth_getProof), decodes the block ref's L1
// origin + sequence number from the block's first transaction (the L1-attributes
// deposit), and computes the v0 output root
// keccak256(0x00..32 ++ stateRoot ++ messagePasserStorageRoot ++ blockHash). The
// op-proposer reads this RPC to build withdrawal proofs and dispute-game claims,
// so a wrong output root is a consensus-level correctness bug.
//
// The output root is a spec value, so two nodes that derived the same safe chain
// must produce byte-identical roots for every safe block; VerifyOutputRootMatches
// walks the shared safe range and compares the root and its inputs (state root,
// withdrawal storage root, block ref) at each height, ignoring the node-local
// embedded syncStatus.

// TestOpConVerifierOutputRootMatches is the strongest correctness check of
// op-con-node's output roots: an op-con-node VERIFIER's optimism_outputAtBlock is
// compared, block by block over the shared safe range, against the op-node
// sequencer that produced the chain. Because the two run different
// implementations (op-con-node vs op-node), agreement pins op-con-node's output
// root to op-node's canonical value — a formula slip, a wrong messagePasser proof
// block, or a bad L1-origin/sequence decode in the block ref would diverge here.
//
// With the default op-node CL kind both slots are op-node, so it degrades to an
// ordinary op-node output-root parity check — a valid suite addition rather than
// an op-con-only test.
//
//	DEVSTACK_L2CL_KIND=op-con-node DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	go test -run TestOpConVerifierOutputRootMatches ./op-acceptance-tests/tests/opcon/
func TestOpConVerifierOutputRootMatches(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t)

	// The op-node sequencer produces + batches; the op-con-node verifier derives the
	// same safe chain from L1. Both advance their safe heads and converge.
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 1, 60),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 60),
	)
	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, 60)

	// The op-con verifier's output roots must match the op-node sequencer's for
	// every block in the shared safe range.
	sys.L2CLB.VerifyOutputRootMatches(sys.L2CL)
}

// TestOpConSequencerOutputRootMatches is the sequencer-slot flip: op-con-node runs
// AS the L2 sequencer and serves optimism_outputAtBlock for the blocks it produced.
// An independently-deriving op-con-node verifier must reproduce the same output
// root for every safe block. This exercises the RPC on the block-producing node
// (whose blocks the op-proposer would prove) and confirms the output root is stable
// across two independent op-con-node instances of the same chain.
//
// With the default op-node CL kind L2CLOpConSequencer() is a no-op, so it degrades
// to an ordinary op-node output-root parity check — a valid suite addition rather
// than an op-con-only test.
//
//	DEVSTACK_L2CL_KIND=op-con-node DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	go test -run TestOpConSequencerOutputRootMatches ./op-acceptance-tests/tests/opcon/
func TestOpConSequencerOutputRootMatches(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t, opConSequencerOpt())

	// The op-con sequencer produces + batches; the op-con verifier derives the same
	// safe chain from L1. Both advance their safe heads and converge.
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 1, 60),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 60),
	)
	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, 60)

	// The verifier's output roots must match the op-con sequencer's (source of
	// truth) for every block in the shared safe range.
	sys.L2CLB.VerifyOutputRootMatches(sys.L2CL)
}
