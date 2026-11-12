package proofs_history_restart

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

const (
	// l2BlockTimeSeconds keeps the runs short: the kill point is a block height, so halving the
	// block time halves the time spent producing blocks.
	l2BlockTimeSeconds = 1
	// memoryBlockBufferTarget is reth's --engine.memory-block-buffer-target. It is passed
	// explicitly, like the persistence threshold, so the gap the kill opens between the proofs
	// store and reth's datadir does not depend on reth's defaults.
	memoryBlockBufferTarget = 5
	// postRestartTrafficBlocks is how far past the head the chain is driven once the follower has
	// caught up, so the sweep also covers blocks the proofs store indexed after coming back.
	postRestartTrafficBlocks = 10
	// proofsStoreFlushBlocks is how many blocks op-reth's proofs store buffers before it flushes
	// them. op-reth has no flag for it.
	proofsStoreFlushBlocks = 5

	// followerImportAttempts bounds the wait for the follower to import the block the sequencer
	// was held at.
	followerImportAttempts = 10
	// killPointHoldAttempts bounds how often the chain is held before the held head fits the case's
	// kill point.
	killPointHoldAttempts = 3
	// rebuildAttempts bounds how long the restarted sequencer may take to rebuild the heights it
	// lost with different contents.
	rebuildAttempts = 30
	// resyncAttempts bounds how long the restarted follower may take to catch up with the
	// sequencer. A follower whose op-reth crash-loops never does.
	resyncAttempts = 60
	// storeCatchUpTimeout bounds the wait for a proofs store flush before the kill.
	storeCatchUpTimeout = 60 * time.Second
)

// killPoint says where, relative to the proofs store's flush cadence, the follower is killed.
type killPoint string

const (
	// killAfterStoreFlush kills the follower as soon as the store has flushed the target block and
	// one more workload transaction has landed, so the store's tip is near the chain head and its
	// lead over reth's datadir is the largest.
	killAfterStoreFlush killPoint = "after_flush"
	// killMidStoreBuffer kills the follower a couple of blocks after a flush, so the store's tip
	// trails the chain head and the store loses blocks of its own on top of reth's.
	killMidStoreBuffer killPoint = "mid_buffer"
)

// syncPath says how the restarted follower's CL brings its EL from reth's datadir tip back to
// the sequencer's head.
type syncPath string

const (
	// clSync has op-node insert the missing blocks itself, fetched from its peers.
	clSync syncPath = "cl_sync"
	// elSync runs op-node in EL sync mode with post-finalization EL sync, as op-node runs it
	// against op-reth, so it hands the gap to op-reth even though op-reth already holds a
	// finalized block past genesis.
	elSync syncPath = "el_sync"
	// elSyncAfterFinality runs op-node in EL sync mode with post-finalization EL sync turned off,
	// as op-node runs it against op-geth. The follower's first EL sync finalized a block past
	// genesis, so the restarted op-node finds it and goes straight to CL sync.
	elSyncAfterFinality syncPath = "el_sync_after_finality"
)

func (p syncPath) followerL2CLOpts(clKind sysgo.MixedL2CLKind) []sysgo.L2CLOption {
	switch p {
	case elSync:
		opts := []sysgo.L2CLOption{sysgo.L2CLVerifierSyncMode(nodeSync.ELSync)}
		// kona-node always EL-syncs the gap and rejects op-node's switch for it.
		if clKind == sysgo.MixedL2CLOpNode {
			opts = append(opts, sysgo.L2CLPostFinalizationELSync(true))
		}
		return opts
	case elSyncAfterFinality:
		return []sysgo.L2CLOption{sysgo.L2CLVerifierSyncMode(nodeSync.ELSync), sysgo.L2CLPostFinalizationELSync(false)}
	default:
		return []sysgo.L2CLOption{sysgo.L2CLVerifierSyncMode(nodeSync.CLSync)}
	}
}

// restartCase is one cell of the matrix the follower runs under.
type restartCase struct {
	killPoint            killPoint
	persistenceThreshold uint64
	// storeFlushPastSetup is how far past the workload's setup block the proofs store must have
	// flushed before the kill. It exceeds persistenceThreshold+1, the most reth's persisted tip
	// trails its head by, so reth has persisted the setup block by the kill.
	storeFlushPastSetup  uint64
	syncPath             syncPath
	verificationInterval uint64
}

