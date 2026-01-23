package supernode

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
)

// rewindTestEnv holds the common test environment for rewind tests.
type rewindTestEnv struct {
	t         helpers.Testing
	sd        *e2eutils.SetupData
	miner     *helpers.L1Miner
	seqEngine *helpers.L2Engine
	sequencer *helpers.L2Sequencer
	batcher   *helpers.L2Batcher
	engCl     *sources.EngineClient
	ec        engine_controller.EngineController
}

// setupRewindTest creates a common test environment for rewind tests.
func setupRewindTest(gt *testing.T) *rewindTestEnv {
	t := helpers.NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, helpers.DefaultRollupTestParams())
	sd := e2eutils.Setup(t, dp, helpers.DefaultAlloc)
	logger := testlog.Logger(t, log.LevelInfo)

	miner, seqEngine, sequencer := helpers.SetupSequencerTest(t, sd, logger)
	rollupSeqCl := sequencer.RollupClient()
	batcher := helpers.NewL2Batcher(logger, sd.RollupCfg, helpers.DefaultBatcherCfg(dp),
		rollupSeqCl, miner.EthClient(), seqEngine.EthClient(), seqEngine.EngineClient(t, sd.RollupCfg))

	// Initialize the pipeline
	sequencer.ActL2PipelineFull(t)

	engCl := seqEngine.EngineClient(t, sd.RollupCfg)
	ec := engine_controller.NewEngineControllerWithL2AndRollup(engCl, sd.RollupCfg)

	return &rewindTestEnv{
		t:         t,
		sd:        sd,
		miner:     miner,
		seqEngine: seqEngine,
		sequencer: sequencer,
		batcher:   batcher,
		engCl:     engCl,
		ec:        ec,
	}
}

// buildUnsafeBlocks builds L2 blocks without submitting them to L1 (unsafe only).
func (env *rewindTestEnv) buildUnsafeBlocks(numL1Blocks int) {
	for i := 0; i < numL1Blocks; i++ {
		env.miner.ActEmptyBlock(env.t)
	}
	env.sequencer.ActL1HeadSignal(env.t)
	env.sequencer.ActBuildToL1Head(env.t)
}

// timestampForBlock returns the timestamp for a given L2 block number.
func (env *rewindTestEnv) timestampForBlock(blockNum uint64) uint64 {
	return env.sd.RollupCfg.Genesis.L2Time + (env.sd.RollupCfg.BlockTime * blockNum)
}

// batchSubmitAndSync submits L2 blocks to L1 and syncs the sequencer to advance safe head.
func (env *rewindTestEnv) batchSubmitAndSync() {
	env.batcher.ActSubmitAll(env.t)
	env.miner.ActL1StartBlock(12)(env.t)
	env.miner.ActL1IncludeTx(env.sd.RollupCfg.Genesis.SystemConfig.BatcherAddr)(env.t)
	env.miner.ActL1EndBlock(env.t)
	env.sequencer.ActL1HeadSignal(env.t)
	env.sequencer.ActL2PipelineFull(env.t)
}

// getHeads returns the current unsafe, safe, and finalized heads from the engine.
func (env *rewindTestEnv) getHeads() (unsafe, safe, finalized eth.L2BlockRef) {
	ctx := context.Background()
	var err error
	unsafe, err = env.engCl.L2BlockRefByLabel(ctx, eth.Unsafe)
	require.NoError(env.t, err)
	safe, err = env.engCl.L2BlockRefByLabel(ctx, eth.Safe)
	require.NoError(env.t, err)
	finalized, err = env.engCl.L2BlockRefByLabel(ctx, eth.Finalized)
	require.NoError(env.t, err)
	return
}

// rewindToBlock rewinds the engine to the given block number and returns the resulting heads.
func (env *rewindTestEnv) rewindToBlock(blockNum uint64) (unsafe, safe, finalized eth.L2BlockRef) {
	err := env.ec.RewindToTimestamp(context.Background(), env.timestampForBlock(blockNum))
	require.NoError(env.t, err)
	return env.getHeads()
}

func TestRewindToTimestamp(gt *testing.T) {
	gt.Run("RewindUnsafeOnly", RewindUnsafeOnly)
	gt.Run("RewindSafeHeadClamped", RewindSafeHeadClamped)
	gt.Run("RewindSafeHeadBackward", RewindSafeHeadBackward)
	gt.Run("RewindFinalizedHeadBackward", RewindFinalizedHeadBackward)
}

