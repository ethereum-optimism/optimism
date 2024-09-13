package proofs

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/actions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// garbageKinds is a list of garbage kinds to test. We don't use `INVALID_COMPRESSION` and `MALFORM_RLP` because
// they submit malformed frames always, and this test models a valid channel with a single invalid frame in the
// middle.
var garbageKinds = []actions.GarbageKind{
	actions.STRIP_VERSION,
	actions.RANDOM,
	actions.TRUNCATE_END,
	actions.DIRTY_APPEND,
}

// Run a test that submits garbage channel data in the middle of a channel.
//
// channel format ([]Frame):
// [f[0 - correct] f_x[1 - bad frame] f[1 - correct]]
func runGarbageChannelTest(gt *testing.T, testCfg *TestCfg[actions.GarbageKind]) {
	t := actions.NewDefaultTesting(gt)
	tp := NewTestParams(func(tp *e2eutils.TestParams) {
		// Set the channel timeout to 10 blocks, 12x lower than the sequencing window.
		tp.ChannelTimeout = 10
	})
	env := NewL2FaultProofEnv(t, testCfg, tp, NewBatcherCfg())

	includeBatchTx := func(env *L2FaultProofEnv) {
		// Instruct the batcher to submit the first channel frame to L1, and include the transaction.
		env.miner.ActL1StartBlock(12)(t)
		env.miner.ActL1IncludeTxByHash(env.batcher.LastSubmitted.Hash())(t)
		env.miner.ActL1EndBlock(t)

		// Finalize the block with the first channel frame on L1.
		env.miner.ActL1SafeNext(t)
		env.miner.ActL1FinalizeNext(t)

		// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
		env.sequencer.ActL1HeadSignal(t)
		env.sequencer.ActL2PipelineFull(t)
	}

	const NumL2Blocks = 10

	// Build NumL2Blocks empty blocks on L2
	for i := 0; i < NumL2Blocks; i++ {
		env.sequencer.ActL2StartBlock(t)
		env.sequencer.ActL2EndBlock(t)
	}

	// Buffer the first half of L2 blocks in the batcher, and submit it.
	for i := 0; i < NumL2Blocks/2; i++ {
		env.batcher.ActL2BatchBuffer(t)
	}
	env.batcher.ActL2BatchSubmit(t)

	// Include the batcher transaction.
	includeBatchTx(env)

	// Ensure that the safe head has not advanced - the channel is incomplete.
	l2SafeHead := env.engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(0), l2SafeHead.Number.Uint64())

	// Buffer the second half of L2 blocks in the batcher.
	for i := 0; i < NumL2Blocks/2; i++ {
		env.batcher.ActL2BatchBuffer(t)
	}
	env.batcher.ActL2ChannelClose(t)
	expectedSecondFrame := env.batcher.ReadNextOutputFrame(t)

	// Submit a garbage frame, modified from the expected second frame.
	env.batcher.ActL2BatchSubmitGarbageRaw(t, expectedSecondFrame, testCfg.Custom)
	// Include the garbage second frame tx
	includeBatchTx(env)

	// Ensure that the safe head has not advanced - the channel is incomplete.
	l2SafeHead = env.engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(0), l2SafeHead.Number.Uint64())

	// Submit the correct second frame.
	env.batcher.ActL2BatchSubmitRaw(t, expectedSecondFrame)
	// Include the corract second frame tx.
	includeBatchTx(env)

	// Ensure that the safe head has advanced - the channel is complete.
	l2SafeHead = env.engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(NumL2Blocks), l2SafeHead.Number.Uint64())

	// Run the FPP on L2 block # NumL2Blocks.
	env.RunFaultProofProgram(t, NumL2Blocks, testCfg.CheckResult, testCfg.InputParams...)
}

func Test_ProgramAction_GarbageChannel(gt *testing.T) {
	matrix := NewMatrix[actions.GarbageKind]()
	defer matrix.Run(gt)

	for _, garbageKind := range garbageKinds {
		matrix.AddTestCase(
			fmt.Sprintf("HonestClaim-%s", garbageKind.String()),
			garbageKind,
			LatestForkOnly,
			runGarbageChannelTest,
			ExpectNoError(),
		)
		matrix.AddTestCase(
			fmt.Sprintf("JunkClaim-%s", garbageKind.String()),
			garbageKind,
			LatestForkOnly,
			runGarbageChannelTest,
			ExpectError(claim.ErrClaimNotValid),
			WithL2Claim(common.HexToHash("0xdeadbeef")),
		)
	}
}
