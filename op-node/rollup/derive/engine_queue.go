package derive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/async"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type AttributesWithParent struct {
	attributes   *eth.PayloadAttributes
	parent       eth.L2BlockRef
	isLastInSpan bool
}

func NewAttributesWithParent(attributes *eth.PayloadAttributes, parent eth.L2BlockRef, isLastInSpan bool) *AttributesWithParent {
	return &AttributesWithParent{attributes, parent, isLastInSpan}
}

func (a *AttributesWithParent) Attributes() *eth.PayloadAttributes {
	return a.attributes
}

type NextAttributesProvider interface {
	Origin() eth.L1BlockRef
	NextAttributes(context.Context, eth.L2BlockRef) (*AttributesWithParent, error)
}

type L2Source interface {
	PayloadByHash(context.Context, common.Hash) (*eth.ExecutionPayloadEnvelope, error)
	PayloadByNumber(context.Context, uint64) (*eth.ExecutionPayloadEnvelope, error)
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
	L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
	SystemConfigL2Fetcher
}

type Engine interface {
	ExecEngine
	L2Source
}

// EngineState provides a read-only interface of the forkchoice state properties of the L2 Engine.
type EngineState interface {
	Finalized() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
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

type LocalEngineControl interface {
	EngineControl
	ResetBuildingState()
	IsEngineSyncing() bool
	TryUpdateEngine(ctx context.Context) error
	TryBackupUnsafeReorg(ctx context.Context) (bool, error)

	PendingSafeL2Head() eth.L2BlockRef
	BackupUnsafeL2Head() eth.L2BlockRef

	SetUnsafeHead(eth.L2BlockRef)
	SetSafeHead(eth.L2BlockRef)
	SetFinalizedHead(eth.L2BlockRef)
	SetPendingSafeL2Head(eth.L2BlockRef)
	SetBackupUnsafeL2Head(block eth.L2BlockRef, triggerReorg bool)
}

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

type FinalizerHooks interface {
	// OnDerivationL1End remembers the given L1 block,
	// and finalizes any prior data with the latest finality signal based on block height.
	OnDerivationL1End(ctx context.Context, derivedFrom eth.L1BlockRef) error
	// PostProcessSafeL2 remembers the L2 block is derived from the given L1 block, for later finalization.
	PostProcessSafeL2(l2Safe eth.L2BlockRef, derivedFrom eth.L1BlockRef)
	// Reset clear recent state, to adapt to reorgs.
	Reset()
}

// EngineQueue queues up payload attributes to consolidate or process with the provided Engine
type EngineQueue struct {
	log log.Logger
	cfg *rollup.Config

	ec LocalEngineControl

	// The queued-up attributes
	safeAttributes *AttributesWithParent

	engine L2Source
	prev   NextAttributesProvider

	origin eth.L1BlockRef   // updated on resets, and whenever we read from the previous stage.
	sysCfg eth.SystemConfig // only used for pipeline resets

	metrics   Metrics
	l1Fetcher L1Fetcher

	syncCfg *sync.Config

	safeHeadNotifs       SafeHeadListener // notified when safe head is updated
	lastNotifiedSafeHead eth.L2BlockRef

	finalizer FinalizerHooks
}

// NewEngineQueue creates a new EngineQueue, which should be Reset(origin) before use.
func NewEngineQueue(log log.Logger, cfg *rollup.Config, l2Source L2Source, engine LocalEngineControl, metrics Metrics,
	prev NextAttributesProvider, l1Fetcher L1Fetcher, syncCfg *sync.Config, safeHeadNotifs SafeHeadListener,
	finalizer FinalizerHooks) *EngineQueue {
	return &EngineQueue{
		log:            log,
		cfg:            cfg,
		ec:             engine,
		engine:         l2Source,
		metrics:        metrics,
		prev:           prev,
		l1Fetcher:      l1Fetcher,
		syncCfg:        syncCfg,
		safeHeadNotifs: safeHeadNotifs,
		finalizer:      finalizer,
	}
}

// Origin identifies the L1 chain (incl.) that included and/or produced all the safe L2 blocks.
func (eq *EngineQueue) Origin() eth.L1BlockRef {
	return eq.origin
}

func (eq *EngineQueue) SystemConfig() eth.SystemConfig {
	return eq.sysCfg
}

func (eq *EngineQueue) BackupUnsafeL2Head() eth.L2BlockRef {
	return eq.ec.BackupUnsafeL2Head()
}

// Determine if the engine is syncing to the target block
func (eq *EngineQueue) isEngineSyncing() bool {
	return eq.ec.IsEngineSyncing()
}

func (eq *EngineQueue) Step(ctx context.Context) error {
	// If we don't need to call FCU to restore unsafeHead using backupUnsafe, keep going b/c
	// this was a no-op(except correcting invalid state when backupUnsafe is empty but TryBackupUnsafeReorg called).
	if fcuCalled, err := eq.ec.TryBackupUnsafeReorg(ctx); fcuCalled {
		// If we needed to perform a network call, then we should yield even if we did not encounter an error.
		return err
	}
	// If we don't need to call FCU, keep going b/c this was a no-op. If we needed to
	// perform a network call, then we should yield even if we did not encounter an error.
	if err := eq.ec.TryUpdateEngine(ctx); !errors.Is(err, ErrNoFCUNeeded) {
		return err
	}
	if eq.isEngineSyncing() {
		// The pipeline cannot move forwards if doing EL sync.
		return EngineELSyncing
	}
	if eq.safeAttributes != nil {
		return eq.tryNextSafeAttributes(ctx)
	}

	newOrigin := eq.prev.Origin()
	// Check if the L2 unsafe head origin is consistent with the new origin
	if err := eq.verifyNewL1Origin(ctx, newOrigin); err != nil {
		return err
	}
	eq.origin = newOrigin
	// make sure we track the last L2 safe head for every new L1 block
	if err := eq.postProcessSafeL2(); err != nil {
		return err
	}
	// try to finalize the L2 blocks we have synced so far (no-op if L1 finality is behind)
	if err := eq.finalizer.OnDerivationL1End(ctx, eq.origin); err != nil {
		return fmt.Errorf("finalizer OnDerivationL1End error: %w", err)
	}
	if next, err := eq.prev.NextAttributes(ctx, eq.ec.PendingSafeL2Head()); err == io.EOF {
		return io.EOF
	} else if err != nil {
		return err
	} else {
		eq.safeAttributes = next
		eq.log.Debug("Adding next safe attributes", "safe_head", eq.ec.SafeL2Head(),
			"pending_safe_head", eq.ec.PendingSafeL2Head(), "next", next)
		return NotEnoughData
	}
}

// verifyNewL1Origin checks that the L2 unsafe head still has a L1 origin that is on the canonical chain.
// If the unsafe head origin is after the new L1 origin it is assumed to still be canonical.
// The check is only required when moving to a new L1 origin.
func (eq *EngineQueue) verifyNewL1Origin(ctx context.Context, newOrigin eth.L1BlockRef) error {
	if newOrigin == eq.origin {
		return nil
	}
	unsafeOrigin := eq.ec.UnsafeL2Head().L1Origin
	if newOrigin.Number == unsafeOrigin.Number && newOrigin.ID() != unsafeOrigin {
		return NewResetError(fmt.Errorf("l1 origin was inconsistent with l2 unsafe head origin, need reset to resolve: l1 origin: %v; unsafe origin: %v",
			newOrigin.ID(), unsafeOrigin))
	}
	// Avoid requesting an older block by checking against the parent hash
	if newOrigin.Number == unsafeOrigin.Number+1 && newOrigin.ParentHash != unsafeOrigin.Hash {
		return NewResetError(fmt.Errorf("l2 unsafe head origin is no longer canonical, need reset to resolve: canonical hash: %v; unsafe origin hash: %v",
			newOrigin.ParentHash, unsafeOrigin.Hash))
	}
	if newOrigin.Number > unsafeOrigin.Number+1 {
		// If unsafe origin is further behind new origin, check it's still on the canonical chain.
		canonical, err := eq.l1Fetcher.L1BlockRefByNumber(ctx, unsafeOrigin.Number)
		if err != nil {
			return NewTemporaryError(fmt.Errorf("failed to fetch canonical L1 block at slot: %v; err: %w", unsafeOrigin.Number, err))
		}
		if canonical.ID() != unsafeOrigin {
			eq.log.Error("Resetting due to origin mismatch")
			return NewResetError(fmt.Errorf("l2 unsafe head origin is no longer canonical, need reset to resolve: canonical: %v; unsafe origin: %v",
				canonical, unsafeOrigin))
		}
	}
	return nil
}

// postProcessSafeL2 buffers the L1 block the safe head was fully derived from,
// to finalize it once the L1 block, or later, finalizes.
func (eq *EngineQueue) postProcessSafeL2() error {
	if err := eq.notifyNewSafeHead(eq.ec.SafeL2Head()); err != nil {
		return err
	}
	eq.finalizer.PostProcessSafeL2(eq.ec.SafeL2Head(), eq.origin)
	return nil
}

// notifyNewSafeHead calls the safe head listener with the current safe head and l1 origin information.
func (eq *EngineQueue) notifyNewSafeHead(safeHead eth.L2BlockRef) error {
	if eq.lastNotifiedSafeHead == safeHead {
		// No change, no need to notify
		return nil
	}
	if err := eq.safeHeadNotifs.SafeHeadUpdated(safeHead, eq.origin.ID()); err != nil {
		// At this point our state is in a potentially inconsistent state as we've updated the safe head
		// in the execution client but failed to post process it. Reset the pipeline so the safe head rolls back
		// a little (it always rolls back at least 1 block) and then it will retry storing the entry
		return NewResetError(fmt.Errorf("safe head notifications failed: %w", err))
	}
	eq.lastNotifiedSafeHead = safeHead
	return nil
}

func (eq *EngineQueue) logSyncProgress(reason string) {
	eq.log.Info("Sync progress",
		"reason", reason,
		"l2_finalized", eq.ec.Finalized(),
		"l2_safe", eq.ec.SafeL2Head(),
		"l2_pending_safe", eq.ec.PendingSafeL2Head(),
		"l2_unsafe", eq.ec.UnsafeL2Head(),
		"l2_backup_unsafe", eq.ec.BackupUnsafeL2Head(),
		"l2_time", eq.ec.UnsafeL2Head().Time,
		"l1_derived", eq.origin,
	)
}

func (eq *EngineQueue) tryNextSafeAttributes(ctx context.Context) error {
	if eq.safeAttributes == nil { // sanity check the attributes are there
		return nil
	}
	// validate the safe attributes before processing them. The engine may have completed processing them through other means.
	if eq.ec.PendingSafeL2Head() != eq.safeAttributes.parent {
		// Previously the attribute's parent was the pending safe head. If the pending safe head advances so pending safe head's parent is the same as the
		// attribute's parent then we need to cancel the attributes.
		if eq.ec.PendingSafeL2Head().ParentHash == eq.safeAttributes.parent.Hash {
			eq.log.Warn("queued safe attributes are stale, safehead progressed",
				"pending_safe_head", eq.ec.PendingSafeL2Head(), "pending_safe_head_parent", eq.ec.PendingSafeL2Head().ParentID(),
				"attributes_parent", eq.safeAttributes.parent)
			eq.safeAttributes = nil
			return nil
		}
		// If something other than a simple advance occurred, perform a full reset
		return NewResetError(fmt.Errorf("pending safe head changed to %s with parent %s, conflicting with queued safe attributes on top of %s",
			eq.ec.PendingSafeL2Head(), eq.ec.PendingSafeL2Head().ParentID(), eq.safeAttributes.parent))

	}
	if eq.ec.PendingSafeL2Head().Number < eq.ec.UnsafeL2Head().Number {
		return eq.consolidateNextSafeAttributes(ctx)
	} else if eq.ec.PendingSafeL2Head().Number == eq.ec.UnsafeL2Head().Number {
		return eq.forceNextSafeAttributes(ctx)
	} else {
		// For some reason the unsafe head is behind the pending safe head. Log it, and correct it.
		eq.log.Error("invalid sync state, unsafe head is behind pending safe head", "unsafe", eq.ec.UnsafeL2Head(), "pending_safe", eq.ec.PendingSafeL2Head())
		eq.ec.SetUnsafeHead(eq.ec.PendingSafeL2Head())
		return nil
	}
}

// consolidateNextSafeAttributes tries to match the next safe attributes against the existing unsafe chain,
// to avoid extra processing or unnecessary unwinding of the chain.
// However, if the attributes do not match, they will be forced with forceNextSafeAttributes.
func (eq *EngineQueue) consolidateNextSafeAttributes(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	envelope, err := eq.engine.PayloadByNumber(ctx, eq.ec.PendingSafeL2Head().Number+1)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			// engine may have restarted, or inconsistent safe head. We need to reset
			return NewResetError(fmt.Errorf("expected engine was synced and had unsafe block to reconcile, but cannot find the block: %w", err))
		}
		return NewTemporaryError(fmt.Errorf("failed to get existing unsafe payload to compare against derived attributes from L1: %w", err))
	}
	if err := AttributesMatchBlock(eq.cfg, eq.safeAttributes.attributes, eq.ec.PendingSafeL2Head().Hash, envelope, eq.log); err != nil {
		eq.log.Warn("L2 reorg: existing unsafe block does not match derived attributes from L1", "err", err, "unsafe", eq.ec.UnsafeL2Head(), "pending_safe", eq.ec.PendingSafeL2Head(), "safe", eq.ec.SafeL2Head())
		// geth cannot wind back a chain without reorging to a new, previously non-canonical, block
		return eq.forceNextSafeAttributes(ctx)
	}
	ref, err := PayloadToBlockRef(eq.cfg, envelope.ExecutionPayload)
	if err != nil {
		return NewResetError(fmt.Errorf("failed to decode L2 block ref from payload: %w", err))
	}
	eq.ec.SetPendingSafeL2Head(ref)
	if eq.safeAttributes.isLastInSpan {
		eq.ec.SetSafeHead(ref)
		if err := eq.postProcessSafeL2(); err != nil {
			return err
		}
	}
	// unsafe head stays the same, we did not reorg the chain.
	eq.safeAttributes = nil
	eq.logSyncProgress("reconciled with L1")

	return nil
}

