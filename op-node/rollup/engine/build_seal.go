package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// PayloadSealInvalidEvent identifies a permanent in-consensus problem with the payload sealing.
type PayloadSealInvalidEvent struct {
	Info eth.PayloadInfo
	Err  error

	IsLastInSpan bool
	DerivedFrom  eth.L1BlockRef
}

func (ev PayloadSealInvalidEvent) String() string {
	return "payload-seal-invalid"
}

// PayloadSealTemporaryErrorEvent identifies temporarily failed payload-sealing.
// The user should re-attempt by starting a new build process.
type PayloadSealTemporaryErrorEvent struct {
	Info eth.PayloadInfo
	Err  error

	IsLastInSpan bool
	DerivedFrom  eth.L1BlockRef
}

func (ev PayloadSealTemporaryErrorEvent) String() string {
	return "payload-seal-temporary-error"
}

type BuildSealEvent struct {
	Info         eth.PayloadInfo
	BuildStarted time.Time
	// if payload should be promoted to safe (must also be pending safe, see DerivedFrom)
	IsLastInSpan bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom eth.L1BlockRef
}

func (ev BuildSealEvent) String() string {
	return "build-seal"
}

func (eq *EngDeriver) onBuildSeal(ev BuildSealEvent) {
	ctx, cancel := context.WithTimeout(eq.ctx, buildSealTimeout)
	defer cancel()

	sealingStart := time.Now()
	envelope, err := eq.ec.engine.GetPayload(ctx, ev.Info)
	if err != nil {
		if x, ok := err.(eth.InputError); ok && x.Code == eth.UnknownPayload { //nolint:all
			eq.log.Warn("Cannot seal block, payload ID is unknown",
				"payloadID", ev.Info.ID, "payload_time", ev.Info.Timestamp,
				"started_time", ev.BuildStarted)
		}
		// As verifier it is safe to ignore this event, attributes will be re-attempted,
		// and any invalid-attributes error should be raised upon
		// the start of block-building and/or later block insertion.
		eq.emitter.Emit(PayloadSealTemporaryErrorEvent{
			Info:         ev.Info,
			Err:          fmt.Errorf("failed to seal execution payload (ID: %s): %w", ev.Info.ID, err),
			IsLastInSpan: ev.IsLastInSpan,
			DerivedFrom:  ev.DerivedFrom,
		})
		return
	}

	if err := sanityCheckPayload(envelope.ExecutionPayload); err != nil {
		eq.emitter.Emit(PayloadSealInvalidEvent{
			Info: ev.Info,
			Err: fmt.Errorf("failed sanity-check of execution payload contents (ID: %s, blockhash: %s): %w",
				ev.Info.ID, envelope.ExecutionPayload.BlockHash, err),
			IsLastInSpan: ev.IsLastInSpan,
			DerivedFrom:  ev.DerivedFrom,
		})
		return
	}

	ref, err := derive.PayloadToBlockRef(eq.cfg, envelope.ExecutionPayload)
	if err != nil {
		eq.emitter.Emit(PayloadSealInvalidEvent{
			Info:         ev.Info,
			Err:          fmt.Errorf("failed to decode L2 block ref from payload: %w", err),
			IsLastInSpan: ev.IsLastInSpan,
			DerivedFrom:  ev.DerivedFrom,
		})
		return
	}

	now := time.Now()
	sealTime := now.Sub(sealingStart)
	buildTime := now.Sub(ev.BuildStarted)
	eq.metrics.RecordSequencerSealingTime(sealTime)
	eq.metrics.RecordSequencerBuildingDiffTime(buildTime - time.Duration(eq.cfg.BlockTime)*time.Second)

	txnCount := len(envelope.ExecutionPayload.Transactions)
	eq.metrics.CountSequencedTxs(txnCount)

	eq.log.Debug("Processed new L2 block", "l2_unsafe", ref, "l1_origin", ref.L1Origin,
		"txs", txnCount, "time", ref.Time, "seal_time", sealTime, "build_time", buildTime)

	eq.emitter.Emit(BuildSealedEvent{
		IsLastInSpan: ev.IsLastInSpan,
		DerivedFrom:  ev.DerivedFrom,
		Info:         ev.Info,
		Envelope:     envelope,
		Ref:          ref,
	})
}
