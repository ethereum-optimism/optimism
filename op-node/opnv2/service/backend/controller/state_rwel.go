package controller

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/payloads"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type RWELIDProvider interface {
	ID() rwel.ID
}

type RWELState struct {
	state State

	id rwel.ID

	rootCtx context.Context

	// cross-unsafe is not available or applicable on read-write engines.

	localUnsafe eth.BlockRef
	crossSafe   eth.BlockRef
	finalized   eth.BlockRef

	crossSafeBackoff backoffState
	finalizedBackoff backoffState

	syncing    bool
	syncTarget eth.BlockID

	// next payload to process
	next        *eth.ExecutionPayloadEnvelope
	nextBackoff backoffState

	// global, for any engine error
	backoffState

	Forkchoice TaskStateV2
	ELSync     TaskStateV2

	PollLocalUnsafe TaskStateV2
	PollCrossSafe   TaskStateV2
	PollFinalized   TaskStateV2

	IncomingBlock        TaskStateV2
	AttributesProcessing TaskStateV2

	L1Sync TaskStateV2

	chainIDState

	pollState
}

func NewRWELState(rootCtx context.Context, emitter event.Emitter, chainID eth.ChainID, id rwel.ID) *RWELState {
	//rootCtx = rwel.WithID(rootCtx, id)
	out := new(RWELState)
	// TODO
	return out
}

func (s *RWELState) ID() rwel.ID {
	return s.id
}

func (s *RWELState) onUpdate(ev event.Event, now time.Time) {
	switch x := ev.(type) {
	case rwel.LocalUnsafeUpdateEvent:
		s.localUnsafe = x.Ref
	case rwel.CrossSafeUpdateEvent:
		s.crossSafe = x.CrossSafe
	case rwel.FinalizedUpdateEvent:
		s.finalized = x.Ref
	case rwel.ForkchoiceUpdateEvent:
		s.localUnsafe = x.LocalUnsafe
		s.crossSafe = x.CrossSafe
		s.finalized = x.Finalized
	case rwel.SyncingUpdateEvent:
		s.syncing = true
		s.syncTarget = x.SyncTarget
	case rwel.NoSyncingEvent:
		s.syncing = false
		s.syncTarget = eth.BlockID{}
	case rollup.EngineTemporaryErrorEvent:
		s.crossSafeBackoff.DoBackoff(x.Err, now)
		s.finalizedBackoff.DoBackoff(x.Err, now)

	case rwel.InvalidPayloadAttributesEvent:
		// TODO
	case rwel.PayloadSealInvalidEvent:
		// TODO
	case rwel.PayloadInvalidEvent:
		s.nextBackoff.DoBackoff(x.Err, now)
		s.next = nil
	case payloads.PayloadResponseEvent:
		s.next = x.Envelope
	}
}

func LatestAtLeast(num uint64) Predicate[*RWELState] {
	return func(v *RWELState) bool {
		return v.localUnsafe.Number > num
	}
}
