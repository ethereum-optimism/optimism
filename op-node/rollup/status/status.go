package status

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

// Compile-time interface compliance check
var _ engine.CrossUpdateHandler = (*StatusTracker)(nil)

type Metrics interface {
	RecordL1ReorgDepth(d uint64)
	RecordL1Ref(name string, ref eth.L1BlockRef)
}

type StatusTracker struct {
	data eth.SyncStatus

	published atomic.Pointer[eth.SyncStatus]

	log log.Logger

	metrics Metrics

	mu sync.RWMutex
}

func NewStatusTracker(log log.Logger, metrics Metrics) *StatusTracker {
	st := &StatusTracker{
		log:     log,
		metrics: metrics,
	}
	st.data = eth.SyncStatus{}
	st.published.Store(&eth.SyncStatus{})
	return st
}

func (st *StatusTracker) OnEvent(ctx context.Context, ev event.Event) bool {
	// TODO(#16917) Remove Event System Refactor Comments
	//  L1UnsafeEvent, L1SafeEvent is removed and OnL1Unsafe is synchronously called at L1Handler
	//  FinalizeL1Event is removed and OnL1Finalized is synchronously called at L1Handler
	st.mu.Lock()
	defer st.mu.Unlock()

	switch x := ev.(type) {
	case engine.ForkchoiceUpdateEvent:
		st.log.Debug("Forkchoice update", "unsafe", x.UnsafeL2Head, "safe", x.SafeL2Head, "finalized", x.FinalizedL2Head)
		st.data.UnsafeL2 = x.UnsafeL2Head
		st.data.SafeL2 = x.SafeL2Head
		if st.data.LocalSafeL2.Number < x.SafeL2Head.Number {
			st.data.LocalSafeL2 = x.SafeL2Head
		}
		st.data.FinalizedL2 = x.FinalizedL2Head
	case engine.PendingSafeUpdateEvent:
		st.data.UnsafeL2 = x.Unsafe
		st.data.PendingSafeL2 = x.PendingSafe
	case engine.LocalSafeUpdateEvent:
		st.log.Debug("Local safe head updated", "local_safe", x.Ref)
		st.data.LocalSafeL2 = x.Ref
	case derive.DeriverL1StatusEvent:
		st.data.CurrentL1 = x.Origin
	case rollup.ResetEvent:
		st.data.UnsafeL2 = eth.L2BlockRef{}
		st.data.SafeL2 = eth.L2BlockRef{}
		st.data.LocalSafeL2 = eth.L2BlockRef{}
		st.data.CurrentL1 = eth.L1BlockRef{}
	case engine.EngineResetConfirmedEvent:
		st.data.UnsafeL2 = x.LocalUnsafe
		st.data.CrossUnsafeL2 = x.CrossUnsafe
		st.data.LocalSafeL2 = x.LocalSafe
		st.data.SafeL2 = x.CrossSafe
		st.data.FinalizedL2 = x.Finalized
	default: // other events do not affect the sync status
		return false
	}

	st.UpdateSyncStatus()
	return true
}

func (st *StatusTracker) UpdateSyncStatus() {
	// If anything changes, then copy the state to the published SyncStatus
	// @dev: If this becomes a performance bottleneck during sync (because mem copies onto heap, and 1KB comparisons),
	// we can rate-limit updates of the published data.
	published := *st.published.Load()
	if st.data != published {
		published = st.data
		st.published.Store(&published)
	}
}

func (st *StatusTracker) OnL1Unsafe(x eth.L1BlockRef) {
	st.metrics.RecordL1Ref("l1_head", x)
	// We don't need to do anything if the head hasn't changed.
	if st.data.HeadL1 == (eth.L1BlockRef{}) {
		st.log.Info("Received first L1 head signal", "l1_head", x)
	} else if st.data.HeadL1.Hash == x.Hash {
		st.log.Trace("Received L1 head signal that is the same as the current head", "l1_head", x)
	} else if st.data.HeadL1.Hash == x.ParentHash {
		// We got a new L1 block whose parent hash is the same as the current L1 head. Means we're
		// dealing with a linear extension (new block is the immediate child of the old one).
		st.log.Debug("L1 head moved forward", "l1_head", x)
	} else {
		if st.data.HeadL1.Number >= x.Number {
			st.metrics.RecordL1ReorgDepth(st.data.HeadL1.Number - x.Number)
		}
		// New L1 block is not the same as the current head or a single step linear extension.
		// This could either be a long L1 extension, or a reorg, or we simply missed a head update.
		st.log.Warn("L1 head signal indicates a possible L1 re-org",
			"old_l1_head", st.data.HeadL1, "new_l1_head_parent", x.ParentHash, "new_l1_head", x)
	}
	st.data.HeadL1 = x
	st.UpdateSyncStatus()
}

func (st *StatusTracker) OnL1Safe(x eth.L1BlockRef) {
	st.log.Info("New L1 safe block", "l1_safe", x)
	st.metrics.RecordL1Ref("l1_safe", x)
	st.data.SafeL1 = x
	st.UpdateSyncStatus()
}

func (st *StatusTracker) OnL1Finalized(x eth.L1BlockRef) {
	st.log.Info("New L1 finalized block", "l1_finalized", x)
	st.metrics.RecordL1Ref("l1_finalized", x)
	st.data.FinalizedL1 = x
	st.data.CurrentL1Finalized = x
	st.UpdateSyncStatus()
}

// SyncStatus is thread safe, and reads the latest view of L1 and L2 block labels
func (st *StatusTracker) SyncStatus() *eth.SyncStatus {
	return st.published.Load()
}

// L1Head is a helper function; the L1 head is closely monitored for confirmation-distance logic.
func (st *StatusTracker) L1Head() eth.L1BlockRef {
	return st.SyncStatus().HeadL1
}

func (st *StatusTracker) OnCrossUnsafeUpdate(ctx context.Context, crossUnsafe eth.L2BlockRef, localUnsafe eth.L2BlockRef) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.log.Debug("Cross unsafe head updated", "cross_unsafe", crossUnsafe, "local_unsafe", localUnsafe)
	st.data.CrossUnsafeL2 = crossUnsafe
	st.data.UnsafeL2 = localUnsafe

	st.UpdateSyncStatus()
}

func (st *StatusTracker) OnCrossSafeUpdate(ctx context.Context, crossSafe eth.L2BlockRef, localSafe eth.L2BlockRef) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.log.Debug("Cross safe head updated", "cross_safe", crossSafe, "local_safe", localSafe)
	st.data.SafeL2 = crossSafe
	st.data.LocalSafeL2 = localSafe

	st.UpdateSyncStatus()
}
