package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type BuildStartEvent struct {
	Attributes *derive.AttributesWithParent
}

func (ev BuildStartEvent) String() string {
	return "build-start"
}

func (e *EngineController) onBuildStart(ctx context.Context, ev BuildStartEvent) {
	result, err := e.startBuild(ctx, ev.Attributes)
	if err != nil {
		return
	}
	e.emitter.Emit(ctx, BuildStartedEvent{
		Info:         result.Info,
		BuildStarted: result.BuildStarted,
		Concluding:   ev.Attributes.Concluding,
		DerivedFrom:  ev.Attributes.DerivedFrom,
		Parent:       result.Parent,
	})
}

// startBuild contains the core logic for starting a block-building job.
// It does NOT acquire e.mu (caller is responsible).
// Emits ForkchoiceUpdateEvent and error events, but NOT BuildStartedEvent (caller's job).
func (e *EngineController) startBuild(ctx context.Context, attrs *derive.AttributesWithParent) (*BuildStartResult, error) {
	rpcCtx, cancel := context.WithTimeout(e.ctx, buildStartTimeout)
	defer cancel()

	if attrs.DerivedFrom != (eth.L1BlockRef{}) &&
		e.pendingSafeHead.Hash != attrs.Parent.Hash {
		e.log.Warn("block-attributes derived from L1 do not build on pending safe head, likely reorg",
			"pending_safe", e.pendingSafeHead, "attributes_parent", attrs.Parent)
	}

	fcEvent := ForkchoiceUpdateEvent{
		UnsafeL2Head:    attrs.Parent,
		SafeL2Head:      e.SafeL2Head(),
		FinalizedL2Head: e.FinalizedHead(),
	}
	if fcEvent.UnsafeL2Head.Number < fcEvent.FinalizedL2Head.Number {
		err := fmt.Errorf("invalid block-building pre-state, unsafe head %s is behind finalized head %s", fcEvent.UnsafeL2Head, fcEvent.FinalizedL2Head)
		e.emitter.Emit(ctx, rollup.CriticalErrorEvent{Err: err})
		return nil, err
	}
	fc := eth.ForkchoiceState{
		HeadBlockHash:      fcEvent.UnsafeL2Head.Hash,
		SafeBlockHash:      fcEvent.SafeL2Head.Hash,
		FinalizedBlockHash: fcEvent.FinalizedL2Head.Hash,
	}
	buildStartTime := time.Now()
	id, errTyp, err := e.startPayload(rpcCtx, fc, attrs.Attributes)
	if err != nil {
		switch errTyp {
		case BlockInsertTemporaryErr:
			e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
				Err: fmt.Errorf("temporarily cannot insert new safe block: %w", err),
			})
			return nil, err
		case BlockInsertPrestateErr:
			e.emitter.Emit(ctx, rollup.ResetEvent{
				Err: fmt.Errorf("need reset to resolve pre-state problem: %w", err),
			})
			return nil, err
		case BlockInsertPayloadErr:
			e.emitter.Emit(ctx, BuildInvalidEvent{Attributes: attrs, Err: err})
			return nil, err
		default:
			e.emitter.Emit(ctx, rollup.CriticalErrorEvent{
				Err: fmt.Errorf("unknown error type %d: %w", errTyp, err),
			})
			return nil, err
		}
	}
	e.emitter.Emit(ctx, fcEvent)

	return &BuildStartResult{
		Info:         eth.PayloadInfo{ID: id, Timestamp: uint64(attrs.Attributes.Timestamp)},
		BuildStarted: buildStartTime,
		Parent:       attrs.Parent,
	}, nil
}

// StartBuild starts a block-building job directly, bypassing the event system.
// Acquires e.mu. Returns the build result or an error.
// Emits ForkchoiceUpdateEvent for other listeners.
// On error, emits appropriate error events before returning.
// Does NOT emit BuildStartedEvent.
func (e *EngineController) StartBuild(ctx context.Context, attrs *derive.AttributesWithParent) (*BuildStartResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.startBuild(ctx, attrs)
}