// RewindUnsafeOnly tests rewinding when only the unsafe head has advanced.
// Safe and finalized remain at genesis.
func RewindUnsafeOnly(gt *testing.T) {
	env := setupRewindTest(gt)

	// Build unsafe blocks only (no batch submission)
	env.buildUnsafeBlocks(5)

	// Verify initial state: unsafe > safe = finalized = genesis
	unsafeBefore, safeBefore, finalizedBefore := env.getHeads()
	require.Greater(gt, unsafeBefore.Number, uint64(2), "unsafe should have advanced past genesis")
	require.Equal(gt, uint64(0), safeBefore.Number, "safe should be at genesis")
	require.Equal(gt, uint64(0), finalizedBefore.Number, "finalized should be at genesis")

	// Rewind to middle of chain (avoiding genesis edge)
	rewindTarget := unsafeBefore.Number / 2
	require.Greater(gt, rewindTarget, uint64(0), "rewind target should be past genesis")
	unsafeAfter, safeAfter, finalizedAfter := env.rewindToBlock(rewindTarget)

	// Verify: unsafe rewound to target, safe/finalized unchanged
	require.Equal(gt, rewindTarget, unsafeAfter.Number, "unsafe should be at rewind target")
	require.Less(gt, unsafeAfter.Number, unsafeBefore.Number, "unsafe should have decreased")
	require.Equal(gt, safeBefore.Number, safeAfter.Number, "safe should not change")
	require.Equal(gt, finalizedBefore.Number, finalizedAfter.Number, "finalized should not change")
}

// RewindSafeHeadClamped tests rewinding to a target ahead of the current safe head.
// The safe head should be clamped (not moved forward).
func RewindSafeHeadClamped(gt *testing.T) {
	env := setupRewindTest(gt)

	// Build and submit blocks to advance safe head
	env.buildUnsafeBlocks(2)
	env.batchSubmitAndSync()

	// Build more unsafe blocks (not batched)
	env.buildUnsafeBlocks(4)

	// Verify initial state: unsafe > safe > genesis
	unsafeBefore, safeBefore, finalizedBefore := env.getHeads()
	require.Greater(gt, unsafeBefore.Number, safeBefore.Number, "unsafe should be ahead of safe")
	require.Greater(gt, safeBefore.Number, uint64(0), "safe should have advanced from genesis")

	// Rewind to a block ahead of current safe head (so safe should be clamped)
	rewindTarget := safeBefore.Number + 2
	require.Greater(gt, rewindTarget, safeBefore.Number, "rewind target should be ahead of safe")
	require.Less(gt, rewindTarget, unsafeBefore.Number, "rewind target should be behind unsafe")
	unsafeAfter, safeAfter, finalizedAfter := env.rewindToBlock(rewindTarget)

	// Verify: unsafe rewound to target, safe clamped (unchanged), finalized unchanged
	require.Equal(gt, rewindTarget, unsafeAfter.Number, "unsafe should be at rewind target")
	require.Equal(gt, safeBefore.Number, safeAfter.Number, "safe should be clamped (unchanged)")
	require.Equal(gt, finalizedBefore.Number, finalizedAfter.Number, "finalized should not change")
}

// RewindSafeHeadBackward tests rewinding to a target behind the current safe head.
// The safe head should actually move backward.
func RewindSafeHeadBackward(gt *testing.T) {
	env := setupRewindTest(gt)

	// Build and submit blocks to advance safe head well past genesis
	env.buildUnsafeBlocks(4)
	env.batchSubmitAndSync()

	// Build more unsafe blocks
	env.buildUnsafeBlocks(2)

	// Verify initial state: unsafe > safe > genesis
	unsafeBefore, safeBefore, finalizedBefore := env.getHeads()
	require.Greater(gt, unsafeBefore.Number, safeBefore.Number, "unsafe should be ahead of safe")
	require.Greater(gt, safeBefore.Number, uint64(2), "safe should have advanced past genesis")

	// Rewind to a block before current safe head
	rewindTarget := safeBefore.Number / 2
	require.Greater(gt, rewindTarget, uint64(0), "rewind target should be past genesis")
	require.Less(gt, rewindTarget, safeBefore.Number, "rewind target should be behind safe")
	unsafeAfter, safeAfter, finalizedAfter := env.rewindToBlock(rewindTarget)

	// Verify: unsafe and safe moved backward to target, finalized unchanged
	require.Equal(gt, rewindTarget, unsafeAfter.Number, "unsafe should be at rewind target")
	require.Equal(gt, rewindTarget, safeAfter.Number, "safe should have moved backward to target")
	require.Less(gt, safeAfter.Number, safeBefore.Number, "safe should have decreased")
	require.Equal(gt, finalizedBefore.Number, finalizedAfter.Number, "finalized should not change")
}

