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

// BuildSealedEvent is emitted by the engine when a payload finished building,
// but is not locally inserted as canonical block yet
type BuildSealedEvent struct {
	// if payload should be promoted to (local) safe (must also be pending safe, see DerivedFrom)
	Concluding bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom  eth.L1BlockRef
	BuildStarted time.Time

	Info     eth.PayloadInfo
	Envelope *eth.ExecutionPayloadEnvelope
	Ref      eth.L2BlockRef
}

func (ev BuildSealedEvent) String() string {
	return "build-sealed"
}

// BuildSeal seals a block and returns the result or error
func (eq *EngDeriver) BuildSeal(ctx context.Context, info eth.PayloadInfo, buildStarted time.Time, concluding bool, derivedFrom eth.L1BlockRef) (*BuildSealedEvent, error) {
	rpcCtx, cancel := context.WithTimeout(eq.ctx, buildSealTimeout)
	defer cancel()

	sealingStart := time.Now()
	envelope, err := eq.ec.engine.GetPayload(rpcCtx, info)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) && eth.ErrorCode(rpcErr.ErrorCode()) == eth.UnknownPayload {
			eq.log.Warn("Cannot seal block, payload ID is unknown",
				"payloadID", info.ID, "payload_time", info.Timestamp,
				"started_time", buildStarted)
		}
		// Although the engine will very likely not be able to continue from here with the same building job,
		// we still call it "temporary", since the exact same payload-attributes have not been invalidated in-consensus.
		// So the user (attributes-handler or sequencer) should be able to re-attempt the exact
		// same attributes with a new block-building job from here to recover from this error.
		// We name it "expired", as this generally identifies a timeout, unknown job, or otherwise invalidated work.
		eq.emitter.Emit(ctx, PayloadSealExpiredErrorEvent{
			Info:        info,
			Err:         fmt.Errorf("failed to seal execution payload (ID: %s): %w", info.ID, err),
			Concluding:  concluding,
			DerivedFrom: derivedFrom,
		})
		return nil, err
	}

	if err := sanityCheckPayload(envelope.ExecutionPayload); err != nil {
		eq.emitter.Emit(ctx, PayloadSealInvalidEvent{
			Info: info,
			Err: fmt.Errorf("failed sanity-check of execution payload contents (ID: %s, blockhash: %s): %w",
				info.ID, envelope.ExecutionPayload.BlockHash, err),
			Concluding:  concluding,
			DerivedFrom: derivedFrom,
		})
		return nil, err
	}

	ref, err := derive.PayloadToBlockRef(eq.cfg, envelope.ExecutionPayload)
	if err != nil {
		eq.emitter.Emit(ctx, PayloadSealInvalidEvent{
			Info:        info,
			Err:         fmt.Errorf("failed to decode L2 block ref from payload: %w", err),
			Concluding:  concluding,
			DerivedFrom: derivedFrom,
		})
		return nil, err
	}

	now := time.Now()
	sealTime := now.Sub(sealingStart)
	buildTime := sealingStart.Sub(buildStarted) // TODO: when we add interrupts (see upstream cannon v2 PR), this will be inaccurate
	// Ensure the metrics don't compare timestamps that may be 0
	if !buildStarted.IsZero() {
		eq.metrics.RecordSequencerBuildingDiffTime(buildTime)
	}
	if sealTime > 0 {
		eq.metrics.RecordSequencerSealingTime(sealTime)
	}

	eq.log.Info("built new block", "id", envelope.ExecutionPayload.ID(), "txs", len(envelope.ExecutionPayload.Transactions),
		"time", uint64(envelope.ExecutionPayload.Timestamp), "build_time", buildTime, "seal_time", sealTime)

	result := &BuildSealedEvent{
		Concluding:   concluding,
		DerivedFrom:  derivedFrom,
		BuildStarted: buildStarted,
		Info:         info,
		Envelope:     envelope,
		Ref:          ref,
	}

	return result, nil
}

func (eq *EngDeriver) onBuildSealed(ctx context.Context, ev BuildSealedEvent) {
	// If a (pending) safe block, immediately process the block
	if ev.DerivedFrom != (eth.L1BlockRef{}) {
		eq.emitter.Emit(ctx, PayloadProcessEvent{
			Concluding:   ev.Concluding,
			DerivedFrom:  ev.DerivedFrom,
			Envelope:     ev.Envelope,
			Ref:          ev.Ref,
			BuildStarted: ev.BuildStarted,
		})
	}
}