func (c restartCase) String() string {
	proofsPath := "index"
	if c.verificationInterval != 0 {
		proofsPath = "execute"
	}
	return fmt.Sprintf("threshold_%d-%s-%s-kill_%s", c.persistenceThreshold, proofsPath, c.syncPath, c.killPoint)
}

// engineArgs pins reth's persistence schedule, which decides how many blocks an unclean stop
// loses. Both nodes get them so the sequencer loses blocks on the same schedule as the follower.
func (c restartCase) engineArgs() []string {
	return []string{
		fmt.Sprintf("--engine.memory-block-buffer-target=%d", memoryBlockBufferTarget),
		fmt.Sprintf("--engine.persistence-threshold=%d", c.persistenceThreshold),
	}
}

func (c restartCase) followerArgs() []string {
	return append(c.engineArgs(), fmt.Sprintf("--proofs-history.verification-interval=%d", c.verificationInterval))
}

// killSchedule pairs the kill point with the persistence threshold rather than crossing them: the
// small threshold gets the mid-buffer kill, where both reth and the store lose blocks, the large
// one the after-flush kill, where the store's lead over reth is largest. Reth persists only up to
// memoryBlockBufferTarget blocks below its head, so its datadir trails the store at the kill: at
// threshold 7 by a few blocks, at 50 by up to 50 blocks it holds only in memory.
type killSchedule struct {
	killPoint            killPoint
	persistenceThreshold uint64
	storeFlushPastSetup  uint64
}

var (
	afterStoreFlushSchedule = killSchedule{killPoint: killAfterStoreFlush, persistenceThreshold: 50, storeFlushPastSetup: 55}
	midStoreBufferSchedule  = killSchedule{killPoint: killMidStoreBuffer, persistenceThreshold: 7, storeFlushPastSetup: 17}
)

// restartMatrix crosses the sync paths with the index proofs path only, and runs the execute
// proofs path once, under EL sync, the one sync path that runs on every CL kind and the one that
// hands the gap to reth's own backfill rather than inserting it block by block.
func restartMatrix(schedule killSchedule) []restartCase {
	cell := func(path syncPath, verificationInterval uint64) restartCase {
		return restartCase{
			killPoint:            schedule.killPoint,
			persistenceThreshold: schedule.persistenceThreshold,
			storeFlushPastSetup:  schedule.storeFlushPastSetup,
			syncPath:             path,
			verificationInterval: verificationInterval,
		}
	}
	return []restartCase{
		cell(clSync, 0),
		cell(elSync, 0),
		cell(elSyncAfterFinality, 0),
		cell(elSync, 1),
	}
}

// skipUnsupportedCell skips the cells the configured clients cannot run.
func skipUnsupportedCell(t devtest.T, c restartCase) {
	sysgo.SkipOnOpGeth(t, "proofs history is an op-reth feature")
	switch c.syncPath {
	case clSync:
		sysgo.SkipOnKonaNode(t, "kona-node has no CL sync mode: it always lets the EL sync the gap")
	case elSyncAfterFinality:
		sysgo.SkipOnKonaNode(t, "kona-node EL-syncs the gap whatever it has finalized, so this cell would repeat el_sync")
	}
}

type restartSystem struct {
	Batcher       *dsl.L2Batcher
	Follower      *dsl.L2ELNode
	FollowerCL    *dsl.L2CLNode
	FollowerCLKey string
	FollowerEnode string
	FunderL2      *dsl.FunderEOA
	Logs          *testlog.CapturingHandler
	Sequencer     *dsl.L2ELNode
	SequencerCL   *dsl.L2CLNode
}

// logCapturingT hands the system a logger that also captures every record it passes on, so a
// cell can check which sync path the follower's op-node logged.
type logCapturingT struct {
	devtest.T
	logger log.Logger
}

func (t logCapturingT) Logger() log.Logger { return t.logger }

