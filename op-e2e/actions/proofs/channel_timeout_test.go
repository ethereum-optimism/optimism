package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/actions"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Run a test that exercises the channel timeout functionality in `op-program`.
//
// Steps:
// 1. Build `NumL2Blocks` empty blocks on L2.
// 2. Buffer the first half of the L2 blocks in the batcher, and submit the frame data.
// 3. Time out the channel by mining `ChannelTimeout + 1` empty blocks on L1.
// 4. Submit the channel frame data across 2 transactions.
// 5. Instruct the sequencer to derive the L2 chain.
// 6. Run the FPP on the safe head.
func runChannelTimeoutTest(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actions.NewDefaultTesting(gt)
	tp := helpers.NewTestParams(func(tp *e2eutils.TestParams) {
		// Set the channel timeout to 10 blocks, 12x lower than the sequencing window.
		tp.ChannelTimeout = 10
	})
	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, helpers.NewBatcherCfg())

	const NumL2Blocks = 10

	// Build NumL2Blocks empty blocks on L2
	for i := 0; i < NumL2Blocks; i++ {
		env.Sequencer.ActL2StartBlock(t)
		env.Sequencer.ActL2EndBlock(t)
	}

	// Buffer the first half of L2 blocks in the batcher, and submit it.
	for i := 0; i < NumL2Blocks/2; i++ {
		env.Batcher.ActL2BatchBuffer(t)
	}
	env.Batcher.ActL2BatchSubmit(t)

	// Instruct the batcher to submit the first channel frame to L1, and include the transaction.
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)

	// Finalize the block with the first channel frame on L1.
	env.Miner.ActL1SafeNext(t)
	env.Miner.ActL1FinalizeNext(t)

	// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	// Ensure that the safe head has not advanced - the channel is incomplete.
	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(0), l2SafeHead.Number.Uint64())

	// Time out the channel by mining `ChannelTimeout + 1` empty blocks on L1.
	for i := uint64(0); i < tp.ChannelTimeout+1; i++ {
		env.Miner.ActEmptyBlock(t)
		env.Miner.ActL1SafeNext(t)
		env.Miner.ActL1FinalizeNext(t)
	}

	// Instruct the sequencer to derive the L2 chain - the channel should now be timed out.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	// Ensure the safe head has still not advanced.
	l2SafeHead = env.Engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(0), l2SafeHead.Number.Uint64())

	// Instruct the batcher to submit the blocks to L1 in a new channel,
	// submitted across 2 transactions.
	for i := 0; i < 2; i++ {
		// Buffer half of the L2 chain's blocks.
		for j := 0; j < NumL2Blocks/2; j++ {
			env.Batcher.ActL2BatchBuffer(t)
		}

		// Close the channel on the second iteration.
		if i == 1 {
			env.Batcher.ActL2ChannelClose(t)
		}

		env.Batcher.ActL2BatchSubmit(t)
		env.Miner.ActL1StartBlock(12)(t)
		env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
		env.Miner.ActL1EndBlock(t)

		// Finalize the block with the frame data on L1.
		env.Miner.ActL1SafeNext(t)
		env.Miner.ActL1FinalizeNext(t)
	}

	// Instruct the sequencer to derive the L2 chain.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	// Ensure the safe head has still advanced to L2 block # NumL2Blocks.
	l2SafeHead = env.Engine.L2Chain().CurrentSafeBlock()
	require.EqualValues(t, NumL2Blocks, l2SafeHead.Number.Uint64())

	// Run the FPP on L2 block # NumL2Blocks/2.
	env.RunFaultProofProgram(t, NumL2Blocks/2, testCfg.CheckResult, testCfg.InputParams...)
}

func Test_ProgramAction_ChannelTimeout(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)

	matrix.AddTestCase(
		"HonestClaim",
		nil,
		helpers.LatestForkOnly,
		runChannelTimeoutTest,
		helpers.ExpectNoError(),
	)
	matrix.AddTestCase(
		"JunkClaim",
		nil,
		helpers.LatestForkOnly,
		runChannelTimeoutTest,
		helpers.ExpectError(claim.ErrClaimNotValid),
		helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
	)
}
