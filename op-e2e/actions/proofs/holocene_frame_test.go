package proofs

import (
	"fmt"
	"testing"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func Test_ProgramAction_HoloceneFrames(gt *testing.T) {

	type testCase struct {
		name                string
		frames              []uint
		safeHeadPreHolocene uint64
		safeHeadHolocene    uint64
	}

	// An ordered list of frames to read from the channel and submit
	// on L1. We expect a different progression of the safe head under Holocene
	// derivation rules, compared with pre Holocene.
	var testCases = []testCase{
		// Standard frame submission,
		{name: "case-0", frames: []uint{0, 1, 2}, safeHeadPreHolocene: 3, safeHeadHolocene: 3},

		// Non-standard frame submission
		{name: "case-1a", frames: []uint{2, 1, 0},
			safeHeadPreHolocene: 3, // frames are buffered, so ordering does not matter
			safeHeadHolocene:    0, // non-first frames will be dropped b/c it is the first seen with that channel Id. The safe head won't move until the channel is closed/completed.
		},
		{name: "case-1b", frames: []uint{0, 1, 0, 2},
			safeHeadPreHolocene: 3, // frames are buffered, so ordering does not matter
			safeHeadHolocene:    0, // non-first frames will be dropped b/c it is the first seen with that channel Id. The safe head won't move until the channel is closed/completed.
		},
		{name: "case-1c", frames: []uint{0, 1, 1, 2},
			safeHeadPreHolocene: 3, // frames are buffered, so ordering does not matter
			safeHeadHolocene:    3, // non-contiguous frames are dropped. So this reduces to case-0.
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

		env.Batcher.ActCreateChannel(t, false)

		blocks := []uint{1, 2, 3}
		targetHeadNumber := 3
		for env.Engine.L2Chain().CurrentBlock().Number.Uint64() < uint64(targetHeadNumber) {

			env.Sequencer.ActL2StartBlock(t)

			// Send an L2 tx
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)
			env.Alice.L2.ActMakeTx(t)
			env.Engine.ActL2IncludeTx(env.Alice.Address())(t)

			env.Sequencer.ActL2EndBlock(t)
		}

		// Build up a local list of frames
		orderedFrames := make([][]byte, 0, len(testCfg.Custom.frames))

		blockLogger := func(block *types.Block) *types.Block {
			t.Log("added block", "num", block.Number(), "txs", block.Transactions(), "time", block.Time(), "l1_origin")
			return block
		}

		// Buffer the blocks in the batcherand populated orderedFrames list
		for i, blockNum := range blocks {
			env.Batcher.ActAddBlockByNumber(t, int64(blockNum), blockLogger)
			if i == len(blocks)-1 {
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

		l2SafeHead := env.Sequencer.L2Safe()

		if testCfg.Hardfork.Precedence < helpers.Holocene.Precedence {
			require.Equal(t, testCfg.Custom.safeHeadPreHolocene, l2SafeHead.Number)
			expectedHash := env.Engine.L2Chain().GetBlockByNumber(testCfg.Custom.safeHeadPreHolocene).Hash()
			require.Equal(t, expectedHash, l2SafeHead.Hash)
		} else {
			require.Equal(t, testCfg.Custom.safeHeadHolocene, l2SafeHead.Number)
			expectedHash := env.Engine.L2Chain().GetBlockByNumber(testCfg.Custom.safeHeadHolocene).Hash()
			require.Equal(t, expectedHash, l2SafeHead.Hash)
		}

		t.Log("Safe head progressed as expected", "l2SafeHeadNumber", l2SafeHead.Number)

		if safeHeadNumber := l2SafeHead.Number; safeHeadNumber > 0 {
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
