package derive

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/async"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type syncStatusEnum int

const (
	syncStatusCL syncStatusEnum = iota
	// We transition between the 4 EL states linearly. We spend the majority of the time in the second & fourth.
	// We only want to EL sync if there is no finalized block & once we finish EL sync we need to mark the last block
	// as finalized so we can switch to consolidation
	// TODO(protocol-quest/91): We can restart EL sync & still consolidate if there finalized blocks on the execution client if the
	// execution client is running in archive mode. In some cases we may want to switch back from CL to EL sync, but that is complicated.
	syncStatusWillStartEL               // First if we are directed to EL sync, check that nothing has been finalized yet
	syncStatusStartedEL                 // Perform our EL sync
	syncStatusFinishedELButNotFinalized // EL sync is done, but we need to mark the final sync block as finalized
	syncStatusFinishedEL                // EL sync is done & we should be performing consolidation
)

var errNoFCUNeeded = errors.New("no FCU call was needed")

var _ EngineControl = (*EngineController)(nil)
var _ LocalEngineControl = (*EngineController)(nil)

type ExecEngine interface {
	GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error)
	ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	NewPayload(ctx context.Context, payload *eth.ExecutionPayload, parentBeaconBlockRoot *common.Hash) (*eth.PayloadStatusV1, error)
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
}

type EngineController struct {
	engine     ExecEngine // Underlying execution engine RPC
	log        log.Logger
	metrics    Metrics
	syncMode   sync.Mode
	syncStatus syncStatusEnum
	rollupCfg  *rollup.Config
	elStart    time.Time
	clock      clock.Clock

	// Block Head State
	unsafeHead      eth.L2BlockRef
	pendingSafeHead eth.L2BlockRef // L2 block processed from the middle of a span batch, but not marked as the safe block yet.
	safeHead        eth.L2BlockRef
	finalizedHead   eth.L2BlockRef
	needFCUCall     bool

	// Building State
	buildingOnto eth.L2BlockRef
	buildingID   eth.PayloadID
	buildingSafe bool
	safeAttrs    *AttributesWithParent
}

func NewEngineController(engine ExecEngine, log log.Logger, metrics Metrics, rollupCfg *rollup.Config, syncMode sync.Mode) *EngineController {
	syncStatus := syncStatusCL
	if syncMode == sync.ELSync {
		syncStatus = syncStatusWillStartEL
	}

	return &EngineController{
		engine:     engine,
		log:        log,
		metrics:    metrics,
		rollupCfg:  rollupCfg,
		syncMode:   syncMode,
		syncStatus: syncStatus,
		clock:      clock.SystemClock,
	}
}

// State Getters

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
	return e.syncStatus == syncStatusWillStartEL || e.syncStatus == syncStatusStartedEL || e.syncStatus == syncStatusFinishedELButNotFinalized
}

// Setters

// SetFinalizedHead implements LocalEngineControl.
func (e *EngineController) SetFinalizedHead(r eth.L2BlockRef) {
	e.metrics.RecordL2Ref("l2_finalized", r)
	e.finalizedHead = r
	e.needFCUCall = true
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
	e.needFCUCall = true
}

