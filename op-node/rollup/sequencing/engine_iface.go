package sequencing

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SequencerEngine provides direct-call methods for the sequencer to interact
// with the engine controller, bypassing the event system for latency-critical
// block production. The event system is still used for derivation.
//
// Outcomes reach the caller as return values: none of these methods emit the
// build-lifecycle events their event-driven counterparts do (BuildStartedEvent,
// BuildSealedEvent, PayloadSuccessEvent, or the seal-error events). Two classes
// of event are still emitted, and callers must not treat either as their own
// recovery signal:
//   - state notifications other subsystems depend on (ForkchoiceUpdateEvent,
//     UnsafeUpdateEvent);
//   - engine-internal recovery the caller cannot perform itself
//     (BuildInvalidEvent, PayloadInvalidEvent), plus the generic
//     EngineTemporaryErrorEvent / ResetEvent / CriticalErrorEvent signals that
//     other subsystems act on.
type SequencerEngine interface {
	// RequestForkchoiceUpdate requests the engine to emit a ForkchoiceUpdateEvent,
	// for when no L2 head data is available yet.
	RequestForkchoiceUpdate(ctx context.Context)

	// StartBuild starts a block-building job, returning the payload info and build
	// metadata. Errors: engine.ErrStaleBuild if the parent is no longer the unsafe
	// head (a forkchoice update has been requested, so waiting for the next head
	// update is the way forward), engine.ErrBuildInvalid if the engine rejected the
	// attributes, otherwise a temporary, reset, or critical failure for which the
	// matching event has been emitted.
	StartBuild(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error)

	// SealBuild retrieves a built payload via GetPayload, returning the sealed
	// envelope and block ref. Emits no events. Errors wrap engine.ErrSealExpired
	// (job timed out or is unknown; the same attributes may be rebuilt) or
	// engine.ErrSealInvalid (payload contents rejected), or are
	// engine.ErrStaleBuild when the sealed block no longer extends the unsafe head.
	SealBuild(ctx context.Context, info eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error)

	// ProcessPayload inserts a payload via NewPayload, updates the unsafe head, and
	// finalizes via forkchoice update, emitting UnsafeUpdateEvent on success.
	// Errors: engine.ErrStaleBuild if the payload no longer extends the unsafe head
	// and engine.ErrPayloadDenied if the SuperAuthority denied it, both emitting
	// nothing; engine.ErrPayloadInvalid if the engine rejected the block; otherwise
	// a temporary failure whose EngineTemporaryErrorEvent has been emitted.
	ProcessPayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error
}
