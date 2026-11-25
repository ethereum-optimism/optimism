package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ErrStaleBuild is returned by the direct-call build methods when the job no
// longer extends the current unsafe head. The caller should wait for the next
// forkchoice update (already requested) and re-attempt on the new head.
var ErrStaleBuild = errors.New("stale block-building job, parent is not the unsafe head")

// ErrBuildInvalid is returned by StartBuild when the engine rejects the payload
// attributes. It is not a temporary failure: the caller must start a new build
// rather than retry the same one.
//
// Unlike the seal errors, the engine keeps emitting BuildInvalidEvent on both
// paths, because onBuildInvalid performs engine-internal recovery that a
// direct caller cannot do for itself (exiting on an invalid deposits-only
// block, reverting the pending-safe head, restoring the backup unsafe head).
// Its downstream InvalidPayloadAttributesEvent is therefore also emitted for
// sequencer-originated builds; recovery stays single-signal because the only
// consumer that acted on it for those builds is the sequencer, which drops the
// handler when it moves to these direct calls.
var ErrBuildInvalid = errors.New("invalid block-building attributes")

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
	if !attrs.IsDerived() && attrs.Parent.ID() != e.unsafeHead.ID() {
		e.log.Warn("dropping stale sequencer build start",
			"attributes_parent", attrs.Parent, "unsafe", e.unsafeHead)
		// Re-emit the forkchoice so the sequencer drops the stale build job and restarts on the current unsafe head.
		e.requestForkchoiceUpdate(ctx)
		return nil, ErrStaleBuild
	}

	rpcCtx, cancel := context.WithTimeout(e.ctx, buildStartTimeout)
	defer cancel()

	if attrs.DerivedFrom != (eth.L1BlockRef{}) &&
		e.pendingSafeHead.Hash != attrs.Parent.Hash {
		// Warn about small reorgs, happens when pending safe head is getting rolled back
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
		e.emitter.Emit(ctx, rollup.CriticalErrorEvent{Err: err}) // make the node exit, things are very wrong.
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
			// RPC errors are recoverable, we can retry the buffered payload attributes later.
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
			return nil, fmt.Errorf("%w: %w", ErrBuildInvalid, err)
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
// On error, emits appropriate error events before returning; a stale parent
// yields ErrStaleBuild after a forkchoice update has been requested.
// Does NOT emit BuildStartedEvent.
func (e *EngineController) StartBuild(ctx context.Context, attrs *derive.AttributesWithParent) (*BuildStartResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.startBuild(ctx, attrs)
}
