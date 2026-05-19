package driver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// safeHeadCall captures a single SafeHeadUpdated call.
type safeHeadCall struct {
	Safe eth.L2BlockRef
	L1   eth.BlockID
}

// recordingSafeHeadListener records every SafeHeadUpdated call.
type recordingSafeHeadListener struct {
	enabled bool
	calls   []safeHeadCall
	resets  []eth.L2BlockRef
}

func (r *recordingSafeHeadListener) Enabled() bool {
	return r.enabled
}

func (r *recordingSafeHeadListener) SafeHeadUpdated(newSafeHead eth.L2BlockRef, l1Block eth.BlockID) error {
	r.calls = append(r.calls, safeHeadCall{Safe: newSafeHead, L1: l1Block})
	return nil
}

func (r *recordingSafeHeadListener) SafeHeadReset(resetSafeHead eth.L2BlockRef) error {
	r.resets = append(r.resets, resetSafeHead)
	return nil
}

func newTestSyncDeriver(t *testing.T, listener *recordingSafeHeadListener) *SyncDeriver {
	emitter := event.EmitterFunc(func(ctx context.Context, ev event.Event) {})
	logger := testlog.Logger(t, log.LevelError)
	sched := NewStepSchedulingDeriver(logger)
	sched.AttachEmitter(emitter)
	s := &SyncDeriver{
		SafeHeadNotifs: listener,
		Emitter:        emitter,
		Log:            logger,
		Ctx:            context.Background(),
		StepDeriver:    sched,
	}
	return s
}

// TestSyncDeriver_SafeHeadUpdatedOncePerL1 asserts that multiple L2 safe-head
// advancements within a single L1 source block result in exactly one
// SafeHeadUpdated call per L1 source, carrying the highest L2 reached.
//
// Today's behavior calls SafeHeadUpdated on every SafeDerivedEvent which
// overwrites the per-L1 SafeDB entry repeatedly and races downstream callers
// that snapshot the value mid-derivation.
func TestSyncDeriver_SafeHeadUpdatedOncePerL1(t *testing.T) {
	listener := &recordingSafeHeadListener{enabled: true}
	s := newTestSyncDeriver(t, listener)
	ctx := context.Background()

	l1A := eth.L1BlockRef{Hash: common.Hash{0x01, 0xaa}, Number: 100}
	l1B := eth.L1BlockRef{Hash: common.Hash{0x01, 0xbb}, Number: 101}

	l2A1 := eth.L2BlockRef{Hash: common.Hash{0x02, 0xa1}, Number: 200, L1Origin: l1A.ID()}
	l2A2 := eth.L2BlockRef{Hash: common.Hash{0x02, 0xa2}, Number: 201, L1Origin: l1A.ID()}
	l2A3 := eth.L2BlockRef{Hash: common.Hash{0x02, 0xa3}, Number: 202, L1Origin: l1A.ID()}
	l2B1 := eth.L2BlockRef{Hash: common.Hash{0x02, 0xb1}, Number: 203, L1Origin: l1B.ID()}
	l2B2 := eth.L2BlockRef{Hash: common.Hash{0x02, 0xb2}, Number: 204, L1Origin: l1B.ID()}

	// Three advancements within L1=A.
	s.OnEvent(ctx, engine.SafeDerivedEvent{Safe: l2A1, Source: l1A})
	s.OnEvent(ctx, engine.SafeDerivedEvent{Safe: l2A2, Source: l1A})
	s.OnEvent(ctx, engine.SafeDerivedEvent{Safe: l2A3, Source: l1A})

	// No SafeHeadUpdated should have fired yet for L1=A: we don't know yet
	// whether more L2 blocks will be derived from the same L1.
	require.Empty(t, listener.calls,
		"SafeHeadUpdated must not fire mid-L1; it should be batched per L1 source")

	// L1 advances. This triggers the flush for L1=A with the final L2 reached.
	s.OnEvent(ctx, engine.SafeDerivedEvent{Safe: l2B1, Source: l1B})

	require.Len(t, listener.calls, 1, "exactly one SafeHeadUpdated for L1=A")
	require.Equal(t, l2A3, listener.calls[0].Safe, "must carry the highest L2 reached during L1=A")
	require.Equal(t, l1A.ID(), listener.calls[0].L1)

	// Another advancement within L1=B; still no extra flush yet.
	s.OnEvent(ctx, engine.SafeDerivedEvent{Safe: l2B2, Source: l1B})
	require.Len(t, listener.calls, 1, "still batching L1=B until a terminator")

	// Derivation idles (caught up). This flushes the pending L1=B entry.
	s.OnEvent(ctx, derive.DeriverIdleEvent{Origin: l1B})
	require.Len(t, listener.calls, 2)
	require.Equal(t, l2B2, listener.calls[1].Safe)
	require.Equal(t, l1B.ID(), listener.calls[1].L1)
}

// TestSyncDeriver_PendingSafeHeadDroppedOnReset asserts that a pipeline reset
// drops the pending in-memory batch without flushing — the engine will
// re-derive the in-progress L1 from scratch after the reset.
func TestSyncDeriver_PendingSafeHeadDroppedOnReset(t *testing.T) {
	listener := &recordingSafeHeadListener{enabled: true}
	s := newTestSyncDeriver(t, listener)
	ctx := context.Background()

	l1A := eth.L1BlockRef{Hash: common.Hash{0x01, 0xaa}, Number: 100}
	l2A1 := eth.L2BlockRef{Hash: common.Hash{0x02, 0xa1}, Number: 200, L1Origin: l1A.ID()}

	s.OnEvent(ctx, engine.SafeDerivedEvent{Safe: l2A1, Source: l1A})
	require.Empty(t, listener.calls)

	// Trigger an ELSync start which clears the safe head DB; pending must be dropped.
	s.OnELSyncStarted()
	require.Empty(t, listener.calls, "pending must not be flushed after reset/elsync")

	// A subsequent idle event must not flush a stale pending entry.
	s.OnEvent(ctx, derive.DeriverIdleEvent{Origin: l1A})
	require.Empty(t, listener.calls, "idle event after drop must not flush stale pending")
}

// TestSyncDeriver_DisabledListenerNoCalls asserts that when the listener is
// disabled, no calls are batched or made.
func TestSyncDeriver_DisabledListenerNoCalls(t *testing.T) {
	listener := &recordingSafeHeadListener{enabled: false}
	s := newTestSyncDeriver(t, listener)
	ctx := context.Background()

	l1A := eth.L1BlockRef{Hash: common.Hash{0x01, 0xaa}, Number: 100}
	l1B := eth.L1BlockRef{Hash: common.Hash{0x01, 0xbb}, Number: 101}
	l2A := eth.L2BlockRef{Hash: common.Hash{0x02, 0xa1}, Number: 200, L1Origin: l1A.ID()}
	l2B := eth.L2BlockRef{Hash: common.Hash{0x02, 0xb1}, Number: 201, L1Origin: l1B.ID()}

	s.OnEvent(ctx, engine.SafeDerivedEvent{Safe: l2A, Source: l1A})
	s.OnEvent(ctx, engine.SafeDerivedEvent{Safe: l2B, Source: l1B})
	s.OnEvent(ctx, derive.DeriverIdleEvent{Origin: l1B})
	require.Empty(t, listener.calls)
}
