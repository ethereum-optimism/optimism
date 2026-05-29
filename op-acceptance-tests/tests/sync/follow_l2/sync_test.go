package follow_l2

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

func TestFollowL2_Safe_Finalized_CurrentL1(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	// assertions.go:387:             ERROR[03-31|11:33:11.255]
	// assertions.go:387:             	Error Trace:	/optimism/op-devstack/sysgo/singlechain_variants.go:143
	// assertions.go:387:             	            				/optimism/op-devstack/sysgo/singlechain_variants.go:53
	// assertions.go:387:             	            				/optimism/op-devstack/presets/singlechain_twoverifiers.go:24
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/follow_l2/setup_test.go:24
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/follow_l2/sync_test.go:18
	// assertions.go:387:             	Error:      	Should be true
	// assertions.go:387:             	Test:       	TestFollowL2_Safe_Finalized_CurrentL1
	// assertions.go:387:             	Messages:   	single-chain test sequencer requires an op-node CL node
	sysgo.SkipOnKonaNode(t, "not supported")
	sysgo.FlakyOnOpReth(t, "timeouts in merge queue but not locally")
	sys := newSingleChainTwoVerifiersFollowL2(t)
	logger := t.Logger()

	// Takes about 2 minutes for L1 finalization
	attempts := 70
	target := uint64(3)

	// L2CL is the sequencer with CL follow source, derivation disabled
	// L2CLB is the verifier without follow source, derivation enabled
	// L2CLC is the verifier with CL follow source, derivation disabled
	// All verifiers must eventually advance unsafe, safe, finalized
	checkMatchedAll := func(lvl safety.Level) {
		dsl.CheckAll(t,
			sys.L2CL.ReachedFn(lvl, target, attempts),
			sys.L2CLB.ReachedFn(lvl, target, attempts),
			sys.L2CLC.ReachedFn(lvl, target, attempts),
		)
		dsl.CheckAll(t,
			sys.L2CLB.InSyncFn(sys.L2CL, lvl, attempts),
			sys.L2CLB.InSyncFn(sys.L2CLC, lvl, attempts),
		)
	}

	checkMatchedAll(safety.LocalUnsafe)
	logger.Info("Unsafe head advanced due to CLP2P", "target", target)

	checkMatchedAll(safety.LocalSafe)
	logger.Info("Safe head followed source", "target", target)

	checkMatchedAll(safety.Finalized)
	logger.Info("Finalized head followed source", "target", target)

	attempts = 10
	dsl.CheckAll(t,
		sys.L2CLC.CurrentL1MatchedFn(sys.L2CLB, attempts),
		sys.L2CL.CurrentL1MatchedFn(sys.L2CLB, attempts),
	)
	logger.Info("CurrentL1 followed source", "currentL1", sys.L2CL.SyncStatus().CurrentL1, "currentL1C", sys.L2CLC.SyncStatus().CurrentL1)
}

