package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/async"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type InvalidPayloadEvent struct {
	Envelope *eth.ExecutionPayloadEnvelope
}

func (ev InvalidPayloadEvent) String() string {
	return "invalid-payload"
}

type InvalidPayloadAttributesEvent struct {
	Attributes *derive.AttributesWithParent
}

func (ev InvalidPayloadAttributesEvent) String() string {
	return "invalid-payload-attributes"
}

// ForkchoiceRequestEvent signals to the engine that it should emit an artificial
// forkchoice-update event, to signal the latest forkchoice to other derivers.
// This helps decouple derivers from the actual engine state,
// while also not making the derivers wait for a forkchoice update at random.
type ForkchoiceRequestEvent struct {
}

func (ev ForkchoiceRequestEvent) String() string {
	return "forkchoice-request"
}

type ForkchoiceUpdateEvent struct {
	UnsafeL2Head, SafeL2Head, FinalizedL2Head eth.L2BlockRef
}

func (ev ForkchoiceUpdateEvent) String() string {
	return "forkchoice-update"
}

type PendingSafeUpdateEvent struct {
	PendingSafe eth.L2BlockRef
	Unsafe      eth.L2BlockRef // tip, added to the signal, to determine if there are existing blocks to consolidate
}

func (ev PendingSafeUpdateEvent) String() string {
	return "pending-safe-update"
}

// PromotePendingSafeEvent signals that a block can be marked as pending-safe, and/or safe.
type PromotePendingSafeEvent struct {
	Ref         eth.L2BlockRef
	Safe        bool
	DerivedFrom eth.L1BlockRef
}

func (ev PromotePendingSafeEvent) String() string {
	return "promote-pending-safe"
}

// SafeDerivedEvent signals that a block was determined to be safe, and derived from the given L1 block
type SafeDerivedEvent struct {
	Safe        eth.L2BlockRef
	DerivedFrom eth.L1BlockRef
}

func (ev SafeDerivedEvent) String() string {
	return "safe-derived"
}

type ProcessAttributesEvent struct {
	Attributes *derive.AttributesWithParent
}

func (ev ProcessAttributesEvent) String() string {
	return "process-attributes"
}

type PendingSafeRequestEvent struct {
}

func (ev PendingSafeRequestEvent) String() string {
	return "pending-safe-request"
}

type ProcessUnsafePayloadEvent struct {
	Envelope *eth.ExecutionPayloadEnvelope
}

func (ev ProcessUnsafePayloadEvent) String() string {
	return "process-unsafe-payload"
}

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

type ForceEngineResetEvent struct {
	Unsafe, Safe, Finalized eth.L2BlockRef
}

func (ev ForceEngineResetEvent) String() string {
	return "force-engine-reset"
}

type EngineResetConfirmedEvent struct {
	Unsafe, Safe, Finalized eth.L2BlockRef
}

func (ev EngineResetConfirmedEvent) String() string {
	return "engine-reset-confirmed"
}

// PromoteFinalizedEvent signals that a block can be marked as finalized.
type PromoteFinalizedEvent struct {
	Ref eth.L2BlockRef
}

func (ev PromoteFinalizedEvent) String() string {
	return "promote-finalized"
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
	switch x := ev.(type) {
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
	case ProcessUnsafePayloadEvent:
		ref, err := derive.PayloadToBlockRef(d.cfg, x.Envelope.ExecutionPayload)
		if err != nil {
			d.log.Error("failed to decode L2 block ref from payload", "err", err)
			return
		}
		if err := d.ec.InsertUnsafePayload(d.ctx, x.Envelope, ref); err != nil {
			d.log.Info("failed to insert payload", "ref", ref,
				"txs", len(x.Envelope.ExecutionPayload.Transactions), "err", err)
			// yes, duplicate error-handling. After all derivers are interacting with the engine
			// through events, we can drop the engine-controller interface:
			// unify the events handler with the engine-controller,
			// remove a lot of code, and not do this error translation.
			if errors.Is(err, derive.ErrReset) {
				d.emitter.Emit(rollup.ResetEvent{Err: err})
			} else if errors.Is(err, derive.ErrTemporary) {
				d.emitter.Emit(rollup.EngineTemporaryErrorEvent{Err: err})
			} else {
				d.emitter.Emit(rollup.CriticalErrorEvent{Err: fmt.Errorf("unexpected InsertUnsafePayload error type: %w", err)})
			}
		} else {
			d.log.Info("successfully processed payload", "ref", ref, "txs", len(x.Envelope.ExecutionPayload.Transactions))
		}
	case ForkchoiceRequestEvent:
		d.emitter.Emit(ForkchoiceUpdateEvent{
			UnsafeL2Head:    d.ec.UnsafeL2Head(),
			SafeL2Head:      d.ec.SafeL2Head(),
			FinalizedL2Head: d.ec.Finalized(),
		})
	case ForceEngineResetEvent:
		ForceEngineReset(d.ec, x)

		// Time to apply the changes to the underlying engine
		d.emitter.Emit(TryUpdateEngineEvent{})

		log.Debug("Reset of Engine is completed",
			"safeHead", x.Safe, "unsafe", x.Unsafe, "safe_timestamp", x.Safe.Time,
			"unsafe_timestamp", x.Unsafe.Time)
		d.emitter.Emit(EngineResetConfirmedEvent(x))
	case ProcessAttributesEvent:
		d.onForceNextSafeAttributes(x.Attributes)
	case PendingSafeRequestEvent:
		d.emitter.Emit(PendingSafeUpdateEvent{
			PendingSafe: d.ec.PendingSafeL2Head(),
			Unsafe:      d.ec.UnsafeL2Head(),
		})
	case PromotePendingSafeEvent:
		// Only promote if not already stale.
		// Resets/overwrites happen through engine-resets, not through promotion.
		if x.Ref.Number > d.ec.PendingSafeL2Head().Number {
			d.ec.SetPendingSafeL2Head(x.Ref)
		}
		if x.Safe && x.Ref.Number > d.ec.SafeL2Head().Number {
			d.ec.SetSafeHead(x.Ref)
			d.emitter.Emit(SafeDerivedEvent{Safe: x.Ref, DerivedFrom: x.DerivedFrom})
		}
	case PromoteFinalizedEvent:
		if x.Ref.Number < d.ec.Finalized().Number {
			d.log.Error("Cannot rewind finality,", "ref", x.Ref, "finalized", d.ec.Finalized())
			return
		}
		if x.Ref.Number > d.ec.SafeL2Head().Number {
			d.log.Error("Block must be safe before it can be finalized", "ref", x.Ref, "safe", d.ec.SafeL2Head())
			return
		}
		d.ec.SetFinalizedHead(x.Ref)
	}
}

