package sequencer

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/stretchr/testify/require"
)

// TestMaxSafeLagStallAndResume verifies the sequencer.max-safe-lag behavior:
//  1. The sequencer produces blocks normally when the safe head is caught up.
//  2. When the batcher is stopped and the unsafe/safe gap exceeds maxSafeLag,
//     the sequencer stalls (stops producing new blocks).
//  3. When the batcher is restarted and the safe head advances past the stall
//     point, the sequencer resumes producing blocks.
//
// This protects the feature introduced in ethereum-optimism/optimism#17936
// against regression.
func TestMaxSafeLagStallAndResume(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// max-safe-lag is enforced inside op-node's Go sequencer; kona-node has its
	// own sequencer implementation and is out of scope for this regression test.
	sysgo.SkipOnKonaNode(t, "max-safe-lag is op-node only")
	const maxSafeLag = uint64(20)
	// This test does not need fault proofs; using the NoFaultProofs variant
	// avoids requiring cannon prestate artifacts in local test runs.
	sys := presets.NewMinimalNoFaultProofs(t,
		presets.WithGlobalL2CLOption(sysgo.L2CLSequencerMaxSafeLag(maxSafeLag)))

	blockTime := sys.L2Chain.Escape().RollupConfig().BlockTime

	// Phase 1: confirm the chain is producing blocks and the safe head is keeping up.
	startingUnsafe := sys.L2CL.SyncStatus().UnsafeL2.Number
	require.Eventually(t, func() bool {
		return sys.L2CL.SyncStatus().UnsafeL2.Number > startingUnsafe+5
	}, time.Duration(blockTime*20)*time.Second, time.Second, "chain did not produce blocks at startup")

	// Phase 2: stop the batcher so the safe head falls behind the unsafe head.
	sys.L2Batcher.Stop()
	t.Logger().Info("batcher stopped, waiting for sequencer to stall at maxSafeLag")

	// Phase 3: wait until the unsafe head stops advancing for several consecutive
	// block times while the gap exceeds maxSafeLag — that is the definitive signal
	// that the sequencer is stalled. A simple "gap >= maxSafeLag then sleep" check
	// would be racy: the batcher can still have in-flight L1 transactions that,
	// once mined, deliver a batch of safe blocks, transiently shrink the gap, and
	// let the sequencer produce a few more unsafe blocks before re-stalling.
	//
	// The confirmation window must be longer than the L1 block time (6s in
	// devstack) so that any L1 tx broadcast by the batcher before Stop() has a
	// chance to mine within the window rather than right after we accept stall.
	const stableSamples = 10
	stallTimeout := time.Duration(blockTime*(maxSafeLag*4+30)) * time.Second
	var stalledUnsafe uint64
	require.Eventually(t, func() bool {
		status := sys.L2CL.SyncStatus()
		gap := status.UnsafeL2.Number - status.SafeL2.Number
		if gap < maxSafeLag {
			stalledUnsafe = 0
			return false
		}
		if stalledUnsafe == 0 || status.UnsafeL2.Number != stalledUnsafe {
			// Either first sighting, or unsafe head advanced — reset the stability window.
			stalledUnsafe = status.UnsafeL2.Number
			return false
		}
		// Unsafe head unchanged since the previous sample with gap >= maxSafeLag.
		// The for-loop below confirms the stall for additional samples.
		return true
	}, stallTimeout, time.Duration(blockTime)*time.Second, "sequencer did not stall after stopping batcher")

	// Confirm the stall holds for enough samples to cover at least one L1 block.
	for range stableSamples {
		require.NoError(t, clock.SystemClock.SleepCtx(t.Ctx(), time.Duration(blockTime)*time.Second)) // nosemgrep: flake-sleep-in-test -- stall confirmation requires wall-clock spacing; no chain event to wait on
		status := sys.L2CL.SyncStatus()
		require.Equal(t, stalledUnsafe, status.UnsafeL2.Number,
			"sequencer was expected to stay stalled but unsafe head advanced: %d -> %d",
			stalledUnsafe, status.UnsafeL2.Number)
	}
	t.Logger().Info("sequencer stalled as expected", "unsafe", stalledUnsafe, "safe", sys.L2CL.SyncStatus().SafeL2.Number)

	// Phase 4: restart the batcher so the safe head can advance again.
	sys.L2Batcher.Start()
	t.Logger().Info("batcher restarted, waiting for sequencer to resume")

	// Phase 5: the sequencer should resume producing blocks past the stalled head.
	// 1 minute is a generous margin for CI jitter.
	require.Eventually(t, func() bool {
		return sys.L2CL.SyncStatus().UnsafeL2.Number > stalledUnsafe
	}, time.Minute, time.Second,
		"sequencer did not resume producing blocks after batcher restart")
	t.Logger().Info("sequencer resumed", "unsafe", sys.L2CL.SyncStatus().UnsafeL2.Number)
}
