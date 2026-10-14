package proofs_history_restart

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
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
	// referenceEthProofWindow lets the sequencer answer eth_getProof for every block of the run
	// from reth's own historical state; reth's default window of zero rejects them all.
	referenceEthProofWindow = 1_000_000

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
	// killAfterStoreFlush kills the follower as soon as the store has flushed the target block, so
	// the store's tip is the chain head and its lead over reth's datadir is the largest.
	killAfterStoreFlush killPoint = "after_flush"
	// killMidStoreBuffer kills the follower a couple of blocks after a flush, so the store's tip
	// trails the chain head and the store loses blocks of its own on top of reth's.
	killMidStoreBuffer killPoint = "mid_buffer"
)

// restartCase is one cell of the matrix the follower runs under.
type restartCase struct {
	killPoint            killPoint
	persistenceThreshold uint64
	// storeFlushBeforeKill is the proofs store tip the kill waits for.
	storeFlushBeforeKill uint64
	syncMode             nodeSync.Mode
	verificationInterval uint64
}

func (c restartCase) name() string {
	proofsPath := "index"
	if c.verificationInterval != 0 {
		proofsPath = "execute"
	}
	syncMode := "cl_sync"
	if c.syncMode == nodeSync.ELSync {
		syncMode = "el_sync"
	}
	return fmt.Sprintf("threshold_%d-%s-%s-kill_%s", c.persistenceThreshold, proofsPath, syncMode, c.killPoint)
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

// restartMatrix pairs the kill point with the persistence threshold rather than crossing them:
// the small threshold gets the mid-buffer kill, where both reth and the store lose blocks, the
// large one the after-flush kill, where the store's lead over reth is largest. The kill heights
// leave reth's datadir behind the store: at threshold 7 reth persists every few blocks, at 50 it
// has persisted once by block 60 (down to block 46) and holds the rest in memory.
func restartMatrix() []restartCase {
	var cases []restartCase
	for _, schedule := range []struct {
		killPoint            killPoint
		persistenceThreshold uint64
		storeFlushBeforeKill uint64
	}{
		{killPoint: killMidStoreBuffer, persistenceThreshold: 7, storeFlushBeforeKill: 25},
		{killPoint: killAfterStoreFlush, persistenceThreshold: 50, storeFlushBeforeKill: 60},
	} {
		for _, verificationInterval := range []uint64{0, 1} {
			for _, syncMode := range []nodeSync.Mode{nodeSync.CLSync, nodeSync.ELSync} {
				cases = append(cases, restartCase{
					killPoint:            schedule.killPoint,
					persistenceThreshold: schedule.persistenceThreshold,
					storeFlushBeforeKill: schedule.storeFlushBeforeKill,
					syncMode:             syncMode,
					verificationInterval: verificationInterval,
				})
			}
		}
	}
	return cases
}

// skipUnsupportedCell skips the cells the configured clients cannot run.
func skipUnsupportedCell(t devtest.T, c restartCase) {
	sysgo.SkipOnOpGeth(t, "proofs history is an op-reth feature")
	if c.syncMode == nodeSync.CLSync {
		sysgo.SkipOnKonaNode(t, "kona-node has no CL sync mode: it always lets the EL sync the gap")
	}
}

type restartSystem struct {
	Batcher     *dsl.L2Batcher
	Follower    *dsl.L2ELNode
	FollowerCL  *dsl.L2CLNode
	FunderL2    *dsl.FunderEOA
	Sequencer   *dsl.L2ELNode
	SequencerCL *dsl.L2CLNode
}

// newRestartSystem starts a sequencer without proofs history, which serves as the independent
// eth_getProof reference, and a follower running proofs history under the case's reth, ExEx and
// sync-mode settings.
func newRestartSystem(t devtest.T, c restartCase) *restartSystem {
	clKind := sysgo.ResolveMixedL2CLKind()
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		DeployerOptions: []sysgo.DeployerOption{sysgo.WithUniformL2BlockTimes(l2BlockTimeSeconds)},
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				CLKey:       "sequencer",
				CLKind:      clKind,
				ELKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpReth,
				IsSequencer: true,
				OpRethOpts: []sysgo.OpRethOption{
					sysgo.OpRethWithoutProofsHistory(),
					sysgo.OpRethWithEthProofWindow(referenceEthProofWindow),
					sysgo.OpRethWithExtraArgs(c.engineArgs()...),
				},
			},
			{
				CLKey:      "follower",
				CLKind:     clKind,
				ELKey:      "proofs-follower",
				ELKind:     sysgo.MixedL2ELOpReth,
				L2CLOpts:   []sysgo.L2CLOption{sysgo.L2CLVerifierSyncMode(c.syncMode)},
				OpRethOpts: []sysgo.OpRethOption{sysgo.OpRethWithExtraArgs(c.followerArgs()...)},
			},
		},
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 2, "expected the sequencer and the proofs follower")

	sys := &restartSystem{Batcher: frontends.L2Batcher, FunderL2: frontends.FunderL2}
	for _, node := range frontends.Nodes {
		if node.Spec.IsSequencer {
			sys.Sequencer, sys.SequencerCL = node.EL, node.CL
		} else {
			sys.Follower, sys.FollowerCL = node.EL, node.CL
		}
	}
	t.Require().NotNil(sys.Follower, "missing the proofs follower")
	return sys
}

