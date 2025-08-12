package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

// ReplaceBlockSource is a magic value for the "Source" attribute,
// used when a L2 block is a replacement of an invalidated block.
// After the replacement has been processed, a reset is performed to derive the next L2 blocks.
var ReplaceBlockSource = eth.L1BlockRef{
	Hash:       common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
	Number:     ^uint64(0),
	ParentHash: common.Hash{},
	Time:       0,
}

type Metrics interface {
	CountSequencedTxsInBlock(txns int, deposits int)

	RecordSequencerBuildingDiffTime(duration time.Duration)
	RecordSequencerSealingTime(duration time.Duration)
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

// UnsafeUpdateEvent signals that the given block is now considered safe.
// This is pre-forkchoice update; the change may not be reflected yet in the EL.
type UnsafeUpdateEvent struct {
	Ref eth.L2BlockRef
}

func (ev UnsafeUpdateEvent) String() string {
	return "unsafe-update"
}

// PromoteCrossUnsafeEvent signals that the given block may be promoted to cross-unsafe.
type PromoteCrossUnsafeEvent struct {
	Ref eth.L2BlockRef
}

func (ev PromoteCrossUnsafeEvent) String() string {
	return "promote-cross-unsafe"
}

// CrossUnsafeUpdateEvent signals that the given block is now considered cross-unsafe.
type CrossUnsafeUpdateEvent struct {
	CrossUnsafe eth.L2BlockRef
	LocalUnsafe eth.L2BlockRef
}

func (ev CrossUnsafeUpdateEvent) String() string {
	return "cross-unsafe-update"
}

type PendingSafeUpdateEvent struct {
	PendingSafe eth.L2BlockRef
	Unsafe      eth.L2BlockRef // tip, added to the signal, to determine if there are existing blocks to consolidate
}

func (ev PendingSafeUpdateEvent) String() string {
	return "pending-safe-update"
}

type CrossSafeUpdateEvent struct {
	CrossSafe eth.L2BlockRef
	LocalSafe eth.L2BlockRef
}

func (ev CrossSafeUpdateEvent) String() string {
	return "cross-safe-update"
}

// LocalSafeUpdateEvent signals that a block is now considered to be local-safe.
type LocalSafeUpdateEvent struct {
	Ref    eth.L2BlockRef
	Source eth.L1BlockRef
}

func (ev LocalSafeUpdateEvent) String() string {
	return "local-safe-update"
}

// PromoteSafeEvent signals that a block can be promoted to cross-safe.
type PromoteSafeEvent struct {
	Ref    eth.L2BlockRef
	Source eth.L1BlockRef
}

func (ev PromoteSafeEvent) String() string {
	return "promote-safe"
}

// SafeDerivedEvent signals that a block was determined to be safe, and derived from the given L1 block.
// This is signaled upon successful processing of PromoteSafeEvent.
type SafeDerivedEvent struct {
	Safe   eth.L2BlockRef
	Source eth.L1BlockRef
}

func (ev SafeDerivedEvent) String() string {
	return "safe-derived"
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

type TryUpdateEngineEvent struct {
	// These fields will be zero-value (BuildStarted,InsertStarted=time.Time{}, Envelope=nil) if
	// this event is emitted outside of engineDeriver.onPayloadSuccess
	BuildStarted  time.Time
	InsertStarted time.Time
	Envelope      *eth.ExecutionPayloadEnvelope
}

func (ev TryUpdateEngineEvent) String() string {
	return "try-update-engine"
}

// Checks for the existence of the Envelope field, which is only
// added by the PayloadSuccessEvent
func (ev TryUpdateEngineEvent) triggeredByPayloadSuccess() bool {
	return ev.Envelope != nil
}

// Returns key/value pairs that can be logged and are useful for plotting
// block build/insert time as a way to measure performance.
func (ev TryUpdateEngineEvent) getBlockProcessingMetrics() []interface{} {
	fcuFinish := time.Now()
	payload := ev.Envelope.ExecutionPayload

	logValues := []interface{}{
		"hash", payload.BlockHash,
		"number", uint64(payload.BlockNumber),
		"state_root", payload.StateRoot,
		"timestamp", uint64(payload.Timestamp),
		"parent", payload.ParentHash,
		"prev_randao", payload.PrevRandao,
		"fee_recipient", payload.FeeRecipient,
		"txs", len(payload.Transactions),
	}

	var totalTime time.Duration
	var mgasps float64
	if !ev.BuildStarted.IsZero() {
		totalTime = fcuFinish.Sub(ev.BuildStarted)
		logValues = append(logValues,
			"build_time", common.PrettyDuration(ev.InsertStarted.Sub(ev.BuildStarted)),
			"insert_time", common.PrettyDuration(fcuFinish.Sub(ev.InsertStarted)),
		)
	} else if !ev.InsertStarted.IsZero() {
		totalTime = fcuFinish.Sub(ev.InsertStarted)
	}

	// Avoid divide-by-zero for mgasps
	if totalTime > 0 {
		mgasps = float64(payload.GasUsed) * 1000 / float64(totalTime)
	}

	logValues = append(logValues,
		"total_time", common.PrettyDuration(totalTime),
		"mgas", float64(payload.GasUsed)/1000000,
		"mgasps", mgasps,
	)

	return logValues
}

type EngineResetConfirmedEvent struct {
	LocalUnsafe eth.L2BlockRef
	CrossUnsafe eth.L2BlockRef
	LocalSafe   eth.L2BlockRef
	CrossSafe   eth.L2BlockRef
	Finalized   eth.L2BlockRef
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

// FinalizedUpdateEvent signals that a block has been marked as finalized.
type FinalizedUpdateEvent struct {
	Ref eth.L2BlockRef
}

func (ev FinalizedUpdateEvent) String() string {
	return "finalized-update"
}

// CrossUpdateRequestEvent triggers update events to be emitted, repeating the current state.
type CrossUpdateRequestEvent struct {
	CrossUnsafe bool
	CrossSafe   bool
}

func (ev CrossUpdateRequestEvent) String() string {
	return "cross-update-request"
}

// InteropInvalidateBlockEvent is emitted when a block needs to be invalidated, and a replacement is needed.
type InteropInvalidateBlockEvent struct {
	Invalidated eth.BlockRef
	Attributes  *derive.AttributesWithParent
}

func (ev InteropInvalidateBlockEvent) String() string {
	return "interop-invalidate-block"
}

// InteropReplacedBlockEvent is emitted when a replacement is done.
type InteropReplacedBlockEvent struct {
	Ref      eth.BlockRef
	Envelope *eth.ExecutionPayloadEnvelope
}

func (ev InteropReplacedBlockEvent) String() string {
	return "interop-replaced-block"
}

type EngDeriver struct {
	metrics Metrics

	log     log.Logger
	cfg     *rollup.Config
	ec      *EngineController
	ctx     context.Context
	emitter event.Emitter
}

type EngDeriverInterface interface {
	TryUpdatePendingSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef)
	TryUpdateLocalSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef)
}

var _ event.Deriver = (*EngDeriver)(nil)
var _ EngDeriverInterface = (*EngDeriver)(nil)

func NewEngDeriver(log log.Logger, ctx context.Context, cfg *rollup.Config,
	metrics Metrics, ec *EngineController,
) *EngDeriver {
	return &EngDeriver{
		log:     log,
		cfg:     cfg,
		ec:      ec,
		ctx:     ctx,
		metrics: metrics,
	}
}

func (d *EngDeriver) AttachEmitter(em event.Emitter) {
	d.emitter = em
}

func (d *EngDeriver) OnEvent(ctx context.Context, ev event.Event) bool {
	d.ec.mu.Lock()
	defer d.ec.mu.Unlock()
	// TODO(#16917) Remove Event System Refactor Comments
	//  PromoteUnsafeEvent, PromotePendingSafeEvent, PromoteLocalSafeEvent fan out is updated to procedural method calls
	switch x := ev.(type) {
	case TryUpdateEngineEvent:
		d.TryUpdateEngine(ctx, x)
	case ProcessUnsafePayloadEvent:
		ref, err := derive.PayloadToBlockRef(d.cfg, x.Envelope.ExecutionPayload)
		if err != nil {
			d.log.Error("failed to decode L2 block ref from payload", "err", err)
			return true
		}
		// Avoid re-processing the same unsafe payload if it has already been processed. Because a FCU event emits the ProcessUnsafePayloadEvent
		// it is possible to have multiple queueed up ProcessUnsafePayloadEvent for the same L2 block. This becomes an issue when processing
		// a large number of unsafe payloads at once (like when iterating through the payload queue after the safe head has advanced).
		if ref.BlockRef().ID() == d.ec.UnsafeL2Head().BlockRef().ID() {
			return true
		}
		if err := d.ec.InsertUnsafePayload(d.ctx, x.Envelope, ref); err != nil {
			d.log.Info("failed to insert payload", "ref", ref,
				"txs", len(x.Envelope.ExecutionPayload.Transactions), "err", err)
			// yes, duplicate error-handling. After all derivers are interacting with the engine
			// through events, we can drop the engine-controller interface:
			// unify the events handler with the engine-controller,
			// remove a lot of code, and not do this error translation.
			if errors.Is(err, derive.ErrReset) {
				d.emitter.Emit(ctx, rollup.ResetEvent{Err: err})
			} else if errors.Is(err, derive.ErrTemporary) {
				d.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: err})
			} else {
				d.emitter.Emit(ctx, rollup.CriticalErrorEvent{
					Err: fmt.Errorf("unexpected InsertUnsafePayload error type: %w", err),
				})
			}
		} else {
			d.log.Info("successfully processed payload", "ref", ref, "txs", len(x.Envelope.ExecutionPayload.Transactions))
		}
	case ForkchoiceRequestEvent:
		d.emitter.Emit(ctx, ForkchoiceUpdateEvent{
			UnsafeL2Head:    d.ec.UnsafeL2Head(),
			SafeL2Head:      d.ec.SafeL2Head(),
			FinalizedL2Head: d.ec.Finalized(),
		})
	case rollup.ForceResetEvent:
		ForceEngineReset(d.ec, x)

		// Time to apply the changes to the underlying engine
		d.emitter.Emit(ctx, TryUpdateEngineEvent{})

		v := EngineResetConfirmedEvent{
			LocalUnsafe: d.ec.UnsafeL2Head(),
			CrossUnsafe: d.ec.CrossUnsafeL2Head(),
			LocalSafe:   d.ec.LocalSafeL2Head(),
			CrossSafe:   d.ec.SafeL2Head(),
			Finalized:   d.ec.Finalized(),
		}
		// We do not emit the original event values, since those might not be set (optional attributes).
		d.emitter.Emit(ctx, v)
		d.log.Info("Reset of Engine is completed",
			"local_unsafe", v.LocalUnsafe,
			"cross_unsafe", v.CrossUnsafe,
			"local_safe", v.LocalSafe,
			"cross_safe", v.CrossSafe,
			"finalized", v.Finalized,
		)
	case UnsafeUpdateEvent:
		// pre-interop everything that is local-unsafe is also immediately cross-unsafe.
		if !d.cfg.IsInterop(x.Ref.Time) {
			d.emitter.Emit(ctx, PromoteCrossUnsafeEvent(x))
		}
		// Try to apply the forkchoice changes
		d.emitter.Emit(ctx, TryUpdateEngineEvent{})
	case PromoteCrossUnsafeEvent:
		d.ec.SetCrossUnsafeHead(x.Ref)
		d.emitter.Emit(ctx, CrossUnsafeUpdateEvent{
			CrossUnsafe: x.Ref,
			LocalUnsafe: d.ec.UnsafeL2Head(),
		})
	case PendingSafeRequestEvent:
		d.emitter.Emit(ctx, PendingSafeUpdateEvent{
			PendingSafe: d.ec.PendingSafeL2Head(),
			Unsafe:      d.ec.UnsafeL2Head(),
		})
	case LocalSafeUpdateEvent:
		// pre-interop everything that is local-safe is also immediately cross-safe.
		if !d.cfg.IsInterop(x.Ref.Time) {
			d.emitter.Emit(ctx, PromoteSafeEvent(x))
		}
	case PromoteSafeEvent:
		d.log.Debug("Updating safe", "safe", x.Ref, "unsafe", d.ec.UnsafeL2Head())
		d.ec.SetSafeHead(x.Ref)
		// Finalizer can pick up this safe cross-block now
		d.emitter.Emit(ctx, SafeDerivedEvent{Safe: x.Ref, Source: x.Source})
		d.emitter.Emit(ctx, CrossSafeUpdateEvent{
			CrossSafe: d.ec.SafeL2Head(),
			LocalSafe: d.ec.LocalSafeL2Head(),
		})
		if x.Ref.Number > d.ec.crossUnsafeHead.Number {
			d.log.Debug("Cross Unsafe Head is stale, updating to match cross safe", "cross_unsafe", d.ec.crossUnsafeHead, "cross_safe", x.Ref)
			d.ec.SetCrossUnsafeHead(x.Ref)
			d.emitter.Emit(ctx, CrossUnsafeUpdateEvent{
				CrossUnsafe: x.Ref,
				LocalUnsafe: d.ec.UnsafeL2Head(),
			})
		}
		// Try to apply the forkchoice changes
		d.emitter.Emit(ctx, TryUpdateEngineEvent{})
	case PromoteFinalizedEvent:
		if x.Ref.Number < d.ec.Finalized().Number {
			d.log.Error("Cannot rewind finality,", "ref", x.Ref, "finalized", d.ec.Finalized())
			return true
		}
		if x.Ref.Number > d.ec.SafeL2Head().Number {
			d.log.Error("Block must be safe before it can be finalized", "ref", x.Ref, "safe", d.ec.SafeL2Head())
			return true
		}
		d.ec.SetFinalizedHead(x.Ref)
		d.emitter.Emit(ctx, FinalizedUpdateEvent(x))
		// Try to apply the forkchoice changes
		d.emitter.Emit(ctx, TryUpdateEngineEvent{})
	case CrossUpdateRequestEvent:
		if x.CrossUnsafe {
			d.emitter.Emit(ctx, CrossUnsafeUpdateEvent{
				CrossUnsafe: d.ec.CrossUnsafeL2Head(),
				LocalUnsafe: d.ec.UnsafeL2Head(),
			})
		}
		if x.CrossSafe {
			d.emitter.Emit(ctx, CrossSafeUpdateEvent{
				CrossSafe: d.ec.SafeL2Head(),
				LocalSafe: d.ec.LocalSafeL2Head(),
			})
		}
	case InteropInvalidateBlockEvent:
		d.emitter.Emit(ctx, BuildStartEvent{Attributes: x.Attributes})
	case BuildStartEvent:
		d.onBuildStart(ctx, x)
	case BuildStartedEvent:
		d.onBuildStarted(ctx, x)
	case BuildSealEvent:
		d.onBuildSeal(ctx, x)
	case BuildSealedEvent:
		d.onBuildSealed(ctx, x)
	case BuildInvalidEvent:
		d.onBuildInvalid(ctx, x)
	case BuildCancelEvent:
		d.onBuildCancel(ctx, x)
	case PayloadProcessEvent:
		d.onPayloadProcess(ctx, x)
	case PayloadSuccessEvent:
		d.onPayloadSuccess(ctx, x)
	case PayloadInvalidEvent:
		d.onPayloadInvalid(ctx, x)
	default:
		return false
	}
	return true
}

