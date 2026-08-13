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
	insertStarted, err := e.processNewPayload(ctx, ev.Envelope, ev.Ref, ev.DerivedFrom)
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
func (e *EngineController) processNewPayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, derivedFrom eth.L1BlockRef) (time.Time, error) {
	rpcCtx, cancel := context.WithTimeout(e.ctx, payloadProcessTimeout)
	defer cancel()

	// Check SuperAuthority denylist before inserting the payload
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
		insertErr := fmt.Errorf("failed to insert execution payload: %w", err)
		e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: insertErr})
		return time.Time{}, insertErr
	}
	switch status.Status {
	case eth.ExecutionInvalid, eth.ExecutionInvalidBlockHash:
		// Depending on execution engine, not all block-validity checks run immediately on build-start
		// at the time of the forkchoiceUpdated engine-API call, nor during getPayload.
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
		statusErr := eth.NewPayloadErr(envelope.ExecutionPayload, status)
		e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: statusErr})
		return time.Time{}, statusErr
	}
}

// ProcessPayload inserts a payload via NewPayload, updates unsafe head, and finalizes via FCU.
// Acquires e.mu. Combines processNewPayload and finalizePayload for the direct-call path.
// Emits UnsafeUpdateEvent on success. On error, emits appropriate error events before returning.
// Returns ErrStaleBuild without inserting when the payload no longer extends the unsafe head.
// A nil return means the payload was inserted (NewPayload valid) and the unsafe head updated;
// the concluding forkchoice update may still have failed transiently — it is logged and
// retried implicitly, since every subsequent build-start and engine update carries the full
// forkchoice state.
// Does NOT emit PayloadSuccessEvent.
func (e *EngineController) ProcessPayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	// The unsafe head may have moved (competing insert, block replacement) since
	// the payload was sealed; inserting it anyway would reorg the head backwards.
	if envelope.ExecutionPayload.ParentHash != e.unsafeHead.Hash {
		e.log.Warn("dropping stale sequencer payload, parent is not the unsafe head",
			"payload", ref, "parent", envelope.ExecutionPayload.ParentHash, "unsafe", e.unsafeHead)
		e.requestForkchoiceUpdate(ctx)
		return ErrStaleBuild
	}
	insertStarted, err := e.processNewPayload(ctx, envelope, ref, eth.L1BlockRef{})
	if err != nil {
		return err
	}
	e.finalizePayload(ctx, ref, false, eth.L1BlockRef{}, envelope, buildStarted, insertStarted)
	return nil
}
