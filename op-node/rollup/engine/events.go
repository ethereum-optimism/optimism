package engine

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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

// BuildHookCtxKey is used to carry a test-only build hook through event contexts.
// The value must be of type: func(context.Context)
type buildHookCtxKeyType struct{}

var BuildHookCtxKey buildHookCtxKeyType

// HookFromContext extracts the test-only build hook from context (may be nil).
func HookFromContext(ctx context.Context) func(context.Context) {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(BuildHookCtxKey).(func(context.Context)); ok {
		return v
	}
	return nil
}

// no local metrics interface; engine depends directly on op-node/metrics.Metricer

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

// PendingSafeUpdateEvent signals that a new pending-safe candidate is selected.
type PendingSafeUpdateEvent struct {
	PendingSafe eth.L2BlockRef
	Unsafe      eth.L2BlockRef // tip, added to the signal, to determine if there are existing blocks to consolidate
}

func (ev PendingSafeUpdateEvent) String() string {
	return "pending-safe-update"
}

// LocalSafeUpdateEvent signals that a block is now considered to be local-safe.
type LocalSafeUpdateEvent struct {
	Ref    eth.L2BlockRef
	Source eth.L1BlockRef
}

func (ev LocalSafeUpdateEvent) String() string {
	return "local-safe-update"
}

// SafeDerivedEvent signals that a block was determined to be safe, and derived from the given L1 block.
// This is signaled upon procedural call of PromoteSafe method
type SafeDerivedEvent struct {
	Safe   eth.L2BlockRef
	Source eth.L1BlockRef
}

func (ev SafeDerivedEvent) String() string {
	return "safe-derived"
}

// ProcessUnsafePayloadEvent signals an unsafe payload to process.
type ProcessUnsafePayloadEvent struct {
	Envelope *eth.ExecutionPayloadEnvelope
}

func (ev ProcessUnsafePayloadEvent) String() string {
	return "process-unsafe-payload"
}

// EngineResetConfirmedEvent signals that an engine reset was completed.
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

// FinalizedUpdateEvent signals that a block has been marked as finalized.
type FinalizedUpdateEvent struct {
	Ref eth.L2BlockRef
}

func (ev FinalizedUpdateEvent) String() string {
	return "finalized-update"
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

// Build lifecycle and payload events

type BuildStartEvent struct {
	Attributes *derive.AttributesWithParent
}

func (ev BuildStartEvent) String() string { return "build-start" }

// Errors and success

type PayloadSealInvalidEvent struct {
	Info eth.PayloadInfo
	Err  error

	Concluding  bool
	DerivedFrom eth.L1BlockRef
}

func (ev PayloadSealInvalidEvent) String() string { return "payload-seal-invalid" }

type PayloadSealExpiredErrorEvent struct {
	Info eth.PayloadInfo
	Err  error

	Concluding  bool
	DerivedFrom eth.L1BlockRef
}

func (ev PayloadSealExpiredErrorEvent) String() string { return "payload-seal-expired-error" }

// Reset control helpers

type ResetEngineControl interface {
	SetUnsafeHead(eth.L2BlockRef)
	SetCrossUnsafeHead(ref eth.L2BlockRef)
	SetLocalSafeHead(ref eth.L2BlockRef)
	SetSafeHead(eth.L2BlockRef)
	SetFinalizedHead(eth.L2BlockRef)
	SetBackupUnsafeL2Head(block eth.L2BlockRef, triggerReorg bool)
	SetPendingSafeL2Head(eth.L2BlockRef)
}

func ForceEngineReset(ec ResetEngineControl, localUnsafe, crossUnsafe, localSafe, crossSafe, finalized eth.L2BlockRef) {
	ec.SetUnsafeHead(localUnsafe)

	// cross-safe is fine to revert back, it does not affect engine logic, just sync-status
	ec.SetCrossUnsafeHead(crossUnsafe)

	// derivation continues at local-safe point
	ec.SetLocalSafeHead(localSafe)
	ec.SetPendingSafeL2Head(localSafe)

	// "safe" in RPC terms is cross-safe
	ec.SetSafeHead(crossSafe)

	// finalized head
	ec.SetFinalizedHead(finalized)

	ec.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)
}

type PayloadInvalidEvent struct {
	Envelope *eth.ExecutionPayloadEnvelope
	Err      error
}

func (ev PayloadInvalidEvent) String() string {
	return "payload-invalid"
}

func (eq *EngineController) onPayloadInvalid(_ context.Context, ev PayloadInvalidEvent) {
	eq.log.Warn("Payload was invalid", "block", ev.Envelope.ExecutionPayload.ID(),
		"err", ev.Err, "timestamp", uint64(ev.Envelope.ExecutionPayload.Timestamp))
}

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

type BuildCancelEvent struct {
	Info  eth.PayloadInfo
	Force bool
}

func (ev BuildCancelEvent) String() string {
	return "build-cancel"
}

// BuildInvalidEvent is an internal engine event, to post-process upon invalid attributes.
// Not for temporary processing problems.
type BuildInvalidEvent struct {
	Attributes *derive.AttributesWithParent
	Err        error
}

func (ev BuildInvalidEvent) String() string {
	return "build-invalid"
}

// InvalidPayloadAttributesEvent is a signal to external derivers that the attributes were invalid.
type InvalidPayloadAttributesEvent struct {
	Attributes *derive.AttributesWithParent
	Err        error
}

func (ev InvalidPayloadAttributesEvent) String() string {
	return "invalid-payload-attributes"
}
