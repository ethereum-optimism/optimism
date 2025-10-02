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

// Max memory used for buffering unsafe payloads
const maxUnsafePayloadsMemory = 500 * 1024 * 1024

// ResetEngineRequestEvent requests the EngineController to walk
// the L2 chain backwards until it finds a plausible unsafe head,
// and find an L2 safe block that is guaranteed to still be from the L1 chain.
// This event is not used in interop.
type ResetEngineRequestEvent struct {
}

func (ev ResetEngineRequestEvent) String() string {
	return "reset-engine-request"
}

type Engine interface {
	ExecEngine
	derive.L2Source
}
type ExecEngine interface {
	GetPayload(ctx context.Context, payloadInfo eth.PayloadInfo) (*eth.ExecutionPayloadEnvelope, error)
	ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	NewPayload(ctx context.Context, payload *eth.ExecutionPayload, parentBeaconBlockRoot *common.Hash) (*eth.PayloadStatusV1, error)
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L2BlockRef, error)
}

// Metrics interface for CLSync functionality
type Metrics interface {
	RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID)
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

	// L1 chain for reset functionality
	l1 sync.L1Chain

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

	// Components that need to be notified during force reset
	attributesResetter     AttributesForceResetter
	pipelineResetter       PipelineForceResetter
	originSelectorResetter OriginSelectorForceResetter

	// Handler for cross-unsafe and cross-safe updates
	crossUpdateHandler CrossUpdateHandler

	unsafePayloads *PayloadsQueue // queue of unsafe payloads, ordered by ascending block number, may have gaps and duplicates
}

var _ event.Deriver = (*EngineController)(nil)

func NewEngineController(ctx context.Context, engine ExecEngine, log log.Logger, m opmetrics.Metricer,
	rollupCfg *rollup.Config, syncCfg *sync.Config, l1 sync.L1Chain, emitter event.Emitter,
) *EngineController {
	syncStatus := syncStatusCL
	if syncCfg.SyncMode == sync.ELSync {
		syncStatus = syncStatusWillStartEL
	}

	return &EngineController{
		engine:         engine,
		log:            log,
		metrics:        m,
		chainSpec:      rollup.NewChainSpec(rollupCfg),
		rollupCfg:      rollupCfg,
		syncCfg:        syncCfg,
		syncStatus:     syncStatus,
		clock:          clock.SystemClock,
		l1:             l1,
		ctx:            ctx,
		emitter:        emitter,
		unsafePayloads: NewPayloadsQueue(log, maxUnsafePayloadsMemory, payloadMemSize),
	}
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

func (e *EngineController) BackupUnsafeL2Head() eth.L2BlockRef {
	return e.backupUnsafeHead
}

func (e *EngineController) RequestForkchoiceUpdate(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.requestForkchoiceUpdate(ctx)
}

func (e *EngineController) requestForkchoiceUpdate(ctx context.Context) {
	e.emitter.Emit(ctx, ForkchoiceUpdateEvent{
		UnsafeL2Head:    e.unsafeHead,
		SafeL2Head:      e.safeHead,
		FinalizedL2Head: e.finalizedHead,
	})
}

func (e *EngineController) IsEngineSyncing() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.isEngineSyncing()
}

func (e *EngineController) isEngineSyncing() bool {
	return e.syncStatus == syncStatusWillStartEL ||
		e.syncStatus == syncStatusStartedEL ||
		e.syncStatus == syncStatusFinishedELButNotFinalized
}

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

func (e *EngineController) SetCrossUpdateHandler(handler CrossUpdateHandler) {
	e.crossUpdateHandler = handler
}

func (e *EngineController) onUnsafeUpdate(ctx context.Context, crossUnsafe, localUnsafe eth.L2BlockRef) {
	// Nil check required because op-program omits this handler.
	if e.crossUpdateHandler != nil {
		e.crossUpdateHandler.OnCrossUnsafeUpdate(ctx, crossUnsafe, localUnsafe)
	}
}

func (e *EngineController) onSafeUpdate(ctx context.Context, crossSafe, localSafe eth.L2BlockRef) {
	// Nil check required because op-program omits this handler.
	if e.crossUpdateHandler != nil {
		e.crossUpdateHandler.OnCrossSafeUpdate(ctx, crossSafe, localSafe)
	}
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
				"l2_time", e.unsafeHead.Time,
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

func (e *EngineController) tryUpdateEngineInternal(ctx context.Context) error {
	if !e.needFCUCall {
		return ErrNoFCUNeeded
	}
	if e.isEngineSyncing() {
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
		e.requestForkchoiceUpdate(ctx)
	}
	if e.unsafeHead == e.safeHead && e.safeHead == e.pendingSafeHead {
		// Remove backupUnsafeHead because this backup will be never used after consolidation.
		e.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)
	}
	e.needFCUCall = false
	return nil
}

