package op_reth_sequencer_crash

import (
	"fmt"
	"testing"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

const (
	// batcherPollInterval is how often the batcher loads the sequencer's new blocks. Below the
	// block time, so the batcher picks up the halted sequencer's head within a poll.
	batcherPollInterval = 250 * time.Millisecond
	// haltRounds bounds how many times the sequencer is halted, each halt that lands the memory
	// block buffer target or more blocks past the workload's last transaction being followed by one
	// more workload round.
	haltRounds = 5
	// inclusionPollAttempts bounds how long the workload waits for its contract write, in
	// inclusionPollInterval polls.
	inclusionPollAttempts = 300
	// inclusionPollInterval is how often the workload polls for its contract write's receipt. A
	// fixed poll far below the block time lets the sequencer halt within about a block of the
	// workload's last transaction; the default exponential backoff can let it build several more.
	inclusionPollInterval = 100 * time.Millisecond
	// l2BlockTimeSeconds keeps the runs short: the kill point is a block height, so halving the
	// block time halves the time spent producing blocks.
	l2BlockTimeSeconds = 1
	// nextSaveMargin is how far below reth's next save the kill target sits: the workload's last
	// round may land a few blocks past the target, and the halt up to one block short of the
	// memory block buffer target past that. A kill past the next save still holds every assertion; it only loses fewer
	// blocks.
	nextSaveMargin = 7
	// postCrashTrafficBlocks is how far the chain is driven once the sequencer is back, so the
	// proofs sweep also covers blocks built after the crash.
	postCrashTrafficBlocks = 10
	// proofsSweepLeadIn is how many blocks below the persisted tip the proofs sweep starts, so it
	// covers blocks the crash left in place as well as every block it lost.
	proofsSweepLeadIn = 5

	// The attempt counts below bound waits in 2s polls.

	// batcherLandingAttempts bounds how long the halted sequencer's blocks may take to be posted
	// to L1 and derived by the followers.
	batcherLandingAttempts = 30
	// followerReorgAttempts bounds how long a follower may take to reorg to the rebuilt chain. A
	// follower running op-node drops a gossiped block at or below its unsafe head, so it only
	// reorgs once the batcher has posted the rebuilt chain and derivation reaches it.
	followerReorgAttempts = 90
	// resumeAttempts bounds how long the restarted sequencer may take to sequence again: to
	// replace the block that held the workload's last transaction, or to build past the head the
	// kill cut short, which includes deriving back from L1 whatever the batcher had posted.
	resumeAttempts = 60
	// resyncAttempts bounds how long the followers may take to catch up with the sequencer.
	resyncAttempts = 60
	// safeAttempts bounds how long the batcher and derivation may take to make a block safe: the
	// posted blocks on the restarted sequencer, and the chain past the head the kill cut short on
	// every node.
	safeAttempts = 90
)

const (
	gethFollowerKey   = "geth-follower"
	proofsFollowerKey = "proofs-follower"
	sequencerKey      = "sequencer"
)

// crashCase is one cell of the matrix the sequencer is killed under.
type crashCase struct {
	persistenceThreshold uint64
}

func (c crashCase) String() string {
	return fmt.Sprintf("threshold_%d", c.persistenceThreshold)
}

// persistenceSchedule is reth's persistence schedule, which decides how many blocks an unclean
// stop loses. Both op-reth nodes get it, so the follower absorbs the reorg under the same schedule.
func (c crashCase) persistenceSchedule() sysgo.OpRethPersistenceSchedule {
	return sysgo.OpRethPersistenceSchedule{
		MemoryBlockBufferTarget:          sysgo.OpRethPinnedMemoryBlockBufferTarget,
		PersistenceBackpressureThreshold: c.persistenceBackpressureThreshold(),
		PersistenceThreshold:             c.persistenceThreshold,
	}
}

// persistenceBackpressureThreshold is the smallest value reth accepts, one above the persistence
// threshold. While a save is in flight, reth stops taking engine API calls once this many blocks
// beyond the memory buffer await persistence, so a slow save stalls the sequencer instead of
// letting unpersisted blocks pile up.
func (c crashCase) persistenceBackpressureThreshold() uint64 {
	return c.persistenceThreshold + 1
}

// maxLostBlocks is the most blocks a kill may lose. Reth starts a save once more than the
// persistence threshold of blocks sit in memory, and while the save runs, backpressure caps the
// blocks in memory at the memory buffer plus the backpressure threshold, however slow the save is.
func (c crashCase) maxLostBlocks() uint64 {
	return sysgo.OpRethPinnedMemoryBlockBufferTarget + c.persistenceBackpressureThreshold()
}

// killTargetBlock is the height the workload drives the chain to before the kill. Reth starts its
// first save at block persistenceThreshold+1, then one every persistence interval. The target lies
// more than the memory block buffer target past the first save that persists setupBlock, since
// backpressure holds the head within that many blocks of a save until the save completes, and
// nextSaveMargin below the save after it, so the kill loses as many blocks as the schedule allows.
func (c crashCase) killTargetBlock(setupBlock uint64) uint64 {
	schedule := c.persistenceSchedule()
	saveHeight := schedule.PersistenceThreshold + 1
	for saveHeight < setupBlock+schedule.MemoryBlockBufferTarget {
		saveHeight += schedule.PersistenceInterval()
	}
	return max(saveHeight+schedule.MemoryBlockBufferTarget+1, saveHeight+schedule.PersistenceInterval()-nextSaveMargin)
}

// crashMatrix kills the sequencer under a threshold that saves every three blocks, so the datadir
// trails the head by 5 to 7 blocks wherever the kill lands, and one that saves every 46 blocks, so
// a kill just before a save loses over 40.
func crashMatrix() []crashCase {
	return []crashCase{
		{persistenceThreshold: 7},
		{persistenceThreshold: 50},
	}
}

type crashSystem struct {
	Batcher        *dsl.L2Batcher
	FunderL2       *dsl.FunderEOA
	GethFollower   presets.MixedSingleChainNodeFrontends
	ProofsFollower presets.MixedSingleChainNodeFrontends
	Sequencer      presets.MixedSingleChainNodeFrontends
	// SequencerEnode identifies the sequencer's op-reth, which keeps its node ID across the kill.
	SequencerEnode string
}

func (sys *crashSystem) followers() []presets.MixedSingleChainNodeFrontends {
	return []presets.MixedSingleChainNodeFrontends{sys.ProofsFollower, sys.GethFollower}
}

func (sys *crashSystem) nodes() []presets.MixedSingleChainNodeFrontends {
	return append([]presets.MixedSingleChainNodeFrontends{sys.Sequencer}, sys.followers()...)
}

// newCrashSystem starts an op-reth sequencer without proofs history, an op-reth follower running
// proofs history and an op-geth follower, which serves eth_getProof from its own archive state and
// is the independent reference the proofs follower is checked against. Both op-reth nodes run the
// case's persistence schedule.
func newCrashSystem(t devtest.T, c crashCase) *crashSystem {
	clKind := sysgo.ResolveMixedL2CLKind()
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		BatcherOptions: []sysgo.BatcherOption{func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
			cfg.PollInterval = batcherPollInterval
		}},
		// op-geth refuses Karst blocks, which the default dev genesis activates.
		DeployerOptions: []sysgo.DeployerOption{
			sysgo.WithJovianAtGenesis,
			sysgo.WithUniformL2BlockTimes(l2BlockTimeSeconds),
		},
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				CLKey:       sequencerKey,
				CLKind:      clKind,
				ELKey:       sequencerKey,
				ELKind:      sysgo.MixedL2ELOpReth,
				IsSequencer: true,
				OpRethOpts: []sysgo.OpRethOption{
					sysgo.OpRethWithoutProofsHistory(),
					sysgo.OpRethWithPersistenceSchedule(c.persistenceSchedule()),
				},
			},
			{
				CLKey:      proofsFollowerKey,
				CLKind:     clKind,
				ELKey:      proofsFollowerKey,
				ELKind:     sysgo.MixedL2ELOpReth,
				OpRethOpts: []sysgo.OpRethOption{sysgo.OpRethWithPersistenceSchedule(c.persistenceSchedule())},
			},
			{
				CLKey:  gethFollowerKey,
				CLKind: clKind,
				ELKey:  gethFollowerKey,
				ELKind: sysgo.MixedL2ELOpGeth,
			},
		},
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	nodesByELKey := make(map[string]presets.MixedSingleChainNodeFrontends, len(frontends.Nodes))
	for _, node := range frontends.Nodes {
		nodesByELKey[node.Spec.ELKey] = node
	}
	t.Require().Len(nodesByELKey, 3, "expected the sequencer and both followers")
	sequencer := nodesByELKey[sequencerKey]
	return &crashSystem{
		Batcher:        frontends.L2Batcher,
		FunderL2:       frontends.FunderL2,
		GethFollower:   nodesByELKey[gethFollowerKey],
		ProofsFollower: nodesByELKey[proofsFollowerKey],
		Sequencer:      sequencer,
		SequencerEnode: sequencer.EL.Enode(),
	}
}

