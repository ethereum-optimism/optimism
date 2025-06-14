package rwel

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type ForkchoiceUpdateRequestEvent struct {
	LocalUnsafe eth.BlockRef
	CrossSafe   eth.BlockRef
	Finalized   eth.BlockRef
}

func (ev ForkchoiceUpdateRequestEvent) String() string {
	return "forkchoice-update-request"
}

func (e *RWEL) onForkchoiceUpdateRequest(ctx context.Context, ev ForkchoiceUpdateRequestEvent) {
	if e.state.IsSyncing() {
		e.log.Warn("Attempting to update forkchoice state while EL syncing")
	}

	if ev.LocalUnsafe.Number < ev.Finalized.Number {
		err := fmt.Errorf("invalid forkchoice state, unsafe head %s is behind finalized head %s",
			ev.LocalUnsafe, ev.Finalized)
		e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: err,
		})
		return
	}

	fc := eth.ForkchoiceState{
		HeadBlockHash:      ev.LocalUnsafe.Hash,
		SafeBlockHash:      ev.CrossSafe.Hash,
		FinalizedBlockHash: ev.Finalized.Hash,
	}
	rpcCtx, cancel := context.WithTimeout(ctx, fcuTimeout)
	defer cancel()
	fcRes, err := e.engine.ForkchoiceUpdate(rpcCtx, &fc, nil)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) {
			switch eth.ErrorCode(rpcErr.ErrorCode()) {
			case eth.InvalidForkchoiceState:
				// TODO improve error, recover
				e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
					Err: fmt.Errorf("no-build forkchoice update was inconsistent with engine: %w", err),
				})
			default:
				e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
					Err: fmt.Errorf("unexpected error code in non-build forkchoice-updated response: %w", err),
				})
			}
		} else {
			e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
				Err: fmt.Errorf("failed to sync forkchoice with engine: %w", err),
			})
		}
		return
	}
	e.state.SetIsSyncing(false)
	e.state.SetSyncTarget(eth.BlockID{})
	switch fcRes.PayloadStatus.Status {
	case eth.ExecutionSyncing:
		e.state.SetIsSyncing(true)
		e.state.SetSyncTarget(ev.LocalUnsafe.ID())
		e.emitter.Emit(ctx, SyncingUpdateEvent{
			SyncTarget: e.state.SyncTarget(),
		})
	case eth.ExecutionValid:
		if e.state.LocalUnsafe() != ev.LocalUnsafe {
			e.state.SetLocalUnsafe(ev.LocalUnsafe)
			e.emitter.Emit(ctx, LocalUnsafeUpdateEvent{
				Ref: e.state.LocalUnsafe(),
			})
		}
		if e.state.CrossSafe() != ev.CrossSafe {
			e.state.SetCrossSafe(ev.CrossSafe)
			e.emitter.Emit(ctx, CrossSafeUpdateEvent{
				CrossSafe: e.state.CrossSafe(),
			})
		}
		if e.state.Finalized() != ev.Finalized {
			e.state.SetCrossSafe(ev.CrossSafe)
			e.emitter.Emit(ctx, FinalizedUpdateEvent{
				Ref: e.state.Finalized(),
			})
		}
		e.emitter.Emit(ctx, ForkchoiceUpdateEvent{
			LocalUnsafe: e.state.LocalUnsafe(),
			CrossSafe:   e.state.CrossSafe(),
			Finalized:   e.state.Finalized(),
		})
		e.emitter.Emit(ctx, NoSyncingEvent{
			LocalUnsafe: e.state.LocalUnsafe(),
		})
	default:
		e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("unexpected engine status: %w", eth.ForkchoiceUpdateErr(fcRes.PayloadStatus)),
		})
	}
}