// workload changes state in nearly every block: alice and bob transfer back and forth and bob
// writes to a contract, so every block an unclean stop loses carries balance and storage changes
// that a rebuilt block would not repeat.
type workload struct {
	alice    *dsl.EOA
	bob      *dsl.EOA
	recorder common.Address
	// setupBlock is the block that completed the funding and the contract deployment; a rebuilt
	// chain must start below it or the workload's accounts no longer exist.
	setupBlock uint64
	t          devtest.T
}

func newWorkload(t devtest.T, funder *dsl.FunderEOA) *workload {
	w := &workload{
		alice: funder.NewFundedEOA(eth.OneEther),
		bob:   funder.NewFundedEOA(eth.OneEther),
		t:     t,
	}
	w.recorder = w.alice.DeployBlockRecorder()
	w.setupBlock = includedBlock(t, w.alice.Transact(w.alice.Plan(), txplan.WithTo(&w.recorder)))
	return w
}

// accounts are the accounts the proofs sweep proves at every block.
func (w *workload) accounts() []dsl.ProofAccount {
	return []dsl.ProofAccount{
		{Address: w.alice.Address()},
		{Address: w.bob.Address()},
		{Address: w.recorder, StorageKeys: []common.Hash{common.HexToHash("0x0"), common.HexToHash("0x1")}},
	}
}

// driveUntilBlock sends transactions one after another until one lands in a block at or past
// target, and returns that block.
func (w *workload) driveUntilBlock(target uint64) uint64 {
	for {
		w.alice.Transfer(w.bob.Address(), eth.OneHundredthEther)
		w.bob.Transfer(w.alice.Address(), eth.OneHundredthEther)
		block := includedBlock(w.t, w.bob.Transact(w.bob.Plan(), txplan.WithTo(&w.recorder)))
		if block >= target {
			return block
		}
	}
}

func includedBlock(t devtest.T, tx *txplan.PlannedTx) uint64 {
	receipt, err := tx.Included.Eval(t.Ctx())
	t.Require().NoError(err, "transaction must be included")
	return bigs.Uint64Strict(receipt.BlockNumber)
}

// uncleanStop is what the follower held when it was killed: the chain head it had imported and
// the last block its proofs store had flushed, both on the chain of that moment.
type uncleanStop struct {
	head     eth.L2BlockRef
	storeTip eth.L2BlockRef
}

// killFollower drives the workload until the follower's proofs store has flushed the case's target
// block and kills the follower's op-reth at the case's kill point. Its CL is left to the caller,
// so a test that kills the sequencer too can do so right away.
func killFollower(t devtest.T, sys *restartSystem, w *workload, c restartCase) uncleanStop {
	// Stop sending transactions two blocks short of the flush: close enough that the last
	// transactions sit above reth's persisted tip, yet with no transaction in flight to delay the
	// kill once the flush is observed.
	w.driveUntilBlock(c.storeFlushBeforeKill - 2)
	window := sys.Follower.WaitForProofsStoreBlock(c.storeFlushBeforeKill, storeCatchUpTimeout)
	if c.killPoint == killMidStoreBuffer {
		sys.Follower.WaitForBlockNumber(*window.Latest + 2)
	}
	// Read the store tip again right before the kill: a flush since the wait would have moved it.
	window, err := sys.Follower.ProofsSyncStatus()
	t.Require().NoError(err, "debug_proofsSyncStatus must answer before the kill")
	storeTip := sys.Follower.BlockRefByNumber(*window.Latest)
	head := sys.Follower.BlockRefByLabel(eth.Unsafe)
	t.Logger().Info("Killing the follower", "head", head, "proofs_store_tip", storeTip, "kill_point", c.killPoint)

	sys.Follower.StopUngracefully()
	return uncleanStop{head: head, storeTip: storeTip}
}

// restartFollower brings the follower's op-reth and CL back and re-peers them with the sequencer,
// since both peerings died with the processes. It returns the block reth resumes from: its datadir
// tip, which must be below the proofs store's tip for the test to exercise the crash gap at all.
func restartFollower(t devtest.T, sys *restartSystem, stop uncleanStop) eth.L2BlockRef {
	sys.Follower.Start()
	sys.Follower.WaitForOnline()
	// Nothing feeds the EL until its CL is back, so its head is what reth had persisted.
	persistedTip := sys.Follower.BlockRefByLabel(eth.Unsafe)
	t.Logger().Info("Follower op-reth is back on its datadir", "persisted_tip", persistedTip,
		"proofs_store_tip", stop.storeTip, "head_at_kill", stop.head)
	t.Require().Lessf(persistedTip.Number, stop.storeTip.Number,
		"the proofs store (tip %d) must be ahead of reth's datadir (tip %d) after the kill; the kill landed too late for this threshold",
		stop.storeTip.Number, persistedTip.Number)

	sys.Follower.PeerWith(sys.Sequencer)
	sys.FollowerCL.Start()
	sys.FollowerCL.ConnectPeer(sys.SequencerCL)
	return persistedTip
}