// newRestartSystem starts a sequencer without proofs history, which serves as the independent
// eth_getProof reference, and a follower running proofs history under the case's reth, ExEx and
// sync-path settings.
func newRestartSystem(t devtest.T, c restartCase) *restartSystem {
	logs, ok := testlog.WrapCaptureLogger(t.Logger().Handler()).(*testlog.CapturingHandler)
	t.Require().True(ok, "the log capture must wrap the test's handler")
	t = logCapturingT{T: t, logger: log.NewLogger(logs).WithContext(t.Ctx())}
	clKind := sysgo.ResolveMixedL2CLKind()
	frontends := presets.NewProofsHistoryFollowerSystem(t, func(cfg *presets.ProofsHistoryFollowerConfig) {
		cfg.DeployerOptions = []sysgo.DeployerOption{sysgo.WithUniformL2BlockTimes(l2BlockTimeSeconds)}
		cfg.FollowerL2CLOpts = c.syncPath.followerL2CLOpts(clKind)
		cfg.FollowerOpRethOpts = []sysgo.OpRethOption{sysgo.OpRethWithExtraArgs(c.followerArgs()...)}
		cfg.SequencerOpRethOpts = []sysgo.OpRethOption{sysgo.OpRethWithExtraArgs(c.engineArgs()...)}
	})
	return &restartSystem{
		Batcher:       frontends.L2Batcher,
		Follower:      frontends.Follower.EL,
		FollowerCL:    frontends.Follower.CL,
		FollowerCLKey: frontends.Follower.Spec.CLKey,
		FollowerEnode: frontends.Follower.EL.Enode(),
		FunderL2:      frontends.FunderL2,
		Logs:          logs,
		Sequencer:     frontends.Sequencer.EL,
		SequencerCL:   frontends.Sequencer.CL,
	}
}

// uncleanStop is what the follower held when it was killed: the chain head it had imported, the
// block that held the workload's last transaction, both on the chain of that moment, and the last
// block its proofs store had flushed.
type uncleanStop struct {
	head              eth.L2BlockRef
	lastWorkloadBlock eth.L2BlockRef
	storeTip          uint64
}

// killFollower drives the workload until the follower's proofs store has flushed the case's target
// block, holds the chain at the case's kill point right after one more workload transaction and
// kills the follower's op-reth there. The sequencer is left stopped and the follower's CL running.
func killFollower(t devtest.T, sys *restartSystem, w *dsl.BlockRecorderWorkload, c restartCase) uncleanStop {
	storeFlushTarget := w.SetupBlock() + c.storeFlushPastSetup
	w.DriveUntilBlock(storeFlushTarget)
	window := sys.Follower.WaitForProofsStoreBlock(storeFlushTarget, storeCatchUpTimeout)
	lastWorkloadTarget := *window.Latest
	if c.killPoint == killMidStoreBuffer {
		lastWorkloadTarget += 2
	}
	stop := holdChainAtKillPoint(t, sys, w, c, w.DriveUntilBlock(lastWorkloadTarget))
	t.Logger().Info("Killing the follower", "head", stop.head, "proofs_store_tip", stop.storeTip,
		"last_workload_block", stop.lastWorkloadBlock, "kill_point", c.killPoint)
	sys.Follower.StopUngracefully()
	return stop
}

