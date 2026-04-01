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
type SequencerEngine interface {
	// RequestForkchoiceUpdate requests the engine to emit a ForkchoiceUpdateEvent.
	// Called from Init() and startBuildingBlock() when no L2 head data is available.
	RequestForkchoiceUpdate(ctx context.Context)

	// StartBuild starts a block-building job. Returns the payload info and build metadata.
	// Emits ForkchoiceUpdateEvent for other listeners.
	// On error, emits appropriate error events before returning.
	StartBuild(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error)

	// SealBuild retrieves a built payload via GetPayload. Returns the sealed envelope and block ref.
	// On error, emits PayloadSealExpiredErrorEvent or PayloadSealInvalidEvent before returning.
	SealBuild(ctx context.Context, info eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error)

	// ProcessPayload inserts a payload via NewPayload, updates unsafe head, and finalizes via FCU.
	// Emits UnsafeUpdateEvent on success.
	// On error, emits appropriate error events before returning.
	ProcessPayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error
}
