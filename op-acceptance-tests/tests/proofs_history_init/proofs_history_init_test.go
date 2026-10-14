package proofs_history_init

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// l2BlockTimeSeconds keeps the runs short: the chain has to grow past a window before the
	// init and past another one for the prune, so halving the block time halves the run.
	l2BlockTimeSeconds = 1
	// memoryBlockBufferTarget is reth's --engine.memory-block-buffer-target. It is passed
	// explicitly, like the persistence threshold, so how far reth's datadir trails its head does
	// not depend on reth's defaults.
	memoryBlockBufferTarget = 5
	// proofsHistoryWindow is the window the init backfills behind its anchor and the node keeps
	// afterwards: small enough that pruning moves the window within the test, large enough that
	// the backfill spans blocks reth has persisted and blocks it still held in memory.
	proofsHistoryWindow = 30
	// windowMargin is how far past a full window the chain is driven before the init, so the
	// backfill target lies above the blocks that funded the workload.
	windowMargin = 5
	// postRestartTrafficBlocks is how far past the head the chain is driven once the follower is
	// back, so the sweeps also cover blocks the store indexed live after the init.
	postRestartTrafficBlocks = 10
	// referenceEthProofWindow lets the sequencer answer eth_getProof for every block of the run
	// from reth's own historical state; reth's default window of zero rejects them all.
	referenceEthProofWindow = 1_000_000

	// pruneTimeout bounds how long the store may take to prune the backfilled blocks once the
	// chain is a window past the anchor.
	pruneTimeout = 90 * time.Second
	// resyncAttempts bounds how long the restarted follower may take to catch up with the
	// sequencer.
	resyncAttempts = 60
	// storeCatchUpTimeout bounds how long the restarted node may take to serve its seeded window.
	storeCatchUpTimeout = 60 * time.Second
)

// followerSpec says how the follower's op-reth runs. Its proofs history is off until a test seeds
// the store with proofs init.
type followerSpec struct {
	persistenceThreshold uint64
	stateMaskingBlocks   uint64
	verificationInterval uint64
}

func (s followerSpec) proofsPath() string {
	if s.verificationInterval != 0 {
		return "execute"
	}
	return "index"
}

// engineArgs pin reth's persistence schedule, which decides how far the datadir trails the head
// at the init. Both nodes get them so the sequencer persists on the same schedule as the follower.
func (s followerSpec) engineArgs() []string {
	args := []string{
		fmt.Sprintf("--engine.memory-block-buffer-target=%d", memoryBlockBufferTarget),
		fmt.Sprintf("--engine.persistence-threshold=%d", s.persistenceThreshold),
	}
	if s.stateMaskingBlocks != 0 {
		args = append(args, fmt.Sprintf("--engine.num-state-masking-blocks=%d", s.stateMaskingBlocks))
	}
	return args
}

func (s followerSpec) followerArgs() []string {
	return append(s.engineArgs(), fmt.Sprintf("--proofs-history.verification-interval=%d", s.verificationInterval))
}

// headWithPersistedWindow is a chain height by which reth's datadir holds more than a window of
// blocks under the spec's threshold, whatever reth still has in memory: reth persists nothing
// until more than the threshold of blocks sit in memory, and from then on its datadir trails the
// head by at most the threshold.
func (s followerSpec) headWithPersistedWindow() uint64 {
	return s.persistenceThreshold + proofsHistoryWindow + windowMargin
}

// initCase is one cell of the matrix the operator path runs under.
type initCase struct {
	spec        followerSpec
	useSnapshot bool
}

func (c initCase) name() string {
	backfill := "snapshot"
	if !c.useSnapshot {
		backfill = "plain"
	}
	return fmt.Sprintf("threshold_%d-%s-backfill_%s", c.spec.persistenceThreshold, c.spec.proofsPath(), backfill)
}

func initMatrix() []initCase {
	var cases []initCase
	for _, persistenceThreshold := range []uint64{7, 50} {
		for _, verificationInterval := range []uint64{0, 1} {
			for _, useSnapshot := range []bool{true, false} {
				cases = append(cases, initCase{
					spec:        followerSpec{persistenceThreshold: persistenceThreshold, verificationInterval: verificationInterval},
					useSnapshot: useSnapshot,
				})
			}
		}
	}
	return cases
}