func TestFollowL2_ReorgRecovery(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	// assertions.go:387:             ERROR[03-31|11:31:11.567]
	// assertions.go:387:             	Error Trace:	/optimism/op-devstack/sysgo/singlechain_variants.go:143
	// assertions.go:387:             	            				/optimism/op-devstack/sysgo/singlechain_variants.go:53
	// assertions.go:387:             	            				/optimism/op-devstack/presets/singlechain_twoverifiers.go:24
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/follow_l2/setup_test.go:24
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/follow_l2/sync_test.go:60
	// assertions.go:387:             	Error:      	Should be true
	// assertions.go:387:             	Test:       	TestFollowL2_ReorgRecovery
	// assertions.go:387:             	Messages:   	single-chain test sequencer requires an op-node CL node
	sysgo.SkipOnKonaNode(t, "not supported")
	sys := newSingleChainTwoVerifiersFollowL2(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	// L2CLB is the verifier without follow source, derivation enabled

	ts := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())
	// Pass the L1 genesis
	sys.L1Network.WaitForBlock()

	// Stop auto advancing L1
	sys.L1CL.Stop()

	startL1Block := sys.L1EL.BlockRefByLabel(eth.Unsafe)

	require.Eventually(func() bool {
		// Advance a single L1 block. Sequencer.Next internally calls New with
		// empty BuildOpts and tolerates ErrConflictingJob, so we do not call
		// ts.New here — that would fail with ErrConflictingJob if a previous
		// Next attempt timed out and left the job state wedged.
		//
		// We must not use require.NoError inside this polling callback: a
		// single transient engine-API stall (CPU starvation under CI load)
		// would otherwise mark the test failed on the first error. Instead we
		// log and return false so Eventually retries until the L1 EL recovers.
		if err := ts.Next(ctx); err != nil {
			logger.Warn("ts.Next failed, will retry", "err", err)
			return false
		}
		l1head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
		l2Safe := sys.L2ELB.BlockRefByLabel(eth.Safe)

		logger.Info("l1 info", "l1_head", l1head, "l1_origin", l2Safe.L1Origin, "l2Safe", l2Safe)
		// Wait until safe L2 block has L1 origin point after the startL1Block
		return l2Safe.Number > 0 && l2Safe.L1Origin.Number > startL1Block.Number
	}, 120*time.Second, 2*time.Second)

	l2BlockBeforeReorg := sys.L2ELB.BlockRefByLabel(eth.Safe)
	logger.Info("Target L2 Block to reorg", "l2", l2BlockBeforeReorg, "l1_origin", l2BlockBeforeReorg.L1Origin)

	// Make sure verifier safe head is also advanced from reorgL2Block or matched
	sys.L2ELB.Reached(eth.Safe, l2BlockBeforeReorg.Number, 3)

	// Reorg L1 block which safe block L1 Origin points to
	l1BlockBeforeReorg := sys.L1EL.BlockRefByNumber(l2BlockBeforeReorg.L1Origin.Number)
	logger.Info("Triggering L1 reorg", "l1", l1BlockBeforeReorg)
	require.NoError(ts.New(ctx, seqtypes.BuildOpts{Parent: l1BlockBeforeReorg.ParentHash}))
	require.NoError(ts.Next(ctx))

	// Start advancing L1
	sys.L1CL.Start()

	// Make sure L1 reorged
	sys.L1EL.WaitForBlockNumber(l1BlockBeforeReorg.Number)
	l1BlockAfterReorg := sys.L1EL.BlockRefByNumber(l1BlockBeforeReorg.Number)
	logger.Info("Triggered L1 reorg", "l1", l1BlockAfterReorg)
	require.NotEqual(l1BlockAfterReorg.Hash, l1BlockBeforeReorg.Hash)

	// Need to poll until the L2CL detects L1 Reorg and trigger L2 Reorg
	// What happens:
	//  L2CL detects L1 Reorg and reset the pipeline. op-node example logs: "reset: detected L1 reorg"
	//  L2ELB detects L2 reorg and replaces the original block. The replacement
	//  block at this height may also come from a different parent chain, so only
	//  assert that the original block is replaced before checking convergence.
	var l2BlockAfterReorg eth.L2BlockRef
	require.Eventually(func() bool {
		l2BlockAfterReorg = sys.L2ELB.BlockRefByNumber(l2BlockBeforeReorg.Number)
		if l2BlockAfterReorg.Hash == l2BlockBeforeReorg.Hash {
			logger.Info("Waiting for L2 reorg", "before", l2BlockBeforeReorg, "current", l2BlockAfterReorg)
			return false
		}
		return true
	}, 60*time.Second, 2*time.Second)
	logger.Info("Triggered L2 reorg", "l2", l2BlockAfterReorg)

	attempts := 30
	dsl.CheckAll(t,
		sys.L2CL.InSyncFn(sys.L2CLB, safety.LocalUnsafe, attempts),
		sys.L2CLC.InSyncFn(sys.L2CLB, safety.LocalUnsafe, attempts),
		sys.L2CL.InSyncFn(sys.L2CLB, safety.LocalSafe, attempts),
		sys.L2CLC.InSyncFn(sys.L2CLB, safety.LocalSafe, attempts),
	)
}