// newWorkload starts the block recorder workload with its contract writes' receipts read from the
// sequencer within inclusionPollInterval, so the sequencer can halt within about a block of the
// workload's last transaction. A chain rebuilt after a kill cannot repeat a block that held one of
// its transactions, since the kill empties the sequencer's transaction pool.
func newWorkload(t devtest.T, sys *crashSystem) *dsl.BlockRecorderWorkload {
	return dsl.NewBlockRecorderWorkload(t, sys.FunderL2).
		WithInclusionPoll(sys.Sequencer.EL, inclusionPollInterval, inclusionPollAttempts)
}

// driveToKillTarget drives the workload to the case's kill target and returns the block that held
// its last transaction.
func driveToKillTarget(t devtest.T, c crashCase, w *dsl.BlockRecorderWorkload) uint64 {
	killTarget := c.killTargetBlock(w.SetupBlock())
	t.Logger().Info("Driving the workload to the kill target", "kill_target", killTarget,
		"persistence_threshold", c.persistenceThreshold, "setup_block", w.SetupBlock())
	return w.DriveUntilBlock(killTarget)
}

// crashOutcome is what the kill cost the sequencer: the head it had when it was killed, the tip
// its datadir held when it came back, the blocks in between as the followers still hold them, and
// among those the block that held the workload's last transaction, which no rebuilt chain can
// repeat.
type crashOutcome struct {
	headAtKill        eth.L2BlockRef
	lastWorkloadBlock eth.L2BlockRef
	lostBlocks        []eth.L2BlockRef
	persistedTip      eth.L2BlockRef
}

