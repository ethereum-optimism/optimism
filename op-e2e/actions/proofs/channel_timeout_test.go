package proofs

import (
	"testing"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Run a test that submits the first channel frame, times out the channel, and then resubmits the full channel.
func runChannelTimeoutTest(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	tp := helpers.NewTestParams()
	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, helpers.NewBatcherCfg())
	channelTimeout := env.Sd.ChainSpec.ChannelTimeout(0)

	var timedOutChannels uint
	env.Sequencer.DerivationMetricsTracer().FnRecordChannelTimedOut = func() {
		timedOutChannels++
	}

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
	firstFrame := env.Batcher.ReadNextOutputFrame(t)
	env.Batcher.ActL2BatchSubmitRaw(t, firstFrame)

	// Include the batcher transaction.
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)

	// Finalize the block with the first channel frame on L1.
	env.Miner.ActL1SafeNext(t)

	// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	// Ensure that the safe head has not advanced - the channel is incomplete.
	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(0), l2SafeHead.Number.Uint64())

	// Time out the channel by mining `channelTimeout + 1` empty blocks on L1.
	for i := uint64(0); i < channelTimeout+1; i++ {
		env.Miner.ActEmptyBlock(t)
		env.Miner.ActL1SafeNext(t)
	}

	// Instruct the sequencer to derive the L2 chain - the channel should now be timed out.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	// Ensure the safe head has still not advanced.
	l2SafeHead = env.Engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(0), l2SafeHead.Number.Uint64())

	// Ensure that the channel was timed out.
	require.EqualValues(t, 1, timedOutChannels)

	// Instruct the batcher to submit the blocks to L1 in a new channel,
	// submitted across 2 transactions.
	for i := 0; i < 2; i++ {
		if i == 0 {
			// Re-submit the first frame
			env.Batcher.ActL2BatchSubmitRaw(t, firstFrame)
		} else {
			// Buffer half of the L2 chain's blocks.
			for j := 0; j < NumL2Blocks/2; j++ {
				env.Batcher.ActL2BatchBuffer(t)
			}
			env.Batcher.ActL2ChannelClose(t)
			env.Batcher.ActL2BatchSubmit(t)
		}

		env.Miner.ActL1StartBlock(12)(t)
		env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
		env.Miner.ActL1EndBlock(t)

		// Finalize the block with the frame data on L1.
		env.Miner.ActL1SafeNext(t)
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

func runChannelTimeoutTest_CloseChannelLate(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	tp := helpers.NewTestParams()
	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, helpers.NewBatcherCfg())
	channelTimeout := env.Sd.ChainSpec.ChannelTimeout(0)

	var timedOutChannels uint
	env.Sequencer.DerivationMetricsTracer().FnRecordChannelTimedOut = func() {
		timedOutChannels++
	}

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
	firstFrame := env.Batcher.ReadNextOutputFrame(t)
	env.Batcher.ActL2BatchSubmitRaw(t, firstFrame)

	// Instruct the batcher to submit the first channel frame to L1, and include the transaction.
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)

	// Finalize the block with the first channel frame on L1.
	env.Miner.ActL1SafeNext(t)

	// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	// Ensure that the safe head has not advanced - the channel is incomplete.
	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(0), l2SafeHead.Number.Uint64())

	// Time out the channel by mining `channelTimeout + 1` empty blocks on L1.
	for i := uint64(0); i < channelTimeout+1; i++ {
		env.Miner.ActEmptyBlock(t)
		env.Miner.ActL1SafeNext(t)
	}

	// Instruct the sequencer to derive the L2 chain.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	// Ensure the safe head has still not advanced.
	l2SafeHead = env.Engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(0), l2SafeHead.Number.Uint64())

	// Ensure that the channel was timed out.
	require.EqualValues(t, 1, timedOutChannels)

	// Cache the second and final frame of the channel from the batcher, but do not submit it yet.
	for i := 0; i < NumL2Blocks/2; i++ {
		env.Batcher.ActL2BatchBuffer(t)
	}
	env.Batcher.ActL2ChannelClose(t)
	finalFrame := env.Batcher.ReadNextOutputFrame(t)

	// Submit the final frame of the timed out channel, now that the channel has timed out.
	env.Batcher.ActL2BatchSubmitRaw(t, finalFrame)

	// Instruct the batcher to submit the second channel frame to L1, and include the transaction.
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)

	// Finalize the block with the second channel frame on L1.
	env.Miner.ActL1SafeNext(t)

	// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	// Ensure the safe head has still not advanced.
	l2SafeHead = env.Engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(0), l2SafeHead.Number.Uint64())

	// Instruct the batcher to submit the blocks to L1 in a new channel.
	for _, frame := range [][]byte{firstFrame, finalFrame} {
		env.Batcher.ActL2BatchSubmitRaw(t, frame)
		env.Miner.ActL1StartBlock(12)(t)
		env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
		env.Miner.ActL1EndBlock(t)

		// Finalize the block with the resubmitted channel frames on L1.
		env.Miner.ActL1SafeNext(t)
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
	matrix.AddTestCase(
		"CloseChannelLate-HonestClaim",
		nil,
		helpers.LatestForkOnly,
		runChannelTimeoutTest_CloseChannelLate,
		helpers.ExpectNoError(),
	)
	matrix.AddTestCase(
		"CloseChannelLate-JunkClaim",
		nil,
		helpers.LatestForkOnly,
		runChannelTimeoutTest_CloseChannelLate,
		helpers.ExpectError(claim.ErrClaimNotValid),
		helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
	)
}