func TestFollowL2_WithoutCLP2P(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	// assertions.go:387:             ERROR[03-31|11:27:57.797]
	// assertions.go:387:             	Error Trace:	/optimism/op-devstack/sysgo/singlechain_variants.go:143
	// assertions.go:387:             	            				/optimism/op-devstack/sysgo/singlechain_variants.go:53
	// assertions.go:387:             	            				/optimism/op-devstack/presets/singlechain_twoverifiers.go:24
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/follow_l2/setup_test.go:24
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/follow_l2/sync_test.go:136
	// assertions.go:387:             	Error:      	Should be true
	// assertions.go:387:             	Test:       	TestFollowL2_WithoutCLP2P
	// assertions.go:387:             	Messages:   	single-chain test sequencer requires an op-node CL nod
	sysgo.SkipOnKonaNode(t, "not supported")
	sys := newSingleChainTwoVerifiersFollowL2(t)
	require := t.Require()
	logger := t.Logger()

	attempts := 20
	target := uint64(3)

	// L2CLB is the verifier without follow source, derivation enabled
	sys.L2CLB.Advanced(safety.LocalUnsafe, target, attempts)

	// The test's primary target is the L2CLC, with follow source and derivation disabled
	// There is often a gap between safe and unsafe before disconnect, but the
	// follow-source verifier may also catch up before we observe it. The actual
	// property this test cares about is the post-disconnect behavior below.
	status := sys.L2CLC.SyncStatus()
	logger.Info("Initial follow-source sync status", "safe", status.LocalSafeL2, "unsafe", status.UnsafeL2)

	logger.Info("Disconnect CLP2P")
	// L2CLC is the verifier with follow source, derivation disabled
	// Disconnect CLP2P of verifier which follow source is enabled
	sys.L2CLC.DisconnectPeer(sys.L2CLB)
	sys.L2CLB.DisconnectPeer(sys.L2CLC)
	sys.L2CLC.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLC)

	// Advance few safe blocks
	sys.L2CLC.Advanced(safety.LocalSafe, target, attempts)
	sys.L2CLC.ReachedRef(safety.LocalSafe, sys.L2CLB.HeadBlockRef(safety.LocalSafe).ID(), attempts)

	// Wait for L2CLC's local-safe to catch up to its (now non-moving)
	// unsafe head via the follow source.
	//
	// Before disconnect, CLP2P pushed L2CLC's unsafe head ahead of its
	// follow-source-driven local-safe head; the size of that gap depends on
	// how far the sequencer's tip is ahead of L2CLB's local-safe at the
	// instant we disconnect (i.e. on L1 block production and batcher
	// latency up to that moment). After disconnect L2CLC has no CLP2P and
	// no derivation, so its unsafe head freezes at that high-water mark
	// and only advances once follow-source sees L2CLB's local-safe catch
	// up to it (op-node/rollup/engine/engine_controller.go FollowSource:
	// tryUpdateUnsafe is only called when eLocalSafeRef > unsafeHead).
	// Local-safe then advances tick-by-tick toward L2CLB's local-safe
	// (op-node/rollup/driver/driver.go upstream-sync ticker, fires every
	// 2*BlockTime), which is itself bounded by L1 block production and
	// batch submission.
	//
	// On a CI runner under load the initial gap can comfortably exceed
	// the previous 20-attempt * 2s = 40s budget (see #20718). Bound this
	// convergence wait by stall detection instead of total wall time:
	// succeed as soon as local-safe.Number == unsafe.Number (the real
	// property under test), and only fail if local-safe stops advancing
	// for stallTimeout (which would indicate follow-source is broken).
	waitFollowSourceLocalSafeReachesUnsafe(t, sys.L2CLC, 5*time.Minute, 30*time.Second)
	// The only data source for L2CLC is the follow source.
	// L2CLC unsafe head will only be advancing with safe head together
	status = sys.L2CLC.SyncStatus()
	require.Equal(status.LocalSafeL2, status.UnsafeL2)
	sys.L2CLC.Advanced(safety.LocalSafe, target, attempts)

	// Advance few safe blocks
	sys.L2CLC.Advanced(safety.LocalSafe, target, attempts)

	// Check once again that the unsafe head is moving together with safe head
	status = sys.L2CLC.SyncStatus()
	require.Equal(status.LocalSafeL2, status.UnsafeL2)
	sys.L2CLC.Advanced(safety.LocalSafe, target, attempts)

	// Recover CLP2P
	logger.Info("Recover CLP2P")
	sys.L2CLC.ConnectPeer(sys.L2CLB)
	sys.L2CLB.ConnectPeer(sys.L2CLC)
	sys.L2CLC.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLC)

	// Sequencer unsafe payload will arrive to the verifier, triggering EL sync and filling in the unsafe gap
	dsl.CheckAll(t,
		// In sync with sequencer, with derivation disabled
		sys.L2CLC.InSyncFn(sys.L2CL, safety.LocalSafe, attempts),
		sys.L2CLC.InSyncFn(sys.L2CL, safety.LocalUnsafe, attempts),
		// In sync with other verifier, with derivation enabled
		sys.L2CLC.InSyncFn(sys.L2CLB, safety.LocalSafe, attempts),
		sys.L2CLC.InSyncFn(sys.L2CLB, safety.LocalUnsafe, attempts),
	)

	t.Cleanup(func() {
		sys.L2CLC.ConnectPeer(sys.L2CLB)
		sys.L2CLB.ConnectPeer(sys.L2CLC)
		sys.L2CLC.ConnectPeer(sys.L2CL)
		sys.L2CL.ConnectPeer(sys.L2CLC)
	})
}

