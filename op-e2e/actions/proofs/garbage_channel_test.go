package proofs

import (
	"fmt"
	"testing"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// garbageKinds is a list of garbage kinds to test. We don't use `INVALID_COMPRESSION` and `MALFORM_RLP` because
// they submit malformed frames always, and this test models a valid channel with a single invalid frame in the
// middle.
var garbageKinds = []actionsHelpers.GarbageKind{
	actionsHelpers.STRIP_VERSION,
	actionsHelpers.RANDOM,
	actionsHelpers.TRUNCATE_END,
	actionsHelpers.DIRTY_APPEND,
}

// Run a test that submits garbage channel data in the middle of a channel.
//
// channel format ([]Frame):
// [f[0 - correct] f_x[1 - bad frame] f[1 - correct]]
func runGarbageChannelTest(gt *testing.T, testCfg *helpers.TestCfg[actionsHelpers.GarbageKind]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	tp := helpers.NewTestParams(func(tp *e2eutils.TestParams) {
		// Set the channel timeout to 10 blocks, 12x lower than the sequencing window.
		tp.ChannelTimeout = 10
	})
	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, helpers.NewBatcherCfg())

	includeBatchTx := func(env *helpers.L2FaultProofEnv) {
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
	env.Batcher.ActL2BatchSubmit(t)

	// Include the batcher transaction.
	includeBatchTx(env)

	// Ensure that the safe head has not advanced - the channel is incomplete.
	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(0), l2SafeHead.Number.Uint64())

	// Buffer the second half of L2 blocks in the batcher.
	for i := 0; i < NumL2Blocks/2; i++ {
		env.Batcher.ActL2BatchBuffer(t)
	}
	env.Batcher.ActL2ChannelClose(t)
	expectedSecondFrame := env.Batcher.ReadNextOutputFrame(t)

	// Submit a garbage frame, modified from the expected second frame.
	env.Batcher.ActL2BatchSubmitGarbageRaw(t, expectedSecondFrame, testCfg.Custom)
	// Include the garbage second frame tx
	includeBatchTx(env)

	// Ensure that the safe head has not advanced - the channel is incomplete.
	l2SafeHead = env.Engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(0), l2SafeHead.Number.Uint64())

	// Submit the correct second frame.
	env.Batcher.ActL2BatchSubmitRaw(t, expectedSecondFrame)
	// Include the corract second frame tx.
	includeBatchTx(env)

	// Ensure that the safe head has advanced - the channel is complete.
	l2SafeHead = env.Engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(NumL2Blocks), l2SafeHead.Number.Uint64())

	// Run the FPP on L2 block # NumL2Blocks.
	env.RunFaultProofProgram(t, NumL2Blocks, testCfg.CheckResult, testCfg.InputParams...)
}

func Test_ProgramAction_GarbageChannel(gt *testing.T) {
	matrix := helpers.NewMatrix[actionsHelpers.GarbageKind]()
	defer matrix.Run(gt)

	for _, garbageKind := range garbageKinds {
		matrix.AddTestCase(
			fmt.Sprintf("HonestClaim-%s", garbageKind.String()),
			garbageKind,
			helpers.LatestForkOnly,
			runGarbageChannelTest,
			helpers.ExpectNoError(),
		)
		matrix.AddTestCase(
			fmt.Sprintf("JunkClaim-%s", garbageKind.String()),
			garbageKind,
			helpers.LatestForkOnly,
			runGarbageChannelTest,
			helpers.ExpectError(claim.ErrClaimNotValid),
			helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
		)
	}
}
