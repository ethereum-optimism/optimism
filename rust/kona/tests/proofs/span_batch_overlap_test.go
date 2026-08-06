package proofs

import (
	"math/big"
	"testing"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestSpanBatchOverlapContent pins the post-Holocene span batch overlap content rule, in both
// op-node (driving derivation below) and kona-client (replaying it as the fault proof program):
// a span batch whose overlap section disagrees with the safe chain must be dropped as a whole,
// with a channel flush, so that the remainder of an invalidated lineage cannot be spliced onto
// the canonical chain.
//
// The scenario mirrors the interop block-replacement incident shape using batcher equivocation:
//
//  1. Channel 1 carries span [1, 2*, 3] where 2*'s transaction signature is invalidated.
//     Deriving it yields block 1, then a deposits-only replacement 2' for the invalid payload,
//     and a channel flush that discards element 3. The safe chain now disagrees with the
//     batched lineage at height 2.
//  2. Channel 2 carries the same invalidated lineage [1, 2*, 3] again — the incident's
//     "re-derivation walk" over a batch containing a since-replaced block. Its overlap
//     (heights 1-2) must be validated against the safe chain: element 2* conflicts with 2'.
//     With the overlap content rule, the whole batch is dropped and the channel flushed.
//     Without it, element 3 — built on the invalidated 2* — is spliced onto 2', producing
//     another deposits-only replacement 3' derived from a lineage that was never canonical.
//  3. Channel 3 carries the canonical continuation 3” (built on 2', with a live transaction).
//     With the rule, it applies and the safe head becomes 3”. Without it, height 3 is already
//     (wrongly) occupied by 3', and the canonical continuation is dropped as a past batch —
//     the equivocating channel's tail wins over the canonical chain.
//
// The fault proof program then replays the same L1 data with honest claims taken from the
// op-node-derived chain at every height 0..3. Height 3 is where a client without the overlap
// rule derives 3' instead of 3”, so an unfixed kona-client rejects the honest claim there,
// pinning that the Go and Rust implementations enforce the same rule.
func TestSpanBatchOverlapContent(gt *testing.T) {
	type testCase struct{}

	// invalidPayload invalidates the signature for the second transaction in the block
	// (the first is the L1 info deposit). This yields an invalid payload in the engine,
	// which post-Holocene is replaced with a deposits-only block.
	invalidPayload := func(block *types.Block) *types.Block {
		alice := types.NewCancunSigner(big.NewInt(901))
		txs := block.Transactions()
		newTx, err := txs[1].WithSignature(alice, make([]byte, 65))
		if err != nil {
			panic(err)
		}
		txs[1] = newTx
		return block
	}

	runTest := func(gt *testing.T, testCfg *helpers.TestCfg[testCase]) {
		t := actionsHelpers.NewDefaultTesting(gt)
		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg())

		buildAliceBlock := func() {
			env.Sequencer.ActL2StartBlock(t)
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)
			env.Alice.L2.ActMakeTx(t)
			env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
			env.Sequencer.ActL2EndBlock(t)
		}

		submitAndIncludeFrame := func(frame []byte) {
			require.NotEmpty(t, frame)
			env.Batcher.ActL2BatchSubmitRaw(t, frame)
			env.Miner.ActL1StartBlock(12)(t)
			env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
			env.Miner.ActL1EndBlock(t)
		}

		// Build L2 blocks 1-3, each with one transaction from Alice.
		for bigs.Uint64Strict(env.Engine.L2Chain().CurrentBlock().Number) < 3 {
			buildAliceBlock()
		}

		// Craft both channels now, while blocks 1-3 are the canonical chain. Once derivation
		// replaces block 2, the original blocks are no longer reachable by number.
		craftInvalidLineageChannel := func() []byte {
			env.Batcher.ActCreateChannel(t, true)
			env.Batcher.ActAddBlockByNumber(t, 1, actionsHelpers.BlockLogger(t))
			env.Batcher.ActAddBlockByNumber(t, 2, invalidPayload, actionsHelpers.BlockLogger(t))
			env.Batcher.ActAddBlockByNumber(t, 3, actionsHelpers.BlockLogger(t))
			env.Batcher.ActL2ChannelClose(t)
			return env.Batcher.ReadNextOutputFrame(t)
		}
		frame1 := craftInvalidLineageChannel()
		frame2 := craftInvalidLineageChannel()

		// Submit channel 1 and channel 2 in consecutive L1 blocks, then derive both.
		submitAndIncludeFrame(frame1)
		submitAndIncludeFrame(frame2)
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActL2PipelineFull(t)

		// Channel 1: block 1 applies, 2* is invalid and replaced with deposits-only 2', and the
		// channel is flushed. Channel 2: its overlap conflicts with 2', so the whole batch must
		// be dropped with a channel flush — the safe head must NOT advance past the replacement.
		l2SafeHead := env.Sequencer.L2Safe()
		require.EqualValues(t, 2, l2SafeHead.Number,
			"the re-batched invalidated lineage must be dropped whole, not spliced onto the replacement")
		require.Equal(t, env.Engine.L2Chain().GetHeaderByNumber(2).Hash(), l2SafeHead.Hash,
			"safe head must be the deposits-only replacement")
		for _, filter := range []string{
			"could not process payload attributes", // 2* rejected by the engine
			"overlapped block's tx count does not match",
			"Dropping invalid span batch, flushing channel (span batch checks)",
		} {
			recs := env.Logs.FindLogs(
				testlog.NewMessageContainsFilter(filter),
				testlog.NewAttributesFilter("role", "sequencer"))
			require.NotEmpty(t, recs, "expected sequencer log: %q", filter)
		}

		// Canonical continuation: build 3'' on the replacement (with a transaction, so it is
		// distinguishable from a deposits-only splice at the same height), batch and derive it.
		buildAliceBlock()
		env.Batcher.ActCreateChannel(t, true)
		env.Batcher.ActAddBlockByNumber(t, 3, actionsHelpers.BlockLogger(t))
		env.Batcher.ActL2ChannelClose(t)
		submitAndIncludeFrame(env.Batcher.ReadNextOutputFrame(t))
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActL2PipelineFull(t)

		l2SafeHead = env.Sequencer.L2Safe()
		require.EqualValues(t, 3, l2SafeHead.Number, "canonical continuation must derive")
		require.Equal(t, env.Engine.L2Chain().GetHeaderByNumber(3).Hash(), l2SafeHead.Hash,
			"the canonical continuation must win over the equivocating channel's tail")

		// Run the fault proof program with honest claims at every height 0..3. A client without
		// the overlap content rule splices the invalidated lineage and derives a different block
		// at height 3, rejecting the honest claim there.
		env.RunFaultProofProgramFromGenesis(t, l2SafeHead.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	matrix := helpers.NewMatrix[testCase]()
	matrix.AddTestCase(
		"HonestClaim",
		testCase{},
		helpers.NewForkMatrix(helpers.Holocene, helpers.LatestFork),
		runTest,
		helpers.ExpectNoError(),
	)
	matrix.AddTestCase(
		"JunkClaim",
		testCase{},
		helpers.NewForkMatrix(helpers.Holocene, helpers.LatestFork),
		runTest,
		helpers.ExpectError(helpers.ErrClaimNotValid),
		helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
	)
	matrix.Run(gt)
}
