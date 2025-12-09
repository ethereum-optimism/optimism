package proofs

import (
	"strconv"
	"testing"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	recoverMode bool
	spanBatches bool
}

// Run a test that proves a deposit-only block generated due to sequence window expiry.
func runSequenceWindowExpireTest(gt *testing.T, testCfg *helpers.TestCfg[testCase]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	tp := helpers.NewTestParams()
	bc := helpers.NewBatcherCfg()

	if testCfg.Custom.spanBatches {
		// It seems more difficult to recover with span batches, since the singular batches within are invalidated atomically.
		// That is to say, if the oldest batch in the span batch fails the sequencing window check (l1 origin + seq window < l1 inclusion)
		// All following batches are invalidated / dropped as well.
		// Although, if the same blocks were batched with singular batches, wouldn't the older blocks being rejected invalidate that later batches anyway?
		// Perhaps not in the case of recover mode since the noTxPool mode means autoderviation actually fills the gap with identical blocks anyway.
		// It seems like this is actually just a problem with derivation, it would be possible to consider modifying the rules to allow for recovery from span batches too.
		bc.ForceSubmitSpanBatch = true
	} else {
		bc.ForceSubmitSingularBatch = true
	}

	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, bc)

	// Mine an empty L1 block for gas estimation purposes.
	env.Miner.ActEmptyBlock(t)

	// Expire the sequence window by building `SequenceWindow + 1` empty blocks on L1.
	// note that tp.SequencerWindowSize is 10.
	// If we were sequencing properly, we would expect the unsafe head to be up to 15*10 = 150
	// because l2 block time is 1s, l1 block time is 15s, and sequence window is 10 blocks.
	for i := 0; i < int(tp.SequencerWindowSize)+1; i++ {
		env.Alice.L1.ActResetTxOpts(t)
		env.Alice.ActDeposit(t)

		env.Miner.ActL1StartBlock(15)(t)
		env.Miner.ActL1IncludeTx(env.Alice.Address())(t)
		env.Miner.ActL1EndBlock(t)

		env.Miner.ActL1SafeNext(t)
		env.Miner.ActL1FinalizeNext(t)
	}

	// Ensure the safe head is still 0.
	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()
	require.EqualValues(t, 0, l2SafeHead.Number.Uint64())

	// Ask the sequencer to derive the deposit-only L2 chain.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	// Ensure the safe head advanced forcefully.
	l2SafeHead = env.Engine.L2Chain().CurrentSafeBlock()
	require.Greater(t, l2SafeHead.Number.Uint64(), uint64(0))

	// Run the FPP on one of the auto-derived blocks.
	env.RunFaultProofProgram(t, l2SafeHead.Number.Uint64()/2, testCfg.CheckResult, testCfg.InputParams...)

	// Set recover mode on the sequencer:
	// I am not actually convinced we need this...
	// and I think it does more harm than good when
	// the lag approaches zero
	env.Sequencer.ActSetRecoverMode(t, testCfg.Custom.recoverMode)

	// Build both chains and assert the l1 origin catches back up with the tip of the L1 chain.
	numL1Blocks := 0
	timeout := tp.SequencerWindowSize * 5
	ss := env.Sequencer.SyncStatus()
	lag := ss.CurrentL1.Number - ss.SafeL2.L1Origin.Number
	require.GreaterOrEqual(t, lag, tp.SequencerWindowSize, "Lag is less than sequencing window size")
	for numL1Blocks < int(timeout) {
		for range tp.L1BlockTime / env.Sd.RollupCfg.BlockTime {
			env.Sequencer.ActL2StartBlock(t)
			env.Alice.L2.ActMakeTx(t)
			env.Engine.ActL2IncludeTx(env.Alice.Address())
			// RecoverMode will ensure no user transactions, so don't mess with that here.
			env.Sequencer.ActL2EndBlock(t)
		}
		if numL1Blocks%2 == 0 {
			env.BatchMineAndSync(t)
		} else {
			env.Miner.ActEmptyBlock(t)
		}
		numL1Blocks++
		l1Head := env.Miner.UnsafeNum()
		ss = env.Sequencer.SyncStatus()
		t.Log(
			"unsafeL2.Number", ss.UnsafeL2.Number,
			"safeL2.Number", ss.SafeL2.Number,
			"currentL1", ss.CurrentL1.Number,
			"l1_origin_unsafe", ss.UnsafeL2.L1Origin.Number,
			"l1_origin_safe", ss.SafeL2.L1Origin.Number,
			"l1_head", l1Head)
		lag = ss.CurrentL1.Number - ss.SafeL2.L1Origin.Number
		t.Log("lag", lag) // This lag starts out equal to the sequencing window, and eventually decreases to a lower value
		if lag <= 1 {     // We can't expect to get a lag of 0, so a lag of 1 is the best we can hope for.
			break
		}
	}

	if uint64(numL1Blocks) > timeout {
		t.Fatal("L1 Origin did not catch up to tip within % L1 blocks", numL1Blocks)
	} else {
		t.Logf("L1 Origin caught up to within %d blocks of the tip within %d L1 blocks (sequencing window size %d)", lag, numL1Blocks, tp.SequencerWindowSize)
	}

}