// waitFollowSourceLocalSafeReachesUnsafe polls cl's sync status and waits for
// local-safe.Number to reach unsafe.Number.
//
// The previous shape — snapshot unsafe once, then retry.Do0(LocalSafe.Reached,
// snapshot, 20) with a fixed 2s strategy — has no way to distinguish "still
// making progress, just slow" from "stuck". Under CI load the post-disconnect
// unsafe/local-safe gap can be larger than 40s of local-safe progression can
// close, producing the flake in #20718 ("expected head to advance: local-safe",
// 20 attempts, ~55s).
//
// maxWait bounds total wall time. stallTimeout is the longest local-safe is
// allowed to sit at the same value before we declare follow-source stuck —
// re-armed each time local-safe advances. Unsafe is re-read on every iteration
// so a (correctly) live unsafe head doesn't matter; the check is against the
// current observed unsafe value.
func waitFollowSourceLocalSafeReachesUnsafe(t devtest.T, cl *dsl.L2CLNode, maxWait, stallTimeout time.Duration) {
	require := t.Require()
	logger := t.Logger()
	logger.Info("Waiting for follow-source local-safe to reach unsafe",
		"max_wait", maxWait, "stall_timeout", stallTimeout)

	deadline := time.Now().Add(maxWait)
	lastLocalSafe := uint64(0)
	lastProgress := time.Now()

	for {
		status := cl.SyncStatus()
		if status.LocalSafeL2.Number == status.UnsafeL2.Number {
			logger.Info("Follow-source local-safe reached unsafe",
				"local_safe", status.LocalSafeL2.Number, "unsafe", status.UnsafeL2.Number)
			return
		}

		now := time.Now()
		if status.LocalSafeL2.Number > lastLocalSafe {
			lastLocalSafe = status.LocalSafeL2.Number
			lastProgress = now
		}
		stalledFor := now.Sub(lastProgress)
		logger.Info("Follow-source convergence pending",
			"local_safe", status.LocalSafeL2.Number, "unsafe", status.UnsafeL2.Number,
			"stalled_for", stalledFor)

		require.LessOrEqualf(stalledFor, stallTimeout,
			"follow-source local-safe stuck at %d while unsafe is %d",
			status.LocalSafeL2.Number, status.UnsafeL2.Number)
		require.Truef(now.Before(deadline),
			"follow-source local-safe did not reach unsafe within %s (local_safe=%d, unsafe=%d)",
			maxWait, status.LocalSafeL2.Number, status.UnsafeL2.Number)

		time.Sleep(2 * time.Second) // nosemgrep: flake-sleep-in-test -- polling sync status with stall and deadline bounds
	}
}
