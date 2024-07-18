package engine

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// BuildSealedEvent is emitted by the engine when a payload finished building,
// but is not locally inserted as canonical block yet
type BuildSealedEvent struct {
	// if payload should be promoted to safe (must also be pending safe, see DerivedFrom)
	IsLastInSpan bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom eth.L1BlockRef

	Info     eth.PayloadInfo
	Envelope *eth.ExecutionPayloadEnvelope
	Ref      eth.L2BlockRef
}

func (ev BuildSealedEvent) String() string {
	return "build-sealed"
}

func (eq *EngDeriver) onBuildSealed(ev BuildSealedEvent) {
	// If a (pending) safe block, immediately process the block
	if ev.DerivedFrom != (eth.L1BlockRef{}) {
		eq.emitter.Emit(PayloadProcessEvent{
			IsLastInSpan: ev.IsLastInSpan,
			DerivedFrom:  ev.DerivedFrom,
			Envelope:     ev.Envelope,
			Ref:          ev.Ref,
		})
	}
}
