package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

// TestSpanBatchCrossesLagoon verifies that transaction validation only inspects the singular
// batches streamed after the safe head. An overlapping span may contain ignored elements on both
// sides of Lagoon, including a PostExec transaction in its first post-Lagoon element.
func TestSpanBatchCrossesLagoon(gt *testing.T) {
	testSpanBatchCrossesLagoon(gt, &helpers.TestCfg[any]{Hardfork: helpers.Karst})
}

func testSpanBatchCrossesLagoon(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	const lagoonOffset = uint64(4)
	testSetup := func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		offset := lagoonOffset
		dc.SetForkTimeOffset(forks.Lagoon, &offset)
	}
	env := helpers.NewL2FaultProofEnv(
		t,
		testCfg,
		helpers.NewTestParams(),
		helpers.NewBatcherCfg(),
		testSetup,
	)

	sequencer := env.Sequencer
	miner := env.Miner
	batcher := env.Batcher
	rollupCfg := env.Sd.RollupCfg
	require.NotNil(t, rollupCfg.LagoonTime)

	miner.ActEmptyBlock(t)
	sequencer.ActL1HeadSignal(t)

	// Build through Lagoon and one block beyond it. The latter is the first block whose parent is
	// already post-Lagoon and is the first element in which SDM may emit a PostExec transaction.
	blocks := make(map[uint64]eth.L2BlockRef)
	for sequencer.L2Unsafe().Time < *rollupCfg.LagoonTime+rollupCfg.BlockTime {
		sequencer.ActL2EmptyBlock(t)
		ref := sequencer.L2Unsafe()
		blocks[ref.Number] = ref
	}
	firstPostLagoon := sequencer.L2Unsafe()
	require.Equal(t, *rollupCfg.LagoonTime+rollupCfg.BlockTime, firstPostLagoon.Time)

	activationNumber := firstPostLagoon.Number - 1
	require.GreaterOrEqual(t, activationNumber, uint64(2))
	safeBeforeBoundary, ok := blocks[activationNumber-2]
	require.True(t, ok)
	targetBeforeLagoon, ok := blocks[activationNumber-1]
	require.True(t, ok)
	require.False(t, rollupCfg.IsSDM(targetBeforeLagoon.Time))

	// First make only an early pre-Lagoon block safe. Later unsafe blocks remain available for the
	// cross-boundary span below, while its first element overlaps this safe head.
	for i := uint64(0); i < safeBeforeBoundary.Number; i++ {
		batcher.ActL2BatchBuffer(t)
	}
	batcher.ActL2ChannelClose(t)
	batcher.ActL2BatchSubmit(t)
	miner.ActL1StartBlock(helpers.L1BlockTime)(t)
	miner.ActL1IncludeTxByHash(batcher.LastSubmitted.Hash())(t)
	miner.ActL1EndBlock(t)
	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)
	require.Equal(t, safeBeforeBoundary, sequencer.L2Safe())

	// Re-submit an overlapping span beginning before Lagoon. Its ignored overlap and its first
	// post-Lagoon element both carry PostExec txs. The batch stage must skip the overlap without
	// transaction introspection, validate each newly streamed singular batch at its own timestamp,
	// and advance through the unmodified pre-Lagoon target and the Lagoon boundary.
	batcher.ActCreateChannel(t, true)
	batcher.ActAddBlockByNumber(t, int64(safeBeforeBoundary.Number), withEmptyPostExec(t))
	batcher.ActAddBlockByNumber(t, int64(targetBeforeLagoon.Number))
	batcher.ActAddBlockByNumber(t, int64(activationNumber))
	batcher.ActAddBlockByNumber(t, int64(firstPostLagoon.Number), withEmptyPostExec(t))
	batcher.ActL2ChannelClose(t)
	batcher.ActL2BatchSubmit(t)

	miner.ActL1StartBlock(helpers.L1BlockTime)(t)
	miner.ActL1IncludeTxByHash(batcher.LastSubmitted.Hash())(t)
	miner.ActL1EndBlock(t)

	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)
	require.Equal(t, firstPostLagoon, sequencer.L2Safe(), "overlapping span should advance across Lagoon")

	// Prove the last pre-Lagoon transition from the cross-boundary channel. Kona decodes the whole
	// span (including its future PostExec element), but validates only the singular batch needed for
	// this claim. Truncate the proof's rollup config at the claim fork: the single-chain host cannot
	// run with Interop scheduled, and future fork times do not affect this pre-Lagoon transition.
	// This also makes any accidental whole-span SDM check reject the fixture again.
	lagoonTime := rollupCfg.LagoonTime
	rollupCfg.LagoonTime = nil
	defer func() { rollupCfg.LagoonTime = lagoonTime }()
	env.RunFaultProofProgram(
		t,
		targetBeforeLagoon.Number,
		helpers.ExpectNoError(),
		testCfg.InputParams...,
	)
}

func withEmptyPostExec(t actionsHelpers.Testing) actionsHelpers.BlockModifier {
	return func(block *types.Block) *types.Block {
		payload, err := rlp.EncodeToBytes(&optypes.PostExecPayload{
			Version:     optypes.PostExecPayloadVersion,
			BlockNumber: block.NumberU64(),
		})
		require.NoError(t, err)
		postExec := types.NewTx(&types.PostExecTx{Data: payload})
		txs := append(types.Transactions(nil), block.Transactions()...)
		txs = append(txs, postExec)
		return block.WithBody(types.Body{Transactions: txs})
	}
}
