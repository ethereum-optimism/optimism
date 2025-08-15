package engine

import (
	"context"
	"errors"
	"fmt"
	gosync "sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	opmetrics "github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type syncStatusEnum int

const (
	syncStatusCL syncStatusEnum = iota
	// We transition between the 4 EL states linearly. We spend the majority of the time in the second & fourth.
	// We only want to EL sync if there is no finalized block & once we finish EL sync we need to mark the last block
	// as finalized so we can switch to consolidation
	// TODO(protocol-quest#91): We can restart EL sync & still consolidate if there finalized blocks on the execution client if the
	// execution client is running in archive mode. In some cases we may want to switch back from CL to EL sync, but that is complicated.
	syncStatusWillStartEL               // First if we are directed to EL sync, check that nothing has been finalized yet
	syncStatusStartedEL                 // Perform our EL sync
	syncStatusFinishedELButNotFinalized // EL sync is done, but we need to mark the final sync block as finalized
	syncStatusFinishedEL                // EL sync is done & we should be performing consolidation
)

var ErrNoFCUNeeded = errors.New("no FCU call was needed")

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

type EngineController struct {
	engine     ExecEngine // Underlying execution engine RPC
	log        log.Logger
	metrics    opmetrics.Metricer
	syncCfg    *sync.Config
	syncStatus syncStatusEnum
	chainSpec  *rollup.ChainSpec
	rollupCfg  *rollup.Config
	elStart    time.Time
	clock      clock.Clock

	// TODO(#16917) Remove Event System Refactor Comments
	// Event system fields (moved from EngDeriver)
	ctx     context.Context
	emitter event.Emitter

	// To lock the engine RPC usage, such that components like the API, which need direct access, can protect their access.
	mu gosync.RWMutex

	// Block Head State
	unsafeHead eth.L2BlockRef
	// Cross-verified unsafeHead, always equal to unsafeHead pre-interop
	crossUnsafeHead eth.L2BlockRef
	// Pending localSafeHead
	// L2 block processed from the middle of a span batch,
	// but not marked as the safe block yet.
	pendingSafeHead eth.L2BlockRef
	// Derived from L1, and known to be a completed span-batch,
	// but not cross-verified yet.
	localSafeHead eth.L2BlockRef
	// Derived from L1 and cross-verified to have cross-safe dependencies.
	safeHead eth.L2BlockRef
	// Derived from finalized L1 data,
	// and cross-verified to only have finalized dependencies.
	finalizedHead eth.L2BlockRef
	// The unsafe head to roll back to,
	// after the pendingSafeHead fails to become safe.
	// This is changing in the Holocene fork.
	backupUnsafeHead eth.L2BlockRef

	needFCUCall bool
	// Track when the rollup node changes the forkchoice to restore previous
	// known unsafe chain. e.g. Unsafe Reorg caused by Invalid span batch.
	// This update does not retry except engine returns non-input error
	// because engine may forgot backupUnsafeHead or backupUnsafeHead is not part
	// of the chain.
	needFCUCallForBackupUnsafeReorg bool

	// For clearing safe head db when EL sync started
	// EngineController is first initialized and used to initialize SyncDeriver.
	// Embed SyncDeriver into EngineController after initializing SyncDeriver
	SyncDeriver SyncDeriver
}

func NewEngineController(ctx context.Context, engine ExecEngine, log log.Logger, m opmetrics.Metricer,
	rollupCfg *rollup.Config, syncCfg *sync.Config, emitter event.Emitter,
) *EngineController {
	syncStatus := syncStatusCL
	if syncCfg.SyncMode == sync.ELSync {
		syncStatus = syncStatusWillStartEL
	}

	return &EngineController{
		engine:     engine,
		log:        log,
		metrics:    m,
		chainSpec:  rollup.NewChainSpec(rollupCfg),
		rollupCfg:  rollupCfg,
		syncCfg:    syncCfg,
		syncStatus: syncStatus,
		clock:      clock.SystemClock,
		ctx:        ctx,
		emitter:    emitter,
	}
}

// State Getters

func (e *EngineController) UnsafeL2Head() eth.L2BlockRef {
	return e.unsafeHead
}

func (e *EngineController) CrossUnsafeL2Head() eth.L2BlockRef {
	return e.crossUnsafeHead
}

func (e *EngineController) PendingSafeL2Head() eth.L2BlockRef {
	return e.pendingSafeHead
}

func (e *EngineController) LocalSafeL2Head() eth.L2BlockRef {
	return e.localSafeHead
}

func (e *EngineController) SafeL2Head() eth.L2BlockRef {
	return e.safeHead
}

func (e *EngineController) Finalized() eth.L2BlockRef {
	return e.finalizedHead
}

func (e *EngineController) BackupUnsafeL2Head() eth.L2BlockRef {
	return e.backupUnsafeHead
}

// RequestForkchoiceUpdate implements attributes.EngineController.
// It reads the current heads under a read lock and emits a ForkchoiceUpdateEvent.
func (e *EngineController) RequestForkchoiceUpdate(ctx context.Context) {
	e.mu.RLock()
	unsafe := e.UnsafeL2Head()
	safe := e.SafeL2Head()
	finalized := e.Finalized()
	e.mu.RUnlock()
	e.emitter.Emit(ctx, ForkchoiceUpdateEvent{
		UnsafeL2Head:    unsafe,
		SafeL2Head:      safe,
		FinalizedL2Head: finalized,
	})
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

// SetLocalSafeHead sets the local-safe head.
func (e *EngineController) SetLocalSafeHead(r eth.L2BlockRef) {
	e.metrics.RecordL2Ref("l2_local_safe", r)
	e.localSafeHead = r
}

// SetSafeHead sets the cross-safe head.
func (e *EngineController) SetSafeHead(r eth.L2BlockRef) {
	e.metrics.RecordL2Ref("l2_safe", r)
	e.safeHead = r
	e.needFCUCall = true
}

// SetUnsafeHead sets the local-unsafe head.
func (e *EngineController) SetUnsafeHead(r eth.L2BlockRef) {
	e.metrics.RecordL2Ref("l2_unsafe", r)
	e.unsafeHead = r
	e.needFCUCall = true
	e.chainSpec.CheckForkActivation(e.log, r)
}

// SetCrossUnsafeHead the cross-unsafe head.
func (e *EngineController) SetCrossUnsafeHead(r eth.L2BlockRef) {
	e.metrics.RecordL2Ref("l2_cross_unsafe", r)
	e.crossUnsafeHead = r
}

// SetBackupUnsafeL2Head implements LocalEngineControl.
func (e *EngineController) SetBackupUnsafeL2Head(r eth.L2BlockRef, triggerReorg bool) {
	e.metrics.RecordL2Ref("l2_backup_unsafe", r)
	e.backupUnsafeHead = r
	e.needFCUCallForBackupUnsafeReorg = triggerReorg
}

// logSyncProgressMaybe helps log forkchoice state-changes when applicable.
// First, the pre-state is registered.
// A callback is returned to then log the changes to the pre-state, if any.
func (e *EngineController) logSyncProgressMaybe() func() {
	prevFinalized := e.finalizedHead
	prevSafe := e.safeHead
	prevPendingSafe := e.pendingSafeHead
	prevUnsafe := e.unsafeHead
	prevBackupUnsafe := e.backupUnsafeHead
	return func() {
		// if forkchoice still needs to be updated, then the last change was unsuccessful, thus no progress to log.
		if e.needFCUCall || e.needFCUCallForBackupUnsafeReorg {
			return
		}
		var reason string
		if prevFinalized != e.finalizedHead {
			reason = "finalized block"
		} else if prevSafe != e.safeHead {
			if prevSafe == prevUnsafe {
				reason = "derived safe block from L1"
			} else {
				reason = "consolidated block with L1"
			}
		} else if prevUnsafe != e.unsafeHead {
			reason = "new chain head block"
		} else if prevPendingSafe != e.pendingSafeHead {
			reason = "pending new safe block"
		} else if prevBackupUnsafe != e.backupUnsafeHead {
			reason = "new backup unsafe block"
		}
		if reason != "" {
			e.log.Info("Sync progress",
				"reason", reason,
				"l2_finalized", e.finalizedHead,
				"l2_safe", e.safeHead,
				"l2_pending_safe", e.pendingSafeHead,
				"l2_unsafe", e.unsafeHead,
				"l2_backup_unsafe", e.backupUnsafeHead,
				"l2_time", e.UnsafeL2Head().Time,
			)
		}
	}
}

// Misc Setters only used by the engine queue

// checkNewPayloadStatus checks returned status of engine_newPayloadV1 request for next unsafe payload.
// It returns true if the status is acceptable.
func (e *EngineController) checkNewPayloadStatus(status eth.ExecutePayloadStatus) bool {
	if e.syncCfg.SyncMode == sync.ELSync {
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
	if e.syncCfg.SyncMode == sync.ELSync {
		if status == eth.ExecutionValid && e.syncStatus == syncStatusStartedEL {
			e.syncStatus = syncStatusFinishedELButNotFinalized
		}
		// Allow SYNCING if engine P2P sync is enabled
		return status == eth.ExecutionValid || status == eth.ExecutionSyncing
	}
	return status == eth.ExecutionValid
}

// initializeUnknowns is important to give the op-node EngineController engine state.
// Pre-interop, the initial reset triggered a find-sync-start, and filled the forkchoice.
// This still happens, but now overrides what may be initialized here.
// Post-interop, the op-supervisor may diff the forkchoice state against the supervisor DB,
// to determine where to perform the initial reset to.
func (e *EngineController) initializeUnknowns(ctx context.Context) error {
	if e.unsafeHead == (eth.L2BlockRef{}) {
		ref, err := e.engine.L2BlockRefByLabel(ctx, eth.Unsafe)
		if err != nil {
			return fmt.Errorf("failed to load local-unsafe head: %w", err)
		}
		e.SetUnsafeHead(ref)
		e.log.Info("Loaded initial local-unsafe block ref", "local_unsafe", ref)
	}
	var finalizedRef eth.L2BlockRef
	if e.finalizedHead == (eth.L2BlockRef{}) {
		var err error
		finalizedRef, err = e.engine.L2BlockRefByLabel(ctx, eth.Finalized)
		if err != nil {
			return fmt.Errorf("failed to load finalized head: %w", err)
		}
		e.SetFinalizedHead(finalizedRef)
		e.log.Info("Loaded initial finalized block ref", "finalized", finalizedRef)
	}
	if e.safeHead == (eth.L2BlockRef{}) {
		ref, err := e.engine.L2BlockRefByLabel(ctx, eth.Safe)
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				// If the engine doesn't have a safe head, then we can use the finalized head
				e.SetSafeHead(finalizedRef)
				e.log.Info("Loaded initial cross-safe block from finalized", "cross_safe", finalizedRef)
			} else {
				return fmt.Errorf("failed to load cross-safe head: %w", err)
			}
		} else {
			e.SetSafeHead(ref)
			e.log.Info("Loaded initial cross-safe block ref", "cross_safe", ref)
		}
	}
	if e.crossUnsafeHead == (eth.L2BlockRef{}) {
		e.SetCrossUnsafeHead(e.safeHead) // preserve cross-safety, don't fall back to a non-cross safety level
		e.log.Info("Set initial cross-unsafe block ref to match cross-safe", "cross_unsafe", e.safeHead)
	}
	if e.localSafeHead == (eth.L2BlockRef{}) {
		e.SetLocalSafeHead(e.safeHead)
		e.log.Info("Set initial local-safe block ref to match cross-safe", "local_safe", e.safeHead)
	}
	return nil
}

// tryUpdateEngine attempts to update the engine with the current forkchoice state of the rollup node,
// this is a no-op if the nodes already agree on the forkchoice state.
func (e *EngineController) tryUpdateEngine(ctx context.Context) error {
	if !e.needFCUCall {
		return ErrNoFCUNeeded
	}
	if e.IsEngineSyncing() {
		e.log.Warn("Attempting to update forkchoice state while EL syncing")
	}
	if err := e.initializeUnknowns(ctx); err != nil {
		return derive.NewTemporaryError(fmt.Errorf("cannot update engine until engine forkchoice is initialized: %w", err))
	}
	if e.unsafeHead.Number < e.finalizedHead.Number {
		err := fmt.Errorf("invalid forkchoice state, unsafe head %s is behind finalized head %s", e.unsafeHead, e.finalizedHead)
		e.emitter.Emit(ctx, rollup.CriticalErrorEvent{Err: err}) // make the node exit, things are very wrong.
		return err
	}
	fc := eth.ForkchoiceState{
		HeadBlockHash:      e.unsafeHead.Hash,
		SafeBlockHash:      e.safeHead.Hash,
		FinalizedBlockHash: e.finalizedHead.Hash,
	}
	logFn := e.logSyncProgressMaybe()
	defer logFn()
	fcRes, err := e.engine.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) {
			switch eth.ErrorCode(rpcErr.ErrorCode()) {
			case eth.InvalidForkchoiceState:
				return derive.NewResetError(fmt.Errorf("forkchoice update was inconsistent with engine, need reset to resolve: %w", err))
			default:
				return derive.NewTemporaryError(fmt.Errorf("unexpected error code in forkchoice-updated response: %w", err))
			}
		} else {
			return derive.NewTemporaryError(fmt.Errorf("failed to sync forkchoice with engine: %w", err))
		}
	}
	if fcRes.PayloadStatus.Status == eth.ExecutionValid {
		e.emitter.Emit(ctx, ForkchoiceUpdateEvent{
			UnsafeL2Head:    e.unsafeHead,
			SafeL2Head:      e.safeHead,
			FinalizedL2Head: e.finalizedHead,
		})
	}
	if e.unsafeHead == e.safeHead && e.safeHead == e.pendingSafeHead {
		// Remove backupUnsafeHead because this backup will be never used after consolidation.
		e.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)
	}
	e.needFCUCall = false
	return nil
}