// tryUpdateEngine attempts to update the engine with the current forkchoice state of the rollup node,
// this is a no-op if the nodes already agree on the forkchoice state.
func (e *EngineController) tryUpdateEngine(ctx context.Context) {
	// If we don't need to call FCU, keep going b/c this was a no-op. If we needed to
	// perform a network call, then we should yield even if we did not encounter an error.
	if err := e.tryUpdateEngineInternal(e.ctx); err != nil && !errors.Is(err, ErrNoFCUNeeded) {
		if errors.Is(err, derive.ErrReset) {
			e.emitter.Emit(ctx, rollup.ResetEvent{Err: err})
		} else if errors.Is(err, derive.ErrTemporary) {
			e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: err})
		} else {
			e.emitter.Emit(ctx, rollup.CriticalErrorEvent{
				Err: fmt.Errorf("unexpected tryUpdateEngine error type: %w", err),
			})
		}
	}
}

func (e *EngineController) InsertUnsafePayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.insertUnsafePayload(ctx, envelope, ref)
}

func (e *EngineController) insertUnsafePayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef) error {
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
		e.onSafeUpdate(ctx, ref, ref)
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
		e.requestForkchoiceUpdate(ctx)
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
	if e.isEngineSyncing() {
		e.log.Warn("Attempting to unsafe reorg using backupUnsafe while EL syncing")
		return false
	}
	if e.backupUnsafeHead == (eth.L2BlockRef{}) { // sanity check backupUnsafeHead is there
		e.log.Warn("Attempting to unsafe reorg using backupUnsafe even though it is empty")
		e.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)
		return false
	}
	return true
}

func (e *EngineController) TryBackupUnsafeReorg(ctx context.Context) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.tryBackupUnsafeReorg(ctx)
}

// tryBackupUnsafeReorg attempts to reorg(restore) unsafe head to backupUnsafeHead.
// If succeeds, update current forkchoice state to the rollup node.
func (e *EngineController) tryBackupUnsafeReorg(ctx context.Context) (bool, error) {
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
		// Execution engine accepted the reorg.
		e.log.Info("successfully reorged unsafe head using backupUnsafe", "unsafe", e.backupUnsafeHead.ID())
		e.SetUnsafeHead(e.backupUnsafeHead)
		e.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)

		e.requestForkchoiceUpdate(ctx)
		return true, nil
	}
	e.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)
	// Execution engine could not reorg back to previous unsafe head.
	return true, derive.NewTemporaryError(fmt.Errorf("cannot restore unsafe chain using backupUnsafe: err: %w",
		eth.ForkchoiceUpdateErr(fcRes.PayloadStatus)))
}

func (e *EngineController) TryUpdateEngine(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tryUpdateEngine(ctx)
}

func (e *EngineController) OnEvent(ctx context.Context, ev event.Event) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	// TODO(#16917) Remove Event System Refactor Comments
	//  PromoteUnsafeEvent, PromotePendingSafeEvent, PromoteLocalSafeEvent fan out is updated to procedural
	//  PromoteSafeEvent fan out is updated to procedural PromoteSafe method call
	switch x := ev.(type) {
	case UnsafeUpdateEvent:
		// pre-interop everything that is local-unsafe is also immediately cross-unsafe.
		if !e.rollupCfg.IsInterop(x.Ref.Time) {
			e.emitter.Emit(ctx, PromoteCrossUnsafeEvent(x))
		}
		// Try to apply the forkchoice changes
		e.tryUpdateEngine(ctx)
	case PromoteCrossUnsafeEvent:
		e.SetCrossUnsafeHead(x.Ref)
		e.onUnsafeUpdate(ctx, x.Ref, e.unsafeHead)
	case LocalSafeUpdateEvent:
		// pre-interop everything that is local-safe is also immediately cross-safe.
		if !e.rollupCfg.IsInterop(x.Ref.Time) {
			e.PromoteSafe(ctx, x.Ref, x.Source)
		}
	case InteropInvalidateBlockEvent:
		e.emitter.Emit(ctx, BuildStartEvent{Attributes: x.Attributes})
	case BuildStartEvent:
		e.onBuildStart(ctx, x)
	case BuildStartedEvent:
		e.onBuildStarted(ctx, x)
	case BuildSealEvent:
		e.onBuildSeal(ctx, x)
	case BuildSealedEvent:
		e.onBuildSealed(ctx, x)
	case BuildInvalidEvent:
		e.onBuildInvalid(ctx, x)
	case BuildCancelEvent:
		e.onBuildCancel(ctx, x)
	case PayloadProcessEvent:
		e.onPayloadProcess(ctx, x)
	case PayloadSuccessEvent:
		e.onPayloadSuccess(ctx, x)
	case PayloadInvalidEvent:
		e.onInvalidPayload(x)
	case ForkchoiceUpdateEvent:
		e.onForkchoiceUpdate(ctx, x)
	case ResetEngineRequestEvent:
		e.onResetEngineRequest(ctx)
	default:
		return false
	}
	return true
}