// forceNextSafeAttributes inserts the provided attributes, reorging away any conflicting unsafe chain.
func (eq *EngineQueue) forceNextSafeAttributes(ctx context.Context) error {
	if eq.safeAttributes == nil {
		return nil
	}
	attrs := eq.safeAttributes.attributes
	lastInSpan := eq.safeAttributes.isLastInSpan
	errType, err := eq.StartPayload(ctx, eq.ec.PendingSafeL2Head(), eq.safeAttributes, true)
	if err == nil {
		_, errType, err = eq.ec.ConfirmPayload(ctx, async.NoOpGossiper{}, &conductor.NoOpConductor{})
	}
	if err != nil {
		switch errType {
		case BlockInsertTemporaryErr:
			// RPC errors are recoverable, we can retry the buffered payload attributes later.
			return NewTemporaryError(fmt.Errorf("temporarily cannot insert new safe block: %w", err))
		case BlockInsertPrestateErr:
			_ = eq.CancelPayload(ctx, true)
			return NewResetError(fmt.Errorf("need reset to resolve pre-state problem: %w", err))
		case BlockInsertPayloadErr:
			_ = eq.CancelPayload(ctx, true)
			eq.log.Warn("could not process payload derived from L1 data, dropping batch", "err", err)
			// Count the number of deposits to see if the tx list is deposit only.
			depositCount := 0
			for _, tx := range attrs.Transactions {
				if len(tx) > 0 && tx[0] == types.DepositTxType {
					depositCount += 1
				}
			}
			// Deposit transaction execution errors are suppressed in the execution engine, but if the
			// block is somehow invalid, there is nothing we can do to recover & we should exit.
			// TODO: Can this be triggered by an empty batch with invalid data (like parent hash or gas limit?)
			if len(attrs.Transactions) == depositCount {
				eq.log.Error("deposit only block was invalid", "parent", eq.safeAttributes.parent, "err", err)
				return NewCriticalError(fmt.Errorf("failed to process block with only deposit transactions: %w", err))
			}
			// drop the payload without inserting it
			eq.safeAttributes = nil
			// Revert the pending safe head to the safe head.
			eq.ec.SetPendingSafeL2Head(eq.ec.SafeL2Head())
			// suppress the error b/c we want to retry with the next batch from the batch queue
			// If there is no valid batch the node will eventually force a deposit only block. If
			// the deposit only block fails, this will return the critical error above.

			// Try to restore to previous known unsafe chain.
			eq.ec.SetBackupUnsafeL2Head(eq.ec.BackupUnsafeL2Head(), true)

			return nil
		default:
			return NewCriticalError(fmt.Errorf("unknown InsertHeadBlock error type %d: %w", errType, err))
		}
	}
	eq.safeAttributes = nil
	eq.logSyncProgress("processed safe block derived from L1")
	if lastInSpan {
		if err := eq.postProcessSafeL2(); err != nil {
			return err
		}
	}

	return nil
}

