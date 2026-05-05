package status

import (
	"context"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type NoopMetrics struct{}

func (m NoopMetrics) RecordL1ReorgDepth(d uint64)                 {}
func (m NoopMetrics) RecordL1Ref(name string, ref eth.L1BlockRef) {}

func TestStatus(t *testing.T) {

	tracker := NewStatusTracker(testlog.Logger(t, log.LevelDebug), NoopMetrics{}, nil)

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

	tracker := NewStatusTracker(testlog.Logger(t, log.LevelDebug), NoopMetrics{}, nil)
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

// TestOnPipelineReset_InvokesCallback proves the StatusTracker invokes the
// onPipelineReset callback exactly when it processes a rollup.ResetEvent.
// Supernodes rely on this to bump their per-chain generation counter so an
// in-flight superroot_atTimestamp gather can detect mid-call resets.
func TestOnPipelineReset_InvokesCallback(t *testing.T) {
	calls := 0
	tracker := NewStatusTracker(testlog.Logger(t, log.LevelDebug), NoopMetrics{}, func() {
		calls++
	})

	// Other events must not fire the callback.
	tracker.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    eth.L2BlockRef{Number: 101},
		SafeL2Head:      eth.L2BlockRef{Number: 102},
		FinalizedL2Head: eth.L2BlockRef{Number: 99},
	})
	require.Equal(t, 0, calls)

	tracker.OnEvent(context.Background(), rollup.ResetEvent{})
	require.Equal(t, 1, calls)

	// Re-fired ResetEvent bumps again — every reset is observable.
	tracker.OnEvent(context.Background(), rollup.ResetEvent{})
	require.Equal(t, 2, calls)
}

// TestOnPipelineReset_NilSafe confirms a nil callback doesn't crash on reset.
func TestOnPipelineReset_NilSafe(t *testing.T) {
	tracker := NewStatusTracker(testlog.Logger(t, log.LevelDebug), NoopMetrics{}, nil)
	require.NotPanics(t, func() {
		tracker.OnEvent(context.Background(), rollup.ResetEvent{})
	})
}
