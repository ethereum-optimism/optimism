package engine

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum/go-ethereum/common"
)

// EngineState provides a read-only interface of the forkchoice state properties of the L2 Engine.
type EngineState interface {
	Finalized() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
}

type Engine interface {
	ExecEngine
	derive.L2Source
}

type LocalEngineState interface {
	EngineState

	PendingSafeL2Head() eth.L2BlockRef
	BackupUnsafeL2Head() eth.L2BlockRef
}

type LocalEngineControl interface {
	LocalEngineState
	ResetEngineControl
}

var _ LocalEngineControl = (*EngineController)(nil)
var _ event.Deriver = (*EngineController)(nil)

type ExecEngine interface {
	GetPayload(ctx context.Context, payloadInfo eth.PayloadInfo) (*eth.ExecutionPayloadEnvelope, error)
	ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	NewPayload(ctx context.Context, payload *eth.ExecutionPayload, parentBeaconBlockRoot *common.Hash) (*eth.PayloadStatusV1, error)
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L2BlockRef, error)
}

type SyncDeriver interface {
	OnELSyncStarted()
}

type AttributesForceResetter interface {
	ForceReset(ctx context.Context, localUnsafe, crossUnsafe, localSafe, crossSafe, finalized eth.L2BlockRef)
}

type PipelineForceResetter interface {
	ResetPipeline()
}

type OriginSelectorForceResetter interface {
	ResetOrigins()
}

// CrossUpdateHandler handles both cross-unsafe and cross-safe L2 head changes.
// Nil check required because op-program omits this handler.
type CrossUpdateHandler interface {
	OnCrossUnsafeUpdate(ctx context.Context, crossUnsafe eth.L2BlockRef, localUnsafe eth.L2BlockRef)
	OnCrossSafeUpdate(ctx context.Context, crossSafe eth.L2BlockRef, localSafe eth.L2BlockRef)
}
