package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type BuildStartEvent struct {
	Attributes *derive.AttributesWithParent
}

func (ev BuildStartEvent) String() string {
	return "build-start"
}

// startBuild contains the core logic for beginning a block build. It is used by
// StartBuildAsync in both event-system (async) and non-event (sync) contexts.
func (eq *EngineController) startBuild(ctx context.Context, attrs *derive.AttributesWithParent) error {
	rpcCtx, cancel := context.WithTimeout(eq.ctx, buildStartTimeout)
	defer cancel()

	if attrs.DerivedFrom != (eth.L1BlockRef{}) &&
		eq.PendingSafeL2Head().Hash != attrs.Parent.Hash {
		// Warn about small reorgs, happens when pending safe head is getting rolled back
		eq.log.Warn("block-attributes derived from L1 do not build on pending safe head, likely reorg",
			"pending_safe", eq.PendingSafeL2Head(), "attributes_parent", attrs.Parent)
	}

	fcEvent := ForkchoiceUpdateEvent{
		UnsafeL2Head:    attrs.Parent,
		SafeL2Head:      eq.safeHead,
		FinalizedL2Head: eq.finalizedHead,
	}
	if fcEvent.UnsafeL2Head.Number < fcEvent.FinalizedL2Head.Number {
		err := fmt.Errorf("invalid block-building pre-state, unsafe head %s is behind finalized head %s", fcEvent.UnsafeL2Head, fcEvent.FinalizedL2Head)
		eq.emitter.Emit(ctx, rollup.CriticalErrorEvent{Err: err}) // make the node exit, things are very wrong.
		return err
	}
	fc := eth.ForkchoiceState{
		HeadBlockHash:      fcEvent.UnsafeL2Head.Hash,
		SafeBlockHash:      fcEvent.SafeL2Head.Hash,
		FinalizedBlockHash: fcEvent.FinalizedL2Head.Hash,
	}
	buildStartTime := time.Now()
	id, errTyp, err := startPayload(rpcCtx, eq.engine, fc, attrs.Attributes)
	if err != nil {
		switch errTyp {
		case BlockInsertTemporaryErr:
			// RPC errors are recoverable, we can retry the buffered payload attributes later.
			eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
				Err: fmt.Errorf("temporarily cannot insert new safe block: %w", err),
			})
		case BlockInsertPrestateErr:
			eq.emitter.Emit(ctx, rollup.ResetEvent{
				Err: fmt.Errorf("need reset to resolve pre-state problem: %w", err),
			})
		case BlockInsertPayloadErr:
			eq.emitter.Emit(ctx, BuildInvalidEvent{Attributes: attrs, Err: err})
		default:
			eq.emitter.Emit(ctx, rollup.CriticalErrorEvent{
				Err: fmt.Errorf("unknown error type %d: %w", errTyp, err),
			})
		}
		return err
	}
	eq.emitter.Emit(ctx, fcEvent)

	eq.emitter.Emit(ctx, BuildStartedEvent{
		Info:         eth.PayloadInfo{ID: id, Timestamp: uint64(attrs.Attributes.Timestamp)},
		BuildStarted: buildStartTime,
		Concluding:   attrs.Concluding,
		DerivedFrom:  attrs.DerivedFrom,
		Parent:       attrs.Parent,
	})
	return nil
}

func (eq *EngineController) StartBuildAsync(ctx context.Context, attrs *derive.AttributesWithParent) event.Promise0[error] {
	return event.Spawn0(ctx, func(ctx context.Context) error {
		return eq.startBuild(ctx, attrs)
	}, event.WithSpawnLegacyEvent(BuildStartEvent{Attributes: attrs}))
}