// RewindFinalizedHeadBackward tests rewinding to a target behind the current finalized head.
// All heads (unsafe, safe, finalized) should move backward.
func RewindFinalizedHeadBackward(gt *testing.T) {
	env := setupRewindTest(gt)

	// Step 1: Build L1 block #1 and mark it as safe
	env.miner.ActEmptyBlock(env.t)
	env.miner.ActL1SafeNext(env.t) // #0 -> #1

	// Build L2 blocks referencing L1 #1
	env.sequencer.ActL1HeadSignal(env.t)
	env.sequencer.ActBuildToL1Head(env.t)
	env.sequencer.ActL2PipelineFull(env.t)
	env.sequencer.ActL1SafeSignal(env.t)

	// Step 2: Build L1 block #2, mark #2 as safe and #1 as finalized
	env.miner.ActEmptyBlock(env.t)
	env.miner.ActL1SafeNext(env.t)     // #1 -> #2
	env.miner.ActL1FinalizeNext(env.t) // #0 -> #1
	env.sequencer.ActL1HeadSignal(env.t)
	env.sequencer.ActBuildToL1Head(env.t)
	env.sequencer.ActL2PipelineFull(env.t)
	env.sequencer.ActL1FinalizedSignal(env.t)
	env.sequencer.ActL1SafeSignal(env.t)
	env.sequencer.ActL2PipelineFull(env.t)

	// Step 3: Submit batch containing current L2 blocks (will be in L1 block #3)
	env.batcher.ActSubmitAll(env.t)
	env.miner.ActL1StartBlock(12)(env.t)
	env.miner.ActL1IncludeTx(env.sd.RollupCfg.Genesis.SystemConfig.BatcherAddr)(env.t)
	env.miner.ActL1EndBlock(env.t)
	env.sequencer.ActL1HeadSignal(env.t)
	env.sequencer.ActL2PipelineFull(env.t)

	// Step 4: Build more L2 blocks and submit another batch (L1 block #4)
	env.sequencer.ActBuildToL1Head(env.t)
	env.batcher.ActSubmitAll(env.t)
	env.miner.ActL1StartBlock(12)(env.t)
	env.miner.ActL1IncludeTx(env.sd.RollupCfg.Genesis.SystemConfig.BatcherAddr)(env.t)
	env.miner.ActL1EndBlock(env.t)

	// Step 5: Add more L1 blocks #5, #6
	env.miner.ActEmptyBlock(env.t)
	env.miner.ActEmptyBlock(env.t)

	// Build more L2 blocks
	env.sequencer.ActL1HeadSignal(env.t)
	env.sequencer.ActBuildToL1Head(env.t)

	// Step 6: Finalize the L1 block containing the first batch (#3)
	env.miner.ActL1SafeNext(env.t)     // #2 -> #3
	env.miner.ActL1SafeNext(env.t)     // #3 -> #4
	env.miner.ActL1FinalizeNext(env.t) // #1 -> #2
	env.miner.ActL1FinalizeNext(env.t) // #2 -> #3

	env.sequencer.ActL2PipelineFull(env.t)
	env.sequencer.ActL1FinalizedSignal(env.t)
	env.sequencer.ActL1SafeSignal(env.t)
	env.sequencer.ActL1HeadSignal(env.t)
	env.sequencer.ActL2PipelineFull(env.t)

	// Verify initial state: unsafe > safe >= finalized > genesis
	unsafeBefore, safeBefore, finalizedBefore := env.getHeads()
	require.Greater(gt, unsafeBefore.Number, safeBefore.Number, "unsafe should be ahead of safe")
	require.GreaterOrEqual(gt, safeBefore.Number, finalizedBefore.Number, "safe should be >= finalized")
	require.Greater(gt, finalizedBefore.Number, uint64(3), "finalized should have advanced well past genesis")

	// Rewind to a block before current finalized head
	rewindTarget := finalizedBefore.Number / 2
	require.Greater(gt, rewindTarget, uint64(0), "rewind target should be past genesis")
	require.Less(gt, rewindTarget, finalizedBefore.Number, "rewind target should be behind finalized")
	unsafeAfter, safeAfter, finalizedAfter := env.rewindToBlock(rewindTarget)

	// Verify: all heads moved backward to target
	require.Equal(gt, rewindTarget, unsafeAfter.Number, "unsafe should be at rewind target")
	require.Equal(gt, rewindTarget, safeAfter.Number, "safe should have moved backward to target")
	require.Equal(gt, rewindTarget, finalizedAfter.Number, "finalized should have moved backward to target")
	require.Less(gt, safeAfter.Number, safeBefore.Number, "safe should have decreased")
	require.Less(gt, finalizedAfter.Number, finalizedBefore.Number, "finalized should have decreased")
}