// haltSequencer stops the sequencer from building blocks while its consensus client keeps serving
// the batcher and gossip, and returns the head it stopped at once every node holds it, with the
// block that held the workload's last transaction. A save persists reth's chain only up to
// the memory block buffer target below its head, so a kill at the halted head loses that block
// only when it lies within that many blocks of the head. A halt that lands further past it
// resumes the sequencer and drives the workload one more round, up to haltRounds times.
func haltSequencer(t devtest.T, sys *crashSystem, w *dsl.BlockRecorderWorkload, lastWorkloadBlock uint64) (eth.L2BlockRef, uint64) {
	haltedHead := sys.Sequencer.EL.BlockRefByHash(sys.Sequencer.CL.StopSequencer())
	for round := 1; ; round++ {
		t.Require().GreaterOrEqualf(haltedHead.Number, lastWorkloadBlock,
			"the sequencer halted at %s, below the block %d that held the workload's last transaction", haltedHead, lastWorkloadBlock)
		blocksPastWorkload := haltedHead.Number - lastWorkloadBlock
		t.Logger().Info("Sequencer halted", "head", haltedHead, "last_workload_block", lastWorkloadBlock,
			"blocks_past_workload", blocksPastWorkload, "round", round)
		if blocksPastWorkload < sysgo.OpRethPinnedMemoryBlockBufferTarget {
			break
		}
		t.Require().Lessf(round, haltRounds,
			"the sequencer halted at %s, %d blocks past the block %d that held the workload's last transaction, in each of %d rounds",
			haltedHead, blocksPastWorkload, lastWorkloadBlock, haltRounds)
		sys.Sequencer.CL.StartSequencer()
		lastWorkloadBlock = w.DriveUntilBlock(haltedHead.Number + 1)
		haltedHead = sys.Sequencer.EL.BlockRefByHash(sys.Sequencer.CL.StopSequencer())
	}
	reachedHaltedHead := make([]dsl.CheckFunc, 0, len(sys.nodes()))
	for _, node := range sys.nodes() {
		reachedHaltedHead = append(reachedHaltedHead, node.EL.ReachedFn(eth.Unsafe, haltedHead.Number, resyncAttempts))
	}
	dsl.CheckAll(t, reachedHaltedHead...)
	for _, node := range sys.nodes() {
		nodeHead := node.EL.BlockRefByLabel(eth.Unsafe)
		t.Require().Equalf(haltedHead.ID(), nodeHead.ID(), "%s must hold the halted head %s, but holds %s", node.EL, haltedHead, nodeHead)
	}
	return haltedHead, lastWorkloadBlock
}

