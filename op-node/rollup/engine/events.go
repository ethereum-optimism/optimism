package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type Resource string

type TryBackupUnsafeReorgEvent struct {
}

func (ev TryBackupUnsafeReorgEvent) String() string {
	return "try-backup-unsafe-reorg"
}

type TryUpdateEngineEvent struct {
}

func (ev TryUpdateEngineEvent) String() string {
	return "try-update-engine"
}

type NewPayloadEvent struct {
	Envelope *eth.ExecutionPayloadEnvelope
}

type ForkchoiceState struct {
	Unsafe    eth.L2BlockRef
	Safe      eth.L2BlockRef
	Finalized eth.L2BlockRef
	// ...
}

type EngineUpdatedEvent struct {
	State ForkchoiceState
}

type EngDeriver struct {
	log     log.Logger
	cfg     *rollup.Config
	ec      *EngineController
	ctx     context.Context
	emitter rollup.EventEmitter
}

var _ rollup.Deriver = (*EngDeriver)(nil)

func NewEngDeriver(log log.Logger, ctx context.Context, cfg *rollup.Config,
	ec *EngineController, emitter rollup.EventEmitter) *EngDeriver {
	return &EngDeriver{
		log:     log,
		cfg:     cfg,
		ec:      ec,
		ctx:     ctx,
		emitter: emitter,
	}
}

func (d *EngDeriver) OnEvent(ev rollup.Event) {
	switch ev.(type) {
	case TryBackupUnsafeReorgEvent:
		// If we don't need to call FCU to restore unsafeHead using backupUnsafe, keep going b/c
		// this was a no-op(except correcting invalid state when backupUnsafe is empty but TryBackupUnsafeReorg called).
		fcuCalled, err := d.ec.TryBackupUnsafeReorg(d.ctx)
		// Dealing with legacy here: it used to skip over the error-handling if fcuCalled was false.
		// But that combination is not actually a code-path in TryBackupUnsafeReorg.
		// We should drop fcuCalled, and make the function emit events directly,
		// once there are no more synchronous callers.
		if !fcuCalled && err != nil {
			d.log.Crit("unexpected TryBackupUnsafeReorg error after no FCU call", "err", err)
		}
		if err != nil {
			// If we needed to perform a network call, then we should yield even if we did not encounter an error.
			if errors.Is(err, derive.ErrReset) {
				d.emitter.Emit(rollup.ResetEvent{Err: err})
			} else if errors.Is(err, derive.ErrTemporary) {
				d.emitter.Emit(rollup.EngineTemporaryErrorEvent{Err: err})
			} else {
				d.emitter.Emit(rollup.CriticalErrorEvent{Err: fmt.Errorf("unexpected TryBackupUnsafeReorg error type: %w", err)})
			}
		}
	case TryUpdateEngineEvent:
		// If we don't need to call FCU, keep going b/c this was a no-op. If we needed to
		// perform a network call, then we should yield even if we did not encounter an error.
		if err := d.ec.TryUpdateEngine(d.ctx); err != nil && !errors.Is(err, ErrNoFCUNeeded) {
			if errors.Is(err, derive.ErrReset) {
				d.emitter.Emit(rollup.ResetEvent{Err: err})
			} else if errors.Is(err, derive.ErrTemporary) {
				d.emitter.Emit(rollup.EngineTemporaryErrorEvent{Err: err})
			} else {
				d.emitter.Emit(rollup.CriticalErrorEvent{Err: fmt.Errorf("unexpected TryUpdateEngine error type: %w", err)})
			}
		}
		// TODO handle more events:
		//case NewPayloadEvent:
		//	ref, err := derive.PayloadToBlockRef(d.cfg, x.Envelope.ExecutionPayload)
		//	err := d.ec.InsertUnsafePayload(ctx, x.Envelope, ref)
		//	// TODO emit events for error / success
		//	// CancelPayload
		//	// SetPendingSafeL2Head
		//	// SetBackupUnsafeL2Head
		//	// SetSafeHead
		//	// InsertUnsafePayload
		//	// SetFinalizedHead
	}
	// TODO emit more events:
	// Emit:
	// - payload processing success
	// - payload processing error
	// - attributes processing success (incl original AttributesWithParent data)
	// - attributes processing error
	// - forkchoice needs update
	// - unsafe head changed
	// - safe head changed
	// - finalized head changed
	//
}
