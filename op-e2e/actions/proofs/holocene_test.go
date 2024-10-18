package proofs

import (
	"fmt"
	"math/big"
	"testing"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type ordering struct {
	name                string
	blocks              []uint // could enhance this to declare either singular or span batches or a mixture
	isSpanBatch         bool
	blockModifiers      []actionsHelpers.BlockModifier
	frames              []uint
	safeHeadPreHolocene uint64
	safeHeadHolocene    uint64
}

// blockFudger invalidates the signature for the second transaction in the block.
// This should result in an invalid payload in the engine queue.
var blockFudger = func(block *types.Block) *types.Block {
	alice := types.NewCancunSigner(big.NewInt(901))
	txs := block.Transactions()
	newTx, err := txs[1].WithSignature(alice, make([]byte, 65))
	newTx.IsDepositTx()
	if err != nil {
		panic(err)
	}
	txs[1] = newTx
	return block
}

// blockSpudger invalidates the parentHash of the block
var blockSpudger = func(block *types.Block) *types.Block {
	headerCopy := block.Header()
	headerCopy.ParentHash = common.MaxHash
	return block.WithSeal(headerCopy)
}

// orderings is a list of orderings which each specify
// an ordered list of blocks (by number) to add to a single channel
// and an ordered list of frames to read from the channel and submit
// on L1. There will be one frame per block.
// Depending on these lists, whether the channel is built as
// as span batch channel, and whether the blocks are modified / invalidated
// we expect a different progression of the safe head under Holocene
// derivation rules, compared with pre Holocene.
var orderings = []ordering{
	{name: "case-0", blocks: []uint{1, 2, 3}, frames: []uint{0, 1, 2}, safeHeadPreHolocene: 3, safeHeadHolocene: 3},       // regular case
	{name: "case-1", blocks: []uint{1, 3, 2}, frames: []uint{0, 1, 2}, safeHeadPreHolocene: 3, safeHeadHolocene: 1},       // out-of-order blocks
	{name: "case-2", blocks: []uint{2, 1, 3}, frames: []uint{0, 1, 2}, safeHeadPreHolocene: 3, safeHeadHolocene: 0},       // out-of-order blocks
	{name: "case-3", blocks: []uint{2, 2, 1, 3}, frames: []uint{0, 1, 2, 3}, safeHeadPreHolocene: 3, safeHeadHolocene: 0}, // duplicate block
	{name: "case-4", blocks: []uint{1, 2, 3}, frames: []uint{2, 1, 0}, safeHeadPreHolocene: 3, safeHeadHolocene: 0},       // bad frame ordering
	{name: "case-5", blocks: []uint{1, 2, 3}, frames: []uint{0, 1, 0, 2}, safeHeadPreHolocene: 3, safeHeadHolocene: 0},    // duplicate frames
	{name: "case-6", blocks: []uint{1, 2, 3}, frames: []uint{0, 1, 2}, safeHeadPreHolocene: 0, safeHeadHolocene: 0,
		isSpanBatch: true, blockModifiers: []actionsHelpers.BlockModifier{nil, blockFudger, nil}}, // partially invalid span batch (invalid payload)
	{name: "case-7", blocks: []uint{1, 2, 3}, frames: []uint{0, 1, 2}, safeHeadPreHolocene: 1, safeHeadHolocene: 1,
		isSpanBatch: false, blockModifiers: []actionsHelpers.BlockModifier{nil, blockSpudger, nil}}, // partially invalid singular batch channel (invalid batch)
	{name: "case-8", blocks: []uint{1, 2, 3}, frames: []uint{0, 1, 2}, safeHeadPreHolocene: 3, safeHeadHolocene: 3,
		isSpanBatch: true, blockModifiers: []actionsHelpers.BlockModifier{nil, blockSpudger, nil}}, // partially invalid span batch (invalid batch?)
}

func max(input []uint) uint {
	max := uint(0)
	for _, val := range input {
		if val > max {
			max = val
		}
	}
	return max
}

func Test_ProgramAction_HoloceneDerivationRules(gt *testing.T) {
	matrix := helpers.NewMatrix[ordering]()
	defer matrix.Run(gt)

	for _, ordering := range orderings {
		matrix.AddTestCase(
			fmt.Sprintf("HonestClaim-%s", ordering.name),
			ordering,
			helpers.NewForkMatrix(helpers.Granite, helpers.LatestFork),
			runHoloceneDerivationTest,
			helpers.ExpectNoError(),
		)
		matrix.AddTestCase(
			fmt.Sprintf("JunkClaim-%s", ordering.name),
			ordering,
			helpers.NewForkMatrix(helpers.Granite, helpers.LatestFork),
			runHoloceneDerivationTest,
			helpers.ExpectError(claim.ErrClaimNotValid),
			helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
		)
	}
}

func runHoloceneDerivationTest(gt *testing.T, testCfg *helpers.TestCfg[ordering]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	tp := helpers.NewTestParams(func(tp *e2eutils.TestParams) {
		// Set the channel timeout to 10 blocks, 12x lower than the sequencing window.
		tp.ChannelTimeout = 10
	})

	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, helpers.NewBatcherCfg())

	includeBatchTx := func() {
		// Include the last transaction submitted by the batcher.
		env.Miner.ActL1StartBlock(12)(t)
		env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
		env.Miner.ActL1EndBlock(t)

		// Finalize the block with the first channel frame on L1.
		env.Miner.ActL1SafeNext(t)
		env.Miner.ActL1FinalizeNext(t)

		// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActL2PipelineFull(t)
	}

	env.Batcher.ActCreateChannel(t, testCfg.Custom.isSpanBatch)

	targetHeadNumber := max(testCfg.Custom.blocks)
	for env.Engine.L2Chain().CurrentBlock().Number.Uint64() < uint64(targetHeadNumber) {
		// Build a block on L2 with 1 tx.
		env.Alice.L2.ActResetTxOpts(t)
		env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)
		env.Alice.L2.ActMakeTx(t)
		env.Sequencer.ActL2StartBlock(t)
		env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
		env.Sequencer.ActL2EndBlock(t)
		env.Alice.L2.ActCheckReceiptStatusOfLastTx(true)(t)
	}

	// Build up a local list of frames
	orderedFrames := make([][]byte, 0, len(testCfg.Custom.frames))

	blockLogger := func(block *types.Block) *types.Block {
		t.Log("added block", "num", block.Number(), "txs", block.Transactions())
		return block
	}

	// Buffer the blocks in the batcher.
	for i, blockNum := range testCfg.Custom.blocks {

		var blockModifier actionsHelpers.BlockModifier
		if len(testCfg.Custom.blockModifiers) > i {
			blockModifier = testCfg.Custom.blockModifiers[i]
		}
		env.Batcher.ActAddBlockByNumber(t, int64(blockNum), blockModifier, blockLogger)
		if i == len(testCfg.Custom.blocks)-1 {
			env.Batcher.ActL2ChannelClose(t)
		}
		frame := env.Batcher.ReadNextOutputFrame(t)
		require.NotEmpty(t, frame, "frame %d", i)
		orderedFrames = append(orderedFrames, frame)
	}

	// Submit frames in specified order order
	for _, j := range testCfg.Custom.frames {
		env.Batcher.ActL2BatchSubmitRaw(t, orderedFrames[j])
		includeBatchTx()
	}

	// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()

	if testCfg.Hardfork.Precedence < helpers.Holocene.Precedence {
		require.Equal(t, testCfg.Custom.safeHeadPreHolocene, l2SafeHead.Number.Uint64())
		expectedHash := env.Engine.L2Chain().GetBlockByNumber(testCfg.Custom.safeHeadPreHolocene).Hash()
		require.Equal(t, expectedHash, l2SafeHead.Hash())
	} else {

		require.Equal(t, testCfg.Custom.safeHeadHolocene, l2SafeHead.Number.Uint64())
		expectedHash := env.Engine.L2Chain().GetBlockByNumber(testCfg.Custom.safeHeadHolocene).Hash()
		require.Equal(t, expectedHash, l2SafeHead.Hash())
	}

	if safeHeadNumber := l2SafeHead.Number.Uint64(); safeHeadNumber > 0 {
		env.RunFaultProofProgram(t, safeHeadNumber, testCfg.CheckResult, testCfg.InputParams...)
	}

}
