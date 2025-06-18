package engine

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
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
	event.Ctx
}

func (ev PayloadSuccessEvent) String() string {
	return "payload-success"
}

func (eq *EngDeriver) onPayloadSuccess(ev PayloadSuccessEvent) {
	if ev.DerivedFrom == ReplaceBlockSource {
		eq.log.Warn("Successfully built replacement block, resetting chain to continue now", "replacement", ev.Ref)
		// Change the engine state to make the replacement block the cross-safe head of the chain,
		// And continue syncing from there.
		eq.emitter.Emit(rollup.ForceResetEvent{
			LocalUnsafe: ev.Ref,
			CrossUnsafe: ev.Ref,
			LocalSafe:   ev.Ref,
			CrossSafe:   ev.Ref,
			Finalized:   eq.ec.Finalized(),
			Ctx:         ev.Ctx,
		})
		eq.emitter.Emit(InteropReplacedBlockEvent{
			Envelope: ev.Envelope,
			Ref:      ev.Ref.BlockRef(),
			Ctx:      ev.Ctx,
		})
		// Apply it to the execution engine
		eq.emitter.Emit(TryUpdateEngineEvent{Ctx: ev.Ctx})
		// Not a regular reset, since we don't wind back to any L2 block.
		// We start specifically from the replacement block.
		return
	}

	eq.emitter.Emit(PromoteUnsafeEvent{Ref: ev.Ref, Ctx: ev.Ctx})

	// If derived from L1, then it can be considered (pending) safe
	if ev.DerivedFrom != (eth.L1BlockRef{}) {
		eq.emitter.Emit(PromotePendingSafeEvent{
			Ref:        ev.Ref,
			Concluding: ev.Concluding,
			Source:     ev.DerivedFrom,
			Ctx:        ev.Ctx,
		})
	}

	eq.emitter.Emit(TryUpdateEngineEvent{
		BuildStarted:  ev.BuildStarted,
		InsertStarted: ev.InsertStarted,
		Envelope:      ev.Envelope,
		Ctx:           ev.Ctx,
	})
}
