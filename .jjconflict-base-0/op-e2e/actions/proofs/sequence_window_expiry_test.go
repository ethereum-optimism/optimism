package proofs

import (
	"testing"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Run a test that proves a deposit-only block generated due to sequence window expiry,
// and then recovers the chain using sequencer recover mode.
func runSequenceWindowExpireTest(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	const SEQUENCER_WINDOW_SIZE = 50 // (short, to keep test fast)
	tp := helpers.NewTestParams(func(p *e2eutils.TestParams) {
		p.SequencerWindowSize = SEQUENCER_WINDOW_SIZE
		p.MaxSequencerDrift = 1800 // use 1800 seconds (30 minutes), which is the protocol constant since Fjord
	})

	// It seems more difficult (almost impossible) to recover from sequencing window expiry with span batches,
	// since the singular batches within are invalidated _atomically_.
	// That is to say, if the oldest batch in the span batch fails the sequencing window check
	// (l1 origin + seq window < l1 inclusion)
	// All following batches are invalidated / dropped as well.
	// https://github.com/ethereum-optimism/optimism/blob/73339162d78a1ebf2daadab01736382eed6f4527/op-node/rollup/derive/batches.go#L96-L100
	//
	// If the same blocks were batched with singular batches, the validation rules are different
	// https://github.com/ethereum-optimism/optimism/blob/73339162d78a1ebf2daadab01736382eed6f4527/op-node/rollup/derive/batches.go#L83-L86
	// In the case of recover mode, the noTxPool=true condition means autoderviation actually fills
	// the gap with identical blocks anyway, meaning the following batches are actually still valid.
	bc := helpers.NewBatcherCfg()
	bc.ForceSubmitSingularBatch = true

	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, bc)

	// Mine an empty L1 block for gas estimation purposes.
	env.Miner.ActEmptyBlock(t)

	// Expire the sequence window by building `SequenceWindow + 1` empty blocks on L1.
	for i := 0; i < int(tp.SequencerWindowSize)+1; i++ {
		env.Alice.L1.ActResetTxOpts(t)
		env.Alice.ActDeposit(t)

		env.Miner.ActL1StartBlock(tp.L1BlockTime)(t)
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
	require.Greater(t, l2SafeHead.Number.Uint64(), uint64(0),
		"The safe head failed to progress after the sequencing window expired (expected deposit-only blocks to be derived).")

	env.RunFaultProofProgram(t, l2SafeHead.Number.Uint64()/2, testCfg.CheckResult, testCfg.InputParams...)

	// Set recover mode on the sequencer:
	env.Sequencer.ActSetRecoverMode(t, true)
	// Since recover mode only affects the L2 CL (op-node),
	// it won't stop the test environment injecting transactions
	// directly into the engine. So we will force the engine
	// to ignore such injections if recover mode is enabled.
	env.Engine.EngineApi.SetForceEmpty(true)

	// Define "lag" as the difference between the current L1 block number and the safe L2 block's L1 origin number.
	computeLag := func() int {
		ss := env.Sequencer.SyncStatus()
		return int(ss.CurrentL1.Number - ss.SafeL2.L1Origin.Number)
	}

	// Define "drift" as the difference between the current L2 block's timestamp and the unsafe L2 block's L1 origin's timestamp.
	computeDrift := func() int {
		ss := env.Sequencer.SyncStatus()
		l2header, err := env.Engine.EthClient().HeaderByHash(t.Ctx(), ss.UnsafeL2.Hash)
		require.NoError(t, err)
		l1header, err := env.Miner.EthClient().HeaderByHash(t.Ctx(), ss.UnsafeL2.L1Origin.Hash)
		require.NoError(t, err)
		t.Log("l2header.Time", l2header.Time)
		t.Log("l1header.Time", l1header.Time)
		return int(l2header.Time) - int(l1header.Time)
	}

	// Build both chains and assert the L1 origin catches back up with the tip of the L1 chain.
	lag := computeLag()
	t.Log("lag", lag)
	drift := computeDrift()
	t.Log("drift", drift)
	require.GreaterOrEqual(t, uint64(lag), tp.SequencerWindowSize, "Lag is less than sequencing window size")
	numL1Blocks := 0
	timeout := tp.SequencerWindowSize * 50

	for numL1Blocks < int(timeout) {
		for range 100 * tp.L1BlockTime / env.Sd.RollupCfg.BlockTime { // go at 100x real time
			err := env.Sequencer.ActMaybeL2StartBlock(t)
			if err != nil {
				break
			}
			env.Bob.L2.ActResetTxOpts(t)
			env.Bob.L2.ActMakeTx(t)
			env.Engine.ActL2IncludeTx(env.Bob.Address())(t)
			// RecoverMode (enabled above) should prevent this
			// transaction from being included in the block, which
			// is critical for recover mode to work.
			env.Sequencer.ActL2EndBlock(t)
			drift = computeDrift()
			t.Log("drift", drift)
		}
		env.BatchMineAndSync(t) // Mines 1 block on L1
		numL1Blocks++
		lag = computeLag()
		t.Log("lag", lag)
		drift = computeDrift()
		t.Log("drift", drift)
		if lag == 1 { // A lag of 1 is the minimum possible.
			break
		}
	}

	if uint64(numL1Blocks) >= timeout {
		t.Fatal("L1 Origin did not catch up to tip within %d L1 blocks (lag is %d)", numL1Blocks, lag)
	} else {
		t.Logf("L1 Origin caught up to within %d blocks of the tip within %d L1 blocks (sequencing window size %d)",
			lag, numL1Blocks, tp.SequencerWindowSize)
	}

	switch {
	case drift == 0:
		t.Fatal("drift is zero, this implies the unsafe l2 head is pinned to the l1 head")
	case drift > int(tp.MaxSequencerDrift):
		t.Fatal("drift is too high")
	default:
		t.Log("drift", drift)
	}

	// Disable recover mode so we can get some user transactions in again.
	env.Sequencer.ActSetRecoverMode(t, false)
	env.Engine.EngineApi.SetForceEmpty(false)
	l2SafeBefore := env.Sequencer.L2Safe()
	env.Sequencer.ActL2StartBlock(t)
	env.Bob.L2.ActResetTxOpts(t)
	env.Bob.L2.ActMakeTx(t)
	env.Engine.ActL2IncludeTx(env.Bob.Address())(t)
	env.Sequencer.ActL2EndBlock(t)
	env.BatchMineAndSync(t)
	l2Safe := env.Sequencer.L2Safe()
	require.Equal(t, l2Safe.Number, l2SafeBefore.Number+1, "safe chain did not progress with user transactions")
	l2SafeBlock, err := env.Engine.EthClient().BlockByHash(t.Ctx(), l2Safe.Hash)
	require.NoError(t, err)
	// Assert safe block has at least two transactions
	require.GreaterOrEqual(t, len(l2SafeBlock.Transactions()), 2, "safe block did not have at least two transactions")

	env.RunFaultProofProgram(t, l2Safe.Number, testCfg.CheckResult, testCfg.InputParams...)
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
	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)

	forks := helpers.ForkMatrix{helpers.Granite, helpers.LatestFork}
	matrix.AddTestCase(
		"HonestClaim",
		nil,
		forks,
		runSequenceWindowExpireTest,
		helpers.ExpectNoError(),
	)
	matrix.AddTestCase(
		"JunkClaim",
		nil,
		forks,
		runSequenceWindowExpireTest,
		helpers.ExpectError(claim.ErrClaimNotValid),
		helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
	)
	matrix.AddTestCase(
		"ChannelCloseAfterWindowExpiry-HonestClaim",
		nil,
		forks,
		runSequenceWindowExpire_ChannelCloseAfterWindowExpiry_Test,
		helpers.ExpectNoError(),
	)
	matrix.AddTestCase(
		"ChannelCloseAfterWindowExpiry-JunkClaim",
		nil,
		forks,
		runSequenceWindowExpire_ChannelCloseAfterWindowExpiry_Test,
		helpers.ExpectError(claim.ErrClaimNotValid),
		helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
	)
}
