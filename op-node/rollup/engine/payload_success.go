package engine

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type PayloadSuccessEvent struct {
	// if payload should be promoted to safe (must also be pending safe, see DerivedFrom)
	IsLastInSpan bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom eth.L1BlockRef

	Envelope *eth.ExecutionPayloadEnvelope
	Ref      eth.L2BlockRef
}

func (ev PayloadSuccessEvent) String() string {
	return "payload-success"
}

func (eq *EngDeriver) onPayloadSuccess(ev PayloadSuccessEvent) {

	// Backup unsafeHead when new block is not built on original unsafe head.
	if eq.ec.unsafeHead.Number >= ev.Ref.Number {
		eq.ec.SetBackupUnsafeL2Head(eq.ec.unsafeHead, false)
	}
	eq.ec.SetUnsafeHead(ev.Ref)

	// If derived from L1, then it can be considered (pending) safe
	if ev.DerivedFrom != (eth.L1BlockRef{}) {
		if ev.IsLastInSpan {
			eq.ec.SetSafeHead(ev.Ref)
			eq.emitter.Emit(SafeDerivedEvent{Safe: ev.Ref, DerivedFrom: ev.DerivedFrom})
		}
		eq.ec.SetPendingSafeL2Head(ev.Ref)
		eq.emitter.Emit(PendingSafeUpdateEvent{
			PendingSafe: eq.ec.PendingSafeL2Head(),
			Unsafe:      eq.ec.UnsafeL2Head(),
		})
	}

	payload := ev.Envelope.ExecutionPayload
	eq.log.Info("Inserted block", "hash", payload.BlockHash, "number", uint64(payload.BlockNumber),
		"state_root", payload.StateRoot, "timestamp", uint64(payload.Timestamp), "parent", payload.ParentHash,
		"prev_randao", payload.PrevRandao, "fee_recipient", payload.FeeRecipient,
		"txs", len(payload.Transactions), "last_in_span", ev.IsLastInSpan, "derived_from", ev.DerivedFrom)

	eq.emitter.Emit(TryUpdateEngineEvent{})
}
