package controller

import (
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rel"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type RELIDProvider interface {
	ID() rel.ID
}

type RELState struct {
	id rel.ID

	// local-safe and cross-unsafe are not
	// available or applicable from read-only endpoints

	localUnsafe eth.L2BlockRef
	crossSafe   eth.L2BlockRef
	finalized   eth.L2BlockRef

	// TODO backoff on last known err

	chainIDState
	pollState
}

func (s *RELState) ID() rel.ID {
	return s.id
}

func (s *RELState) onUpdate(ev event.Event) {
	switch x := ev.(type) {
	case rel.LocalUnsafeUpdateEvent:
		s.localUnsafe = x.LocalUnsafe
	case rel.CrossSafeUpdateEvent:
		s.crossSafe = x.CrossSafe
	case rel.FinalizedUpdateEvent:
		s.finalized = x.Finalized
	case rollup.EngineTemporaryErrorEvent:
		// TODO backoff
	}
}
