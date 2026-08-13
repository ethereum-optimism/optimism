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
	e.finalizePayload(ctx, ev.Ref, ev.Concluding, ev.DerivedFrom, ev.Envelope, ev.BuildStarted, ev.InsertStarted)
}

// finalizePayload handles unsafe/safe head updates and the engine forkchoice update after a successful NewPayload.
// It does NOT acquire e.mu (caller is responsible).
func (e *EngineController) finalizePayload(ctx context.Context, ref eth.L2BlockRef, concluding bool, derivedFrom eth.L1BlockRef, envelope *eth.ExecutionPayloadEnvelope, buildStarted time.Time, insertStarted time.Time) {
	if derivedFrom == ReplaceBlockSource {
		e.log.Warn("Successfully built replacement block, resetting chain to continue now", "replacement", ref)
		// Change the engine state to make the replacement block the cross-safe head of the chain,
		// And continue syncing from there.
		e.forceReset(ctx, ref, ref, ref, ref, e.Finalized(), false)
		e.emitter.Emit(ctx, InteropReplacedBlockEvent{
			Envelope: envelope,
			Ref:      ref.BlockRef(),
		})
		// Apply it to the execution engine
		e.tryUpdateEngine(ctx)
		// Not a regular reset, since we don't wind back to any L2 block.
		// We start specifically from the replacement block.
		return
	}

	// TryUpdateUnsafe, TryUpdatePendingSafe, TryUpdateLocalSafe, tryUpdateEngine must be sequentially invoked
	e.tryUpdateUnsafe(ctx, ref)
	// If derived from L1, then it can be considered (pending) safe
	if derivedFrom != (eth.L1BlockRef{}) {
		e.tryUpdatePendingSafe(ctx, ref, concluding, derivedFrom)
		e.tryUpdateLocalSafe(ctx, ref, concluding, derivedFrom)
	}
	// Now if possible synchronously call FCU
	err := e.tryUpdateEngineInternal(ctx)
	if err != nil {
		e.log.Error("Failed to update engine", "error", err)
	} else {
		updateEngineFinish := time.Now()
		e.logBlockProcessingMetrics(updateEngineFinish, envelope, buildStarted, insertStarted)
	}
}

func (e *EngineController) logBlockProcessingMetrics(updateEngineFinish time.Time, envelope *eth.ExecutionPayloadEnvelope, buildStarted, insertStarted time.Time) {
	// Protect against nil pointer dereferences
	if envelope == nil || envelope.ExecutionPayload == nil {
		e.log.Info("Envelope.ExecutionPayload not found, skipping block processing metrics")
		return
	}

	mgas := float64(envelope.ExecutionPayload.GasUsed) / 1e6
	buildTime := time.Duration(0)
	insertTime := updateEngineFinish.Sub(insertStarted)
	totalTime := insertTime

	// BuildStarted may be zero if sequencer already built + gossiped a block, but failed during
	// insertion and needed a retry of the insertion. In that case we use the default values above,
	// otherwise we calculate buildTime and totalTime below
	if !buildStarted.IsZero() {
		buildTime = insertStarted.Sub(buildStarted)
		totalTime = updateEngineFinish.Sub(buildStarted)
	}

	// Protect against divide-by-zero
	var mgasps float64 // Mgas/s
	if totalTime > 0 {
		// Calculate "block-processing" Mgas/s.
		// NOTE: "realtime" mgasps (chain throughput) is a different calculation: (GasUsed / blockPeriod)
		mgasps = mgas / totalTime.Seconds()
	}

	e.log.Info("Inserted new L2 unsafe block",
		"hash", envelope.ExecutionPayload.BlockHash,
		"number", uint64(envelope.ExecutionPayload.BlockNumber),
		"build_time", common.PrettyDuration(buildTime),
		"insert_time", common.PrettyDuration(insertTime),
		"total_time", common.PrettyDuration(totalTime),
		"mgas", mgas,
		"mgasps", mgasps,
	)
}