// killSequencer stops the halted sequencer's consensus client, kills its op-reth and drops the
// followers' trusted entries for it. Left in place, a follower's trusted redial of the restarted
// sequencer can cross a dial from the sequencer to that follower, and reth tears down both
// sessions. The entries are dropped only once the sequencer is down, since a live sequencer
// redials the followers through its own trusted entries for them.
func killSequencer(sys *crashSystem) {
	sys.Sequencer.CL.Stop()
	sys.Sequencer.EL.StopUngracefully()
	for _, follower := range sys.followers() {
		follower.EL.ForgetPeer(sys.SequencerEnode)
	}
}

// reopenSequencerDatadir brings the sequencer's op-reth back on its datadir and reads the tip it
// persisted, which nothing moves until its consensus client is back. The tip must trail the head
// by at most maxLostBlocks, must lie at or above the workload's setup or a chain rebuilt from it
// lacks the workload's accounts, and must lie below the block that held the workload's last
// transaction or nothing tells a rebuilt chain from the lost one.
func reopenSequencerDatadir(t devtest.T, sys *crashSystem, c crashCase, w *dsl.BlockRecorderWorkload, headAtKill eth.L2BlockRef, lastWorkloadBlock uint64) crashOutcome {
	sys.Sequencer.EL.Start()
	sys.Sequencer.EL.WaitForOnline()
	persistedTip := sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe)
	t.Require().Lessf(persistedTip.Number, headAtKill.Number,
		"the sequencer's op-reth came back at %s, not below the head %s it was killed at, so the kill lost nothing", persistedTip, headAtKill)
	lostBlockCount := headAtKill.Number - persistedTip.Number
	t.Logger().Info("Sequencer op-reth is back on its datadir", "persisted_tip", persistedTip, "head_at_kill", headAtKill,
		"lost_blocks", lostBlockCount, "max_lost_blocks", c.maxLostBlocks(), "last_workload_block", lastWorkloadBlock)
	t.Require().LessOrEqualf(lostBlockCount, c.maxLostBlocks(),
		"the kill lost %d blocks, more than the memory buffer %d plus the persistence backpressure threshold %d",
		lostBlockCount, sysgo.OpRethPinnedMemoryBlockBufferTarget, c.persistenceBackpressureThreshold())
	t.Require().LessOrEqualf(w.SetupBlock(), persistedTip.Number,
		"the workload's funding and contract (block %d) must survive the crash, but the persisted tip is %d", w.SetupBlock(), persistedTip.Number)
	t.Require().Greaterf(lastWorkloadBlock, persistedTip.Number,
		"the block %d that held the workload's last transaction must be among the lost blocks, but the persisted tip is %d", lastWorkloadBlock, persistedTip.Number)
	t.Require().Truef(sys.GethFollower.EL.IsCanonical(persistedTip.ID()),
		"the persisted tip %s is not on the chain the followers hold", persistedTip)
	lostBlocks := blockRefsInRange(sys.GethFollower.EL, persistedTip.Number+1, headAtKill.Number)
	return crashOutcome{
		headAtKill:        headAtKill,
		lastWorkloadBlock: lostBlocks[lastWorkloadBlock-persistedTip.Number-1],
		lostBlocks:        lostBlocks,
		persistedTip:      persistedTip,
	}
}

// rejoinSequencer waits for the proofs follower's store to index every lost block, so the chain
// the sequencer comes back on replaces blocks the store holds, then re-peers the sequencer's
// op-reth with the followers, starts its consensus client and re-peers that too, since both
// peerings died with the processes.
func rejoinSequencer(sys *crashSystem, outcome crashOutcome) {
	// The store flushes a partly filled buffer only after it has sat idle for 10s, well within the
	// default timeout.
	sys.ProofsFollower.EL.WaitForProofsStoreBlock(outcome.headAtKill.Number, dsl.DefaultProofsCatchUpTimeout)
	for _, follower := range sys.followers() {
		sys.Sequencer.EL.PeerWith(follower.EL)
	}
	sys.Sequencer.CL.Start()
	for _, follower := range sys.followers() {
		sys.Sequencer.CL.ConnectPeer(follower.CL)
	}
}

