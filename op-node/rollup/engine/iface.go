package engine

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup/async"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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

// EngineControl enables other components to build blocks with the Engine,
// while keeping the forkchoice state and payload-id management internal to
// avoid state inconsistencies between different users of the EngineControl.
type EngineControl interface {
	EngineState

	// StartPayload requests the engine to start building a block with the given attributes.
	// If updateSafe, the resulting block will be marked as a safe block.
	StartPayload(ctx context.Context, parent eth.L2BlockRef, attrs *derive.AttributesWithParent, updateSafe bool) (errType BlockInsertionErrType, err error)
	// ConfirmPayload requests the engine to complete the current block. If no block is being built, or if it fails, an error is returned.
	ConfirmPayload(ctx context.Context, agossip async.AsyncGossiper, sequencerConductor conductor.SequencerConductor) (out *eth.ExecutionPayloadEnvelope, errTyp BlockInsertionErrType, err error)
	// CancelPayload requests the engine to stop building the current block without making it canonical.
	// This is optional, as the engine expires building jobs that are left uncompleted, but can still save resources.
	CancelPayload(ctx context.Context, force bool) error
	// BuildingPayload indicates if a payload is being built, and onto which block it is being built, and whether or not it is a safe payload.
	BuildingPayload() (onto eth.L2BlockRef, id eth.PayloadID, safe bool)
}

type LocalEngineState interface {
	EngineState

	PendingSafeL2Head() eth.L2BlockRef
	BackupUnsafeL2Head() eth.L2BlockRef
}

type LocalEngineControl interface {
	LocalEngineState
	EngineControl
	ResetEngineControl
}

type FinalizerHooks interface {
	// OnDerivationL1End remembers the given L1 block,
	// and finalizes any prior data with the latest finality signal based on block height.
	OnDerivationL1End(ctx context.Context, derivedFrom eth.L1BlockRef) error
	// PostProcessSafeL2 remembers the L2 block is derived from the given L1 block, for later finalization.
	PostProcessSafeL2(l2Safe eth.L2BlockRef, derivedFrom eth.L1BlockRef)
	// Reset clear recent state, to adapt to reorgs.
	Reset()
}

var _ EngineControl = (*EngineController)(nil)
var _ LocalEngineControl = (*EngineController)(nil)