type ResetEngineControl interface {
	SetUnsafeHead(eth.L2BlockRef)
	SetCrossUnsafeHead(ref eth.L2BlockRef)
	SetLocalSafeHead(ref eth.L2BlockRef)
	SetSafeHead(eth.L2BlockRef)
	SetFinalizedHead(eth.L2BlockRef)
	SetBackupUnsafeL2Head(block eth.L2BlockRef, triggerReorg bool)
	SetPendingSafeL2Head(eth.L2BlockRef)
}

func ForceEngineReset(ec ResetEngineControl, x rollup.ForceResetEvent) {
	ec.SetUnsafeHead(x.LocalUnsafe)

	// cross-safe is fine to revert back, it does not affect engine logic, just sync-status
	ec.SetCrossUnsafeHead(x.CrossUnsafe)

	// derivation continues at local-safe point
	ec.SetLocalSafeHead(x.LocalSafe)
	ec.SetPendingSafeL2Head(x.LocalSafe)

	// "safe" in RPC terms is cross-safe
	ec.SetSafeHead(x.CrossSafe)

	// finalized head
	ec.SetFinalizedHead(x.Finalized)

	ec.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)
}

// Signals that the given block may now become a canonical unsafe block.
// This is pre-forkchoice update; the change may not be reflected yet in the EL.
func (d *EngDeriver) TryUpdateUnsafe(ctx context.Context, ref eth.L2BlockRef) {
	// Backup unsafeHead when new block is not built on original unsafe head.
	if d.ec.unsafeHead.Number >= ref.Number {
		d.ec.SetBackupUnsafeL2Head(d.ec.unsafeHead, false)
	}
	d.ec.SetUnsafeHead(ref)
	d.emitter.Emit(ctx, UnsafeUpdateEvent{Ref: ref})
}

func (d *EngDeriver) TryUpdatePendingSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
	// Only promote if not already stale.
	// Resets/overwrites happen through engine-resets, not through promotion.
	if ref.Number > d.ec.PendingSafeL2Head().Number {
		d.log.Debug("Updating pending safe", "pending_safe", ref, "local_safe", d.ec.LocalSafeL2Head(), "unsafe", d.ec.UnsafeL2Head(), "concluding", concluding)
		d.ec.SetPendingSafeL2Head(ref)
		d.emitter.Emit(ctx, PendingSafeUpdateEvent{
			PendingSafe: d.ec.PendingSafeL2Head(),
			Unsafe:      d.ec.UnsafeL2Head(),
		})
	}
}

func (d *EngDeriver) TryUpdateLocalSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
	if concluding && ref.Number > d.ec.LocalSafeL2Head().Number {
		// Promote to local safe
		d.log.Debug("Updating local safe", "local_safe", ref, "safe", d.ec.SafeL2Head(), "unsafe", d.ec.UnsafeL2Head())
		d.ec.SetLocalSafeHead(ref)
		d.emitter.Emit(ctx, LocalSafeUpdateEvent{Ref: ref, Source: source})
	}
}