// holdChainAtKillPoint holds the chain at a head that fits the case's kill point. Reth persists
// only up to memoryBlockBufferTarget blocks below its head, so a head fewer blocks than that past
// the workload's last transaction keeps that block out of reth's datadir; a mid-buffer kill also
// needs a head the proofs store has not flushed. While the held head misses either, the sequencer
// resumes for one more workload transaction and the chain is held again.
func holdChainAtKillPoint(t devtest.T, sys *restartSystem, w *dsl.BlockRecorderWorkload, c restartCase, lastWorkloadBlockNumber uint64) uncleanStop {
	stop, err := retry.Do(t.Ctx(), killPointHoldAttempts, retry.Fixed(0), func() (uncleanStop, error) {
		stop := uncleanStop{head: holdChain(t, sys)}
		stop.lastWorkloadBlock = sys.Follower.BlockRefByNumber(lastWorkloadBlockNumber)
		// Read last, right before the kill: op-reth flushes a buffer left idle for 10s, and a flush
		// of the held head turns a mid-buffer kill into an after-flush one.
		stop.storeTip = proofsStoreTip(t, sys.Follower)
		mismatch := killPointMismatch(c, stop)
		if mismatch == "" {
			return stop, nil
		}
		t.Logger().Info("Held head does not fit the kill point; resuming the sequencer for another workload transaction",
			"head", stop.head, "proofs_store_tip", stop.storeTip, "last_workload_block", stop.lastWorkloadBlock,
			"mismatch", mismatch)
		sys.SequencerCL.StartSequencer()
		lastWorkloadBlockNumber = w.DriveUntilBlock(stop.head.Number + 1)
		return uncleanStop{}, errors.New(mismatch)
	})
	t.Require().NoErrorf(err, "the chain never held at a head fitting the %s kill point", c.killPoint)
	return stop
}

// killPointMismatch says why the held chain does not fit the case's kill point, or returns "" when
// it does.
func killPointMismatch(c restartCase, stop uncleanStop) string {
	if blocksPastWorkload := stop.head.Number - stop.lastWorkloadBlock.Number; blocksPastWorkload >= memoryBlockBufferTarget {
		return fmt.Sprintf("the held head %d is %d blocks past the workload's last block %d, so reth may persist that block",
			stop.head.Number, blocksPastWorkload, stop.lastWorkloadBlock.Number)
	}
	if c.killPoint == killMidStoreBuffer && stop.storeTip >= stop.head.Number {
		return fmt.Sprintf("the proofs store flushed the held head %d (store tip %d)", stop.head.Number, stop.storeTip)
	}
	return ""
}

// holdChain stops block production, waits for the follower to import the last block and returns
// that head once the proofs store can no longer flush on block count: it flushes once
// proofsStoreFlushBlocks blocks are buffered, so a tip within that many blocks of the held head
// stays put until op-reth's 10s idle flush.
func holdChain(t devtest.T, sys *restartSystem) eth.L2BlockRef {
	heldHead := sys.Sequencer.BlockRefByHash(sys.SequencerCL.StopSequencer())
	sys.Follower.Reached(eth.Unsafe, heldHead.Number, followerImportAttempts)
	head := sys.Follower.BlockRefByLabel(eth.Unsafe)
	t.Require().Equalf(heldHead.ID(), head.ID(), "the follower must hold the sequencer's head %s, but holds %s", heldHead, head)
	sys.Follower.WaitForProofsStoreBlock(head.Number-(proofsStoreFlushBlocks-1), storeCatchUpTimeout)
	return head
}

func proofsStoreTip(t devtest.T, el *dsl.L2ELNode) uint64 {
	status, err := el.ProofsSyncStatus(t.Ctx())
	t.Require().NoErrorf(err, "debug_proofsSyncStatus on %s failed", el)
	t.Require().NotNilf(status.Latest, "the proofs store on %s has indexed no block", el)
	return *status.Latest
}

// restartFollower brings the follower's op-reth and CL back and re-peers them with the sequencer,
// since both peerings died with the processes. It returns the block reth resumes from, its datadir
// tip, which must be below the proofs store's tip for the test to exercise the crash gap at all.
// It clears the captured logs before the follower's CL starts, so the follower op-node records
// left in them are the restarted node's.
func restartFollower(t devtest.T, sys *restartSystem, stop uncleanStop) eth.L2BlockRef {
	// The follower keeps its node ID but listens on a new port, so the sequencer's trusted entry
	// would keep redialing the old one, and reach the restarted follower if the OS reuses that
	// port: a redial that crosses the follower's own dial makes reth tear down both sessions.
	sys.Sequencer.ForgetPeer(sys.FollowerEnode)
	sys.Follower.Start()
	sys.Follower.WaitForOnline()
	// Nothing feeds the EL until its CL is back, so its head is what reth had persisted.
	persistedTip := sys.Follower.BlockRefByLabel(eth.Unsafe)
	t.Logger().Info("Follower op-reth is back on its datadir", "persisted_tip", persistedTip,
		"proofs_store_tip", stop.storeTip, "head_at_kill", stop.head)
	t.Require().Lessf(persistedTip.Number, stop.storeTip,
		"the proofs store (tip %d) must be ahead of reth's datadir (tip %d) after the kill; the kill landed too late for this threshold",
		stop.storeTip, persistedTip.Number)

	sys.Follower.PeerWith(sys.Sequencer)
	sys.Logs.Clear()
	sys.FollowerCL.Start()
	sys.FollowerCL.ConnectPeer(sys.SequencerCL)
	return persistedTip
}