func (e *EngineController) RequestPendingSafeUpdate(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.emitter.Emit(ctx, PendingSafeUpdateEvent{
		PendingSafe: e.pendingSafeHead,
		Unsafe:      e.unsafeHead,
	})
}

// TryUpdatePendingSafe updates the pending safe head if the new reference is newer, acquiring lock
func (e *EngineController) TryUpdatePendingSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tryUpdatePendingSafe(ctx, ref, concluding, source)
}

// tryUpdatePendingSafe updates the pending safe head if the new reference is newer
func (e *EngineController) tryUpdatePendingSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
	// Only promote if not already stale.
	// Resets/overwrites happen through engine-resets, not through promotion.
	if ref.Number > e.pendingSafeHead.Number {
		e.log.Debug("Updating pending safe", "pending_safe", ref, "local_safe", e.localSafeHead, "unsafe", e.unsafeHead, "concluding", concluding)
		e.SetPendingSafeL2Head(ref)
		e.emitter.Emit(ctx, PendingSafeUpdateEvent{
			PendingSafe: e.pendingSafeHead,
			Unsafe:      e.unsafeHead,
		})
	}
}

// TryUpdateLocalSafe updates the local safe head if the new reference is newer and concluding, acquiring lock
func (e *EngineController) TryUpdateLocalSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tryUpdateLocalSafe(ctx, ref, concluding, source)
}

// tryUpdateLocalSafe updates the local safe head if the new reference is newer and concluding
func (e *EngineController) tryUpdateLocalSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
	if concluding && ref.Number > e.localSafeHead.Number {
		// Promote to local safe
		e.log.Debug("Updating local safe", "local_safe", ref, "safe", e.safeHead, "unsafe", e.unsafeHead)
		e.SetLocalSafeHead(ref)
		e.emitter.Emit(ctx, LocalSafeUpdateEvent{Ref: ref, Source: source})
	}
}

// TryUpdateUnsafe updates the unsafe head and backs up the previous one if needed
func (e *EngineController) tryUpdateUnsafe(ctx context.Context, ref eth.L2BlockRef) {
	// Backup unsafeHead when new block is not built on original unsafe head.
	if e.unsafeHead.Number >= ref.Number {
		e.SetBackupUnsafeL2Head(e.unsafeHead, false)
	}
	e.SetUnsafeHead(ref)
	e.emitter.Emit(ctx, UnsafeUpdateEvent{Ref: ref})
}

func (e *EngineController) PromoteSafe(ctx context.Context, ref eth.L2BlockRef, source eth.L1BlockRef) {
	e.log.Debug("Updating safe", "safe", ref, "unsafe", e.unsafeHead)
	e.SetSafeHead(ref)
	// Finalizer can pick up this safe cross-block now
	e.emitter.Emit(ctx, SafeDerivedEvent{Safe: ref, Source: source})
	e.onSafeUpdate(ctx, e.safeHead, e.localSafeHead)
	if ref.Number > e.crossUnsafeHead.Number {
		e.log.Debug("Cross Unsafe Head is stale, updating to match cross safe", "cross_unsafe", e.crossUnsafeHead, "cross_safe", ref)
		e.SetCrossUnsafeHead(ref)
		e.onUnsafeUpdate(ctx, ref, e.unsafeHead)
	}
	// Try to apply the forkchoice changes
	e.tryUpdateEngine(ctx)
}

