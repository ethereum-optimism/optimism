package rwel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

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

func (eq *RWEL) onBuildStart(ctx context.Context, ev BuildStartEvent) {
	rpcCtx, cancel := context.WithTimeout(eq.ctx, buildStartTimeout)
	defer cancel()

	fcEvent := ForkchoiceUpdateEvent{
		LocalUnsafe: ev.Attributes.Parent.BlockRef(),
		CrossSafe:   eq.state.CrossSafe(),
		Finalized:   eq.state.Finalized(),
	}
	if fcEvent.LocalUnsafe.Number < fcEvent.Finalized.Number {
		err := fmt.Errorf("invalid block-building pre-state, unsafe head %s is behind finalized head %s",
			fcEvent.LocalUnsafe, fcEvent.Finalized)
		eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: err})
		return
	}
	fc := eth.ForkchoiceState{
		HeadBlockHash:      fcEvent.LocalUnsafe.Hash,
		SafeBlockHash:      fcEvent.CrossSafe.Hash,
		FinalizedBlockHash: fcEvent.Finalized.Hash,
	}
	buildStartTime := time.Now()

	fcRes, err := eq.engine.ForkchoiceUpdate(rpcCtx, &fc, ev.Attributes.Attributes)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) {
			switch code := eth.ErrorCode(rpcErr.ErrorCode()); code {
			case eth.InvalidForkchoiceState:
				eq.emitter.Emit(ctx, rollup.ResetEvent{
					Err: fmt.Errorf("need reset to resolve pre-state problem: %w", err),
				})
				return
			case eth.InvalidPayloadAttributes:
				eq.state.SetLocalUnsafe(ev.Attributes.Parent.BlockRef())
				eq.emitter.Emit(ctx, fcEvent) // the forkchoice was valid, but the attributes were not
				eq.emitter.Emit(ctx, BuildInvalidEvent{Attributes: ev.Attributes, Err: err})
				return
			default:
				if code.IsEngineError() {
					err = fmt.Errorf("unexpected engine error code in FCU build start response: %w", err)
					eq.emitter.Emit(ctx, BuildInvalidEvent{Attributes: ev.Attributes, Err: err})
					return
				} else {
					eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
						Err: fmt.Errorf("unexpected generic RPC on FCU build start: %w", err),
					})
				}
			}
		} else {
			eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
				Err: fmt.Errorf("temporary error: %w", err),
			})
			return
		}
	}

	eq.state.SetIsSyncing(false)
	eq.state.SetSyncTarget(eth.BlockID{})
	switch fcRes.PayloadStatus.Status {
	case eth.ExecutionInvalid, eth.ExecutionInvalidBlockHash:
		// Even if the building is invalid, the FCU applies, and may cause reorgs and such.
		// Let's check if we're in sync, and try to repair if we mismatch.
		if latest := fcRes.PayloadStatus.LatestValidHash; latest != nil && *latest != (common.Hash{}) && *latest != eq.state.LocalUnsafe().Hash {
			eq.emitter.Emit(ctx, PollLocalUnsafeRequestEvent{})
		}
		err := eth.ForkchoiceUpdateErr(fcRes.PayloadStatus)
		eq.emitter.Emit(ctx, BuildInvalidEvent{Attributes: ev.Attributes, Err: err})
		return
	case eth.ExecutionValid:
		id := fcRes.PayloadID
		if id == nil {
			eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
				Err: fmt.Errorf("unexpected nil payload ID (does the engine support block building?): %w", err),
			})
			return
		}
		// engine-API spec says we will get the FCU head back, let's warn if we don't
		if latest := fcRes.PayloadStatus.LatestValidHash; latest != nil && ev.Attributes.Parent.Hash != *latest {
			eq.log.Warn("Returned latest hash does not match", "latest", *latest, "expected", ev.Attributes.Parent)
		}
		eq.state.SetLocalUnsafe(ev.Attributes.Parent.BlockRef())
		eq.emitter.Emit(ctx, fcEvent)
		eq.emitter.Emit(ctx, BuildStartedEvent{
			Info:         eth.PayloadInfo{ID: *id, Timestamp: uint64(ev.Attributes.Attributes.Timestamp)},
			BuildStarted: buildStartTime,
			DerivedFrom:  ev.Attributes.DerivedFrom,
			Parent:       ev.Attributes.Parent.BlockRef(),
		})
		eq.emitter.Emit(ctx, NoSyncingEvent{
			LocalUnsafe: eq.state.LocalUnsafe(),
		})
	case eth.ExecutionSyncing:
		eq.state.SetIsSyncing(true)
		eq.state.SetSyncTarget(ev.Attributes.Parent.ID())
		eq.emitter.Emit(ctx, SyncingUpdateEvent{
			SyncTarget: eq.state.SyncTarget(),
		})
		eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("node is busy syncing, cannot build block on top of %s", ev.Attributes.Parent),
		})
	default:
		eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: eth.ForkchoiceUpdateErr(fcRes.PayloadStatus),
		})
		return
	}
}
