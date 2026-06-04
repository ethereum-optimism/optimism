package reorg

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

// TestUnsafeGapFillAfterSafeReorg demonstrates the sequence:
//  1. Verifier CLP2P is disconnected and Verifier CL is stopped.
//  2. Safe reorg occurs because L1 reorged.
//  3. Verifier restarts, and consolidation drops the verifier previously-unsafe blocks.
//  4. CLP2P is restored, the verifier backfills and the unsafe gap is closed.
func TestUnsafeGapFillAfterSafeReorg(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	// assertions.go:387:             ERROR[03-30|22:17:00.549]
	// assertions.go:387:             	Error Trace:	/op-devstack/dsl/l2_el.go:204
	// assertions.go:387:             	            				/op-acceptance-tests/tests/sync/elsync/reorg/sync_test.go:78
	// assertions.go:387:             	Error:      	Received unexpected error:
	// assertions.go:387:             	            	operation failed permanently after 30 attempts: expected head to reorg 0xae5a516a6654d4ee6a2edfb9a8e2db12106991b1a29fbb3953dd5afb8a60914e:12, but got 0xae5a516a6654d4ee6a2edfb9a8e2db12106991b1a29fbb3953dd5afb8a60914e:12
	// assertions.go:387:             	Test:       	TestUnsafeGapFillAfterSafeReorg
	sysgo.SkipOnKonaNode(t, "not supported (timeout)")
	sys := newReorgSystem(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

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
		l2Safe := sys.L2EL.BlockRefByLabel(eth.Safe)
		logger.Info("l1 info", "l1_head", l1head, "l1_origin", l2Safe.L1Origin, "l2Safe", l2Safe)
		// Wait until safe L2 block has L1 origin point after the startL1Block
		return l2Safe.Number > 0 && l2Safe.L1Origin.Number > startL1Block.Number
	}, 120*time.Second, 2*time.Second)

	l2BlockBeforeReorg := sys.L2EL.BlockRefByLabel(eth.Safe)
	logger.Info("Target L2 Block to reorg", "l2", l2BlockBeforeReorg, "l1_origin", l2BlockBeforeReorg.L1Origin)

	// Make sure verifier safe head is also advanced from reorgL2Block or matched
	sys.L2ELB.Reached(eth.Safe, l2BlockBeforeReorg.Number, 3)

	// Disconnect CLP2P
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLB)

	// Stop verifier CL
	sys.L2CLB.Stop()

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

	// Wait for the sequencer's safe L2 head to reference the new L1 chain.
	// After the L1 reorg, the L2CL detects it, resets its pipeline, and the batcher
	// re-submits batches with the new L1 view. We verify this by polling until the
	// safe head's L1 origin hash matches the post-reorg L1 block — proving L2
	// actually switched to the new L1 fork, not just that block numbers advanced.
	sys.L2EL.WaitL1OriginHash(eth.Safe, l1BlockAfterReorg.ID(), 30)

	// Restart verifier CL and verify it applies the reorg
	sys.L2CLB.Start()
	sys.L2ELB.InSync(sys.L2EL, eth.Safe, 5)

	// Reconnect CLP2P so verifier can backfill the unsafe gap
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)
	sys.L2ELB.InSync(sys.L2EL, safety.LocalUnsafe, 50)
}