// verifyFollowerSyncPath requires the restarted op-node to have logged the sync path the cell
// claims: el_sync must finish an EL sync, el_sync_after_finality must skip it for CL sync. op-node
// logs either outcome no later than its first block past the head it caught up to, so once the
// follower has indexed the post-restart traffic the record is in the capture. kona-node is not
// checked: it always EL-syncs and logs neither.
func verifyFollowerSyncPath(t devtest.T, sys *restartSystem, c restartCase) {
	if sysgo.ResolveMixedL2CLKind() != sysgo.MixedL2CLOpNode {
		return
	}
	var expectedMessage string
	switch c.syncPath {
	case elSync:
		expectedMessage = "Finished EL sync"
	case elSyncAfterFinality:
		expectedMessage = "Skipping EL sync"
	default:
		return
	}
	record := sys.Logs.FindLog(
		testlog.NewAttributesFilter("component", "l2cl-"+sys.FollowerCLKey),
		testlog.NewMessageContainsFilter(expectedMessage),
	)
	t.Require().NotNilf(record, "the restarted follower op-node never logged %q, so it did not take the %s path", expectedMessage, c.syncPath)
}

// Each kill schedule is a top-level test of its own, so its four cells, which each start a devnet
// and run one after another, get a go test timeout to themselves.

func TestProofsHistoryFollowerRecoversFromUncleanShutdownAfterStoreFlush(gt *testing.T) {
	testFollowerRecoversFromUncleanShutdown(gt, afterStoreFlushSchedule)
}

func TestProofsHistoryFollowerRecoversFromUncleanShutdownMidStoreBuffer(gt *testing.T) {
	testFollowerRecoversFromUncleanShutdown(gt, midStoreBufferSchedule)
}

func TestProofsHistoryFollowerServesRebuiltBlocksAfterUncleanShutdownAfterStoreFlush(gt *testing.T) {
	testFollowerServesRebuiltBlocksAfterUncleanShutdown(gt, afterStoreFlushSchedule)
}

func TestProofsHistoryFollowerServesRebuiltBlocksAfterUncleanShutdownMidStoreBuffer(gt *testing.T) {
	testFollowerServesRebuiltBlocksAfterUncleanShutdown(gt, midStoreBufferSchedule)
}

// testFollowerRecoversFromUncleanShutdown kills a proofs-history follower once its store has
// flushed blocks reth has not yet persisted, restarts it into the same chain and requires it to
// catch up with the sequencer, keep indexing, and serve proofs that match the sequencer's for the
// blocks it lost, the blocks it re-imported and the blocks that followed. The follower catches up
// by CL sync, by EL sync, or by the CL sync an EL sync config without post-finalization EL sync
// falls back to once a block past genesis is finalized.
func testFollowerRecoversFromUncleanShutdown(gt *testing.T, schedule killSchedule) {
	t := devtest.ParallelT(gt)
	devtest.RunParallel(t, restartMatrix(schedule), func(t devtest.T, c restartCase) {
		skipUnsupportedCell(t, c)
		sys := newRestartSystem(t, c)
		w := dsl.NewBlockRecorderWorkload(t, sys.FunderL2)

		stop := killFollower(t, sys, w, c)
		sys.SequencerCL.StartSequencer()
		sys.FollowerCL.Stop()
		persistedTip := restartFollower(t, sys, stop)

		sys.Follower.InSync(sys.Sequencer, safety.LocalUnsafe, resyncAttempts)
		lastBlock := w.DriveUntilBlock(sys.Sequencer.BlockRefByLabel(eth.Unsafe).Number + postRestartTrafficBlocks)
		sys.Follower.VerifyHistoricalProofsMatch(sys.Sequencer, func(opts *dsl.HistoricalProofsOpts) {
			opts.Accounts = w.Accounts()
			opts.FirstBlock = persistedTip.Number
			opts.LastBlock = lastBlock
		})
		verifyFollowerSyncPath(t, sys, c)
	})
}

