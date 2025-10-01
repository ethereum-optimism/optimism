package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// PayloadSealInvalidEvent identifies a permanent in-consensus problem with the payload sealing.
type PayloadSealInvalidEvent struct {
	Info eth.PayloadInfo
	Err  error

	Concluding  bool
	DerivedFrom eth.L1BlockRef
}

func (ev PayloadSealInvalidEvent) String() string {
	return "payload-seal-invalid"
}

// PayloadSealExpiredErrorEvent identifies a form of failed payload-sealing that is not coupled
// to the attributes themselves, but rather the build-job process.
// The user should re-attempt by starting a new build process. The payload-sealing job should not be re-attempted,
// as it most likely expired, timed out, or referenced an otherwise invalidated block-building job identifier.
type PayloadSealExpiredErrorEvent struct {
	Info eth.PayloadInfo
	Err  error

	Concluding  bool
	DerivedFrom eth.L1BlockRef
}

func (ev PayloadSealExpiredErrorEvent) String() string {
	return "payload-seal-expired-error"
}

type BuildSealEvent struct {
	Info         eth.PayloadInfo
	BuildStarted time.Time
	// if payload should be promoted to safe (must also be pending safe, see DerivedFrom)
	Concluding bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom eth.L1BlockRef
}

func (ev BuildSealEvent) String() string {
	return "build-seal"
}

func (e *EngineController) onBuildSeal(ctx context.Context, ev BuildSealEvent) {
	rpcCtx, cancel := context.WithTimeout(e.ctx, buildSealTimeout)
	defer cancel()

	sealingStart := time.Now()
	envelope, err := e.engine.GetPayload(rpcCtx, ev.Info)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) && eth.ErrorCode(rpcErr.ErrorCode()) == eth.UnknownPayload {
			e.log.Warn("Cannot seal block, payload ID is unknown",
				"payloadID", ev.Info.ID, "payload_time", ev.Info.Timestamp,
				"started_time", ev.BuildStarted)
		}
		// Although the engine will very likely not be able to continue from here with the same building job,
		// we still call it "temporary", since the exact same payload-attributes have not been invalidated in-consensus.
		// So the user (attributes-handler or sequencer) should be able to re-attempt the exact
		// same attributes with a new block-building job from here to recover from this error.
		// We name it "expired", as this generally identifies a timeout, unknown job, or otherwise invalidated work.
		e.emitter.Emit(ctx, PayloadSealExpiredErrorEvent{
			Info:        ev.Info,
			Err:         fmt.Errorf("failed to seal execution payload (ID: %s): %w", ev.Info.ID, err),
			Concluding:  ev.Concluding,
			DerivedFrom: ev.DerivedFrom,
		})
		return
	}

	if err := sanityCheckPayload(envelope.ExecutionPayload); err != nil {
		e.emitter.Emit(ctx, PayloadSealInvalidEvent{
			Info: ev.Info,
			Err: fmt.Errorf("failed sanity-check of execution payload contents (ID: %s, blockhash: %s): %w",
				ev.Info.ID, envelope.ExecutionPayload.BlockHash, err),
			Concluding:  ev.Concluding,
			DerivedFrom: ev.DerivedFrom,
		})
		return
	}

	ref, err := derive.PayloadToBlockRef(e.rollupCfg, envelope.ExecutionPayload)
	if err != nil {
		e.emitter.Emit(ctx, PayloadSealInvalidEvent{
			Info:        ev.Info,
			Err:         fmt.Errorf("failed to decode L2 block ref from payload: %w", err),
			Concluding:  ev.Concluding,
			DerivedFrom: ev.DerivedFrom,
		})
		return
	}

	now := time.Now()
	sealTime := now.Sub(sealingStart)
	buildTime := now.Sub(ev.BuildStarted)
	e.metrics.RecordSequencerSealingTime(sealTime)
	e.metrics.RecordSequencerBuildingDiffTime(buildTime - time.Duration(e.rollupCfg.BlockTime)*time.Second)

	txnCount := len(envelope.ExecutionPayload.Transactions)
	depositCount, _ := lastDeposit(envelope.ExecutionPayload.Transactions)
	e.metrics.CountSequencedTxsInBlock(txnCount, depositCount)

	e.log.Debug("Built new L2 block", "l2_unsafe", ref, "l1_origin", ref.L1Origin,
		"txs", txnCount, "deposits", depositCount, "time", ref.Time, "seal_time", sealTime, "build_time", buildTime)

	e.emitter.Emit(ctx, BuildSealedEvent{
		Concluding:   ev.Concluding,
		DerivedFrom:  ev.DerivedFrom,
		BuildStarted: ev.BuildStarted,
		Info:         ev.Info,
		Envelope:     envelope,
		Ref:          ref,
	})
}