// guardCase is one cell of the matrix the init guards run under. The guards are about reth's
// datadir, so the cells vary what shapes it: the persistence schedule and state masking.
type guardCase struct {
	spec followerSpec
}

func (c guardCase) name() string {
	return fmt.Sprintf("threshold_%d-masking_%d", c.spec.persistenceThreshold, c.spec.stateMaskingBlocks)
}

func guardMatrix() []guardCase {
	var cases []guardCase
	for _, persistenceThreshold := range []uint64{7, 50} {
		for _, stateMaskingBlocks := range []uint64{0, 30} {
			cases = append(cases, guardCase{spec: followerSpec{persistenceThreshold: persistenceThreshold, stateMaskingBlocks: stateMaskingBlocks}})
		}
	}
	return cases
}

// skipMaskingCell skips a cell that needs state masking, which develop cannot run.
func skipMaskingCell(t devtest.T, spec followerSpec) {
	if spec.stateMaskingBlocks != 0 {
		t.Skip("state masking is compiled out of reth 2.5.2 on develop and #23046 rejects a nonzero --engine.num-state-masking-blocks; re-enable with ethereum-optimism/protocol-team#457, where the init against a killed node must fail with InitializeSourceStateTrieLagging until a clean restart repairs the datadir")
	}
}

type initSystem struct {
	Follower    *dsl.L2ELNode
	FollowerCL  *dsl.L2CLNode
	FunderL2    *dsl.FunderEOA
	Sequencer   *dsl.L2ELNode
	SequencerCL *dsl.L2CLNode
}

// newInitSystem starts a sequencer without proofs history, which serves as the independent
// eth_getProof reference, and a follower without proofs history under the spec's reth and ExEx
// settings, configured with the window its store gets once a test seeds it.
func newInitSystem(t devtest.T, spec followerSpec) *initSystem {
	sysgo.SkipOnOpGeth(t, "proofs history is an op-reth feature")
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
					sysgo.OpRethWithExtraArgs(spec.engineArgs()...),
				},
			},
			{
				CLKey:  "follower",
				CLKind: clKind,
				ELKey:  "follower",
				ELKind: sysgo.MixedL2ELOpReth,
				OpRethOpts: []sysgo.OpRethOption{
					sysgo.OpRethWithoutProofsHistory(),
					sysgo.OpRethWithProofsHistoryWindow(proofsHistoryWindow),
					sysgo.OpRethWithExtraArgs(spec.followerArgs()...),
				},
			},
		},
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 2, "expected the sequencer and the follower")

	sys := &initSystem{FunderL2: frontends.FunderL2}
	for _, node := range frontends.Nodes {
		if node.Spec.IsSequencer {
			sys.Sequencer, sys.SequencerCL = node.EL, node.CL
		} else {
			sys.Follower, sys.FollowerCL = node.EL, node.CL
		}
	}
	t.Require().NotNil(sys.Follower, "missing the follower")
	return sys
}

// workload changes state in nearly every block: alice and bob transfer back and forth and bob
// writes to a contract, so every block the backfill reconstructs carries balance and storage
// changes.
type workload struct {
	alice    *dsl.EOA
	bob      *dsl.EOA
	recorder common.Address
	t        devtest.T
}

func newWorkload(t devtest.T, funder *dsl.FunderEOA) *workload {
	w := &workload{
		alice: funder.NewFundedEOA(eth.OneEther),
		bob:   funder.NewFundedEOA(eth.OneEther),
		t:     t,
	}
	w.recorder = w.alice.DeployBlockRecorder()
	return w
}

// accounts are the accounts the proofs sweeps prove at every block.
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

// driveChainTo runs the workload until the chain reaches target and the follower has imported
// all of it, and returns the follower's head.
func driveChainTo(sys *initSystem, w *workload, target uint64) eth.L2BlockRef {
	w.driveUntilBlock(target)
	sys.Follower.InSync(sys.Sequencer, safety.LocalUnsafe, resyncAttempts)
	return sys.Follower.BlockRefByLabel(eth.Unsafe)
}

