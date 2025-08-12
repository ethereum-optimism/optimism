package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type PayloadSuccessEvent struct {
	// if payload should be promoted to (local) safe (must also be pending safe, see DerivedFrom)
	Concluding bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom   eth.L1BlockRef
	BuildStarted  time.Time
	InsertStarted time.Time

	Envelope *eth.ExecutionPayloadEnvelope
	Ref      eth.L2BlockRef
}

func (ev PayloadSuccessEvent) String() string {
	return "payload-success"
}

func (eq *EngDeriver) onPayloadSuccess(ctx context.Context, ev PayloadSuccessEvent) {
	if ev.DerivedFrom == ReplaceBlockSource {
		eq.log.Warn("Successfully built replacement block, resetting chain to continue now", "replacement", ev.Ref)
		// Change the engine state to make the replacement block the cross-safe head of the chain,
		// And continue syncing from there.
		eq.emitter.Emit(ctx, rollup.ForceResetEvent{
			LocalUnsafe: ev.Ref,
			CrossUnsafe: ev.Ref,
			LocalSafe:   ev.Ref,
			CrossSafe:   ev.Ref,
			Finalized:   eq.ec.Finalized(),
		})
		eq.emitter.Emit(ctx, InteropReplacedBlockEvent{
			Envelope: ev.Envelope,
			Ref:      ev.Ref.BlockRef(),
		})
		// Apply it to the execution engine
		eq.emitter.Emit(ctx, TryUpdateEngineEvent{})
		// Not a regular reset, since we don't wind back to any L2 block.
		// We start specifically from the replacement block.
		return
	}

	// TryUpdateUnsafe, TryUpdatePendingSafe, TryUpdateLocalSafe, TryUpdateEngine must be sequentially invoked
	eq.TryUpdateUnsafe(ctx, ev.Ref)
	// If derived from L1, then it can be considered (pending) safe
	if ev.DerivedFrom != (eth.L1BlockRef{}) {
		eq.TryUpdatePendingSafe(ctx, ev.Ref, ev.Concluding, ev.DerivedFrom)
		eq.TryUpdateLocalSafe(ctx, ev.Ref, ev.Concluding, ev.DerivedFrom)
	}
	// Now if possible synchronously call FCU
	eq.TryUpdateEngine(ctx, TryUpdateEngineEvent{
		BuildStarted:  ev.BuildStarted,
		InsertStarted: ev.InsertStarted,
		Envelope:      ev.Envelope,
	})
}

func (d *EngDeriver) TryUpdateEngine(ctx context.Context, x TryUpdateEngineEvent) {
	// If we don't need to call FCU, keep going b/c this was a no-op. If we needed to
	// perform a network call, then we should yield even if we did not encounter an error.
	if err := d.ec.TryUpdateEngine(d.ctx); err != nil && !errors.Is(err, ErrNoFCUNeeded) {
		if errors.Is(err, derive.ErrReset) {
			d.emitter.Emit(ctx, rollup.ResetEvent{Err: err})
		} else if errors.Is(err, derive.ErrTemporary) {
			d.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: err})
		} else {
			d.emitter.Emit(ctx, rollup.CriticalErrorEvent{
				Err: fmt.Errorf("unexpected TryUpdateEngine error type: %w", err),
			})
		}
	} else if x.triggeredByPayloadSuccess() {
		logValues := x.getBlockProcessingMetrics()
		d.log.Info("Inserted new L2 unsafe block", logValues...)
	}
}