// blockRefsInRange returns the blocks from first to last inclusive as node holds them.
func blockRefsInRange(node *dsl.L2ELNode, first, last uint64) []eth.L2BlockRef {
	refs := make([]eth.L2BlockRef, 0, last-first+1)
	for number := first; number <= last; number++ {
		refs = append(refs, node.BlockRefByNumber(number))
	}
	return refs
}

// keptThrough returns the highest lost block node still holds, or the persisted tip when it holds
// none. The lost blocks a node holds form an unbroken run above the tip, since a child of a
// replaced block cannot stay canonical.
func keptThrough(node *dsl.L2ELNode, outcome crashOutcome) uint64 {
	kept := outcome.persistedTip.Number
	for _, lost := range outcome.lostBlocks {
		if !node.IsCanonical(lost.ID()) {
			break
		}
		kept = lost.Number
	}
	return kept
}

// verifyChainSettled drives the workload from fresh senders past the head the kill cut short and
// requires every node to settle on one chain that is safe past that head, that keeps the persisted
// tip and the lost blocks through a height between keptAtLeast and keptAtMost and replaces the
// rest, and whose proofs the proofs follower serves like the op-geth follower from below the
// persisted tip to the last block driven.
func verifyChainSettled(t devtest.T, sys *crashSystem, w *dsl.BlockRecorderWorkload, outcome crashOutcome, keptAtLeast, keptAtMost uint64) {
	postCrashWorkload := w.WithFreshSenders(sys.FunderL2)
	lastBlock := postCrashWorkload.DriveUntilBlock(outcome.headAtKill.Number + postCrashTrafficBlocks)
	followersInSync := make([]dsl.CheckFunc, 0, len(sys.followers()))
	for _, follower := range sys.followers() {
		followersInSync = append(followersInSync, follower.EL.InSyncFn(sys.Sequencer.EL, safety.LocalUnsafe, resyncAttempts))
	}
	dsl.CheckAll(t, followersInSync...)
	safePastKill := make([]dsl.CheckFunc, 0, len(sys.nodes()))
	for _, node := range sys.nodes() {
		safePastKill = append(safePastKill, node.EL.ReachedFn(eth.Safe, outcome.headAtKill.Number+1, safeAttempts))
	}
	dsl.CheckAll(t, safePastKill...)
	for _, node := range sys.nodes() {
		t.Require().Truef(node.EL.IsCanonical(outcome.persistedTip.ID()),
			"%s replaced the persisted tip %s, which the crash left in place", node.EL, outcome.persistedTip)
		kept := keptThrough(node.EL, outcome)
		t.Logger().Info("Lost blocks settled", "node", node.EL, "persisted_tip", outcome.persistedTip.Number,
			"kept_through", kept, "head_at_kill", outcome.headAtKill.Number)
		t.Require().GreaterOrEqualf(kept, keptAtLeast,
			"%s keeps the lost blocks only through %d, but the batcher had posted them through %d before the sequencer came back", node.EL, kept, keptAtLeast)
		t.Require().LessOrEqualf(kept, keptAtMost,
			"%s keeps the lost blocks through %d, but at most those through %d could come back", node.EL, kept, keptAtMost)
	}
	// A FirstBlock of 0 would let the proofs store pick the start and skip blocks it prunes.
	sweepStart := uint64(1)
	if outcome.persistedTip.Number > proofsSweepLeadIn {
		sweepStart = outcome.persistedTip.Number - proofsSweepLeadIn
	}
	sys.ProofsFollower.EL.VerifyHistoricalProofsMatch(sys.GethFollower.EL, func(opts *dsl.HistoricalProofsOpts) {
		opts.Accounts = append(w.Accounts(), postCrashWorkload.SenderAccounts()...)
		opts.FirstBlock = sweepStart
		opts.LastBlock = lastBlock
	})
}

