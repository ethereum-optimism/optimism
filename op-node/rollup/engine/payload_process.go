package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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

func (eq *EngineController) onPayloadProcess(ctx context.Context, ev PayloadProcessEvent) {
	rpcCtx, cancel := context.WithTimeout(eq.ctx, payloadProcessTimeout)
	defer cancel()

	insertStart := time.Now()
	status, err := eq.engine.NewPayload(rpcCtx,
		ev.Envelope.ExecutionPayload, ev.Envelope.ParentBeaconBlockRoot)
	if err != nil {
		eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("failed to insert execution payload: %w", err),
		})
		return
	}
	switch status.Status {
	case eth.ExecutionInvalid, eth.ExecutionInvalidBlockHash:
		// Depending on execution engine, not all block-validity checks run immediately on build-start
		// at the time of the forkchoiceUpdated engine-API call, nor during getPayload.
		if ev.DerivedFrom != (eth.L1BlockRef{}) && eq.rollupCfg.IsHolocene(ev.DerivedFrom.Time) {
			eq.emitDepositsOnlyPayloadAttributesRequest(ctx, ev.Ref.ParentID(), ev.DerivedFrom)
			return
		}

		eq.emitter.Emit(ctx, PayloadInvalidEvent{
			Envelope: ev.Envelope,
			Err:      eth.NewPayloadErr(ev.Envelope.ExecutionPayload, status),
		})
		return
	case eth.ExecutionValid:
		eq.emitter.Emit(ctx, PayloadSuccessEvent{
			Concluding:    ev.Concluding,
			DerivedFrom:   ev.DerivedFrom,
			BuildStarted:  ev.BuildStarted,
			InsertStarted: insertStart,
			Envelope:      ev.Envelope,
			Ref:           ev.Ref,
		})
		return
	default:
		eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: eth.NewPayloadErr(ev.Envelope.ExecutionPayload, status),
		})
		return
	}
}
