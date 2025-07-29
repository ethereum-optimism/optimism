package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// BuildStartedEvent represents the result of starting a block build
type BuildStartedEvent struct {
	Info eth.PayloadInfo

	BuildStarted time.Time

	Parent eth.L2BlockRef

	// if payload should be promoted to (local) safe (must also be pending safe, see DerivedFrom)
	Concluding bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom eth.L1BlockRef
}

func (ev BuildStartedEvent) String() string {
	return "build-started"
}

// BuildStart initiates block building and returns the result or error
func (eq *EngDeriver) BuildStart(ctx context.Context, attributes *derive.AttributesWithParent) (*BuildStartedEvent, error) {
	rpcCtx, cancel := context.WithTimeout(eq.ctx, buildStartTimeout)
	defer cancel()

	if attributes.DerivedFrom != (eth.L1BlockRef{}) &&
		eq.ec.PendingSafeL2Head().Hash != attributes.Parent.Hash {
		// Warn about small reorgs, happens when pending safe head is getting rolled back
		eq.log.Warn("block-attributes derived from L1 do not build on pending safe head, likely reorg",
			"pending_safe", eq.ec.PendingSafeL2Head(), "attributes_parent", attributes.Parent)
	}

	fcEvent := ForkchoiceUpdateEvent{
		L2ChainState: eth.L2ChainState{
			UnsafeL2Head:    attributes.Parent,
			SafeL2Head:      eq.ec.safeHead,
			FinalizedL2Head: eq.ec.finalizedHead,
		},
	}
	if fcEvent.UnsafeL2Head.Number < fcEvent.FinalizedL2Head.Number {
		err := fmt.Errorf("invalid block-building pre-state, unsafe head %s is behind finalized head %s", fcEvent.UnsafeL2Head, fcEvent.FinalizedL2Head)
		eq.emitter.Emit(ctx, rollup.CriticalErrorEvent{Err: err}) // make the node exit, things are very wrong.
		return nil, err
	}
	fc := eth.ForkchoiceState{
		HeadBlockHash:      fcEvent.UnsafeL2Head.Hash,
		SafeBlockHash:      fcEvent.SafeL2Head.Hash,
		FinalizedBlockHash: fcEvent.FinalizedL2Head.Hash,
	}
	buildStartTime := time.Now()
	id, errTyp, err := startPayload(rpcCtx, eq.ec.engine, fc, attributes.Attributes)
	if err != nil {
		switch errTyp {
		case BlockInsertTemporaryErr:
			// RPC errors are recoverable, we can retry the buffered payload attributes later.
			eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
				Err: fmt.Errorf("temporarily cannot insert new safe block: %w", err),
			})
			return nil, err
		case BlockInsertPrestateErr:
			eq.emitter.Emit(ctx, rollup.ResetEvent{
				Err: fmt.Errorf("need reset to resolve pre-state problem: %w", err),
			})
			return nil, err
		case BlockInsertPayloadErr:
			eq.emitter.Emit(ctx, BuildInvalidEvent{Attributes: attributes, Err: err})
			return nil, err
		default:
			eq.emitter.Emit(ctx, rollup.CriticalErrorEvent{
				Err: fmt.Errorf("unknown error type %d: %w", errTyp, err),
			})
			return nil, err
		}
	}
	eq.emitter.Emit(ctx, fcEvent)

	result := &BuildStartedEvent{
		Info:         eth.PayloadInfo{ID: id, Timestamp: uint64(attributes.Attributes.Timestamp)},
		BuildStarted: buildStartTime,
		Concluding:   attributes.Concluding,
		DerivedFrom:  attributes.DerivedFrom,
		Parent:       attributes.Parent,
	}

	return result, nil
}

// onBuildStarted handles the result of a successful build start
func (eq *EngDeriver) onBuildStarted(ctx context.Context, ev BuildStartedEvent) {
	// If a (pending) safe block, immediately seal the block
	if ev.DerivedFrom != (eth.L1BlockRef{}) {
		if result, err := eq.BuildSeal(ctx, ev.Info, ev.BuildStarted, ev.Concluding, ev.DerivedFrom); err != nil {
			eq.log.Error("Failed to seal block", "err", err)
		} else {
			eq.onBuildSealed(ctx, *result)
		}
	}
}
