package engine

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
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

func (e *EngineController) logBlockProcessingMetrics(updateEngineFinish time.Time, ev PayloadSuccessEvent) {
	// Protect against nil pointer dereferences
	if ev.Envelope == nil || ev.Envelope.ExecutionPayload == nil {
		e.log.Info("Envelope.ExecutionPayload not found, skipping block processing metrics")
		return
	}

	mgas := float64(ev.Envelope.ExecutionPayload.GasUsed) / 1e6
	buildTime := time.Duration(0)
	insertTime := updateEngineFinish.Sub(ev.InsertStarted)
	totalTime := insertTime

	// BuildStarted may be zero if sequencer already built + gossiped a block, but failed during
	// insertion and needed a retry of the insertion. In that case we use the default values above,
	// otherwise we calculate buildTime and totalTime below
	if !ev.BuildStarted.IsZero() {
		buildTime = ev.InsertStarted.Sub(ev.BuildStarted)
		totalTime = updateEngineFinish.Sub(ev.BuildStarted)
	}

	// Protect against divide-by-zero
	var mgasps float64 // Mgas/s
	if totalTime > 0 {
		// Calculate "block-processing" Mgas/s.
		// NOTE: "realtime" mgasps (chain throughput) is a different calculation: (GasUsed / blockPeriod)
		mgasps = mgas / totalTime.Seconds()
	}

	e.log.Info("Inserted new L2 unsafe block",
		"hash", ev.Envelope.ExecutionPayload.BlockHash,
		"number", uint64(ev.Envelope.ExecutionPayload.BlockNumber),
		"build_time", common.PrettyDuration(buildTime),
		"insert_time", common.PrettyDuration(insertTime),
		"total_time", common.PrettyDuration(totalTime),
		"mgas", mgas,
		"mgasps", mgasps,
	)
}