// onForceNextSafeAttributes inserts the provided attributes, reorging away any conflicting unsafe chain.
func (eq *EngDeriver) onForceNextSafeAttributes(attributes *derive.AttributesWithParent) {
	ctx, cancel := context.WithTimeout(eq.ctx, time.Second*10)
	defer cancel()

	attrs := attributes.Attributes
	errType, err := eq.ec.StartPayload(ctx, eq.ec.PendingSafeL2Head(), attributes, true)
	var envelope *eth.ExecutionPayloadEnvelope
	if err == nil {
		envelope, errType, err = eq.ec.ConfirmPayload(ctx, async.NoOpGossiper{}, &conductor.NoOpConductor{})
	}
	if err != nil {
		switch errType {
		case BlockInsertTemporaryErr:
			// RPC errors are recoverable, we can retry the buffered payload attributes later.
			eq.emitter.Emit(rollup.EngineTemporaryErrorEvent{Err: fmt.Errorf("temporarily cannot insert new safe block: %w", err)})
			return
		case BlockInsertPrestateErr:
			_ = eq.ec.CancelPayload(ctx, true)
			eq.emitter.Emit(rollup.ResetEvent{Err: fmt.Errorf("need reset to resolve pre-state problem: %w", err)})
			return
		case BlockInsertPayloadErr:
			if !errors.Is(err, derive.ErrTemporary) {
				eq.emitter.Emit(InvalidPayloadAttributesEvent{Attributes: attributes})
			}
			_ = eq.ec.CancelPayload(ctx, true)
			eq.log.Warn("could not process payload derived from L1 data, dropping attributes", "err", err)
			// Count the number of deposits to see if the tx list is deposit only.
			depositCount := 0
			for _, tx := range attrs.Transactions {
				if len(tx) > 0 && tx[0] == types.DepositTxType {
					depositCount += 1
				}
			}
			// Deposit transaction execution errors are suppressed in the execution engine, but if the
			// block is somehow invalid, there is nothing we can do to recover & we should exit.
			if len(attrs.Transactions) == depositCount {
				eq.log.Error("deposit only block was invalid", "parent", attributes.Parent, "err", err)
				eq.emitter.Emit(rollup.CriticalErrorEvent{Err: fmt.Errorf("failed to process block with only deposit transactions: %w", err)})
				return
			}
			// Revert the pending safe head to the safe head.
			eq.ec.SetPendingSafeL2Head(eq.ec.SafeL2Head())
			// suppress the error b/c we want to retry with the next batch from the batch queue
			// If there is no valid batch the node will eventually force a deposit only block. If
			// the deposit only block fails, this will return the critical error above.

			// Try to restore to previous known unsafe chain.
			eq.ec.SetBackupUnsafeL2Head(eq.ec.BackupUnsafeL2Head(), true)

			// drop the payload without inserting it into the engine
			return
		default:
			eq.emitter.Emit(rollup.CriticalErrorEvent{Err: fmt.Errorf("unknown InsertHeadBlock error type %d: %w", errType, err)})
		}
	}
	ref, err := derive.PayloadToBlockRef(eq.cfg, envelope.ExecutionPayload)
	if err != nil {
		eq.emitter.Emit(rollup.ResetEvent{Err: fmt.Errorf("failed to decode L2 block ref from payload: %w", err)})
		return
	}
	eq.ec.SetPendingSafeL2Head(ref)
	if attributes.IsLastInSpan {
		eq.ec.SetSafeHead(ref)
		eq.emitter.Emit(SafeDerivedEvent{Safe: ref, DerivedFrom: attributes.DerivedFrom})
	}
	eq.emitter.Emit(PendingSafeUpdateEvent{
		PendingSafe: eq.ec.PendingSafeL2Head(),
		Unsafe:      eq.ec.UnsafeL2Head(),
	})
}

type ResetEngineControl interface {
	SetUnsafeHead(eth.L2BlockRef)
	SetSafeHead(eth.L2BlockRef)
	SetFinalizedHead(eth.L2BlockRef)
	SetBackupUnsafeL2Head(block eth.L2BlockRef, triggerReorg bool)
	SetPendingSafeL2Head(eth.L2BlockRef)
	ResetBuildingState()
}

// ForceEngineReset is not to be used. The op-program needs it for now, until event processing is adopted there.
func ForceEngineReset(ec ResetEngineControl, x ForceEngineResetEvent) {
	ec.SetUnsafeHead(x.Unsafe)
	ec.SetSafeHead(x.Safe)
	ec.SetPendingSafeL2Head(x.Safe)
	ec.SetFinalizedHead(x.Finalized)
	ec.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)
	ec.ResetBuildingState()
}
