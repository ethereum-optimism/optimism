package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

// lagoonActivationOffset schedules Lagoon with enough pre-activation runway for a safe head below the span
// plus the span's own pre-Lagoon element. The block-count assertions below pin what the test needs,
// so this only has to be generous.
const lagoonActivationOffset uint64 = 8

// TestInvalidPostExecTxInCrossLagoonSpan tests that a span batch crossing the Lagoon activation
// block is dropped in its entirety when its first singular batch - a pre-Lagoon one - carries a
// PostExec transaction, and that the honest variant of the same span derives afterwards.
//
// It has to be the first element for the drop to be total. BatchDrop discards more than the offending
// batch: it flushes the channel, cascading through the batch stage (the span's undelivered elements),
// the channel reader (the channel's remaining batch stream) and the channel assembler (the staged
// channel), so the span's later elements are never validated at all, despite being legal post-Lagoon.
// Holocene does not backwards-invalidate, though, so poisoning a later element would instead leave
// the ones streamed before it safe.
//
// The chain is built so that the span begins one block above the safe head - no overlap, whose
// elements would be skipped as past and never transaction-checked - and crosses the boundary:
//
//	safeHead -- target -- activation -- firstPostLagoon
//	            │         └── first Lagoon block
//	            └── last pre-Lagoon block: carries a user tx, and is what the proof claims
//
// Three channels reach L1, one per L1 block: the blocks up to safeHead, then the span
// [target, activation, firstPostLagoon] with a PostExec transaction injected into target, then that
// same span unmodified. The injected transaction is the only difference between the two spans.
//
// op-node, with Lagoon scheduled, must leave the safe head below the span for the invalid variant,
// and then derive the honest one all the way across the boundary, activation block included.
//
// kona-client proves the transition into target, the block the invalid span poisoned, starting from
// the safe head below it. It therefore has to reach the same verdict on the same L1 bytes: drop the
// span, flush the channel, and derive target from the honest span that follows. Two fixture
// properties are what make that meaningful rather than decorative - the claim height and the user
// transaction; see the comments at the block-building loop and at the proof call.
func TestInvalidPostExecTxInCrossLagoonSpan(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	// Karst at genesis; the deploy config schedules Lagoon a few blocks in. Lagoon cannot be a
	// matrix fork: it is the interop feature gate, and the single-chain client asserts that interop
	// is not scheduled (see the rollup config truncation at the end of the test).
	matrix.AddDefaultTestCases(
		nil,
		helpers.NewForkMatrix(helpers.Karst),
		testInvalidPostExecTxInCrossLagoonSpan,
	)
	matrix.Run(gt)
}