func (e *EngineController) InsertUnsafePayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef) error {
	// Check if there is a finalized head once when doing EL sync. If so, transition to CL sync
	if e.syncStatus == syncStatusWillStartEL {
		b, err := e.engine.L2BlockRefByLabel(ctx, eth.Finalized)
		rollupGenesisIsFinalized := b.Hash == e.rollupCfg.Genesis.L2.Hash
		if errors.Is(err, ethereum.NotFound) || rollupGenesisIsFinalized || e.syncCfg.SupportsPostFinalizationELSync {
			e.syncStatus = syncStatusStartedEL
			e.log.Info("Starting EL sync")
			e.elStart = e.clock.Now()
			e.SyncDeriver.OnELSyncStarted()
		} else if err == nil {
			e.syncStatus = syncStatusFinishedEL
			e.log.Info("Skipping EL sync and going straight to CL sync because there is a finalized block", "id", b.ID())
			return nil
		} else {
			return derive.NewTemporaryError(fmt.Errorf("failed to fetch finalized head: %w", err))
		}
	}
	// Insert the payload & then call FCU
	newPayloadStart := time.Now()
	status, err := e.engine.NewPayload(ctx, envelope.ExecutionPayload, envelope.ParentBeaconBlockRoot)
	if err != nil {
		return derive.NewTemporaryError(fmt.Errorf("failed to update insert payload: %w", err))
	}
	if status.Status == eth.ExecutionInvalid {
		e.emitter.Emit(ctx, PayloadInvalidEvent{
			Envelope: envelope,
			Err:      eth.NewPayloadErr(envelope.ExecutionPayload, status),
		})
	}
	if !e.checkNewPayloadStatus(status.Status) {
		payload := envelope.ExecutionPayload
		return derive.NewTemporaryError(fmt.Errorf("cannot process unsafe payload: new - %v; parent: %v; err: %w",
			payload.ID(), payload.ParentID(), eth.NewPayloadErr(payload, status)))
	}
	newPayloadFinish := time.Now()

	// Mark the new payload as valid
	fc := eth.ForkchoiceState{
		HeadBlockHash:      envelope.ExecutionPayload.BlockHash,
		SafeBlockHash:      e.safeHead.Hash,
		FinalizedBlockHash: e.finalizedHead.Hash,
	}
	if e.syncStatus == syncStatusFinishedELButNotFinalized {
		fc.SafeBlockHash = envelope.ExecutionPayload.BlockHash
		fc.FinalizedBlockHash = envelope.ExecutionPayload.BlockHash
		e.SetUnsafeHead(ref) // ensure that the unsafe head stays ahead of safe/finalized labels.
		e.emitter.Emit(ctx, UnsafeUpdateEvent{Ref: ref})
		e.SetLocalSafeHead(ref)
		e.SetSafeHead(ref)
		e.emitter.Emit(ctx, CrossSafeUpdateEvent{LocalSafe: ref, CrossSafe: ref})
		e.SetFinalizedHead(ref)
	}
	logFn := e.logSyncProgressMaybe()
	defer logFn()
	fcu2Start := time.Now()
	fcRes, err := e.engine.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) {
			switch eth.ErrorCode(rpcErr.ErrorCode()) {
			case eth.InvalidForkchoiceState:
				return derive.NewResetError(fmt.Errorf("pre-unsafe-block forkchoice update was inconsistent with engine, need reset to resolve: %w", err))
			default:
				return derive.NewTemporaryError(fmt.Errorf("unexpected error code in forkchoice-updated response: %w", err))
			}
		} else {
			return derive.NewTemporaryError(fmt.Errorf("failed to update forkchoice to prepare for new unsafe payload: %w", err))
		}
	}
	if !e.checkForkchoiceUpdatedStatus(fcRes.PayloadStatus.Status) {
		payload := envelope.ExecutionPayload
		return derive.NewTemporaryError(fmt.Errorf("cannot prepare unsafe chain for new payload: new - %v; parent: %v; err: %w",
			payload.ID(), payload.ParentID(), eth.ForkchoiceUpdateErr(fcRes.PayloadStatus)))
	}
	fcu2Finish := time.Now()
	e.SetUnsafeHead(ref)
	e.needFCUCall = false
	e.emitter.Emit(ctx, UnsafeUpdateEvent{Ref: ref})

	if e.syncStatus == syncStatusFinishedELButNotFinalized {
		e.log.Info("Finished EL sync", "sync_duration", e.clock.Since(e.elStart), "finalized_block", ref.ID().String())
		e.syncStatus = syncStatusFinishedEL
	}

	if fcRes.PayloadStatus.Status == eth.ExecutionValid {
		e.emitter.Emit(ctx, ForkchoiceUpdateEvent{
			UnsafeL2Head:    e.unsafeHead,
			SafeL2Head:      e.safeHead,
			FinalizedL2Head: e.finalizedHead,
		})
	}

	totalTime := fcu2Finish.Sub(newPayloadStart)
	e.log.Info("Inserted new L2 unsafe block (synchronous)",
		"hash", envelope.ExecutionPayload.BlockHash,
		"number", uint64(envelope.ExecutionPayload.BlockNumber),
		"newpayload_time", common.PrettyDuration(newPayloadFinish.Sub(newPayloadStart)),
		"fcu2_time", common.PrettyDuration(fcu2Finish.Sub(fcu2Start)),
		"total_time", common.PrettyDuration(totalTime),
		"mgas", float64(envelope.ExecutionPayload.GasUsed)/1000000,
		"mgasps", float64(envelope.ExecutionPayload.GasUsed)*1000/float64(totalTime))

	return nil
}

