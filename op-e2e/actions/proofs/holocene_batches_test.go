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

func Test_ProgramAction_HoloceneBatches(gt *testing.T) {

	type testCase struct {
		name                string
		blocks              []uint // could enhance this to declare either singular or span batches or a mixture
		isSpanBatch         bool
		safeHeadPreHolocene uint64
		safeHeadHolocene    uint64
	}

	// An ordered list of blocks (by number) to add to a single channel.
	// Depending on the list,  we expect a different progression of the safe head under Holocene
	// derivation rules, compared with pre Holocene.
	var testCases = []testCase{
		// Standard channel composition
		{name: "case-0", blocks: []uint{1, 2, 3}, safeHeadPreHolocene: 3, safeHeadHolocene: 3},

		// Non-standard channel composition
		{name: "case-2a", blocks: []uint{1, 3, 2},
			safeHeadPreHolocene: 3, // batches are buffered, so the block ordering does not matter
			safeHeadHolocene:    1, // batch for block 3 is considered invalid because it is from the future. This batch + remaining channel is dropped.
		},
		{name: "case-2b", blocks: []uint{2, 1, 3},
			safeHeadPreHolocene: 3, // batches are buffered, so the block ordering does not matter
			safeHeadHolocene:    0, // batch for block 2 is considered invalid because it is from the future. This batch + remaining channel is dropped.
		},
		{name: "case-2c", blocks: []uint{1, 1, 2, 3},
			safeHeadPreHolocene: 3, // duplicate batches are silently dropped, so this reduceds to case-0
			safeHeadHolocene:    3, // duplicate batches are silently dropped
		},
		{name: "case-2d", blocks: []uint{2, 2, 1, 3},
			safeHeadPreHolocene: 3, // duplicate batches are silently dropped, so this reduces to case-2b
			safeHeadHolocene:    0, // duplicate batches are silently dropped, so this reduces to case-2b
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

			// Send an L2 tx
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)
			env.Alice.L2.ActMakeTx(t)
			env.Engine.ActL2IncludeTx(env.Alice.Address())(t)

			env.Sequencer.ActL2EndBlock(t)
		}

		blockLogger := func(block *types.Block) *types.Block {
			t.Log("added block", "num", block.Number(), "txs", block.Transactions(), "time", block.Time(), "l1_origin")
			return block
		}

		// Buffer the blocks in the batcher.
		for _, blockNum := range testCfg.Custom.blocks {
			env.Batcher.ActAddBlockByNumber(t, int64(blockNum), blockLogger)
		}
		env.Batcher.ActL2ChannelClose(t)
		frame := env.Batcher.ReadNextOutputFrame(t)
		require.NotEmpty(t, frame)
		env.Batcher.ActL2BatchSubmitRaw(t, frame)
		includeBatchTx()

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
