package status

import (
	"context"
	"math/rand"
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type NoopMetrics struct{}

func (m NoopMetrics) RecordL1ReorgDepth(d uint64)                 {}
func (m NoopMetrics) RecordL1Ref(name string, ref eth.L1BlockRef) {}

func TestStatus(t *testing.T) {

	tracker := NewStatusTracker(testlog.Logger(t, log.LevelDebug), NoopMetrics{})

	status := tracker.SyncStatus()
	require.Equal(t, eth.SyncStatus{}, *status)

	tracker.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    eth.L2BlockRef{Number: 101},
		SafeL2Head:      eth.L2BlockRef{Number: 102},
		FinalizedL2Head: eth.L2BlockRef{Number: 99},
	})
	status = tracker.SyncStatus()

	// this is a general invariant which should hold both pre and post interop
	require.GreaterOrEqual(t, status.LocalSafeL2.Number, status.SafeL2.Number)

	require.Equal(t, status.UnsafeL2.Number, uint64(101))
	require.Equal(t, status.SafeL2.Number, uint64(102))
	require.Equal(t, status.FinalizedL2.Number, uint64(99))

	// If this were to happen while other fields remain nonzero
	// the batcher might try and load blocks from genesis
	// which would cause a major issue:
	require.NotZero(t, status.LocalSafeL2.Number)

	tracker.OnEvent(context.Background(), rollup.ResetEvent{})
	status = tracker.SyncStatus()

	require.Zero(t, status.LocalSafeL2.Number)
	require.Zero(t, status.SafeL2.Number)
	require.Zero(t, status.UnsafeL2.Number)
	require.Zero(t, status.CurrentL1.Number)
}

// During EL sync the engine answers the forkchoice update with SYNCING, so
// insertUnsafePayload emits UnsafeUpdateEvent without a following
// ForkchoiceUpdateEvent, and derivation is not yet running to emit a
// PendingSafeUpdateEvent. UnsafeUpdateEvent is then the only event that keeps
// the reported unsafe head tracking sync progress.
func TestUnsafeUpdateSetsUnsafeHead(t *testing.T) {
	tracker := NewStatusTracker(testlog.Logger(t, log.LevelDebug), NoopMetrics{})

	ref := eth.L2BlockRef{Number: 42}
	require.True(t, tracker.OnEvent(context.Background(), engine.UnsafeUpdateEvent{Ref: ref}),
		"unsafe update must affect the sync status")
	require.Equal(t, ref, tracker.SyncStatus().UnsafeL2)

	// A reorg moves the unsafe head backwards, so the update must be applied
	// unconditionally rather than only when the head advances.
	reorged := eth.L2BlockRef{Number: 40}
	require.True(t, tracker.OnEvent(context.Background(), engine.UnsafeUpdateEvent{Ref: reorged}))
	require.Equal(t, reorged, tracker.SyncStatus().UnsafeL2)
}

func TestForkchoiceUpdateSeedsLocalSafeWithGenesisSafe(t *testing.T) {
	rng := rand.New(rand.NewSource(20295))
	l1Origin := testutils.RandomBlockRef(rng)
	genesis := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         0,
		Time:           l1Origin.Time,
		L1Origin:       l1Origin.ID(),
		SequenceNumber: 0,
	}

	tracker := NewStatusTracker(testlog.Logger(t, log.LevelDebug), NoopMetrics{})
	tracker.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    genesis,
		SafeL2Head:      genesis,
		FinalizedL2Head: genesis,
	})

	status := tracker.SyncStatus()
	require.Equal(t, genesis, status.SafeL2)
	require.Equal(t, genesis, status.LocalSafeL2)
	require.Equal(t, genesis, status.FinalizedL2)
}

// The L1 handlers run on three separate goroutines — the head subscription and
// the safe/finalized pollers — while OnEvent runs on the driver's event loop.
// All four mutate st.data, so they must serialise on the same mutex. Run with
// -race: without the locks this reports writes racing with UpdateSyncStatus.
func TestStatusTrackerConcurrentUpdates(t *testing.T) {
	tracker := NewStatusTracker(testlog.Logger(t, log.LevelError), NoopMetrics{})

	const iterations = 200
	writers := []func(i uint64){
		func(i uint64) { tracker.OnL1Unsafe(eth.L1BlockRef{Number: i, Hash: common.Hash{byte(i)}}) },
		func(i uint64) { tracker.OnL1Safe(eth.L1BlockRef{Number: i}) },
		func(i uint64) { tracker.OnL1Finalized(eth.L1BlockRef{Number: i}) },
		func(i uint64) {
			tracker.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{
				UnsafeL2Head: eth.L2BlockRef{Number: i},
			})
		},
		func(i uint64) { require.NotNil(t, tracker.SyncStatus()) },
	}

	var wg sync.WaitGroup
	for _, writer := range writers {
		wg.Add(1)
		go func(fn func(uint64)) {
			defer wg.Done()
			for i := uint64(0); i < iterations; i++ {
				fn(i)
			}
		}(writer)
	}
	wg.Wait()
}
