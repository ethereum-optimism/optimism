package engine

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type PayloadSuccessEvent struct {
	// if payload should be promoted to (local) safe (must also be pending safe, see DerivedFrom)
	Concluding bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom   eth.L1BlockRef
	BuildStarted  time.Time
	InsertStarted time.Time

	Envelope *eth.ExecutionPayloadEnvelope
	Ref      eth.L2BlockRef
}

func (ev PayloadSuccessEvent) String() string {
	return "payload-success"
}

func (e *EngineController) onPayloadSuccess(ctx context.Context, ev PayloadSuccessEvent) {
	if ev.DerivedFrom == ReplaceBlockSource {
		e.log.Warn("Successfully built replacement block, resetting chain to continue now", "replacement", ev.Ref)
		// Change the engine state to make the replacement block the cross-safe head of the chain,
		// And continue syncing from there.
		e.forceReset(ctx, ev.Ref, ev.Ref, ev.Ref, ev.Ref, e.Finalized())
		e.emitter.Emit(ctx, InteropReplacedBlockEvent{
			Envelope: ev.Envelope,
			Ref:      ev.Ref.BlockRef(),
		})
		// Apply it to the execution engine
		e.tryUpdateEngine(ctx)
		// Not a regular reset, since we don't wind back to any L2 block.
		// We start specifically from the replacement block.
		return
	}

	// TryUpdateUnsafe, TryUpdatePendingSafe, TryUpdateLocalSafe, tryUpdateEngine must be sequentially invoked
	e.tryUpdateUnsafe(ctx, ev.Ref)
	// If derived from L1, then it can be considered (pending) safe
	if ev.DerivedFrom != (eth.L1BlockRef{}) {
		e.tryUpdatePendingSafe(ctx, ev.Ref, ev.Concluding, ev.DerivedFrom)
		e.tryUpdateLocalSafe(ctx, ev.Ref, ev.Concluding, ev.DerivedFrom)
	}
	// Now if possible synchronously call FCU
	err := e.tryUpdateEngineInternal(ctx)
	if err != nil {
		e.log.Error("Failed to update engine", "error", err)
	} else {
		updateEngineFinish := time.Now()
		e.logBlockProcessingMetrics(updateEngineFinish, ev)
	}
}

func (e *EngineController) logBlockProcessingMetrics(updateEngineFinish time.Time, ev PayloadSuccessEvent) {
	// Protect against nil pointer dereferences
	if ev.Envelope == nil || ev.Envelope.ExecutionPayload == nil {
		e.log.Debug("Envelope.ExecutionPayload not found, skipping block processing metrics")
		return
	}

	buildTime := ev.InsertStarted.Sub(ev.BuildStarted)
	insertTime := updateEngineFinish.Sub(ev.InsertStarted)

	var totalTime time.Duration
	if !ev.BuildStarted.IsZero() {
		totalTime = updateEngineFinish.Sub(ev.BuildStarted)
	} else {
		totalTime = insertTime
	}

	// Protect against divide-by-zero
	var mgasps float64
	if totalTime > 0 {
		mgasps = float64(ev.Envelope.ExecutionPayload.GasUsed) * 1000 / float64(totalTime)
	}

	e.log.Info("Inserted new L2 unsafe block",
		"hash", ev.Envelope.ExecutionPayload.BlockHash,
		"number", uint64(ev.Envelope.ExecutionPayload.BlockNumber),
		"build_time", common.PrettyDuration(buildTime),
		"insert_time", common.PrettyDuration(insertTime),
		"total_time", common.PrettyDuration(totalTime),
		"mgas", float64(ev.Envelope.ExecutionPayload.GasUsed)/1000000,
		"mgasps", mgasps,
	)
}