// shouldTryBackupUnsafeReorg checks reorging(restoring) unsafe head to backupUnsafeHead is needed.
// Returns boolean which decides to trigger FCU.
func (e *EngineController) shouldTryBackupUnsafeReorg() bool {
	if !e.needFCUCallForBackupUnsafeReorg {
		return false
	}
	// This method must be never called when EL sync. If EL sync is in progress, early return.
	if e.IsEngineSyncing() {
		e.log.Warn("Attempting to unsafe reorg using backupUnsafe while EL syncing")
		return false
	}
	if e.BackupUnsafeL2Head() == (eth.L2BlockRef{}) { // sanity check backupUnsafeHead is there
		e.log.Warn("Attempting to unsafe reorg using backupUnsafe even though it is empty")
		e.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)
		return false
	}
	return true
}

// TryBackupUnsafeReorg attempts to reorg(restore) unsafe head to backupUnsafeHead.
// If succeeds, update current forkchoice state to the rollup node.
func (e *EngineController) TryBackupUnsafeReorg(ctx context.Context) (bool, error) {
	if !e.shouldTryBackupUnsafeReorg() {
		// Do not need to perform FCU.
		return false, nil
	}
	// Only try FCU once because execution engine may forgot backupUnsafeHead
	// or backupUnsafeHead is not part of the chain.
	// Exception: Retry when forkChoiceUpdate returns non-input error.
	e.needFCUCallForBackupUnsafeReorg = false
	// Reorg unsafe chain. Safe/Finalized chain will not be updated.
	e.log.Warn("trying to restore unsafe head", "backupUnsafe", e.backupUnsafeHead.ID(), "unsafe", e.unsafeHead.ID())
	fc := eth.ForkchoiceState{
		HeadBlockHash:      e.backupUnsafeHead.Hash,
		SafeBlockHash:      e.safeHead.Hash,
		FinalizedBlockHash: e.finalizedHead.Hash,
	}
	logFn := e.logSyncProgressMaybe()
	defer logFn()
	fcRes, err := e.engine.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) {
			switch eth.ErrorCode(rpcErr.ErrorCode()) {
			case eth.InvalidForkchoiceState:
				e.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)
				return true, derive.NewResetError(fmt.Errorf("forkchoice update was inconsistent with engine, need reset to resolve: %w", err))
			default:
				// Retry when forkChoiceUpdate returns non-input error.
				// Do not reset backupUnsafeHead because it will be used again.
				e.needFCUCallForBackupUnsafeReorg = true
				return true, derive.NewTemporaryError(fmt.Errorf("unexpected error code in forkchoice-updated response: %w", err))
			}
		} else {
			// Retry when forkChoiceUpdate returns non-input error.
			// Do not reset backupUnsafeHead because it will be used again.
			e.needFCUCallForBackupUnsafeReorg = true
			return true, derive.NewTemporaryError(fmt.Errorf("failed to sync forkchoice with engine: %w", err))
		}
	}
	if fcRes.PayloadStatus.Status == eth.ExecutionValid {
		e.emitter.Emit(ctx, ForkchoiceUpdateEvent{
			UnsafeL2Head:    e.backupUnsafeHead,
			SafeL2Head:      e.safeHead,
			FinalizedL2Head: e.finalizedHead,
		})
		// Execution engine accepted the reorg.
		e.log.Info("successfully reorged unsafe head using backupUnsafe", "unsafe", e.backupUnsafeHead.ID())
		e.SetUnsafeHead(e.BackupUnsafeL2Head())
		e.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)
		return true, nil
	}
	e.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)
	// Execution engine could not reorg back to previous unsafe head.
	return true, derive.NewTemporaryError(fmt.Errorf("cannot restore unsafe chain using backupUnsafe: err: %w",
		eth.ForkchoiceUpdateErr(fcRes.PayloadStatus)))
}