// SetUnsafeHead implements LocalEngineControl.
func (e *EngineController) SetUnsafeHead(r eth.L2BlockRef) {
	e.metrics.RecordL2Ref("l2_unsafe", r)
	e.unsafeHead = r
	e.needFCUCall = true
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

func (e *EngineController) ConfirmPayload(ctx context.Context, agossip async.AsyncGossiper, sequencerConductor conductor.SequencerConductor) (out *eth.ExecutionPayloadEnvelope, errTyp BlockInsertionErrType, err error) {
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
	envelope, errTyp, err := confirmPayload(ctx, e.log, e.engine, fc, e.buildingID, updateSafe, agossip, sequencerConductor)
	if err != nil {
		return nil, errTyp, fmt.Errorf("failed to complete building on top of L2 chain %s, id: %s, error (%d): %w", e.buildingOnto, e.buildingID, errTyp, err)
	}
	ref, err := PayloadToBlockRef(e.rollupCfg, envelope.ExecutionPayload)
	if err != nil {
		return nil, BlockInsertPayloadErr, NewResetError(fmt.Errorf("failed to decode L2 block ref from payload: %w", err))
	}

	e.unsafeHead = ref

	e.metrics.RecordL2Ref("l2_unsafe", ref)
	if e.buildingSafe {
		e.metrics.RecordL2Ref("l2_pending_safe", ref)
		e.pendingSafeHead = ref
		if updateSafe {
			e.safeHead = ref
			e.metrics.RecordL2Ref("l2_safe", ref)
		}
	}

	e.resetBuildingState()
	return envelope, BlockInsertOK, nil
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

// checkNewPayloadStatus checks returned status of engine_newPayloadV1 request for next unsafe payload.
// It returns true if the status is acceptable.
func (e *EngineController) checkNewPayloadStatus(status eth.ExecutePayloadStatus) bool {
	if e.syncMode == sync.ELSync {
		if status == eth.ExecutionValid && e.syncStatus == syncStatusStartedEL {
			e.syncStatus = syncStatusFinishedELButNotFinalized
		}
		// Allow SYNCING and ACCEPTED if engine EL sync is enabled
		return status == eth.ExecutionValid || status == eth.ExecutionSyncing || status == eth.ExecutionAccepted
	}
	return status == eth.ExecutionValid
}

// checkForkchoiceUpdatedStatus checks returned status of engine_forkchoiceUpdatedV1 request for next unsafe payload.
// It returns true if the status is acceptable.
func (e *EngineController) checkForkchoiceUpdatedStatus(status eth.ExecutePayloadStatus) bool {
	if e.syncMode == sync.ELSync {
		if status == eth.ExecutionValid && e.syncStatus == syncStatusStartedEL {
			e.syncStatus = syncStatusFinishedELButNotFinalized
		}
		// Allow SYNCING if engine P2P sync is enabled
		return status == eth.ExecutionValid || status == eth.ExecutionSyncing
	}
	return status == eth.ExecutionValid
}

// TryUpdateEngine attempts to update the engine with the current forkchoice state of the rollup node,
// this is a no-op if the nodes already agree on the forkchoice state.
func (e *EngineController) TryUpdateEngine(ctx context.Context) error {
	if !e.needFCUCall {
		return errNoFCUNeeded
	}
	if e.IsEngineSyncing() {
		e.log.Warn("Attempting to update forkchoice state while engine is P2P syncing")
	}
	fc := eth.ForkchoiceState{
		HeadBlockHash:      e.unsafeHead.Hash,
		SafeBlockHash:      e.safeHead.Hash,
		FinalizedBlockHash: e.finalizedHead.Hash,
	}
	_, err := e.engine.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		var inputErr eth.InputError
		if errors.As(err, &inputErr) {
			switch inputErr.Code {
			case eth.InvalidForkchoiceState:
				return NewResetError(fmt.Errorf("forkchoice update was inconsistent with engine, need reset to resolve: %w", inputErr.Unwrap()))
			default:
				return NewTemporaryError(fmt.Errorf("unexpected error code in forkchoice-updated response: %w", err))
			}
		} else {
			return NewTemporaryError(fmt.Errorf("failed to sync forkchoice with engine: %w", err))
		}
	}
	e.needFCUCall = false
	return nil
}

func (e *EngineController) InsertUnsafePayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef) error {
	// Check if there is a finalized head once when doing EL sync. If so, transition to CL sync
	if e.syncStatus == syncStatusWillStartEL {
		b, err := e.engine.L2BlockRefByLabel(ctx, eth.Finalized)
		if errors.Is(err, ethereum.NotFound) {
			e.syncStatus = syncStatusStartedEL
			e.log.Info("Starting EL sync")
			e.elStart = e.clock.Now()
		} else if err == nil {
			e.syncStatus = syncStatusFinishedEL
			e.log.Info("Skipping EL sync and going straight to CL sync because there is a finalized block", "id", b.ID())
			return nil
		} else {
			return NewTemporaryError(fmt.Errorf("failed to fetch finalized head: %w", err))
		}
	}
	// Insert the payload & then call FCU
	status, err := e.engine.NewPayload(ctx, envelope.ExecutionPayload, envelope.ParentBeaconBlockRoot)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to update insert payload: %w", err))
	}
	if !e.checkNewPayloadStatus(status.Status) {
		payload := envelope.ExecutionPayload
		return NewTemporaryError(fmt.Errorf("cannot process unsafe payload: new - %v; parent: %v; err: %w",
			payload.ID(), payload.ParentID(), eth.NewPayloadErr(payload, status)))
	}

	// Mark the new payload as valid
	fc := eth.ForkchoiceState{
		HeadBlockHash:      envelope.ExecutionPayload.BlockHash,
		SafeBlockHash:      e.safeHead.Hash,
		FinalizedBlockHash: e.finalizedHead.Hash,
	}
	if e.syncStatus == syncStatusFinishedELButNotFinalized {
		fc.SafeBlockHash = envelope.ExecutionPayload.BlockHash
		fc.FinalizedBlockHash = envelope.ExecutionPayload.BlockHash
		e.SetSafeHead(ref)
		e.SetFinalizedHead(ref)
	}
	fcRes, err := e.engine.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		var inputErr eth.InputError
		if errors.As(err, &inputErr) {
			switch inputErr.Code {
			case eth.InvalidForkchoiceState:
				return NewResetError(fmt.Errorf("pre-unsafe-block forkchoice update was inconsistent with engine, need reset to resolve: %w", inputErr.Unwrap()))
			default:
				return NewTemporaryError(fmt.Errorf("unexpected error code in forkchoice-updated response: %w", err))
			}
		} else {
			return NewTemporaryError(fmt.Errorf("failed to update forkchoice to prepare for new unsafe payload: %w", err))
		}
	}
	if !e.checkForkchoiceUpdatedStatus(fcRes.PayloadStatus.Status) {
		payload := envelope.ExecutionPayload
		return NewTemporaryError(fmt.Errorf("cannot prepare unsafe chain for new payload: new - %v; parent: %v; err: %w",
			payload.ID(), payload.ParentID(), eth.ForkchoiceUpdateErr(fcRes.PayloadStatus)))
	}
	e.SetUnsafeHead(ref)
	e.needFCUCall = false

	if e.syncStatus == syncStatusFinishedELButNotFinalized {
		e.log.Info("Finished EL sync", "sync_duration", e.clock.Since(e.elStart))
		e.syncStatus = syncStatusFinishedEL
	}

	return nil
}

// ResetBuildingState implements LocalEngineControl.
func (e *EngineController) ResetBuildingState() {
	e.resetBuildingState()
}