// testFollowerServesRebuiltBlocksAfterUncleanShutdown kills a proofs-history follower, then kills
// the sequencer's op-reth too while the blocks the follower's store holds are still only in the
// sequencer's memory, so the restarted sequencer rebuilds those heights with different contents.
// The restarted follower must follow the rebuilt chain and serve its proofs, not the ones its
// store recorded for the chain that no longer exists.
func testFollowerServesRebuiltBlocksAfterUncleanShutdown(gt *testing.T, schedule killSchedule) {
	t := devtest.ParallelT(gt)
	devtest.RunParallel(t, restartMatrix(schedule), func(t devtest.T, c restartCase) {
		skipUnsupportedCell(t, c)
		sys := newRestartSystem(t, c)
		// Blocks that reached L1 would be derived back as they were; blocks that never did
		// are re-sequenced, which is the rebuild this test needs.
		sys.Batcher.Stop()
		w := dsl.NewBlockRecorderWorkload(t, sys.FunderL2)

		// Both op-reths die back to back on the held chain, so the sequencer still holds the
		// blocks the follower's store flushed; the CLs follow once the processes are gone. The
		// restarted sequencer CL sequences again.
		stop := killFollower(t, sys, w, c)
		sys.Sequencer.StopUngracefully()
		sys.FollowerCL.Stop()
		sys.SequencerCL.Stop()

		sys.Sequencer.Start()
		sys.Sequencer.WaitForOnline()
		sequencerPersistedTip := sys.Sequencer.BlockRefByLabel(eth.Unsafe)
		t.Logger().Info("Sequencer op-reth is back on its datadir", "persisted_tip", sequencerPersistedTip,
			"follower_proofs_store_tip", stop.storeTip, "last_workload_block", stop.lastWorkloadBlock,
			"workload_setup_block", w.SetupBlock())
		t.Require().Lessf(sequencerPersistedTip.Number, stop.storeTip,
			"the sequencer must rebuild blocks the follower's proofs store holds, but its datadir tip %d is not below the store tip %d",
			sequencerPersistedTip.Number, stop.storeTip)
		// A lost block without a workload transaction can come back identical, so only losing the
		// workload's last block guarantees the rebuilt chain differs.
		t.Require().Lessf(sequencerPersistedTip.Number, stop.lastWorkloadBlock.Number,
			"the block %d that held the workload's last transaction must be among the sequencer's lost blocks, but its datadir tip is %d",
			stop.lastWorkloadBlock.Number, sequencerPersistedTip.Number)
		t.Require().LessOrEqualf(w.SetupBlock(), sequencerPersistedTip.Number,
			"the workload's funding and contract (block %d) must survive the sequencer's crash, but its datadir tip is %d",
			w.SetupBlock(), sequencerPersistedTip.Number)
		sys.SequencerCL.Start()
		sys.Sequencer.ReorgTriggered(stop.lastWorkloadBlock, rebuildAttempts)

		persistedTip := restartFollower(t, sys, stop)

		sys.Follower.InSync(sys.Sequencer, safety.LocalUnsafe, resyncAttempts)
		lastBlock := w.DriveUntilBlock(sys.Sequencer.BlockRefByLabel(eth.Unsafe).Number + postRestartTrafficBlocks)
		sys.Follower.VerifyHistoricalProofsMatch(sys.Sequencer, func(opts *dsl.HistoricalProofsOpts) {
			opts.Accounts = w.Accounts()
			// The follower's datadir can outlast the sequencer's, so start at whichever comes
			// first: its own tip or the first height the sequencer rebuilt.
			opts.FirstBlock = min(persistedTip.Number, sequencerPersistedTip.Number+1)
			opts.LastBlock = lastBlock
		})
		verifyFollowerSyncPath(t, sys, c)
	})
}
