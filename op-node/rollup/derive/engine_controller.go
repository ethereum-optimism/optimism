package derive

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var _ EngineControl = (*EngineController)(nil)
var _ LocalEngineControl = (*EngineController)(nil)

type ExecEngine interface {
	GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error)
	ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	NewPayload(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error)
}

type EngineController struct {
	engine  ExecEngine // Underlying execution engine RPC
	log     log.Logger
	metrics Metrics
	genesis *rollup.Genesis

	// Block Head State
	syncTarget      eth.L2BlockRef
	unsafeHead      eth.L2BlockRef
	pendingSafeHead eth.L2BlockRef
	safeHead        eth.L2BlockRef
	finalizedHead   eth.L2BlockRef

	// Building State
	buildingOnto eth.L2BlockRef
	buildingID   eth.PayloadID
	buildingSafe bool
	safeAttrs    *AttributesWithParent
}

func NewEngineController(engine ExecEngine, log log.Logger, metrics Metrics, genesis rollup.Genesis) *EngineController {
	return &EngineController{
		engine:  engine,
		log:     log,
		metrics: metrics,
		genesis: &genesis,
	}
}

// State Getters

func (e *EngineController) EngineSyncTarget() eth.L2BlockRef {
	return e.syncTarget
}

func (e *EngineController) UnsafeL2Head() eth.L2BlockRef {
	return e.unsafeHead
}

func (e *EngineController) PendingSafeL2Head() eth.L2BlockRef {
	return e.pendingSafeHead
}

func (e *EngineController) SafeL2Head() eth.L2BlockRef {
	return e.safeHead
}

func (e *EngineController) Finalized() eth.L2BlockRef {
	return e.finalizedHead
}

func (e *EngineController) BuildingPayload() (eth.L2BlockRef, eth.PayloadID, bool) {
	return e.buildingOnto, e.buildingID, e.buildingSafe
}

func (e *EngineController) IsEngineSyncing() bool {
	return e.unsafeHead.Hash != e.syncTarget.Hash
}

// Setters

// SetEngineSyncTarget implements LocalEngineControl.
func (e *EngineController) SetEngineSyncTarget(r eth.L2BlockRef) {
	e.metrics.RecordL2Ref("l2_engineSyncTarget", r)
	e.syncTarget = r
}

// SetFinalizedHead implements LocalEngineControl.
func (e *EngineController) SetFinalizedHead(r eth.L2BlockRef) {
	e.metrics.RecordL2Ref("l2_finalized", r)
	e.finalizedHead = r
}

// SetPendingSafeL2Head implements LocalEngineControl.
func (e *EngineController) SetPendingSafeL2Head(r eth.L2BlockRef) {
	e.metrics.RecordL2Ref("l2_pending_safe", r)
	e.pendingSafeHead = r
}

// SetSafeHead implements LocalEngineControl.
func (e *EngineController) SetSafeHead(r eth.L2BlockRef) {
	e.metrics.RecordL2Ref("l2_safe", r)
	e.safeHead = r
}

// SetUnsafeHead implements LocalEngineControl.
func (e *EngineController) SetUnsafeHead(r eth.L2BlockRef) {
	e.metrics.RecordL2Ref("l2_unsafe", r)
	e.unsafeHead = r
}

// Engine Methods

func (e *EngineController) StartPayload(ctx context.Context, parent eth.L2BlockRef, attrs *AttributesWithParent, updateSafe bool) (errType BlockInsertionErrType, err error) {
	if e.IsEngineSyncing() {
		return BlockInsertTemporaryErr, fmt.Errorf("engine is in progess of p2p sync")
	}
	if e.buildingID != (eth.PayloadID{}) {
		e.log.Warn("did not finish previous block building, starting new building now", "prev_onto", e.buildingOnto, "prev_payload_id", e.buildingID, "new_onto", parent)
		// TODO(8841): maybe worth it to force-cancel the old payload ID here.
	}
	fc := eth.ForkchoiceState{
		HeadBlockHash:      parent.Hash,
		SafeBlockHash:      e.safeHead.Hash,
		FinalizedBlockHash: e.finalizedHead.Hash,
	}

	id, errTyp, err := startPayload(ctx, e.engine, fc, attrs.attributes)
	if err != nil {
		return errTyp, err
	}

	e.buildingID = id
	e.buildingSafe = updateSafe
	e.buildingOnto = parent
	if updateSafe {
		e.safeAttrs = attrs
	}

	return BlockInsertOK, nil
}

