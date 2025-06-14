package controller

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/derive2"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/l1access"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type PipelineState struct {
	state State

	id derive2.ID

	rootCtx context.Context

	lastL1Source  eth.BlockRef // if this is zero, a reset is needed
	lastLocalSafe eth.L2BlockRef

	attributes *derive.AttributesWithParent
	envelope   *eth.ExecutionPayloadEnvelope
	confirming eth.L2BlockRef

	// non-nil when attributes were confirmed, this will become the next local-safe
	nextL1Source eth.BlockRef // may be zero if not set

	more bool // true if we know there is more derivation processing left

	l1Backoff  backoffState // when we encounter a L1 error
	engBackoff backoffState // when we encounter an engine related error

	deriveTask TaskStateV2
	nextL1Task TaskStateV2

	chainIDState
}

func NewPipelineState(rootCtx context.Context, emitter event.Emitter, chainID eth.ChainID, id derive2.ID) *PipelineState {
	//rootCtx = derive2.WithID(rootCtx, id)
	out := new(PipelineState)
	// TODO
	return out
}

func (s *PipelineState) ID() derive2.ID {
	return s.id
}

func (s *PipelineState) onUpdate(ev event.Event, now time.Time) {
	switch x := ev.(type) {
	case derive.DeriverMoreEvent:
		s.more = true
	case derive.DerivedAttributesEvent:
		s.lastLocalSafe = x.Attributes.Parent
		s.lastL1Source = x.Attributes.DerivedFrom
		s.more = true
		s.attributes = x.Attributes
		if s.nextL1Source.ParentHash != s.lastL1Source.Hash {
			s.nextL1Source = eth.BlockRef{}
		}
	case derive.ExhaustedL1Event:
		s.nextL1Source = eth.BlockRef{}
		s.lastL1Source = x.L1Ref
		s.lastLocalSafe = x.LastL2
		s.more = false
	case rollup.ResetEvent:
		*s = PipelineState{more: true}
		s.engBackoff.DoBackoff(x.Err, now)
	case rollup.L1TemporaryErrorEvent:
		s.more = true
		s.l1Backoff.DoBackoff(x.Err, now)
	case rollup.EngineTemporaryErrorEvent:
		s.more = true
		s.engBackoff.DoBackoff(x.Err, now)
	case l1access.RetrievedL1BlockEvent:
		s.nextL1Source = x.Ref
	case l1access.TemporaryL1AccessErrorEvent:
		s.l1Backoff.DoBackoff(x.Err, now)
	}
}
