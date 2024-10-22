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

func Test_ProgramAction_HoloceneDerivationRules(gt *testing.T) {

	type testCase struct {
		name                    string
		blocks                  []uint // could enhance this to declare either singular or span batches or a mixture
		isSpanBatch             bool
		blockModifiers          []actionsHelpers.BlockModifier
		frames                  []uint // ignored if isSpanBatch
		safeHeadPreHolocene     uint64
		safeHeadHolocene        uint64
		breachMaxSequencerDrift bool
	}

	// invalidPayload invalidates the signature for the second transaction in the block.
	// This should result in an invalid payload in the engine queue.
	var invalidPayload = func(block *types.Block) *types.Block {
		alice := types.NewCancunSigner(big.NewInt(901))
		txs := block.Transactions()
		newTx, err := txs[1].WithSignature(alice, make([]byte, 65))
		if err != nil {
			panic(err)
		}
		txs[1] = newTx
		return block
	}

	// invalidParentHash invalidates the parentHash of the block.
	// This should result in an invalid batch being derived,
	// but only for singular (not for span) batches.
	var invalidParentHash = func(block *types.Block) *types.Block {
		headerCopy := block.Header()
		headerCopy.ParentHash = common.MaxHash
		return block.WithSeal(headerCopy)
	}

	k := 2000
	var twoThousandBlocks = make([]uint, k)
	for i := 0; i < k; i++ {
		twoThousandBlocks[i] = uint(i) + 1
	}

	// testCases is a list of testCases which each specify
	// an ordered list of blocks (by number) to add to a single channel
	// and an ordered list of frames to read from the channel and submit
	// on L1. There will be one frame per block unless isSpanBatch=true,
	// in which case all blocks are added to a single span batch which is
	// sent as a single frame.
	// Depending on these lists, whether the channel is built as
	// as span batch channel, and whether the blocks are modified / invalidated
	// we expect a different progression of the safe head under Holocene
	// derivation rules, compared with pre Holocene.
	var testCases = []testCase{
		// Standard frame submission, standard channel composition
		{name: "case-0", blocks: []uint{1, 2, 3}, frames: []uint{0, 1, 2}, safeHeadPreHolocene: 3, safeHeadHolocene: 3},

		// Non-standard frame submission, standard channel composition
		{name: "case-1a", blocks: []uint{1, 2, 3}, frames: []uint{2, 1, 0},
			safeHeadPreHolocene: 3, // frames are buffered, so ordering does not matter
			safeHeadHolocene:    0, // non-first frames will be dropped b/c it is the first seen with that channel Id. The safe head won't move until the channel is closed/completed.
		},
		{name: "case-1b", blocks: []uint{1, 2, 3}, frames: []uint{0, 1, 0, 2},
			safeHeadPreHolocene: 3, // frames are buffered, so ordering does not matter
			safeHeadHolocene:    0, // non-first frames will be dropped b/c it is the first seen with that channel Id. The safe head won't move until the channel is closed/completed.
		},
		{name: "case-1c", blocks: []uint{1, 2, 3}, frames: []uint{0, 1, 1, 2},
			safeHeadPreHolocene: 3, // frames are buffered, so ordering does not matter
			safeHeadHolocene:    3, // non-contiguous frames are dropped. So this reduces to case-0.
		},

		// Standard frame submission, non-standard channel composition
		{name: "case-2a", blocks: []uint{1, 3, 2}, frames: []uint{0, 1, 2},
			safeHeadPreHolocene: 3, // batches are buffered, so the block ordering does not matter
			safeHeadHolocene:    1, // batch for block 3 is considered invalid because it is from the future. This batch + remaining channel is dropped.
		},
		{name: "case-2b", blocks: []uint{2, 1, 3}, frames: []uint{0, 1, 2},
			safeHeadPreHolocene: 3, // batches are buffered, so the block ordering does not matter
			safeHeadHolocene:    0, // batch for block 2 is considered invalid because it is from the future. This batch + remaining channel is dropped.
		},
		{name: "case-2c", blocks: []uint{1, 1, 2, 3}, frames: []uint{0, 1, 2, 3},
			safeHeadPreHolocene: 3, // duplicate batches are silently dropped, so this reduceds to case-0
			safeHeadHolocene:    3, // duplicate batches are silently dropped
		},
		{name: "case-2d", blocks: []uint{2, 2, 1, 3}, frames: []uint{0, 1, 2, 3},
			safeHeadPreHolocene: 3, // duplicate batches are silently dropped, so this reduces to case-2b
			safeHeadHolocene:    0, // duplicate batches are silently dropped, so this reduces to case-2b
		},
		{name: "case-3a", blocks: []uint{1, 2, 3}, blockModifiers: []actionsHelpers.BlockModifier{nil, invalidPayload, nil},
			isSpanBatch:         true,
			safeHeadPreHolocene: 0, // Invalid signature in block 2 causes an invalid _payload_ in the engine queue. Entire span batch is invalidated.
			safeHeadHolocene:    0, // TODO with full Holocene implementation, we expect the safe head to move to 2 due to creation of an deposit-only block.
		},
		{name: "case-3b", blocks: []uint{1, 2, 3}, blockModifiers: []actionsHelpers.BlockModifier{nil, invalidParentHash, nil},
			frames:              []uint{0, 1, 2},
			isSpanBatch:         false,
			safeHeadPreHolocene: 1, // Invalid parentHash in block 2 causes an invalid batch to be derived.
			safeHeadHolocene:    1, // Invalid parentHash in block 2 causes an invalid batch to be derived. This batch + remaining channel is dropped.
		},
		{name: "case-3c", blocks: twoThousandBlocks, // if we artificially stall the l1 origin, this should be enough to trigger violation of the max sequencer drift
			isSpanBatch:             true,
			safeHeadPreHolocene:     0, // entire span batch invalidated
			safeHeadHolocene:        0, // TODO we expect partial validity around block 1800
			breachMaxSequencerDrift: true,
		},
	}

	runHoloceneDerivationTest := func(gt *testing.T, testCfg *helpers.TestCfg[testCase]) {
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
		}

		if testCfg.Custom.breachMaxSequencerDrift {
			env.Sequencer.ActL2KeepL1Origin(t)
		} // prevent L1 origin from progressing

		env.Batcher.ActCreateChannel(t, testCfg.Custom.isSpanBatch)

		var max = func(input []uint) uint {
			max := uint(0)
			for _, val := range input {
				if val > max {
					max = val
				}
			}
			return max
		}

		targetHeadNumber := max(testCfg.Custom.blocks)
		for env.Engine.L2Chain().CurrentBlock().Number.Uint64() < uint64(targetHeadNumber) {
			env.Sequencer.ActL2StartBlock(t)

			if !testCfg.Custom.breachMaxSequencerDrift ||
				env.Engine.L2Chain().CurrentBlock().Number.Uint64() == 1799 ||
				env.Engine.L2Chain().CurrentBlock().Number.Uint64() == 1800 ||
				env.Engine.L2Chain().CurrentBlock().Number.Uint64() == 1801 {
				// Send an L2 tx
				env.Alice.L2.ActResetTxOpts(t)
				env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)
				env.Alice.L2.ActMakeTx(t)
				env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
			}
			env.Sequencer.ActL2EndBlock(t)
		}

		// Build up a local list of frames
		orderedFrames := make([][]byte, 0, len(testCfg.Custom.frames))

		blockLogger := func(block *types.Block) *types.Block {
			t.Log("added block", "num", block.Number(), "txs", block.Transactions(), "time", block.Time())
			return block
		}

		// Buffer the blocks in the batcher.
		for i, blockNum := range testCfg.Custom.blocks {

			var blockModifier actionsHelpers.BlockModifier
			if len(testCfg.Custom.blockModifiers) > i {
				blockModifier = testCfg.Custom.blockModifiers[i]
			}
			env.Batcher.ActAddBlockByNumber(t, int64(blockNum), blockModifier, blockLogger)

			if !testCfg.Custom.isSpanBatch {
				if i == len(testCfg.Custom.blocks)-1 {
					env.Batcher.ActL2ChannelClose(t)
				}
				frame := env.Batcher.ReadNextOutputFrame(t)
				require.NotEmpty(t, frame, "frame %d", i)
				orderedFrames = append(orderedFrames, frame)
			}
		}

		if testCfg.Custom.isSpanBatch { // Make a single frame for the span batch and submit it
			env.Batcher.ActL2ChannelClose(t)
			frame := env.Batcher.ReadNextOutputFrame(t)
			require.NotEmpty(t, frame)
			env.Batcher.ActL2BatchSubmitRaw(t, frame)
			includeBatchTx()
		} else {
			// Submit frames in specified order order
			for _, j := range testCfg.Custom.frames {
				env.Batcher.ActL2BatchSubmitRaw(t, orderedFrames[j])
				includeBatchTx()
			}
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

	matrix := helpers.NewMatrix[testCase]()
	defer matrix.Run(gt)

	for _, ordering := range testCases {
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