// TestUnsafeGapFillAfterUnsafeReorg_RestartL2CL demonstrates the flow where:
//  1. Verifier L2CL is stopped.
//  2. Unsafe reorg occurs because L1 reorged,
//  3. Verifier restarts and detects the L1 reorg, triggering its own unsafe reorg,
//  4. Verifier then backfills and closes the unsafe gap once reconnected via CLP2P.
func TestUnsafeGapFillAfterUnsafeReorg_RestartL2CL(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	// assertions.go:387:             ERROR[03-30|22:17:07.231]
	// assertions.go:387:             	Error Trace:	/optimism/op-devstack/dsl/l2_el.go:204
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/elsync/reorg/sync_test.go:211
	// assertions.go:387:             	Error:      	Received unexpected error:
	// assertions.go:387:             	            	operation failed permanently after 30 attempts: expected head to reorg 0x893d77533b0ff9b37a92090679bf256d987b4535f06186ec71f29e68ddccd9a5:14, but got 0x893d77533b0ff9b37a92090679bf256d987b4535f06186ec71f29e68ddccd9a5:14
	// assertions.go:387:             	Test:       	TestUnsafeGapFillAfterUnsafeReorg_RestartL2CL
	sysgo.SkipOnKonaNode(t, "not supported (timeout)")
	// Example error with op-reth:
	//
	// assertions.go:387:
	// Error Trace:	/op-devstack/dsl/l2_el.go:430
	//             				/op-acceptance-tests/tests/sync/elsync/reorg/sync_test.go:218
	// Error:      	Received unexpected error:
	//             	operation failed permanently after 50 attempts: expected head to match: unsafe
	// Test:       	TestUnsafeGapFillAfterUnsafeReorg_RestartL2CL
	sysgo.FlakyOnOpReth(t, "")
	sys := newReorgSystem(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	// Stop the batcher not to advance safe head
	sys.L2Batcher.Stop()

	ts := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())
	// Pass the L1 genesis
	sys.L1Network.WaitForBlock()

	// Stop auto advancing L1
	sys.L1CL.Stop()

	startL1Block := sys.L1EL.BlockRefByLabel(eth.Unsafe)

	require.Eventually(func() bool {
		// Advance a single L1 block. See the comment on the matching loop in
		// TestUnsafeGapFillAfterSafeReorg for why we use ts.Next alone and do
		// not require.NoError inside the polling callback.
		if err := ts.Next(ctx); err != nil {
			logger.Warn("ts.Next failed, will retry", "err", err)
			return false
		}
		l1head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
		l2Unsafe := sys.L2EL.BlockRefByLabel(eth.Unsafe)
		logger.Info("l1 info", "l1_head", l1head, "l1_origin", l2Unsafe.L1Origin, "l2Unsafe", l2Unsafe)
		// Wait until unsafe L2 block has L1 origin point after the startL1Block
		return l2Unsafe.Number > 0 && l2Unsafe.L1Origin.Number > startL1Block.Number
	}, 120*time.Second, 2*time.Second)

	sys.L2ELB.InSync(sys.L2EL, safety.LocalUnsafe, 30)

	// Pick reorg block
	l2BlockBeforeReorg := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	logger.Info("Target L2 Block to reorg", "l2", l2BlockBeforeReorg, "l1_origin", l2BlockBeforeReorg.L1Origin)

	// Make few more unsafe blocks which will be reorged out
	sys.L2EL.Advanced(eth.Unsafe, 4)
	sys.L2ELB.InSync(sys.L2EL, safety.LocalUnsafe, 30)

	// Stop Verifier CL
	sys.L2CLB.Stop()
	// Capture verifier's frozen unsafe head to use as a lower bound for what
	// the sequencer must exceed after the reorg.
	verUnsafeFrozen := sys.L2ELB.BlockRefByLabel(eth.Unsafe)

	// Reorg L1 block which unsafe block L1 Origin points to
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
	//  L2EL detects L2 Reorg and reorgs. op-geth example logs: "Chain reorg detected"
	sys.L2EL.ReorgTriggered(l2BlockBeforeReorg, 30)
	l2BlockAfterReorg := sys.L2EL.BlockRefByNumber(l2BlockBeforeReorg.Number)
	require.NotEqual(l2BlockAfterReorg.Hash, l2BlockBeforeReorg.Hash)
	logger.Info("Triggered L2 reorg", "l2", l2BlockAfterReorg)

	// Wait for the sequencer to strictly exceed the verifier's frozen height
	// before asserting seq > ver — ReorgTriggered only guarantees the block at
	// l2BlockBeforeReorg.Number has been rewritten, not that the sequencer has
	// rebuilt past the verifier.
	sys.L2EL.Reached(eth.Unsafe, verUnsafeFrozen.Number+1, 30)

	// Check the divergence before restarting verifier L2CLB
	verUnsafe := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	seqUnsafe := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	logger.Info("Unsafe heads", "seq", seqUnsafe, "ver", verUnsafe)
	// Verifier unsafe head cannot advance yet because L2CLB is down
	require.Greater(seqUnsafe.Number, verUnsafe.Number)
	// Verifier unsafe head diverged
	canonicalFromSeq := sys.L2EL.BlockRefByNumber(verUnsafe.Number)
	require.NotEqual(canonicalFromSeq.Hash, verUnsafe.Hash)
	logger.Info("Verifer unsafe head diverged", "verUnsafe", verUnsafe, "canonical", canonicalFromSeq)
	var rewindTo eth.L2BlockRef
	for i := verUnsafe.Number; i > 0; i-- {
		ver := sys.L2ELB.BlockRefByNumber(i)
		seq := sys.L2EL.BlockRefByNumber(i)
		if ver.Hash == seq.Hash {
			rewindTo = ver
			break
		}
	}
	logger.Info("Verifier diverged", "rewindTo", rewindTo)
	require.Greater(l1BlockAfterReorg.Number, rewindTo.L1Origin.Number)

	// Restart verifier L2CL
	// L2CL walks back. op-node example logs "walking sync start"
	// Dropping L2 blocks which has invalid L1 origin, until we reach rewindTo
	sys.L2CLB.Start()

	// Make sure CLP2P is connected
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	// L2CLB will receive unsafe payloads from sequencer
	// Unsafe gap will be observed by the L2CLB, and it will be smart enough to close the gap,
	// using RR Sync(soon be deprecated), or rely on EL Sync(desired)

	// Verifier converged with sequencer's canonical unsafe chain
	sys.L2ELB.InSync(sys.L2EL, safety.LocalUnsafe, 50)

	seqUnsafe = sys.L2EL.BlockRefByLabel(eth.Unsafe)
	verUnsafe = sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	logger.Info("Verifier unsafe gap closed", "gap", seqUnsafe.Number-verUnsafe.Number, "seqUnsafe", seqUnsafe.Number, "verUnsafe", verUnsafe.Number)

	gt.Cleanup(func() {
		sys.L2Batcher.Start()
		sys.L2CLB.Start()
		sys.L2CLB.ConnectPeer(sys.L2CL)
		sys.L2CL.ConnectPeer(sys.L2CLB)
	})
}

