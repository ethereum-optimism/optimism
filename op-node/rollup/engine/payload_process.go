package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var (
	// ErrPayloadDenied is returned when a payload is denied by the SuperAuthority denylist.
	ErrPayloadDenied = errors.New("payload denied by SuperAuthority")
	// ErrPayloadInvalid is returned when a payload's execution is invalid.
	ErrPayloadInvalid = errors.New("payload execution invalid")
)

type PayloadProcessEvent struct {
	// if payload should be promoted to (local) safe (must also be pending safe, see DerivedFrom)
	Concluding bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom  eth.L1BlockRef
	BuildStarted time.Time

	Envelope *eth.ExecutionPayloadEnvelope
	Ref      eth.L2BlockRef
}

func (ev PayloadProcessEvent) String() string {
	return "payload-process"
}

func (e *EngineController) onPayloadProcess(ctx context.Context, ev PayloadProcessEvent) {
	insertStarted, err := e.processNewPayload(ctx, ev.Envelope, ev.Ref, ev.Concluding, ev.DerivedFrom)
	if err != nil {
		return
	}
	e.emitter.Emit(ctx, PayloadSuccessEvent{
		Concluding:    ev.Concluding,
		DerivedFrom:   ev.DerivedFrom,
		BuildStarted:  ev.BuildStarted,
		InsertStarted: insertStarted,
		Envelope:      ev.Envelope,
		Ref:           ev.Ref,
	})
}

// processNewPayload handles the SuperAuthority check and NewPayload RPC call.
// It does NOT acquire e.mu (caller is responsible).
// Returns the insert start time on success, or an error.
// Emits error events for other listeners, but does NOT emit PayloadSuccessEvent (caller's job).
func (e *EngineController) processNewPayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, concluding bool, derivedFrom eth.L1BlockRef) (time.Time, error) {
	rpcCtx, cancel := context.WithTimeout(e.ctx, payloadProcessTimeout)
	defer cancel()

	if e.superAuthority != nil && envelope != nil && envelope.ExecutionPayload != nil {
		payload := envelope.ExecutionPayload
		denied, err := e.superAuthority.IsDenied(uint64(payload.BlockNumber), payload.BlockHash)
		if err != nil {
			e.log.Error("Failed to check SuperAuthority denylist, proceeding with payload",
				"blockNumber", payload.BlockNumber,
				"blockHash", payload.BlockHash,
				"err", err,
			)
		} else if denied {
			if derivedFrom != (eth.L1BlockRef{}) {
				e.log.Warn("Requesting deposits-only replacement for derived payload",
					"blockNumber", payload.BlockNumber,
					"blockHash", payload.BlockHash,
					"derivedFrom", derivedFrom,
				)
				e.emitDepositsOnlyPayloadAttributesRequest(ctx, ref.ParentID(), derivedFrom)
			} else {
				e.log.Warn("Unsafe payload denied by SuperAuthority, dropping",
					"blockNumber", payload.BlockNumber,
					"blockHash", payload.BlockHash,
				)
			}
			return time.Time{}, ErrPayloadDenied
		}
	}

	insertStart := time.Now()
	status, err := e.engine.NewPayload(rpcCtx,
		envelope.ExecutionPayload, envelope.ParentBeaconBlockRoot)
	if err != nil {
		e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("failed to insert execution payload: %w", err),
		})
		return time.Time{}, fmt.Errorf("failed to insert execution payload: %w", err)
	}
	switch status.Status {
	case eth.ExecutionInvalid, eth.ExecutionInvalidBlockHash:
		if derivedFrom != (eth.L1BlockRef{}) && e.rollupCfg.IsHolocene(derivedFrom.Time) {
			e.emitDepositsOnlyPayloadAttributesRequest(ctx, ref.ParentID(), derivedFrom)
			return time.Time{}, ErrPayloadInvalid
		}

		e.emitter.Emit(ctx, PayloadInvalidEvent{
			Envelope: envelope,
			Err:      eth.NewPayloadErr(envelope.ExecutionPayload, status),
		})
		return time.Time{}, ErrPayloadInvalid
	case eth.ExecutionValid:
		return insertStart, nil
	default:
		e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: eth.NewPayloadErr(envelope.ExecutionPayload, status),
		})
		return time.Time{}, eth.NewPayloadErr(envelope.ExecutionPayload, status)
	}
}

// finalizePayload handles tryUpdateUnsafe, safe head updates, and tryUpdateEngineInternal after a successful NewPayload.
// It does NOT acquire e.mu (caller is responsible).
func (e *EngineController) finalizePayload(ctx context.Context, ref eth.L2BlockRef, concluding bool, derivedFrom eth.L1BlockRef, envelope *eth.ExecutionPayloadEnvelope, buildStarted time.Time, insertStarted time.Time) {
	if derivedFrom == ReplaceBlockSource {
		e.log.Warn("Successfully built replacement block, resetting chain to continue now", "replacement", ref)
		e.forceReset(ctx, ref, ref, ref, ref, e.Finalized(), false)
		e.emitter.Emit(ctx, InteropReplacedBlockEvent{
			Envelope: envelope,
			Ref:      ref.BlockRef(),
		})
		e.tryUpdateEngine(ctx)
		return
	}

	e.tryUpdateUnsafe(ctx, ref)
	if derivedFrom != (eth.L1BlockRef{}) {
		e.tryUpdatePendingSafe(ctx, ref, concluding, derivedFrom)
		e.tryUpdateLocalSafe(ctx, ref, concluding, derivedFrom)
	}
	err := e.tryUpdateEngineInternal(ctx)
	if err != nil {
		e.log.Error("Failed to update engine", "error", err)
	} else {
		updateEngineFinish := time.Now()
		e.logBlockProcessingMetrics(updateEngineFinish, PayloadSuccessEvent{
			Concluding:    concluding,
			DerivedFrom:   derivedFrom,
			BuildStarted:  buildStarted,
			InsertStarted: insertStarted,
			Envelope:      envelope,
			Ref:           ref,
		})
	}
}

// ProcessPayload inserts a payload via NewPayload, updates unsafe head, and finalizes via FCU.
// Acquires e.mu. Combines processNewPayload and finalizePayload for the direct-call path.
// Does NOT emit PayloadSuccessEvent.
// Emits UnsafeUpdateEvent on success. On error, emits appropriate error events before returning.
func (e *EngineController) ProcessPayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	insertStarted, err := e.processNewPayload(ctx, envelope, ref, false, eth.L1BlockRef{})
	if err != nil {
		return err
	}
	e.finalizePayload(ctx, ref, false, eth.L1BlockRef{}, envelope, buildStarted, insertStarted)
	return nil
}