// restartFollowerEL brings the follower's op-reth back on its datadir, now with proofs history,
// and returns the datadir tip it resumed from. Nothing feeds the node until its CL is back, so
// the tip is what reth had persisted, and the store serves exactly what the init seeded.
func restartFollowerEL(t devtest.T, sys *initSystem) eth.L2BlockRef {
	sys.Follower.Start()
	sys.Follower.WaitForOnline()
	tip := sys.Follower.BlockRefByLabel(eth.Unsafe)
	t.Logger().Info("Follower op-reth is back on its datadir with proofs history", "datadir_tip", tip)
	return tip
}

// reconnectFollower re-peers the follower's op-reth with the sequencer, brings its CL back and
// waits until the follower has caught up with the sequencer.
func reconnectFollower(sys *initSystem) {
	sys.Follower.PeerWith(sys.Sequencer)
	sys.FollowerCL.Start()
	sys.FollowerCL.ConnectPeer(sys.SequencerCL)
	sys.Follower.InSync(sys.Sequencer, safety.LocalUnsafe, resyncAttempts)
}

// verifySeededWindow requires the store seeded at anchor to span exactly one window behind it and
// sweeps that window against the sequencer. It runs before the follower's CL is back, so nothing
// extends or prunes the store meanwhile.
func verifySeededWindow(t devtest.T, sys *initSystem, w *workload, anchor eth.L2BlockRef) {
	t.Require().GreaterOrEqualf(anchor.Number, uint64(proofsHistoryWindow),
		"the anchor %s must be at least a window (%d blocks) above genesis for the backfill to span a full window", anchor, proofsHistoryWindow)
	window := sys.Follower.WaitForProofsStoreBlock(anchor.Number, storeCatchUpTimeout)
	t.Require().Equalf(anchor.Number, *window.Latest, "the store must be anchored at the datadir tip %s; store window %s", anchor, window)
	t.Require().Equalf(anchor.Number-proofsHistoryWindow, *window.Earliest,
		"the backfill must extend the store one window (%d blocks) behind the anchor %s; store window %s", proofsHistoryWindow, anchor, window)
	sys.Follower.VerifyHistoricalProofsMatch(sys.Sequencer, func(opts *dsl.HistoricalProofsOpts) {
		opts.Accounts = w.accounts()
		opts.FirstBlock = *window.Earliest
		opts.LastBlock = anchor.Number
	})
}

// verifyLiveWindow drives traffic past the current head and sweeps the window the store serves
// then, which now holds blocks the store indexed live.
func verifyLiveWindow(sys *initSystem, w *workload) {
	lastBlock := w.driveUntilBlock(sys.Sequencer.BlockRefByLabel(eth.Unsafe).Number + postRestartTrafficBlocks)
	sys.Follower.VerifyHistoricalProofsMatch(sys.Sequencer, func(opts *dsl.HistoricalProofsOpts) {
		opts.Accounts = w.accounts()
		opts.LastBlock = lastBlock
	})
}

// TestFollowerServesTheWholeWindowAfterProofsInitWithBackfill runs the operator path: a follower
// that synced without proofs history is stopped cleanly, `op-reth proofs init` seeds its store
// at the datadir tip and backfills a window behind it, and the restarted follower must serve
// proofs for the whole window that verify against the headers and match the sequencer's, then
// keep doing so once pruning has moved the window past the init's anchor.
func TestFollowerServesTheWholeWindowAfterProofsInitWithBackfill(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range initMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			sys := newInitSystem(t, c.spec)
			w := newWorkload(t, sys.FunderL2)
			// A clean stop persists what reth holds in memory, so the datadir tip is the head
			// whatever the threshold, and a window plus margin of blocks is enough.
			head := driveChainTo(sys, w, proofsHistoryWindow+windowMargin)

			sys.FollowerCL.Stop()
			sys.Follower.Stop()
			report := sys.Follower.InitProofsHistory(func(init *stack.ProofsHistoryInit) {
				init.UseSnapshot = c.useSnapshot
			})
			t.Logger().Info("Proofs init seeded the follower's store", "cell", c.name(), "init_duration", report.Duration,
				"head_at_stop", head, "window", proofsHistoryWindow)

			anchor := restartFollowerEL(t, sys)
			verifySeededWindow(t, sys, w, anchor)

			reconnectFollower(sys)
			lastBlock := w.driveUntilBlock(anchor.Number + proofsHistoryWindow + windowMargin)
			window := sys.Follower.WaitForProofsStorePrunedPast(anchor.Number, pruneTimeout)
			t.Logger().Info("Pruning moved the window past the init's anchor", "anchor", anchor, "window", window)
			sys.Follower.VerifyHistoricalProofsMatch(sys.Sequencer, func(opts *dsl.HistoricalProofsOpts) {
				opts.Accounts = w.accounts()
				opts.LastBlock = lastBlock
			})
		})
	}
}