func (e *EngineController) PromoteFinalized(ctx context.Context, ref eth.L2BlockRef) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.promoteFinalized(ctx, ref)
}
func (e *EngineController) promoteFinalized(ctx context.Context, ref eth.L2BlockRef) {
	if ref.Number < e.finalizedHead.Number {
		e.log.Error("Cannot rewind finality,", "ref", ref, "finalized", e.finalizedHead)
		return
	}
	if ref.Number > e.safeHead.Number {
		e.log.Error("Block must be safe before it can be finalized", "ref", ref, "safe", e.safeHead)
		return
	}
	e.SetFinalizedHead(ref)
	e.emitter.Emit(ctx, FinalizedUpdateEvent{Ref: ref})
	// Try to apply the forkchoice changes
	e.tryUpdateEngine(ctx)
}

// SetAttributesResetter sets the attributes component that needs force reset notifications
func (e *EngineController) SetAttributesResetter(resetter AttributesForceResetter) {
	e.attributesResetter = resetter
}

// SetPipelineResetter sets the pipeline component that needs force reset notifications
func (e *EngineController) SetPipelineResetter(resetter PipelineForceResetter) {
	e.pipelineResetter = resetter
}

// SetOriginSelectorResetter sets the origin selector component that needs force reset notifications
func (e *EngineController) SetOriginSelectorResetter(resetter OriginSelectorForceResetter) {
	e.originSelectorResetter = resetter
}

// ForceReset performs a forced reset to the specified block references, acquiring lock
func (e *EngineController) ForceReset(ctx context.Context, localUnsafe, crossUnsafe, localSafe, crossSafe, finalized eth.L2BlockRef) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.forceReset(ctx, localUnsafe, crossUnsafe, localSafe, crossSafe, finalized)
}

// forceReset performs a forced reset to the specified block references
func (e *EngineController) forceReset(ctx context.Context, localUnsafe, crossUnsafe, localSafe, crossSafe, finalized eth.L2BlockRef) {
	// Reset other components before resetting the engine
	if e.attributesResetter != nil {
		e.attributesResetter.ForceReset(ctx, localUnsafe, crossUnsafe, localSafe, crossSafe, finalized)
	}
	if e.pipelineResetter != nil {
		e.pipelineResetter.ResetPipeline()
	}
	// originSelectorResetter is only present when sequencing is enabled
	if e.originSelectorResetter != nil {
		e.originSelectorResetter.ResetOrigins()
	}

	ForceEngineReset(e, localUnsafe, crossUnsafe, localSafe, crossSafe, finalized)

	if e.pipelineResetter != nil {
		e.emitter.Emit(ctx, derive.ConfirmPipelineResetEvent{})
	}

	// Time to apply the changes to the underlying engine
	e.tryUpdateEngine(ctx)

	v := EngineResetConfirmedEvent{
		LocalUnsafe: e.unsafeHead,
		CrossUnsafe: e.crossUnsafeHead,
		LocalSafe:   e.localSafeHead,
		CrossSafe:   e.safeHead,
		Finalized:   e.finalizedHead,
	}
	// We do not emit the original event values, since those might not be set (optional attributes).
	e.emitter.Emit(ctx, v)
	e.log.Info("Reset of Engine is completed",
		"local_unsafe", v.LocalUnsafe,
		"cross_unsafe", v.CrossUnsafe,
		"local_safe", v.LocalSafe,
		"cross_safe", v.CrossSafe,
		"finalized", v.Finalized,
	)
}

// LowestQueuedUnsafeBlock retrieves the first queued-up L2 unsafe payload, or a zeroed reference if there is none.
func (e *EngineController) LowestQueuedUnsafeBlock() eth.L2BlockRef {
	payload := e.unsafePayloads.Peek()
	if payload == nil {
		return eth.L2BlockRef{}
	}
	ref, err := derive.PayloadToBlockRef(e.rollupCfg, payload.ExecutionPayload)
	if err != nil {
		return eth.L2BlockRef{}
	}
	return ref
}

