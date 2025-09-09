package sequencing

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum/go-ethereum/log"
)

// verifierBuilder handles BuildStartEvent for L1-derived attributes when no sequencer is present.
// It starts the payload build, retrieves the payload, and emits a PayloadProcessEvent.
type verifierBuilder struct {
	ctx        context.Context
	log        log.Logger
	rollup     *rollup.Config
	controller *engine.EngineController
	emitter    event.Emitter
	engine     engine.ExecEngine
}

var _ event.Deriver = (*verifierBuilder)(nil)
var _ event.AttachEmitter = (*verifierBuilder)(nil)

func NewVerifierBuilder(ctx context.Context, log log.Logger, cfg *rollup.Config, ec *engine.EngineController, engine engine.ExecEngine) event.Deriver {
	return &verifierBuilder{
		ctx:        ctx,
		log:        log,
		rollup:     cfg,
		controller: ec,
		engine:     engine,
	}
}

func (eq *verifierBuilder) AttachEmitter(em event.Emitter) {
	eq.emitter = em
}

func (eq *verifierBuilder) OnEvent(ctx context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case engine.BuildStartEvent:
		eq.onBuildStart(ctx, x)
	default:
		return false
	}
	return true
}

func (eq *verifierBuilder) onBuildStart(ctx context.Context, ev engine.BuildStartEvent) {
	rpcCtx, cancel := context.WithTimeout(eq.ctx, buildStartTimeout)
	defer cancel()

	if ev.Attributes.DerivedFrom != (eth.L1BlockRef{}) &&
		eq.controller.PendingSafeL2Head().Hash != ev.Attributes.Parent.Hash {
		// Warn about small reorgs, happens when pending safe head is getting rolled back
		eq.log.Warn("block-attributes derived from L1 do not build on pending safe head, likely reorg",
			"pending_safe", eq.controller.PendingSafeL2Head(), "attributes_parent", ev.Attributes.Parent)
	}

	fcEvent := engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    ev.Attributes.Parent,
		SafeL2Head:      eq.controller.SafeL2Head(),
		FinalizedL2Head: eq.controller.Finalized(),
	}
	if fcEvent.UnsafeL2Head.Number < fcEvent.FinalizedL2Head.Number {
		err := fmt.Errorf("invalid block-building pre-state, unsafe head %s is behind finalized head %s", fcEvent.UnsafeL2Head, fcEvent.FinalizedL2Head)
		eq.emitter.Emit(ctx, rollup.CriticalErrorEvent{Err: err}) // make the node exit, things are very wrong.
		return
	}
	fc := eth.ForkchoiceState{
		HeadBlockHash:      fcEvent.UnsafeL2Head.Hash,
		SafeBlockHash:      fcEvent.SafeL2Head.Hash,
		FinalizedBlockHash: fcEvent.FinalizedL2Head.Hash,
	}
	buildStartTime := time.Now()
	id, errTyp, err := engine.StartPayload(rpcCtx, eq.engine, fc, ev.Attributes.Attributes)
	if err != nil {
		switch errTyp {
		case engine.BlockInsertTemporaryErr:
			// RPC errors are recoverable, we can retry the buffered payload attributes later.
			eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
				Err: fmt.Errorf("temporarily cannot insert new safe block: %w", err),
			})
			return
		case engine.BlockInsertPrestateErr:
			eq.emitter.Emit(ctx, rollup.ResetEvent{
				Err: fmt.Errorf("need reset to resolve pre-state problem: %w", err),
			})
			return
		case engine.BlockInsertPayloadErr:
			eq.emitter.Emit(ctx, BuildInvalidEvent{Attributes: ev.Attributes, Err: err})
			return
		default:
			eq.emitter.Emit(ctx, rollup.CriticalErrorEvent{
				Err: fmt.Errorf("unknown error type %d: %w", errTyp, err),
			})
			return
		}
	}
	eq.emitter.Emit(ctx, fcEvent)

	eq.emitter.Emit(ctx, BuildStartedEvent{
		Info:         eth.PayloadInfo{ID: id, Timestamp: uint64(ev.Attributes.Attributes.Timestamp)},
		BuildStarted: buildStartTime,
		Concluding:   ev.Attributes.Concluding,
		DerivedFrom:  ev.Attributes.DerivedFrom,
		Parent:       ev.Attributes.Parent,
	})
}