// TestProofsInitSeedsAConsistentStoreFromAKilledNodeDatadir kills a follower that synced without
// proofs history, so reth's datadir trails the head by the blocks it still held in memory, and
// runs `op-reth proofs init` against that datadir. The init must seed a store anchored at the
// persisted tip with a full window behind it, and once the follower is back it must serve that
// window and the blocks it re-imports with proofs that match the sequencer's.
func TestProofsInitSeedsAConsistentStoreFromAKilledNodeDatadir(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range guardMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			skipMaskingCell(t, c.spec)
			sys := newInitSystem(t, c.spec)
			w := newWorkload(t, sys.FunderL2)
			head := driveChainTo(sys, w, c.spec.headWithPersistedWindow())

			sys.Follower.StopUngracefully()
			sys.FollowerCL.Stop()
			report := sys.Follower.InitProofsHistory()
			t.Logger().Info("Proofs init seeded the killed follower's store", "cell", c.name(), "init_duration", report.Duration, "head_at_kill", head)

			persistedTip := restartFollowerEL(t, sys)
			t.Require().Lessf(persistedTip.Number, head.Number,
				"the kill must leave reth's datadir (tip %d) behind the head %d, or the init ran against a datadir no different from a clean stop",
				persistedTip.Number, head.Number)
			verifySeededWindow(t, sys, w, persistedTip)

			reconnectFollower(sys)
			verifyLiveWindow(sys, w)
		})
	}
}

// TestProofsInitAgainstARunningNodeNeverSeedsAStoreThatMatchesNoBlock runs `op-reth proofs init`
// against the datadir of a follower that is still running and persisting, which an operator can
// do by mistake. Whatever the init makes of that datadir, the store the follower ends up running
// on must describe the chain: after a clean stop, a seeding of a store the init refused, and a
// restart with proofs history, every proof of the window must verify against its header and match
// the sequencer's.
func TestProofsInitAgainstARunningNodeNeverSeedsAStoreThatMatchesNoBlock(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range guardMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			skipMaskingCell(t, c.spec)
			sys := newInitSystem(t, c.spec)
			w := newWorkload(t, sys.FunderL2)
			// The init reads reth's persisted tip, so the chain is driven until that tip is a
			// window above genesis under the spec's threshold.
			head := driveChainTo(sys, w, c.spec.headWithPersistedWindow())

			report, err := sys.Follower.TryInitProofsHistory()
			if err != nil {
				// #23046 refuses the init when reth persisted past the anchor between the init's
				// two database reads. develop has no such guard, so any failure is unexpected.
				t.Require().ErrorContainsf(err, "reth database tip moved",
					"proofs init against a running node may only fail because reth's tip moved (#23046); it failed with: %v", err)
			}
			t.Logger().Info("Proofs init ran against the running follower", "cell", c.name(), "init_duration", report.Duration,
				"head_at_init", head, "refused", err != nil)

			sys.FollowerCL.Stop()
			sys.Follower.Stop()
			if err != nil {
				sys.Follower.InitProofsHistory()
			}

			tip := restartFollowerEL(t, sys)
			window := sys.Follower.WaitForProofsStoreBlock(tip.Number, storeCatchUpTimeout)
			t.Logger().Info("Follower store caught up with its datadir", "datadir_tip", tip, "window", window)
			sys.Follower.VerifyHistoricalProofsMatch(sys.Sequencer, func(opts *dsl.HistoricalProofsOpts) {
				opts.Accounts = w.accounts()
				opts.LastBlock = tip.Number
			})

			reconnectFollower(sys)
			verifyLiveWindow(sys, w)
		})
	}
}