// onInvalidPayload checks if the first next-up payload matches the invalid payload.
// If so, the payload is dropped, to give the next payloads a try.
func (e *EngineController) onInvalidPayload(x PayloadInvalidEvent) {
	e.log.Debug("Received invalid payload report", "block", x.Envelope.ExecutionPayload.ID(),
		"err", x.Err, "timestamp", uint64(x.Envelope.ExecutionPayload.Timestamp))

	block := x.Envelope.ExecutionPayload
	if peek := e.unsafePayloads.Peek(); peek != nil &&
		block.BlockHash == peek.ExecutionPayload.BlockHash {
		e.log.Warn("Dropping invalid unsafe payload",
			"hash", block.BlockHash, "number", uint64(block.BlockNumber),
			"timestamp", uint64(block.Timestamp))
		e.unsafePayloads.Pop()
	}
}

// onForkchoiceUpdate refreshes unsafe payload queue and peeks at the next applicable unsafe payload, if any,
// to apply on top of the received forkchoice pre-state.
// The payload is held on to until the forkchoice changes (success case) or the payload is reported to be invalid.
func (e *EngineController) onForkchoiceUpdate(ctx context.Context, event ForkchoiceUpdateEvent) {
	e.log.Debug("Received forkchoice update",
		"unsafe", event.UnsafeL2Head, "safe", event.SafeL2Head, "finalized", event.FinalizedL2Head)

	e.unsafePayloads.DropInapplicableUnsafePayloads(event)
	nextEnvelope := e.unsafePayloads.Peek()
	if nextEnvelope == nil {
		e.log.Debug("No unsafe payload to process")
		return
	}

	// Only process the next payload if it is applicable on top of the current unsafe head.
	// This avoids prematurely attempting to insert non-adjacent payloads (e.g. height gaps),
	// which could otherwise trigger EL sync behavior.
	refParentHash := nextEnvelope.ExecutionPayload.ParentHash
	refBlockNumber := uint64(nextEnvelope.ExecutionPayload.BlockNumber)
	if refParentHash != event.UnsafeL2Head.Hash || refBlockNumber != event.UnsafeL2Head.Number+1 {
		e.log.Debug("Next unsafe payload is not applicable yet",
			"nextHash", nextEnvelope.ExecutionPayload.BlockHash, "nextNumber", refBlockNumber, "unsafe", event.UnsafeL2Head)
		return
	}

	// We don't pop from the queue. If there is a temporary error then we can retry.
	// Upon next forkchoice update or invalid-payload event we can remove it from the queue.
	e.processUnsafePayload(ctx, nextEnvelope)
}

// processUnsafePayload processes an unsafe payload by inserting it into the engine.
func (e *EngineController) processUnsafePayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) {
	ref, err := derive.PayloadToBlockRef(e.rollupCfg, envelope.ExecutionPayload)
	if err != nil {
		e.log.Error("failed to decode L2 block ref from payload", "err", err)
		return
	}
	// Avoid re-processing the same unsafe payload if it has already been processed. Because a FCU event calls processUnsafePayload
	// it is possible to have multiple queued up calls for the same L2 block. This becomes an issue when processing
	// a large number of unsafe payloads at once (like when iterating through the payload queue after the safe head has advanced).
	if ref.BlockRef().ID() == e.unsafeHead.BlockRef().ID() {
		return
	}
	if err := e.insertUnsafePayload(e.ctx, envelope, ref); err != nil {
		e.log.Info("failed to insert payload", "ref", ref,
			"txs", len(envelope.ExecutionPayload.Transactions), "err", err)
		// yes, duplicate error-handling. After all derivers are interacting with the engine
		// through events, we can drop the engine-controller interface:
		// unify the events handler with the engine-controller,
		// remove a lot of code, and not do this error translation.
		if errors.Is(err, derive.ErrReset) {
			e.emitter.Emit(ctx, rollup.ResetEvent{Err: err})
		} else if errors.Is(err, derive.ErrTemporary) {
			e.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: err})
		} else {
			e.emitter.Emit(ctx, rollup.CriticalErrorEvent{
				Err: fmt.Errorf("unexpected InsertUnsafePayload error type: %w", err),
			})
		}
	} else {
		e.log.Info("successfully processed payload", "ref", ref, "txs", len(envelope.ExecutionPayload.Transactions))
	}
}

