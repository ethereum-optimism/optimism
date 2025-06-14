package rwel

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type TriggerSyncEvent struct {
	Target eth.BlockID
}

func (ev TriggerSyncEvent) String() string {
	return "trigger-sync"
}

func (e *RWEL) onTriggerSync(ctx context.Context, ev TriggerSyncEvent) {
	fc := eth.ForkchoiceState{
		HeadBlockHash:      ev.Target.Hash,
		SafeBlockHash:      e.state.CrossSafe().Hash,
		FinalizedBlockHash: e.state.Finalized().Hash,
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
					Err: fmt.Errorf("sync forkchoice update was inconsistent with engine: %w", err),
				})
			default:
				e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
					Err: fmt.Errorf("unexpected error code in sync forkchoice-updated response: %w", err),
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
	switch fcRes.PayloadStatus.Status {
	case eth.ExecutionSyncing:
		e.state.SetIsSyncing(true)
		e.state.SetSyncTarget(ev.Target)
		e.emitter.Emit(ctx, SyncingUpdateEvent{
			SyncTarget: e.state.SyncTarget(),
		})
	case eth.ExecutionValid:
		if latest := fcRes.PayloadStatus.LatestValidHash; latest != nil && ev.Target.Hash != *latest {
			e.log.Warn("Engine already synced to different block", "latest", *latest, "expected", ev.Target)
		}
		e.state.SetSyncTarget(eth.BlockID{})
		e.emitter.Emit(ctx, PollLocalUnsafeRequestEvent{})
	default:
		e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("unexpected engine status: %w", eth.ForkchoiceUpdateErr(fcRes.PayloadStatus)),
		})
	}
}