// TestSequencerOpRethCrashKeepsPostedBlocksWithBatcherRunning kills the sequencer's op-reth with
// blocks it has not persisted, while the batcher keeps running, and requires the chain to settle
// on the blocks the batcher had posted: the restarted sequencer derives every lost block back from
// L1 and replaces none of them, both followers keep up, the batcher keeps submitting until the
// chain past the crash is safe, and the proofs follower's proofs match the op-geth follower's
// across the crash. The sequencer stops building before the kill, and the batcher posts its chain
// through that head while the sequencer's consensus client still serves it.
func TestSequencerOpRethCrashKeepsPostedBlocksWithBatcherRunning(gt *testing.T) {
	t := devtest.ParallelT(gt)
	devtest.RunParallel(t, crashMatrix(), func(t devtest.T, c crashCase) {
		sysgo.SkipOnOpGeth(t, "the crash exercises op-reth's persistence threshold")
		sysgo.SkipOnKonaNode(t, "fails on develop under kona-node (ethereum-optimism/protocol-team#483): at threshold 50 the restarted sequencer's "+
			"build loop fails every forkchoice update with `error code -38002: Invalid forkchoice state` and reth's safe "+
			"label stays unresolvable, so the wait for the posted blocks to be safe fails with `failed to determine "+
			"L2BlockRef of safe, could not get payload: not found`; at threshold 7 the sequencer resumes, but the "+
			"op-geth follower stops inside the blocks the sequencer rebuilt before derivation brought the posted ones "+
			"back and never catches up: `nodes not in sync: slower head at 78, must reach 106: unsafe`")
		sys := newCrashSystem(t, c)
		w := newWorkload(t, sys)
		lastWorkloadBlock := driveToKillTarget(t, c, w)

		haltedHead, lastWorkloadBlock := haltSequencer(t, sys, w, lastWorkloadBlock)
		sys.Batcher.FlushUntilSafe(sys.GethFollower.EL, haltedHead.Number, batcherLandingAttempts)
		killSequencer(sys)
		outcome := reopenSequencerDatadir(t, sys, c, w, haltedHead, lastWorkloadBlock)
		rejoinSequencer(sys, outcome)

		sys.Sequencer.EL.Reached(eth.Safe, outcome.headAtKill.Number, safeAttempts)
		sys.Sequencer.EL.Reached(eth.Unsafe, outcome.headAtKill.Number+1, resumeAttempts)
		verifyChainSettled(t, sys, w, outcome, outcome.headAtKill.Number, outcome.headAtKill.Number)
	})
}

// TestSequencerOpRethCrashRebuildsLostBlocksWithBatcherStopped kills the sequencer's op-reth with
// blocks it has not persisted and none of them on L1, since the batcher was stopped before the
// workload began, and requires the restarted sequencer to rebuild the chain from the tip its
// datadir held, both followers to reorg to the rebuilt chain once the restarted batcher posts it,
// the chain past the crash to become safe, and the proofs follower's proofs to match the op-geth
// follower's across the reorg. The sequencer stops building, and the followers take its head over
// gossip, before its consensus client stops and op-reth is killed.
func TestSequencerOpRethCrashRebuildsLostBlocksWithBatcherStopped(gt *testing.T) {
	t := devtest.ParallelT(gt)
	devtest.RunParallel(t, crashMatrix(), func(t devtest.T, c crashCase) {
		sysgo.SkipOnOpGeth(t, "the crash exercises op-reth's persistence threshold")
		sys := newCrashSystem(t, c)
		sys.Batcher.Stop()
		w := newWorkload(t, sys)
		lastWorkloadBlock := driveToKillTarget(t, c, w)

		haltedHead, lastWorkloadBlock := haltSequencer(t, sys, w, lastWorkloadBlock)
		killSequencer(sys)
		outcome := reopenSequencerDatadir(t, sys, c, w, haltedHead, lastWorkloadBlock)
		rejoinSequencer(sys, outcome)

		// A lost block without a workload transaction can come back identical, so the chain
		// is known to have diverged by the block that held the last one.
		sys.Sequencer.EL.ReorgTriggered(outcome.lastWorkloadBlock, resumeAttempts)
		sys.Batcher.Start()
		for _, follower := range sys.followers() {
			follower.EL.ReorgTriggered(outcome.lastWorkloadBlock, followerReorgAttempts)
		}
		verifyChainSettled(t, sys, w, outcome, outcome.persistedTip.Number, outcome.lastWorkloadBlock.Number-1)
	})
}
