package rwel

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type PayloadProcessEvent struct {
	// payload is promoted to pending-safe if non-zero
	DerivedFrom  eth.L1BlockRef
	BuildStarted time.Time

	Envelope *eth.ExecutionPayloadEnvelope
}

func (ev PayloadProcessEvent) String() string {
	return "payload-process"
}

func (eq *RWEL) onPayloadProcess(ctx context.Context, ev PayloadProcessEvent) {
	rpcCtx, cancel := context.WithTimeout(ctx, payloadProcessTimeout)
	defer cancel()

	insertStart := time.Now()
	status, err := eq.engine.NewPayload(rpcCtx,
		ev.Envelope.ExecutionPayload, ev.Envelope.ParentBeaconBlockRoot)
	if err != nil {
		// Engine API enshrines status codes:
		// -38005 invalid fork -> if wrong engine API method version was used
		// -32602 invalid params -> if wrong engine API method content was used
		// But we support the right methods, and if the assumption is
		// broken it's best to stall on temporary errors until the discrepancy is identified.
		eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("failed to insert execution payload: %w", err),
		})
		return
	}
	ref := ev.Envelope.BlockRef()
	switch status.Status {
	case eth.ExecutionInvalid, eth.ExecutionInvalidBlockHash:
		// Depending on execution engine, not all block-validity checks run immediately on build-start
		// at the time of the forkchoiceUpdated engine-API call, nor during getPayload.
		if ev.DerivedFrom != (eth.L1BlockRef{}) {
			req := derive.DepositsOnlyPayloadAttributesRequestEvent{
				Parent:      ev.Envelope.BlockRef().ParentID(),
				DerivedFrom: ev.DerivedFrom,
			}
			eq.log.Warn("Payload processing was invalid, requesting deposits-only attributes",
				"parent", req.Parent, "derived_from", req.DerivedFrom)
			eq.emitter.Emit(ctx, req)
			return
		}

		eq.emitter.Emit(ctx, PayloadInvalidEvent{
			Envelope: ev.Envelope,
			Err:      eth.NewPayloadErr(ev.Envelope.ExecutionPayload, status),
		})
		return
	case eth.ExecutionAccepted:
		if eq.state.IsSyncing() {
			eq.log.Info("Execution engine is syncing, and buffered the new block", "ref", ref)
		} else {
			eq.log.Warn("Execution engine accepted block but never signaled it is syncing", "ref", ref)
		}
		eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("cannot validate payload yet, but tentatively accepted block %s", ref),
		})
	case eth.ExecutionSyncing:
		eq.state.SetIsSyncing(true)
		eq.emitter.Emit(ctx, SyncingUpdateEvent{
			SyncTarget: eq.state.SyncTarget(),
		})
		eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: eth.NewPayloadErr(ev.Envelope.ExecutionPayload, status),
		})
	case eth.ExecutionValid:
		eq.state.SetIsSyncing(false)
		eq.state.SetSyncTarget(eth.BlockID{})
		eq.emitter.Emit(ctx, PayloadSuccessEvent{
			DerivedFrom:   ev.DerivedFrom,
			BuildStarted:  ev.BuildStarted,
			InsertStarted: insertStart,
			Envelope:      ev.Envelope,
		})
		eq.emitter.Emit(ctx, NoSyncingEvent{
			LocalUnsafe: eq.state.LocalUnsafe(),
		})
		return
	default:
		eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: eth.NewPayloadErr(ev.Envelope.ExecutionPayload, status),
		})
		return
	}
}