func (e *EngineController) ConfirmPayload(ctx context.Context) (out *eth.ExecutionPayload, errTyp BlockInsertionErrType, err error) {
	if e.buildingID == (eth.PayloadID{}) {
		return nil, BlockInsertPrestateErr, fmt.Errorf("cannot complete payload building: not currently building a payload")
	}
	if e.buildingOnto.Hash != e.unsafeHead.Hash { // E.g. when safe-attributes consolidation fails, it will drop the existing work.
		e.log.Warn("engine is building block that reorgs previous unsafe head", "onto", e.buildingOnto, "unsafe", e.unsafeHead)
	}
	fc := eth.ForkchoiceState{
		HeadBlockHash:      common.Hash{}, // gets overridden
		SafeBlockHash:      e.safeHead.Hash,
		FinalizedBlockHash: e.finalizedHead.Hash,
	}
	// Update the safe head if the payload is built with the last attributes in the batch.
	updateSafe := e.buildingSafe && e.safeAttrs != nil && e.safeAttrs.isLastInSpan
	payload, errTyp, err := confirmPayload(ctx, e.log, e.engine, fc, e.buildingID, updateSafe)
	if err != nil {
		return nil, errTyp, fmt.Errorf("failed to complete building on top of L2 chain %s, id: %s, error (%d): %w", e.buildingOnto, e.buildingID, errTyp, err)
	}
	ref, err := PayloadToBlockRef(payload, e.genesis)
	if err != nil {
		return nil, BlockInsertPayloadErr, NewResetError(fmt.Errorf("failed to decode L2 block ref from payload: %w", err))
	}

	e.unsafeHead = ref
	e.syncTarget = ref

	e.metrics.RecordL2Ref("l2_unsafe", ref)
	e.metrics.RecordL2Ref("l2_engineSyncTarget", ref)
	if e.buildingSafe {
		e.metrics.RecordL2Ref("l2_pending_safe", ref)
		e.pendingSafeHead = ref
		if updateSafe {
			e.safeHead = ref
			e.metrics.RecordL2Ref("l2_safe", ref)
		}
	}

	e.resetBuildingState()
	return payload, BlockInsertOK, nil
}

func (e *EngineController) CancelPayload(ctx context.Context, force bool) error {
	if e.buildingID == (eth.PayloadID{}) { // only cancel if there is something to cancel.
		return nil
	}
	// the building job gets wrapped up as soon as the payload is retrieved, there's no explicit cancel in the Engine API
	e.log.Error("cancelling old block sealing job", "payload", e.buildingID)
	_, err := e.engine.GetPayload(ctx, e.buildingID)
	if err != nil {
		e.log.Error("failed to cancel block building job", "payload", e.buildingID, "err", err)
		if !force {
			return err
		}
	}
	e.resetBuildingState()
	return nil
}

func (e *EngineController) resetBuildingState() {
	e.buildingID = eth.PayloadID{}
	e.buildingOnto = eth.L2BlockRef{}
	e.buildingSafe = false
	e.safeAttrs = nil
}

// Misc Setters only used by the engine queue

// ResetBuildingState implements LocalEngineControl.
func (e *EngineController) ResetBuildingState() {
	e.resetBuildingState()
}

// ForkchoiceUpdate implements LocalEngineControl.
func (e *EngineController) ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return e.engine.ForkchoiceUpdate(ctx, state, attr)
}

// NewPayload implements LocalEngineControl.
func (e *EngineController) NewPayload(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	return e.engine.NewPayload(ctx, payload)
}