// AddUnsafePayload schedules an execution payload to be processed, ahead of deriving it from L1.
func (e *EngineController) AddUnsafePayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) {
	if envelope == nil {
		e.log.Error("AddUnsafePayload cannot add nil unsafe payload")
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	e.log.Debug("Received payload", "payload", envelope.ExecutionPayload.ID())

	if err := e.unsafePayloads.Push(envelope); err != nil {
		e.log.Warn("Could not add unsafe payload", "id", envelope.ExecutionPayload.ID(), "timestamp", uint64(envelope.ExecutionPayload.Timestamp), "err", err)
		return
	}
	p := e.unsafePayloads.Peek()
	e.metrics.RecordUnsafePayloadsBuffer(uint64(e.unsafePayloads.Len()), e.unsafePayloads.MemSize(), p.ExecutionPayload.ID())
	e.log.Trace("Next unsafe payload to process", "next", p.ExecutionPayload.ID(), "timestamp", uint64(p.ExecutionPayload.Timestamp))

	// request forkchoice update directly so we can process the payload
	e.requestForkchoiceUpdate(ctx)
}

// onResetEngineRequest handles the ResetEngineRequestEvent by finding L2 heads and performing a force reset
func (e *EngineController) onResetEngineRequest(ctx context.Context) {
	result, err := sync.FindL2Heads(e.ctx, e.rollupCfg, e.l1, e.engine, e.log, e.syncCfg)
	if err != nil {
		e.emitter.Emit(ctx, rollup.ResetEvent{
			Err: fmt.Errorf("failed to find the L2 Heads to start from: %w", err),
		})
		return
	}
	e.forceReset(ctx, result.Unsafe, result.Unsafe, result.Safe, result.Safe, result.Finalized)
}

var ErrEngineSyncing = errors.New("engine is syncing")

type BlockInsertionErrType uint

const (
	// BlockInsertOK indicates that the payload was successfully executed and appended to the canonical chain.
	BlockInsertOK BlockInsertionErrType = iota
	// BlockInsertTemporaryErr indicates that the insertion failed but may succeed at a later time without changes to the payload.
	BlockInsertTemporaryErr
	// BlockInsertPrestateErr indicates that the pre-state to insert the payload could not be prepared, e.g. due to missing chain data.
	BlockInsertPrestateErr
	// BlockInsertPayloadErr indicates that the payload was invalid and cannot become canonical.
	BlockInsertPayloadErr
)

// startPayload starts an execution payload building process in the engine, with the given attributes.
// The severity of the error is distinguished to determine whether the same payload attributes may be re-attempted later.
func (e *EngineController) startPayload(ctx context.Context, fc eth.ForkchoiceState, attrs *eth.PayloadAttributes) (id eth.PayloadID, errType BlockInsertionErrType, err error) {
	fcRes, err := e.engine.ForkchoiceUpdate(ctx, &fc, attrs)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) {
			switch code := eth.ErrorCode(rpcErr.ErrorCode()); code {
			case eth.InvalidForkchoiceState:
				return eth.PayloadID{}, BlockInsertPrestateErr, fmt.Errorf("pre-block-creation forkchoice update was inconsistent with engine, need reset to resolve: %w", err)
			case eth.InvalidPayloadAttributes:
				return eth.PayloadID{}, BlockInsertPayloadErr, fmt.Errorf("payload attributes are not valid, cannot build block: %w", err)
			default:
				if code.IsEngineError() {
					return eth.PayloadID{}, BlockInsertPrestateErr, fmt.Errorf("unexpected engine error code in forkchoice-updated response: %w", err)
				}
				return eth.PayloadID{}, BlockInsertTemporaryErr, fmt.Errorf("unexpected generic error code in forkchoice-updated response: %w", err)
			}
		}

		return eth.PayloadID{}, BlockInsertTemporaryErr, fmt.Errorf("failed to create new block via forkchoice: %w", err)
	}

	switch fcRes.PayloadStatus.Status {
	// TODO: snap sync - specify explicit different error type if node is syncing
	case eth.ExecutionInvalid, eth.ExecutionInvalidBlockHash:
		return eth.PayloadID{}, BlockInsertPayloadErr, eth.ForkchoiceUpdateErr(fcRes.PayloadStatus)
	case eth.ExecutionValid:
		if fcRes.PayloadID == nil {
			return eth.PayloadID{}, BlockInsertTemporaryErr, errors.New("nil id in forkchoice result when expecting a valid ID")
		}
		return *fcRes.PayloadID, BlockInsertOK, nil
	case eth.ExecutionSyncing:
		return eth.PayloadID{}, BlockInsertTemporaryErr, ErrEngineSyncing
	default:
		return eth.PayloadID{}, BlockInsertTemporaryErr, eth.ForkchoiceUpdateErr(fcRes.PayloadStatus)
	}
}