func (eq *EngineQueue) StartPayload(ctx context.Context, parent eth.L2BlockRef, attrs *AttributesWithParent, updateSafe bool) (errType BlockInsertionErrType, err error) {
	return eq.ec.StartPayload(ctx, parent, attrs, updateSafe)
}

func (eq *EngineQueue) ConfirmPayload(ctx context.Context, agossip async.AsyncGossiper, sequencerConductor conductor.SequencerConductor) (out *eth.ExecutionPayloadEnvelope, errTyp BlockInsertionErrType, err error) {
	return eq.ec.ConfirmPayload(ctx, agossip, sequencerConductor)
}

func (eq *EngineQueue) CancelPayload(ctx context.Context, force bool) error {
	return eq.ec.CancelPayload(ctx, force)
}

func (eq *EngineQueue) BuildingPayload() (onto eth.L2BlockRef, id eth.PayloadID, safe bool) {
	return eq.ec.BuildingPayload()
}

// Reset walks the L2 chain backwards until it finds an L2 block whose L1 origin is canonical.
// The unsafe head is set to the head of the L2 chain, unless the existing safe head is not canonical.
func (eq *EngineQueue) Reset(ctx context.Context, _ eth.L1BlockRef, _ eth.SystemConfig) error {
	result, err := sync.FindL2Heads(ctx, eq.cfg, eq.l1Fetcher, eq.engine, eq.log, eq.syncCfg)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to find the L2 Heads to start from: %w", err))
	}
	finalized, safe, unsafe := result.Finalized, result.Safe, result.Unsafe
	l1Origin, err := eq.l1Fetcher.L1BlockRefByHash(ctx, safe.L1Origin.Hash)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to fetch the new L1 progress: origin: %v; err: %w", safe.L1Origin, err))
	}
	if safe.Time < l1Origin.Time {
		return NewResetError(fmt.Errorf("cannot reset block derivation to start at L2 block %s with time %d older than its L1 origin %s with time %d, time invariant is broken",
			safe, safe.Time, l1Origin, l1Origin.Time))
	}

	// Walk back L2 chain to find the L1 origin that is old enough to start buffering channel data from.
	pipelineL2 := safe
	for {
		afterL2Genesis := pipelineL2.Number > eq.cfg.Genesis.L2.Number
		afterL1Genesis := pipelineL2.L1Origin.Number > eq.cfg.Genesis.L1.Number
		afterChannelTimeout := pipelineL2.L1Origin.Number+eq.cfg.ChannelTimeout > l1Origin.Number
		if afterL2Genesis && afterL1Genesis && afterChannelTimeout {
			parent, err := eq.engine.L2BlockRefByHash(ctx, pipelineL2.ParentHash)
			if err != nil {
				return NewResetError(fmt.Errorf("failed to fetch L2 parent block %s", pipelineL2.ParentID()))
			}
			pipelineL2 = parent
		} else {
			break
		}
	}
	pipelineOrigin, err := eq.l1Fetcher.L1BlockRefByHash(ctx, pipelineL2.L1Origin.Hash)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to fetch the new L1 progress: origin: %s; err: %w", pipelineL2.L1Origin, err))
	}
	l1Cfg, err := eq.engine.SystemConfigByL2Hash(ctx, pipelineL2.Hash)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to fetch L1 config of L2 block %s: %w", pipelineL2.ID(), err))
	}
	eq.log.Debug("Reset engine queue", "safeHead", safe, "unsafe", unsafe, "safe_timestamp", safe.Time, "unsafe_timestamp", unsafe.Time, "l1Origin", l1Origin)
	eq.ec.SetUnsafeHead(unsafe)
	eq.ec.SetSafeHead(safe)
	eq.ec.SetPendingSafeL2Head(safe)
	eq.ec.SetFinalizedHead(finalized)
	eq.ec.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)
	eq.safeAttributes = nil
	eq.ec.ResetBuildingState()
	eq.finalizer.Reset()
	// note: finalizedL1 and triedFinalizeAt do not reset, since these do not change between reorgs.
	// note: we do not clear the unsafe payloads queue; if the payloads are not applicable anymore the parent hash checks will clear out the old payloads.
	eq.origin = pipelineOrigin
	eq.sysCfg = l1Cfg
	eq.lastNotifiedSafeHead = safe
	if err := eq.safeHeadNotifs.SafeHeadReset(safe); err != nil {
		return err
	}
	if eq.safeHeadNotifs.Enabled() && safe.Number == eq.cfg.Genesis.L2.Number && safe.Hash == eq.cfg.Genesis.L2.Hash {
		// The rollup genesis block is always safe by definition. So if the pipeline resets this far back we know
		// we will process all safe head updates and can record genesis as always safe from L1 genesis.
		// Note that it is not safe to use cfg.Genesis.L1 here as it is the block immediately before the L2 genesis
		// but the contracts may have been deployed earlier than that, allowing creating a dispute game
		// with a L1 head prior to cfg.Genesis.L1
		l1Genesis, err := eq.l1Fetcher.L1BlockRefByNumber(ctx, 0)
		if err != nil {
			return fmt.Errorf("failed to retrieve L1 genesis: %w", err)
		}
		if err := eq.safeHeadNotifs.SafeHeadUpdated(safe, l1Genesis.ID()); err != nil {
			return err
		}
	}
	eq.logSyncProgress("reset derivation work")
	return io.EOF
}
