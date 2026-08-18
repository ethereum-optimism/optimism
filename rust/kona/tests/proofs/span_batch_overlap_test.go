package proofs

import (
	"testing"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestSpanBatchOverlapContent pins the span batch overlap content rule, in both op-node (driving
// derivation below) and kona-client (replaying the same L1 data as the fault proof program): a
// span batch whose overlap section disagrees with the safe chain must be dropped as a whole, so
// that its elements past the safe head cannot be applied. Pre-Holocene, the rule has always been
// part of the full span batch checks; this test pins that it holds identically on the Holocene
// batch stage, where a content-blind past-skip would otherwise splice the remainder of a stale
// or equivocating lineage onto the canonical chain (the interop block-replacement lineage bug).
//
// The scenario uses batcher equivocation with alternative *valid* blocks (the canonical blocks
// with their user transactions stripped), so it is independent of invalid-payload replacement
// rules and runs unchanged across forks:
//
//  1. Channel 1 carries span [1, 2]: the canonical chain, derived to the safe head.
//  2. Channel 2 carries span [2*, 3*], the same heights with user transactions stripped —
//     an alternative valid lineage. Its overlap (height 2) disagrees with the safe chain, so
//     the whole batch must be dropped and the safe head stay at 2. Without the overlap content
//     rule, 2* is past-skipped and 3* — a valid empty block — is spliced onto canonical 2.
//  3. Channel 3 carries the canonical block 3: it must derive (fixed), instead of being dropped
//     as a past batch because height 3 is already wrongly occupied by 3* (unfixed) — the
//     equivocating channel's tail must not win over the canonical chain.
//
// The fault proof program then replays the same L1 data with honest claims taken from the
// op-node-derived chain at every height 0..3. Height 3 is where a client without the overlap
// rule derives 3* instead of the canonical block 3 and rejects the honest claim, pinning that
// the Go and Rust implementations enforce the same rule.
func TestSpanBatchOverlapContent(gt *testing.T) {
	// dropUserTxs strips all non-deposit transactions from the block, yielding an alternative
	// valid block at the same height: same parent and timestamp, different content.
	dropUserTxs := func(block *types.Block) *types.Block {
		var deposits []*types.Transaction
		for _, tx := range block.Transactions() {
			if tx.Type() == types.DepositTxType {
				deposits = append(deposits, tx)
			}
		}
		return block.WithBody(types.Body{Transactions: deposits})
	}

	runTest := func(gt *testing.T, testCfg *helpers.TestCfg[any]) {
		t := actionsHelpers.NewDefaultTesting(gt)
		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg())

		// Build L2 blocks 1-3, each with one transaction from Alice.
		for bigs.Uint64Strict(env.Engine.L2Chain().CurrentBlock().Number) < 3 {
			env.Sequencer.ActL2StartBlock(t)
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)
			env.Alice.L2.ActMakeTx(t)
			env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
			env.Sequencer.ActL2EndBlock(t)
		}
		// The canonical block 3, captured before derivation can reorg the engine: the safe head
		// must end up here, not on the equivocating lineage's 3*.
		canonicalHash3 := env.Engine.L2Chain().GetHeaderByNumber(3).Hash()

		craftSpanChannel := func(blocks []int64, modifier actionsHelpers.BlockModifier) []byte {
			env.Batcher.ActCreateChannel(t, true)
			for _, num := range blocks {
				env.Batcher.ActAddBlockByNumber(t, num, modifier, actionsHelpers.BlockLogger(t))
			}
			env.Batcher.ActL2ChannelClose(t)
			frame := env.Batcher.ReadNextOutputFrame(t)
			require.NotEmpty(t, frame)
			return frame
		}
		submitAndIncludeFrame := func(frame []byte) {
			env.Batcher.ActL2BatchSubmitRaw(t, frame)
			env.Miner.ActL1StartBlock(helpers.L1BlockTime)(t)
			env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
			env.Miner.ActL1EndBlock(t)
		}

		// Channel 1: the canonical chain up to height 2. Channel 2: the equivocating lineage
		// [2*, 3*]. Both are crafted upfront and submitted in consecutive L1 blocks.
		frame1 := craftSpanChannel([]int64{1, 2}, nil)
		frame2 := craftSpanChannel([]int64{2, 3}, dropUserTxs)
		submitAndIncludeFrame(frame1)
		submitAndIncludeFrame(frame2)
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActL2PipelineFull(t)

		// Channel 2's overlap disagrees with the safe chain at height 2: the whole batch must be
		// dropped and the safe head stay at 2, instead of 3* being spliced onto canonical 2.
		l2SafeHead := env.Sequencer.L2Safe()
		require.EqualValues(t, 2, l2SafeHead.Number,
			"the equivocating overlapping span batch must be dropped whole, not tail-spliced")
		isHolocene := testCfg.Hardfork.Precedence >= helpers.Holocene.Precedence
		expectedLogs := []string{"overlapped block's tx count does not match"}
		if isHolocene {
			expectedLogs = append(expectedLogs,
				"Dropping invalid span batch, flushing channel (span batch checks)")
		}
		for _, filter := range expectedLogs {
			recs := env.Logs.FindLogs(
				testlog.NewMessageContainsFilter(filter),
				testlog.NewAttributesFilter("role", "sequencer"))
			require.NotEmpty(t, recs, "expected sequencer log: %q", filter)
		}

		// Channel 3: the canonical block 3. It must derive on top of the safe head — with a
		// content-blind past-skip, height 3 is already occupied by 3* and the canonical
		// continuation would be dropped as a past batch.
		submitAndIncludeFrame(craftSpanChannel([]int64{3}, nil))
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActL2PipelineFull(t)

		l2SafeHead = env.Sequencer.L2Safe()
		require.EqualValues(t, 3, l2SafeHead.Number, "canonical continuation must derive")
		require.Equal(t, canonicalHash3, l2SafeHead.Hash,
			"the canonical block must win over the equivocating channel's tail")

		// Run the fault proof program with honest claims at every height 0..3. A client without
		// the overlap content rule splices the equivocating lineage and derives a different
		// block at height 3, rejecting the honest claim there.
		env.RunFaultProofProgramFromGenesis(t, l2SafeHead.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	matrix := helpers.NewMatrix[any]()
	matrix.AddDefaultTestCases(
		nil,
		helpers.NewForkMatrix(helpers.Fjord, helpers.Holocene, helpers.LatestFork),
		runTest,
	)
	matrix.Run(gt)
}