func (d *EngineController) TryUpdateEngine(ctx context.Context) {
	// If we don't need to call FCU, keep going b/c this was a no-op. If we needed to
	// perform a network call, then we should yield even if we did not encounter an error.
	if err := d.tryUpdateEngine(d.ctx); err != nil && !errors.Is(err, ErrNoFCUNeeded) {
		if errors.Is(err, derive.ErrReset) {
			d.emitter.Emit(ctx, rollup.ResetEvent{Err: err})
		} else if errors.Is(err, derive.ErrTemporary) {
			d.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: err})
		} else {
			d.emitter.Emit(ctx, rollup.CriticalErrorEvent{
				Err: fmt.Errorf("unexpected tryUpdateEngine error type: %w", err),
			})
		}
	}
}

// TODO(#16917) Remove Event System Refactor Comments
// OnEvent implements event.Deriver (moved from EngDeriver)
// TryUpdateEngineEvent is replaced with TryUpdateEngine
func (d *EngineController) OnEvent(ctx context.Context, ev event.Event) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	// TODO(#16917) Remove Event System Refactor Comments
	//  PromoteUnsafeEvent, PromotePendingSafeEvent, PromoteLocalSafeEvent fan out is updated to procedural
	switch x := ev.(type) {
	case ProcessUnsafePayloadEvent:
		ref, err := derive.PayloadToBlockRef(d.rollupCfg, x.Envelope.ExecutionPayload)
		if err != nil {
			d.log.Error("failed to decode L2 block ref from payload", "err", err)
			return true
		}
		// Avoid re-processing the same unsafe payload if it has already been processed. Because a FCU event emits the ProcessUnsafePayloadEvent
		// it is possible to have multiple queued up ProcessUnsafePayloadEvent for the same L2 block. This becomes an issue when processing
		// a large number of unsafe payloads at once (like when iterating through the payload queue after the safe head has advanced).
		if ref.BlockRef().ID() == d.UnsafeL2Head().BlockRef().ID() {
			return true
		}
		if err := d.InsertUnsafePayload(d.ctx, x.Envelope, ref); err != nil {
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
	case rollup.ForceResetEvent:
		ForceEngineReset(d, x)

		// Time to apply the changes to the underlying engine
		d.TryUpdateEngine(ctx)

		v := EngineResetConfirmedEvent{
			LocalUnsafe: d.UnsafeL2Head(),
			CrossUnsafe: d.CrossUnsafeL2Head(),
			LocalSafe:   d.LocalSafeL2Head(),
			CrossSafe:   d.SafeL2Head(),
			Finalized:   d.Finalized(),
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
		if !d.rollupCfg.IsInterop(x.Ref.Time) {
			d.emitter.Emit(ctx, PromoteCrossUnsafeEvent(x))
		}
		// Try to apply the forkchoice changes
		d.TryUpdateEngine(ctx)
	case PromoteCrossUnsafeEvent:
		d.SetCrossUnsafeHead(x.Ref)
		d.emitter.Emit(ctx, CrossUnsafeUpdateEvent{
			CrossUnsafe: x.Ref,
			LocalUnsafe: d.UnsafeL2Head(),
		})
	case PendingSafeRequestEvent:
		d.emitter.Emit(ctx, PendingSafeUpdateEvent{
			PendingSafe: d.PendingSafeL2Head(),
			Unsafe:      d.UnsafeL2Head(),
		})

	case LocalSafeUpdateEvent:
		// pre-interop everything that is local-unsafe is also immediately cross-unsafe.
		if !d.rollupCfg.IsInterop(x.Ref.Time) {
			d.emitter.Emit(ctx, PromoteSafeEvent(x))
		}
	case PromoteSafeEvent:
		d.log.Debug("Updating safe", "safe", x.Ref, "unsafe", d.UnsafeL2Head())
		d.SetSafeHead(x.Ref)
		// Finalizer can pick up this safe cross-block now
		d.emitter.Emit(ctx, SafeDerivedEvent{Safe: x.Ref, Source: x.Source})
		d.emitter.Emit(ctx, CrossSafeUpdateEvent{
			CrossSafe: d.SafeL2Head(),
			LocalSafe: d.LocalSafeL2Head(),
		})
		if x.Ref.Number > d.crossUnsafeHead.Number {
			d.log.Debug("Cross Unsafe Head is stale, updating to match cross safe", "cross_unsafe", d.crossUnsafeHead, "cross_safe", x.Ref)
			d.SetCrossUnsafeHead(x.Ref)
			d.emitter.Emit(ctx, CrossUnsafeUpdateEvent{
				CrossUnsafe: x.Ref,
				LocalUnsafe: d.UnsafeL2Head(),
			})
		}
		// Try to apply the forkchoice changes
		d.TryUpdateEngine(ctx)
	case PromoteFinalizedEvent:
		if x.Ref.Number < d.Finalized().Number {
			d.log.Error("Cannot rewind finality,", "ref", x.Ref, "finalized", d.Finalized())
			return true
		}
		if x.Ref.Number > d.SafeL2Head().Number {
			d.log.Error("Block must be safe before it can be finalized", "ref", x.Ref, "safe", d.SafeL2Head())
			return true
		}
		d.SetFinalizedHead(x.Ref)
		d.emitter.Emit(ctx, FinalizedUpdateEvent(x))
		// Try to apply the forkchoice changes
		d.TryUpdateEngine(ctx)
	case CrossUpdateRequestEvent:
		if x.CrossUnsafe {
			d.emitter.Emit(ctx, CrossUnsafeUpdateEvent{
				CrossUnsafe: d.CrossUnsafeL2Head(),
				LocalUnsafe: d.UnsafeL2Head(),
			})
		}
		if x.CrossSafe {
			d.emitter.Emit(ctx, CrossSafeUpdateEvent{
				CrossSafe: d.SafeL2Head(),
				LocalSafe: d.LocalSafeL2Head(),
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

// TryUpdatePendingSafe updates the pending safe head if the new reference is newer
func (e *EngineController) TryUpdatePendingSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
	// Only promote if not already stale.
	// Resets/overwrites happen through engine-resets, not through promotion.
	if ref.Number > e.PendingSafeL2Head().Number {
		e.log.Debug("Updating pending safe", "pending_safe", ref, "local_safe", e.LocalSafeL2Head(), "unsafe", e.UnsafeL2Head(), "concluding", concluding)
		e.SetPendingSafeL2Head(ref)
		e.emitter.Emit(ctx, PendingSafeUpdateEvent{
			PendingSafe: e.PendingSafeL2Head(),
			Unsafe:      e.UnsafeL2Head(),
		})
	}
}

// TryUpdateLocalSafe updates the local safe head if the new reference is newer and concluding
func (e *EngineController) TryUpdateLocalSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
	if concluding && ref.Number > e.LocalSafeL2Head().Number {
		// Promote to local safe
		e.log.Debug("Updating local safe", "local_safe", ref, "safe", e.SafeL2Head(), "unsafe", e.UnsafeL2Head())
		e.SetLocalSafeHead(ref)
		e.emitter.Emit(ctx, LocalSafeUpdateEvent{Ref: ref, Source: source})
	}
}

// TryUpdateUnsafe updates the unsafe head and backs up the previous one if needed
func (e *EngineController) TryUpdateUnsafe(ctx context.Context, ref eth.L2BlockRef) {
	// Backup unsafeHead when new block is not built on original unsafe head.
	if e.unsafeHead.Number >= ref.Number {
		e.SetBackupUnsafeL2Head(e.unsafeHead, false)
	}
	e.SetUnsafeHead(ref)
	e.emitter.Emit(ctx, UnsafeUpdateEvent{Ref: ref})
}