func testInvalidPostExecTxInCrossLagoonSpan(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	testSetup := func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		offset := lagoonActivationOffset
		dc.SetForkTimeOffset(forks.Lagoon, &offset)
	}
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), testSetup)

	sequencer := env.Sequencer
	batcher := env.Batcher
	rollupCfg := env.Sd.RollupCfg
	require.NotNil(t, rollupCfg.LagoonTime)

	env.Miner.ActEmptyBlock(t)
	sequencer.ActL1HeadSignal(t)

	// Build L2 blocks through the Lagoon activation block and one block past it. The last pre-Lagoon
	// block gets a user transaction: it is the block this test claims, and Holocene's recovery from a
	// payload that fails to execute is a deposits-only replacement block, which would be
	// indistinguishable from an empty honest block. Without a user transaction the proof passes even
	// for a client that skips the batch rule and only rejects the transaction during execution.
	// Activation blocks stay empty - they may not carry user transactions.
	firstPostLagoonTime := *rollupCfg.LagoonTime + rollupCfg.BlockTime
	blocksByNumber := make(map[uint64]eth.L2BlockRef)
	for sequencer.L2Unsafe().Time < firstPostLagoonTime {
		nextTime := sequencer.L2Unsafe().Time + rollupCfg.BlockTime
		isLastPreLagoonBlock := !rollupCfg.IsSDM(nextTime) && rollupCfg.IsSDM(nextTime+rollupCfg.BlockTime)
		if isLastPreLagoonBlock {
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)
			env.Alice.L2.ActMakeTx(t)
			sequencer.ActL2StartBlock(t)
			env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
			sequencer.ActL2EndBlock(t)
		} else {
			sequencer.ActL2EmptyBlock(t)
		}
		ref := sequencer.L2Unsafe()
		blocksByNumber[ref.Number] = ref
	}
	firstPostLagoon := sequencer.L2Unsafe()
	require.Equal(t, firstPostLagoonTime, firstPostLagoon.Time)

	activation := blocksByNumber[firstPostLagoon.Number-1] // the Lagoon activation block
	target := blocksByNumber[activation.Number-1]          // last pre-Lagoon block: span element #1 and proof claim
	require.GreaterOrEqual(t, target.Number, uint64(2), "need a pre-Lagoon safe head below the span")
	safeHead := blocksByNumber[target.Number-1]
	require.False(t, rollupCfg.IsSDM(target.Time), "the span's first element must be pre-Lagoon")
	require.True(t, rollupCfg.IsSDM(activation.Time), "the span must cross the Lagoon boundary")
	require.Len(t, env.Engine.L2Chain().GetBlockByHash(target.Hash).Transactions(), 2,
		"the claimed block must carry a user tx, so a deposits-only replacement differs from it")

	// Make safeHead safe, so the cross-boundary span below starts at safeHead+1 with no overlap. An
	// overlapping span would have its first elements skipped as past, and past batches are never
	// transaction-checked.
	for range safeHead.Number {
		batcher.ActL2BatchBuffer(t)
	}
	submitChannelAndDerive(t, env)
	require.Equal(t, safeHead, sequencer.L2Safe(), "safe head must stop below the span")

	// submitSpan submits the cross-boundary span in a fresh channel, applying the given modifiers to
	// its first, pre-Lagoon element. Both variants of the span go through here, so they can only
	// differ by those modifiers.
	submitSpan := func(targetModifiers ...actionsHelpers.BlockModifier) {
		batcher.ActCreateChannel(t, true)
		batcher.ActAddBlockByNumber(t, int64(target.Number), targetModifiers...)
		batcher.ActAddBlockByNumber(t, int64(activation.Number))
		batcher.ActAddBlockByNumber(t, int64(firstPostLagoon.Number))
		submitChannelAndDerive(t, env)
	}

	// Submit the invalid variant of the span: a PostExec transaction in its first, pre-Lagoon
	// element. The batch stage must drop that singular batch and flush the channel, so nothing
	// derives at all - not even the later elements, which are legal on their own.
	submitSpan(withPostExecTx(t))

	require.Equal(t, safeHead, sequencer.L2Safe(), "invalid span must not derive anything")
	require.Equal(t, firstPostLagoon, sequencer.L2Unsafe(), "unsafe chain must be untouched")
	require.NotEmpty(t, env.Logs.FindLogs(testlog.NewMessageFilter(
		"sequencers may not embed any PostExec transactions before SDM")))
	require.NotEmpty(t, env.Logs.FindLogs(testlog.NewMessageFilter(
		"Dropping invalid singular batch, flushing channel")))

	// Submit the honest variant of the same span. It is byte-identical to the canonical chain, so it
	// consolidates and advances the safe head across the Lagoon boundary.
	submitSpan()
	require.Equal(t, firstPostLagoon, sequencer.L2Safe(), "honest span must derive across Lagoon")

	// Prove the transition into the block the invalid span poisoned. The proof starts from safeHead,
	// so the poisoned singular batch is a candidate the client must validate and drop before
	// deriving this block from the honest span that follows.
	//
	// Truncate the rollup config at the claim fork: Lagoon is the interop feature gate, and the
	// single-chain client runs without a dependency set, so the attributes builder asserts that
	// interop is not scheduled. The claim is pre-Lagoon, so this does not change the transition
	// being proven - and it makes any accidental whole-span SDM check reject the fixture again.
	lagoonTime := rollupCfg.LagoonTime
	rollupCfg.LagoonTime = nil
	defer func() { rollupCfg.LagoonTime = lagoonTime }()
	env.RunFaultProofProgram(t, target.Number, testCfg.CheckResult, testCfg.InputParams...)
}

// withPostExecTx appends a well-formed PostExec transaction with no refund entries to the block as
// it is batched. Only the transaction type drives the batch rule, but keep the payload valid so the
// fixture stays realistic.
func withPostExecTx(t actionsHelpers.Testing) actionsHelpers.BlockModifier {
	return func(block *types.Block) *types.Block {
		payload, err := rlp.EncodeToBytes(&optypes.PostExecPayload{
			Version:     optypes.PostExecPayloadVersion,
			BlockNumber: block.NumberU64(),
		})
		require.NoError(t, err)

		body := block.Body()
		txs := make(types.Transactions, 0, len(body.Transactions)+1)
		txs = append(txs, body.Transactions...)
		txs = append(txs, types.NewTx(&types.PostExecTx{Data: payload}))
		body.Transactions = txs
		return block.WithBody(*body)
	}
}

// submitChannelAndDerive closes and submits the batcher's channel, mines the submission on L1, and
// derives up to the new L1 head.
func submitChannelAndDerive(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv) {
	env.Batcher.ActL2ChannelClose(t)
	env.Batcher.ActL2BatchSubmit(t)
	env.Miner.ActL1StartBlock(helpers.L1BlockTime)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)
}