// TestProofsHistoryFollowerRecoversFromUncleanShutdown kills a proofs-history follower once its
// store has flushed blocks reth has not yet persisted, restarts it into the same chain and
// requires it to catch up with the sequencer, keep indexing, and serve proofs that match the
// sequencer's for the blocks it lost, the blocks it re-imported and the blocks that followed.
func TestProofsHistoryFollowerRecoversFromUncleanShutdown(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range restartMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			skipUnsupportedCell(t, c)
			sys := newRestartSystem(t, c)
			w := newWorkload(t, sys.FunderL2)

			stop := killFollower(t, sys, w, c)
			sys.FollowerCL.Stop()
			persistedTip := restartFollower(t, sys, stop)

			sys.Follower.InSync(sys.Sequencer, safety.LocalUnsafe, resyncAttempts)
			lastBlock := w.driveUntilBlock(sys.Sequencer.BlockRefByLabel(eth.Unsafe).Number + postRestartTrafficBlocks)
			sys.Follower.VerifyHistoricalProofsMatch(sys.Sequencer, func(opts *dsl.HistoricalProofsOpts) {
				opts.Accounts = w.accounts()
				opts.FirstBlock = persistedTip.Number
				opts.LastBlock = lastBlock
			})
		})
	}
}

// TestProofsHistoryFollowerServesRebuiltBlocksAfterUncleanShutdown kills a proofs-history
// follower, then kills the sequencer's op-reth too while the blocks the follower's store holds are
// still only in the sequencer's memory, so the restarted sequencer rebuilds those heights with
// different contents. The restarted follower must follow the rebuilt chain and serve its proofs,
// not the ones its store recorded for the chain that no longer exists.
func TestProofsHistoryFollowerServesRebuiltBlocksAfterUncleanShutdown(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range restartMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			skipUnsupportedCell(t, c)
			t.Skip("fails on develop: the restarted follower's op-reth exits with `Critical task exex panicked: ExEx proofs-history crashed: Parent hash mismatch` at the block after its proofs store's tip, because the store is never reconciled with reth's datadir (ethereum-optimism/protocol-team#464); un-skip with #23358")
			sys := newRestartSystem(t, c)
			// Blocks that reached L1 would be derived back as they were; blocks that never did
			// are re-sequenced, which is the rebuild this test needs.
			sys.Batcher.Stop()
			w := newWorkload(t, sys.FunderL2)

			// Both op-reths die back to back, so the sequencer still holds the blocks the
			// follower's store flushed; the CLs follow once the processes are gone.
			stop := killFollower(t, sys, w, c)
			sys.Sequencer.StopUngracefully()
			sys.FollowerCL.Stop()
			sys.SequencerCL.Stop()

			sys.Sequencer.Start()
			sys.Sequencer.WaitForOnline()
			sequencerPersistedTip := sys.Sequencer.BlockRefByLabel(eth.Unsafe)
			t.Logger().Info("Sequencer op-reth is back on its datadir", "persisted_tip", sequencerPersistedTip,
				"follower_proofs_store_tip", stop.storeTip, "workload_setup_block", w.setupBlock)
			t.Require().Lessf(sequencerPersistedTip.Number, stop.storeTip.Number,
				"the sequencer must rebuild blocks the follower's proofs store holds, but its datadir tip %d is not below the store tip %d",
				sequencerPersistedTip.Number, stop.storeTip.Number)
			t.Require().LessOrEqualf(w.setupBlock, sequencerPersistedTip.Number,
				"the workload's funding and contract (block %d) must survive the sequencer's crash, but its datadir tip is %d",
				w.setupBlock, sequencerPersistedTip.Number)
			sys.SequencerCL.Start()
			sys.Sequencer.ReorgTriggered(stop.storeTip, rebuildAttempts)

			persistedTip := restartFollower(t, sys, stop)

			sys.Follower.InSync(sys.Sequencer, safety.LocalUnsafe, resyncAttempts)
			lastBlock := w.driveUntilBlock(sys.Sequencer.BlockRefByLabel(eth.Unsafe).Number + postRestartTrafficBlocks)
			sys.Follower.VerifyHistoricalProofsMatch(sys.Sequencer, func(opts *dsl.HistoricalProofsOpts) {
				opts.Accounts = w.accounts()
				opts.FirstBlock = persistedTip.Number
				opts.LastBlock = lastBlock
			})
		})
	}
}