// TestUnsafeGapFillAfterUnsafeReorg_RestartCLP2P demonstrates the flow where:
//  1. Verifier CLP2P is disconnected.
//  2. Unsafe reorg occurs because L1 reorged.
//  3. Verifier detects the L1 reorg, triggering its own unsafe reorg.
//  4. CLP2P is restored Verifier, the verifier backfills and the unsafe gap is closed.
func TestUnsafeGapFillAfterUnsafeReorg_RestartCLP2P(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	// assertions.go:387:             ERROR[03-31|11:15:39.398]
	// assertions.go:387:             	Error Trace:	/optimism/op-devstack/dsl/l2_el.go:204
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/elsync/reorg/sync_test.go:356
	// assertions.go:387:             	Error:      	Received unexpected error:
	// assertions.go:387:             	            	operation failed permanently after 30 attempts: expected head to reorg 0x166970054ad16ad090210e5d1045538eeccd2afd88ea991b010de026d0106870:18, but got 0x166970054ad16ad090210e5d1045538eeccd2afd88ea991b010de026d0106870:18
	// assertions.go:387:             	Test:       	TestUnsafeGapFillAfterUnsafeReorg_RestartCLP2P
	sysgo.SkipOnKonaNode(t, "not supported (timeout)")
	sys := newReorgSystem(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	// Stop the batcher not to advance safe head
	sys.L2Batcher.Stop()

	ts := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())
	// Pass the L1 genesis
	sys.L1Network.WaitForBlock()

	// Stop auto advancing L1
	sys.L1CL.Stop()

	startL1Block := sys.L1EL.BlockRefByLabel(eth.Unsafe)

	require.Eventually(func() bool {
		// Advance a single L1 block. See the comment on the matching loop in
		// TestUnsafeGapFillAfterSafeReorg for why we use ts.Next alone and do
		// not require.NoError inside the polling callback.
		if err := ts.Next(ctx); err != nil {
			logger.Warn("ts.Next failed, will retry", "err", err)
			return false
		}
		l1head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
		l2Unsafe := sys.L2EL.BlockRefByLabel(eth.Unsafe)
		logger.Info("l1 info", "l1_head", l1head, "l1_origin", l2Unsafe.L1Origin, "l2Unsafe", l2Unsafe)
		// Wait until unsafe L2 block has L1 origin point after the startL1Block
		return l2Unsafe.Number > 0 && l2Unsafe.L1Origin.Number > startL1Block.Number
	}, 120*time.Second, 2*time.Second)

	sys.L2ELB.InSync(sys.L2EL, safety.LocalUnsafe, 5)

	// Pick reorg block
	l2BlockBeforeReorg := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	logger.Info("Target L2 Block to reorg", "l2", l2BlockBeforeReorg, "l1_origin", l2BlockBeforeReorg.L1Origin)

	// Make few more unsafe blocks which will be reorged out
	sys.L2EL.Advanced(eth.Unsafe, 4)
	sys.L2ELB.InSync(sys.L2EL, safety.LocalUnsafe, 5)

	// Disconnect CLP2P
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLB)

	// verUnsafe will eventually reorged out
	verUnsafe := sys.L2ELB.BlockRefByLabel(eth.Unsafe)

	// Reorg L1 block which unsafe block L1 Origin points to
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
	//  L2EL detects L2 Reorg and reorgs. op-geth example logs: "Chain reorg detected"
	sys.L2EL.ReorgTriggered(l2BlockBeforeReorg, 30)
	l2BlockAfterReorg := sys.L2EL.BlockRefByNumber(l2BlockBeforeReorg.Number)
	require.NotEqual(l2BlockAfterReorg.Hash, l2BlockBeforeReorg.Hash)
	logger.Info("Triggered L2 reorg", "l2", l2BlockAfterReorg)

	// L2CLB is still up but only have access to L1 to update canonical view
	// verifier cannot advance unsafe head, but only reorging out blocks
	// Test can still independently find rewindTo
	rewindTo := sys.L2ELB.BlockRefByNumber(0)
	for i := verUnsafe.Number; i > 0; i-- {
		ref, err := sys.L2ELB.Escape().L2EthClient().L2BlockRefByNumber(ctx, i)
		if err != nil {
			// May be not found since verifier EL reorging
			continue
		}
		if ref.L1Origin.Number < l1BlockAfterReorg.Number {
			rewindTo = ref
			break
		}
	}
	logger.Info("Verifier diverged", "rewindTo", rewindTo)

	// Wait until verifier reset and dropped all reorg blocks
	sys.L2CLB.Reset(safety.LocalUnsafe, rewindTo)
	logger.Info("Verifier rewind done", "rewindTo", rewindTo)

	// Make sure CLP2P is connected
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	// L2CLB will receive unsafe payloads from sequencer
	// Unsafe gap will be observed by the L2CLB, and it will be smart enough to close the gap,
	// using RR Sync(soon be deprecated), or rely on EL Sync(desired)

	// Verifier converged with sequencer's canonical unsafe chain
	sys.L2ELB.InSync(sys.L2EL, safety.LocalUnsafe, 50)

	seqUnsafe := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	verUnsafe = sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	logger.Info("Verifier unsafe gap closed", "gap", seqUnsafe.Number-verUnsafe.Number, "seqUnsafe", seqUnsafe.Number, "verUnsafe", verUnsafe.Number)

	gt.Cleanup(func() {
		sys.L2Batcher.Start()
		sys.L2CLB.ConnectPeer(sys.L2CL)
		sys.L2CL.ConnectPeer(sys.L2CLB)
	})
}
