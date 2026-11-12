package follow_l2

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

func TestFollowL2_Safe_Finalized_CurrentL1(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSingleChainTwoVerifiersFollowL2(t)
	logger := t.Logger()

	// Takes about 2 minutes for L1 finalization
	attempts := 70
	target := uint64(3)

	// L2CL is the sequencer and L2CLB the deriving verifier, both op-node by the preset.
	// L2CLC follows L2CLB with derivation disabled and runs the CL kind DEVSTACK_L2CL_KIND selects.
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
	// The follow-source verifier must hold the block about to be reorged as local-safe, so the
	// recovery checks below start from a real safe head rather than from genesis.
	require.NoError(sys.L2CLC.ReachedWithProgressFn(
		safety.LocalSafe, safety.LocalUnsafe, l2BlockBeforeReorg.Number,
		2*time.Minute, 30*time.Second,
	)())
	require.Equal(l2BlockBeforeReorg.Hash, sys.L2ELC.BlockRefByNumber(l2BlockBeforeReorg.Number).Hash,
		"follow-source verifier must hold the deriving verifier's block before the L1 reorg")

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

	sys.L2ELC.ReorgTriggered(l2BlockBeforeReorg, 60)
	// A local-safe head past the reorged height shows the follow-source verifier recovered; the
	// in-sync checks below also pass when both verifiers sit at genesis. Waiting for the EL reorg
	// first means the unsafe head that bounds this wait is already climbing on the new chain.
	require.NoError(sys.L2CLC.ReachedWithProgressFn(
		safety.LocalSafe, safety.LocalUnsafe, l2BlockBeforeReorg.Number+1,
		5*time.Minute, time.Minute,
	)())
	require.Equal(sys.L2ELB.BlockRefByNumber(l2BlockBeforeReorg.Number).Hash,
		sys.L2ELC.BlockRefByNumber(l2BlockBeforeReorg.Number).Hash,
		"follow-source verifier must replace the reorged block with the deriving verifier's")

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

	// Wait for L2CLC's local-safe to catch up to its now-frozen unsafe head.
	// The initial gap can exceed a fixed 40s budget on loaded CI runners
	// (#20718), so bound the wait by stall detection on local-safe progress.
	require.NoError(sys.L2CLC.MatchedWithProgressFn(
		unsafeAsLocalSafe{cl: sys.L2CLC},
		safety.LocalSafe, safety.LocalSafe,
		5*time.Minute, 30*time.Second,
	)())
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

	// The deriving verifier finalizes past genesis first, so the comparison cannot pass with both
	// nodes at genesis.
	require.NoError(sys.L2CLB.ReachedWithProgressFn(
		safety.Finalized, safety.LocalSafe, 1,
		5*time.Minute, time.Minute,
	)())
	sys.L2CLC.InSync(sys.L2CLB, safety.Finalized, attempts)

	t.Cleanup(func() {
		sys.L2CLC.ConnectPeer(sys.L2CLB)
		sys.L2CLB.ConnectPeer(sys.L2CLC)
		sys.L2CLC.ConnectPeer(sys.L2CL)
		sys.L2CL.ConnectPeer(sys.L2CLC)
	})
}

// unsafeAsLocalSafe lets MatchedWithProgressFn compare cl.LocalSafe to
// cl.UnsafeL2: it takes two providers at one level, so we synthesize a
// reference whose LocalSafe reports cl's unsafe head.
type unsafeAsLocalSafe struct{ cl *dsl.L2CLNode }

func (u unsafeAsLocalSafe) ChainSyncStatus(chainID eth.ChainID, lvl safety.Level) eth.BlockID {
	if lvl != safety.LocalSafe {
		panic(fmt.Sprintf("unsafeAsLocalSafe queried at %s, expected LocalSafe", lvl))
	}
	return u.cl.ChainSyncStatus(chainID, safety.LocalUnsafe)
}

func (u unsafeAsLocalSafe) String() string {
	return u.cl.String() + "/unsafe-as-local-safe"
}
