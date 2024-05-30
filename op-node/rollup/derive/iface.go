package derive

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/rollup/async"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SafeHeadListener is called when the safe head is updated.
// The safe head may advance by more than one block in a single update
// The l1Block specified is the first L1 block that includes sufficient information to derive the new safe head
type SafeHeadListener interface {

	// Enabled reports if this safe head listener is actively using the posted data. This allows the engine queue to
	// optionally skip making calls that may be expensive to prepare.
	// Callbacks may still be made if Enabled returns false but are not guaranteed.
	Enabled() bool

	// SafeHeadUpdated indicates that the safe head has been updated in response to processing batch data
	// The l1Block specified is the first L1 block containing all required batch data to derive newSafeHead
	SafeHeadUpdated(newSafeHead eth.L2BlockRef, l1Block eth.BlockID) error

	// SafeHeadReset indicates that the derivation pipeline reset back to the specified safe head
	// The L1 block that made the new safe head safe is unknown.
	SafeHeadReset(resetSafeHead eth.L2BlockRef) error
}

// EngineState provides a read-only interface of the forkchoice state properties of the L2 Engine.
type EngineState interface {
	Finalized() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
}

type Engine interface {
	ExecEngine
	L2Source
}

// EngineControl enables other components to build blocks with the Engine,
// while keeping the forkchoice state and payload-id management internal to
// avoid state inconsistencies between different users of the EngineControl.
type EngineControl interface {
	EngineState

	// StartPayload requests the engine to start building a block with the given attributes.
	// If updateSafe, the resulting block will be marked as a safe block.
	StartPayload(ctx context.Context, parent eth.L2BlockRef, attrs *AttributesWithParent, updateSafe bool) (errType BlockInsertionErrType, err error)
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

type ResetEngineControl interface {
	SetUnsafeHead(eth.L2BlockRef)
	SetSafeHead(eth.L2BlockRef)
	SetFinalizedHead(eth.L2BlockRef)

	SetBackupUnsafeL2Head(block eth.L2BlockRef, triggerReorg bool)
	SetPendingSafeL2Head(eth.L2BlockRef)

	ResetBuildingState()
}

type LocalEngineControl interface {
	LocalEngineState
	EngineControl
	ResetEngineControl
}

type L2Source interface {
	PayloadByHash(context.Context, common.Hash) (*eth.ExecutionPayloadEnvelope, error)
	PayloadByNumber(context.Context, uint64) (*eth.ExecutionPayloadEnvelope, error)
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
	L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
	SystemConfigL2Fetcher
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

type AttributesHandler interface {
	// HasAttributes returns if there are any block attributes to process.
	// HasAttributes is for EngineQueue testing only, and can be removed when attribute processing is fully independent.
	HasAttributes() bool
	// SetAttributes overwrites the set of attributes. This may be nil, to clear what may be processed next.
	SetAttributes(attributes *AttributesWithParent)
	// Proceed runs one attempt of processing attributes, if any.
	// Proceed returns io.EOF if there are no attributes to process.
	Proceed(ctx context.Context) error
}