// Runs a that proves a block in a chain where the batcher opens a channel, the sequence window expires, and then the
// batcher attempts to close the channel afterwards.
func runSequenceWindowExpire_ChannelCloseAfterWindowExpiry_Test(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	tp := helpers.NewTestParams()
	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, helpers.NewBatcherCfg())

	// Mine 2 empty blocks on L2.
	for i := 0; i < 2; i++ {
		env.Sequencer.ActL2StartBlock(t)
		env.Alice.L2.ActResetTxOpts(t)
		env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)
		env.Alice.L2.ActMakeTx(t)
		env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
		env.Sequencer.ActL2EndBlock(t)
	}

	// Open the channel on L1.
	env.Batcher.ActL2BatchBuffer(t)
	env.Batcher.ActL2BatchSubmit(t)
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)

	// Finalize the block with the first channel frame on L1.
	env.Miner.ActL1SafeNext(t)
	env.Miner.ActL1FinalizeNext(t)

	// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	// Ensure the safe head is still 0.
	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()
	require.EqualValues(t, 0, l2SafeHead.Number.Uint64())

	// Cache the next frame data before expiring the sequence window, but don't submit it yet.
	env.Batcher.ActL2BatchBuffer(t)
	env.Batcher.ActL2ChannelClose(t)
	finalFrame := env.Batcher.ReadNextOutputFrame(t)

	// Expire the sequence window by building `SequenceWindow + 1` empty blocks on L1.
	for i := 0; i < int(tp.SequencerWindowSize)+1; i++ {
		env.Alice.L1.ActResetTxOpts(t)
		env.Alice.ActDeposit(t)

		env.Miner.ActL1StartBlock(12)(t)
		env.Miner.ActL1IncludeTx(env.Alice.Address())(t)
		env.Miner.ActL1EndBlock(t)

		env.Miner.ActL1SafeNext(t)
		env.Miner.ActL1FinalizeNext(t)
	}

	// Instruct the batcher to closethe channel on L1, after the sequence window + channel timeout has elapsed.
	env.Batcher.ActL2BatchSubmitRaw(t, finalFrame)
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)

	// Finalize the block with the second channel frame on L1.
	env.Miner.ActL1SafeNext(t)
	env.Miner.ActL1FinalizeNext(t)

	// Ensure the safe head is still 0.
	l2SafeHead = env.Engine.L2Chain().CurrentSafeBlock()
	require.EqualValues(t, 0, l2SafeHead.Number.Uint64())

	// Ask the sequencer to derive the deposit-only L2 chain.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	// Ensure the safe head advanced forcefully.
	l2SafeHead = env.Engine.L2Chain().CurrentSafeBlock()
	require.Greater(t, l2SafeHead.Number.Uint64(), uint64(0))

	// Run the FPP on one of the auto-derived blocks.
	env.RunFaultProofProgram(t, l2SafeHead.Number.Uint64()/2, testCfg.CheckResult, testCfg.InputParams...)
}

func Test_ProgramAction_SequenceWindowExpired(gt *testing.T) {
	matrix := helpers.NewMatrix[testCase]()
	defer matrix.Run(gt)

	forks := helpers.ForkMatrix{helpers.LatestFork}
	// TODO add recoverMode=true test case
	for _, testCase := range []testCase{
		{recoverMode: false, spanBatches: false},
		{recoverMode: false, spanBatches: true},
	} {
		matrix.AddTestCase(
			"HonestClaim-recoverMode-"+strconv.FormatBool(testCase.recoverMode)+"-spanBatches-"+strconv.FormatBool(testCase.spanBatches),
			testCase,
			forks,
			runSequenceWindowExpireTest,
			helpers.ExpectNoError(),
		)
	}

	// matrix.AddTestCase(
	// 	"JunkClaim",
	// 	nil,
	// 	forks,
	// 	runSequenceWindowExpireTest,
	// 	helpers.ExpectError(claim.ErrClaimNotValid),
	// 	helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
	// )
	// matrix.AddTestCase(
	// 	"ChannelCloseAfterWindowExpiry-HonestClaim",
	// 	nil,
	// 	forks,
	// 	runSequenceWindowExpire_ChannelCloseAfterWindowExpiry_Test,
	// 	helpers.ExpectNoError(),
	// )
	// matrix.AddTestCase(
	// 	"ChannelCloseAfterWindowExpiry-JunkClaim",
	// 	nil,
	// 	forks,
	// 	runSequenceWindowExpire_ChannelCloseAfterWindowExpiry_Test,
	// 	helpers.ExpectError(claim.ErrClaimNotValid),
	// 	helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
	// )
}
