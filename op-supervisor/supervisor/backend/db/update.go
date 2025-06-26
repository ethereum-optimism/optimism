package db

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/superevents"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func (db *ChainsDB) AddLog(
	chain eth.ChainID,
	logHash common.Hash,
	parentBlock eth.BlockID,
	logIdx uint32,
	execMsg *types.ExecutingMessage,
) error {
	logDB, ok := db.logDBs.Get(chain)
	if !ok {
		return fmt.Errorf("cannot AddLog: %w: %v", types.ErrUnknownChain, chain)
	}
	return logDB.AddLog(logHash, parentBlock, logIdx, execMsg)
}

// SealBlock seals the block in the logDB.
// The database needs to be initialized.
func (db *ChainsDB) SealBlock(chain eth.ChainID, block eth.BlockRef) error {
	return db.sealBlock(chain, block, false)
}

func (db *ChainsDB) sealBlock(chain eth.ChainID, block eth.BlockRef, mayInit bool) error {
	logDB, ok := db.logDBs.Get(chain)
	if !ok {
		return fmt.Errorf("cannot SealBlock: %w: %v", types.ErrUnknownChain, chain)
	}
	if !mayInit && logDB.IsEmpty() {
		return fmt.Errorf("cannot SealBlock on uninitialized database: %w", types.ErrUninitialized)
	}
	err := logDB.SealBlock(block.ParentHash, block.ID(), block.Time)
	if err != nil {
		return fmt.Errorf("failed to seal block %v: %w", block, err)
	}
	db.logger.Info("Updated local unsafe", "chain", chain, "block", block)
	db.emitter.Emit(db.rootCtx, superevents.LocalUnsafeUpdateEvent{
		ChainID:        chain,
		NewLocalUnsafe: block,
	})
	db.m.RecordLocalUnsafe(chain, types.BlockSealFromRef(block))
	return nil
}

func (db *ChainsDB) RewindByNumber(chain eth.ChainID, number uint64) error {
	logDB, ok := db.logDBs.Get(chain)
	if !ok {
		return fmt.Errorf("cannot RewindByNumber: %w: %s", types.ErrUnknownChain, chain)
	}
	block, err := logDB.FindSealedBlock(number)
	if err != nil {
		return fmt.Errorf("failed to find sealed block %d: %w", number, err)
	}
	return logDB.Rewind(db.readRegistry, block.ID())
}

func (db *ChainsDB) Rewind(chain eth.ChainID, headBlock eth.BlockID) error {
	// Rewind the logDB
	logDB, ok := db.logDBs.Get(chain)
	if !ok {
		return fmt.Errorf("cannot Rewind: %w: %s", types.ErrUnknownChain, chain)
	}
	if err := logDB.Rewind(db.readRegistry, headBlock); err != nil {
		return fmt.Errorf("failed to rewind to block %v: %w", headBlock, err)
	}

	localDB, ok := db.localDBs.Get(chain)
	if !ok {
		return fmt.Errorf("cannot Rewind (localDB not found): %w: %s", types.ErrUnknownChain, chain)
	}
	crossDB, ok := db.crossDBs.Get(chain)
	if !ok {
		return fmt.Errorf("cannot Rewind (crossDB not found): %w: %s", types.ErrUnknownChain, chain)
	}

	revision, err := crossDB.DerivedToRevision(headBlock)
	if err != nil {
		return fmt.Errorf("cannot determine revision of %s on %s: %w", headBlock, chain, err)
	}

	if err := localDB.RewindToFirstDerived(db.readRegistry, headBlock, revision); err != nil {
		return fmt.Errorf("failed to rewind localDB to block %s on %s: %w", headBlock, chain, err)
	}
	if err := crossDB.RewindToFirstDerived(db.readRegistry, headBlock, revision); err != nil {
		return fmt.Errorf("failed to rewind crossDB to block %s on %s: %w", headBlock, chain, err)
	}

	return nil
}

// UpdateLocalSafe updates the local-safe database with the given source and lastDerived blocks.
// It wraps an inner function, blocking the call if the database is not initialized.
func (db *ChainsDB) UpdateLocalSafe(chain eth.ChainID, source eth.BlockRef, lastDerived eth.BlockRef, nodeId string) {
	logger := db.logger.New("chain", chain, "source", source, "lastDerived", lastDerived, "nodeId", nodeId)
	if !db.isLocalInitialized(chain) {
		logger.Error("cannot UpdateLocalSafe on uninitialized local-safe database")
		return
	}
	db.initializedUpdateLocalSafe(chain, source, lastDerived, nodeId)
}

func (db *ChainsDB) initializedUpdateLocalSafe(chain eth.ChainID, source eth.BlockRef, lastDerived eth.BlockRef, nodeId string) {
	logger := db.logger.New("chain", chain, "source", source, "lastDerived", lastDerived, "nodeId", nodeId)
	localDB, ok := db.localDBs.Get(chain)
	if !ok {
		logger.Error("Cannot update local-safe DB, unknown chain")
		return
	}
	logger.Debug("Updating local safe DB")
	if err := localDB.AddDerived(source, lastDerived, types.RevisionAny); err != nil {
		if errors.Is(err, types.ErrIneffective) {
			logger.Info("Node is syncing known source blocks on known latest local-safe block", "err", err)
			return
		} else if errors.Is(err, types.ErrDataCorruption) {
			logger.Error("DB coruption occurred", "err", err)
			return
		}
		logger.Warn("Failed to update local safe", "err", err)
		db.emitter.Emit(db.rootCtx, superevents.UpdateLocalSafeFailedEvent{
			ChainID: chain,
			Err:     err,
			NodeID:  nodeId,
		})
		return
	}
	logger.Info("Updated local safe DB")
	db.emitter.Emit(db.rootCtx, superevents.LocalSafeUpdateEvent{
		ChainID:      chain,
		NewLocalSafe: types.DerivedBlockSealPairFromRefs(source, lastDerived),
	})
	db.m.RecordLocalSafe(chain, types.BlockSealFromRef(lastDerived))
}

func (db *ChainsDB) updateSafePreInterop(chain eth.ChainID, source eth.BlockRef, lastDerived eth.BlockRef, nodeId string) error {
	logger := db.logger.New("chain", chain, "source", source, "lastDerived", lastDerived, "nodeId", nodeId, "preInterop", true)
	crossDB, ok := db.crossDBs.Get(chain)
	if !ok {
		return types.ErrUnknownChain
	}
	logger.Debug("Updating cross safe DB")
	if err := crossDB.AddDerived(source, lastDerived, types.RevisionAny); err != nil {
		if errors.Is(err, types.ErrIneffective) {
			logger.Info("Node is syncing known source blocks on known latest cross-safe block", "err", err)
			return nil
		}
		if errors.Is(err, types.ErrDataCorruption) {
			return err
		}
		logger.Warn("Failed to update cross safe", "err", err)
		db.emitter.Emit(db.rootCtx, superevents.UpdateLocalSafeFailedEvent{
			ChainID: chain,
			Err:     err,
			NodeID:  nodeId,
		})
		return err
	}

	logger.Info("Updated cross safe DB pre-Interop")
	seal := types.DerivedBlockSealPairFromRefs(source, lastDerived)
	db.emitter.Emit(db.rootCtx, superevents.LocalSafeUpdateEvent{
		ChainID:      chain,
		NewLocalSafe: seal,
	})
	db.emitter.Emit(db.rootCtx, superevents.CrossSafeUpdateEvent{
		ChainID:      chain,
		NewCrossSafe: seal,
	})
	derivedSeal := types.BlockSealFromRef(lastDerived)
	db.m.RecordLocalSafe(chain, derivedSeal)
	db.m.RecordCrossSafe(chain, derivedSeal)

	return db.consolidateCrossUnsafe(chain, lastDerived)
}

func (db *ChainsDB) UpdateCrossUnsafe(chain eth.ChainID, crossUnsafe types.BlockSeal) error {
	v, ok := db.crossUnsafe.Get(chain)
	if !ok {
		return fmt.Errorf("cannot UpdateCrossUnsafe: %w: %s", types.ErrUnknownChain, chain)
	}
	// Cross unsafe is stateless, fine to always update to latest value.
	// Also allows to already track it during Interop activation phase when the safe chain hasn't
	// crossed Interop yet, so the ChainsDB isn't fully initialized yet.
	v.Set(crossUnsafe)
	db.logger.Info("Updated cross-unsafe", "chain", chain, "crossUnsafe", crossUnsafe)
	db.emitter.Emit(db.rootCtx, superevents.CrossUnsafeUpdateEvent{
		ChainID:        chain,
		NewCrossUnsafe: crossUnsafe,
	})
	db.m.RecordCrossUnsafe(chain, crossUnsafe)
	return nil
}

func (db *ChainsDB) UpdateCrossSafe(chain eth.ChainID, l1View eth.BlockRef, lastCrossDerived eth.BlockRef) error {
	if !db.isCrossInitialized(chain) {
		return fmt.Errorf("cannot UpdateCrossSafe on uninitialized database: %w", types.ErrUninitialized)
	}
	return db.initializedUpdateCrossSafe(chain, l1View, lastCrossDerived)
}

func (db *ChainsDB) initializedUpdateCrossSafe(chain eth.ChainID, l1View eth.BlockRef, lastCrossDerived eth.BlockRef) error {
	crossDB, ok := db.crossDBs.Get(chain)
	if !ok {
		return fmt.Errorf("no cross-safe DB: %w: %s", types.ErrUnknownChain, chain)
	}
	localDB, ok := db.localDBs.Get(chain)
	if !ok {
		return fmt.Errorf("no local-safe DB: %w: %s", types.ErrUnknownChain, chain)
	}
	// local DB here already has the new block, incl. replacement data, to sync with the cross-db
	revision, err := localDB.SourceToRevision(l1View.ID())
	if err != nil {
		return fmt.Errorf("failed to lookup revision: %w", err)
	}
	if err := crossDB.AddDerived(l1View, lastCrossDerived, revision); err != nil {
		return err
	}
	db.logger.Info("Updated cross-safe", "chain", chain, "l1View", l1View, "lastCrossDerived", lastCrossDerived)
	db.emitter.Emit(db.rootCtx, superevents.CrossSafeUpdateEvent{
		ChainID:      chain,
		NewCrossSafe: types.DerivedBlockSealPairFromRefs(l1View, lastCrossDerived),
	})
	db.m.RecordCrossSafe(chain, types.BlockSealFromRef(lastCrossDerived))

	return db.consolidateCrossUnsafe(chain, lastCrossDerived)
}

func (db *ChainsDB) consolidateCrossUnsafe(chain eth.ChainID, lastCrossDerived eth.BlockRef) error {
	// compare new cross-safe to recorded cross-unsafe
	crossUnsafe, err := db.CrossUnsafe(chain)
	if err != nil {
		db.logger.Warn("Error querying CrossUnsafe", "err", err)
		return nil // log warning for errorneous cross-unsafe query, but don't return it
	}
	if crossUnsafe.Number > lastCrossDerived.Number { // nothing to do
		return nil
	}

	// if cross-unsafe block number is same or smaller than new cross-safe, make sure to update cross-unsafe to new cross-safe
	if crossUnsafe.Hash.Cmp(lastCrossDerived.Hash) != 0 {
		db.logger.Warn("Updating cross-unsafe due to cross-safe update", "chain", chain, "crossSafe", lastCrossDerived, "crossUnsafe", crossUnsafe)
		err := db.UpdateCrossUnsafe(chain, types.BlockSealFromRef(lastCrossDerived))
		if err != nil {
			return fmt.Errorf("updating cross-unsafe: %w", err)
		}
	}
	return nil
}

func (db *ChainsDB) onLocalUnsafe(id eth.ChainID, unsafe eth.BlockRef) bool {
	logger := db.logger.New("chain", id, "unsafe", unsafe)
	if !db.cfg.IsInterop(id, unsafe.Time) {
		logger.Warn("ignoring local unsafe received event for pre-interop block")
		return false
	} else if db.cfg.IsInteropActivationBlock(id, unsafe.Time) {
		// Interop-activation unsafe blocks trigger the initialization of the logs DB.
		logger.Info("Initializing logs DB from unsafe activation block")
		if err := db.maybeInitFromUnsafe(id, unsafe); err != nil {
			logger.Error("Error initializing logs DB from unsafe activation block")
			return false
		}
	} else if !db.isLogsInitialized(id) {
		logger.Error("First seen unsafe block is post-Interop activation")
		// TODO: need to handle the case that this unsafe block is post-activation.
		//  Reset currently only really reset safe, so a PreInteropReset doesn't work.
		return false
	} else {
		// Non-activation Interop unsafe blocks trigger the chain processor.
		db.emitter.Emit(db.rootCtx, superevents.ChainProcessEvent{
			ChainID: id,
			Target:  unsafe.Number,
		})
	}
	return true
}

func (db *ChainsDB) onLocalDerived(id eth.ChainID, derived types.DerivedBlockRefPair, nodeID string) bool {
	logger := db.logger.New("chain", id, "derived", derived.Derived, "source", derived.Source, "nodeID", nodeID)

	// We initialize the cross-safe DB from any derived block at or before the Interop activation block.
	// If the derived DBs are uninitialized and the derived block is past the activation block, the
	// node needs to be reset to provide the activation block.
	if !db.isCrossInitialized(id) {
		if db.cfg.IsInteropPostActivation(id, derived.Derived.Time) {
			logger.Warn("Received post-activation derived block on uninitialized cross-safe DB, requesting reset")
			db.emitter.Emit(db.rootCtx, superevents.ResetPreInteropRequestEvent{
				ChainID: id,
			})
			return true
		}
		if err := db.initCrossSafe(id, derived); err != nil {
			logger.Error("Error initializing cross-safe DB from derived block", "err", err)
			return false
		}
	} else if !db.cfg.IsInteropPostActivation(id, derived.Derived.Time) {
		// Pre-interop derived blocks directly update the cross-safe DB and skip the local-safe DB
		if err := db.updateSafePreInterop(id, derived.Source, derived.Derived, nodeID); err != nil {
			logger.Error("Error updating cross-safe DB from derived block", "err", err)
			return false
		}
	}

	// This captures the case where we initialized the cross-safe DB pre-Interop
	if !db.cfg.IsInterop(id, derived.Derived.Time) {
		return true
	}

	// Post-interop derived blocks update the local-safe DB, the activation block initializes it.
	if !db.isLocalInitialized(id) {
		if db.cfg.IsInteropPostActivation(id, derived.Derived.Time) {
			logger.Error("Received post-activation derived block on uninitialized local-safe, requesting reset")
			db.emitter.Emit(db.rootCtx, superevents.ResetPreInteropRequestEvent{
				ChainID: id,
			})
			return true
		}

		if err := db.initLocalSafe(id, derived); err != nil {
			logger.Error("Error initializing safe DBs from derived block", "err", err)
			return false
		}
		// Don't call UpdateLocalSafe if we just initialized.
		return true
	}

	db.UpdateLocalSafe(id, derived.Source, derived.Derived, nodeID)
	return true
}

func (db *ChainsDB) onFinalizedL1(finalized eth.BlockRef) {
	// Lock, so we avoid race-conditions in-between getting (for comparison) and setting.
	// Unlock is managed explicitly, in this function so we can call NotifyL2Finalized after releasing the lock.
	db.finalizedL1.Lock()

	if v := db.finalizedL1.Value; v != (eth.BlockRef{}) && v.Number > finalized.Number {
		db.finalizedL1.Unlock()
		db.logger.Warn("Cannot rewind finalized L1 block", "current", v, "signal", finalized)
		return
	}
	db.finalizedL1.Value = finalized
	db.logger.Debug("Updated finalized L1", "finalizedL1", finalized)
	db.finalizedL1.Unlock()

	db.emitter.Emit(db.rootCtx, superevents.FinalizedL1UpdateEvent{
		FinalizedL1: finalized,
	})
	// whenever the L1 Finalized changes, the L2 Finalized may change, notify subscribers
	for _, chain := range db.cfg.Chains() {
		if !db.isCrossInitialized(chain) {
			continue
		}

		fin, err := db.Finalized(chain)
		if err != nil {
			db.logger.Warn("Unable to determine finalized L2 block", "chain", chain, "l1Finalized", finalized)
			continue
		}
		db.emitter.Emit(db.rootCtx, superevents.FinalizedL2UpdateEvent{
			ChainID: chain, FinalizedL2: fin,
		})
	}
}

func (db *ChainsDB) InvalidateLocalSafe(chainID eth.ChainID, candidate types.DerivedBlockRefPair) error {
	// Get databases to invalidate data in.
	eventsDB, ok := db.logDBs.Get(chainID)
	if !ok {
		return fmt.Errorf("cannot find events DB of chain %s for invalidation: %w", chainID, types.ErrUnknownChain)
	}
	localSafeDB, ok := db.localDBs.Get(chainID)
	if !ok {
		return fmt.Errorf("cannot find local-safe DB of chain %s for invalidation: %w", chainID, types.ErrUnknownChain)
	}

	// Now invalidate the local-safe data.
	// We insert a marker, so we don't build on top of the invalidated block, until it is replaced.
	// And we won't index unsafe blocks, until it is replaced.
	if err := localSafeDB.RewindAndInvalidate(db.readRegistry, candidate); err != nil {
		return fmt.Errorf("failed to invalidate entry in local-safe DB: %w", err)
	}

	// Change cross-unsafe, if it's equal or past the invalidated block.
	if err := db.ResetCrossUnsafeIfNewerThan(chainID, candidate.Derived.Number); err != nil {
		return fmt.Errorf("failed to reset cross-unsafe: %w", err)
	}

	// Drop the events of the invalidated block and after,
	// by rewinding to only keep the parent-block.
	if err := eventsDB.Rewind(db.readRegistry, candidate.Derived.ParentID()); err != nil {
		return fmt.Errorf("failed to rewind unsafe-chain: %w", err)
	}

	// Create an event, that subscribed sync-nodes can listen to,
	// to start finding the replacement block.
	db.emitter.Emit(db.rootCtx, superevents.InvalidateLocalSafeEvent{
		ChainID:   chainID,
		Candidate: candidate,
	})
	return nil
}

func (db *ChainsDB) RewindToSource(chainID eth.ChainID, source eth.BlockID) error {
	logger := db.logger.New("chain", chainID, "source", source)
	crossSafeDB, ok := db.crossDBs.Get(chainID)
	if !ok {
		return fmt.Errorf("cannot find cross-safe DB of chain %s for invalidation: %w", chainID, types.ErrUnknownChain)
	}
	logger.Warn("Rewinding cross-safe DB")
	target, err := crossSafeDB.RewindToSource(db.readRegistry, source)
	if errors.Is(err, types.ErrFuture) {
		logger.Warn("Cross-safe DB behind rewind target", "err", err)
	} else if err != nil {
		return fmt.Errorf("failed to rewind cross-safe: %w", err)
	}
	// Pre-interop, only the cross-safe DB is maintained.
	if !db.cfg.IsInterop(chainID, target.Derived.Timestamp) {
		// TODO: clear local and logs DBs, rewind may have crossed activation boundary
		return nil
	}

	localSafeDB, ok := db.localDBs.Get(chainID)
	if !ok {
		return fmt.Errorf("cannot find local-safe DB of chain %s for invalidation: %w", chainID, types.ErrUnknownChain)
	}
	logger.Warn("Rewinding local-safe DB")
	target, err = localSafeDB.RewindToSource(db.readRegistry, source)
	if errors.Is(err, types.ErrFuture) {
		logger.Warn("Local-safe DB behind rewind target", "err", err)
		// Won't rewind logs DB, no target L2 block
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to rewind local-safe: %w", err)
	}

	return db.RewindLogs(chainID, target.Derived)
}

// TODO: remove
// RewindLocalSafeSource removes all local-safe blocks after the given new derived-from source.
// If the source is before the start of the DB, the DB will be emptied.
// Note that this drop L1 blocks that resulted in a previously invalidated local-safe block.
// This returns ErrFuture if the block is newer than the last known block.
// This returns ErrConflict if a different block at the given height is known.
func (db *ChainsDB) RewindLocalSafeSource(chainID eth.ChainID, source eth.BlockID) error {
	localSafeDB, ok := db.localDBs.Get(chainID)
	if !ok {
		return fmt.Errorf("cannot find local-safe DB of chain %s for invalidation: %w", chainID, types.ErrUnknownChain)
	}
	if _, err := localSafeDB.RewindToSource(db.readRegistry, source); err != nil {
		return fmt.Errorf("failed to rewind local-safe: %w", err)
	}
	return nil
}

// TODO: remove
// RewindCrossSafeSource removes all cross-safe blocks after the given new derived-from source.
// If the source is before the start of the DB, the DB will be emptied.
// This returns ErrFuture if the block is newer than the last known block.
// This returns ErrConflict if a different block at the given height is known.
func (db *ChainsDB) RewindCrossSafeSource(chainID eth.ChainID, source eth.BlockID) error {
	crossSafeDB, ok := db.crossDBs.Get(chainID)
	if !ok {
		return fmt.Errorf("cannot find cross-safe DB of chain %s for invalidation: %w", chainID, types.ErrUnknownChain)
	}
	if _, err := crossSafeDB.RewindToSource(db.readRegistry, source); err != nil {
		return fmt.Errorf("failed to rewind cross-safe: %w", err)
	}
	return nil
}

func (db *ChainsDB) RewindLogs(chainID eth.ChainID, newHead types.BlockSeal) error {
	eventsDB, ok := db.logDBs.Get(chainID)
	if !ok {
		return fmt.Errorf("cannot find events DB of chain %s for invalidation: %w", chainID, types.ErrUnknownChain)
	}
	db.logger.Warn("Rewinding logs DB", "chain", chainID, "newHead", newHead)
	if err := eventsDB.Rewind(db.readRegistry, newHead.ID()); err != nil {
		return fmt.Errorf("failed to rewind logs of chain %s: %w", chainID, err)
	}

	return nil
}

func (db *ChainsDB) ResetCrossUnsafeIfNewerThan(chainID eth.ChainID, number uint64) error {
	crossUnsafe, ok := db.crossUnsafe.Get(chainID)
	if !ok {
		return nil
	}

	crossSafeDB, ok := db.crossDBs.Get(chainID)
	if !ok {
		return fmt.Errorf("cannot find cross-safe DB of chain %s for invalidation: %w", chainID, types.ErrUnknownChain)
	}
	crossSafe, err := crossSafeDB.Last()
	if err != nil {
		return fmt.Errorf("cannot get cross-safe of chain %s: %w", chainID, err)
	}

	// Reset cross-unsafe if it's equal or newer than the given block number
	crossUnsafe.Lock()
	x := crossUnsafe.Value
	defer crossUnsafe.Unlock()
	if x.Number >= number {
		db.logger.Warn("Resetting cross-unsafe to cross-safe, since prior block was invalidated",
			"crossUnsafe", x, "crossSafe", crossSafe, "number", number)
		crossUnsafe.Value = crossSafe.Derived
	}
	return nil
}

func (db *ChainsDB) onReplaceBlock(chainID eth.ChainID, replacement eth.BlockRef, invalidated common.Hash) {
	localSafeDB, ok := db.localDBs.Get(chainID)
	if !ok {
		db.logger.Error("Cannot find DB for replacement block", "chain", chainID)
		return
	}

	db.logger.Warn("Replacing local block", "replacement", replacement)
	result, err := localSafeDB.ReplaceInvalidatedBlock(db.readRegistry, replacement, invalidated)
	if err != nil {
		db.logger.Error("Cannot replace invalidated block in local-safe DB",
			"invalidated", invalidated, "replacement", replacement, "err", err)
		return
	}

	revision := types.Revision(result.Derived.Number)
	db.logger.Info("Replaced block", "chain", chainID, "replacement", replacement, "revision", revision)

	// Consider the replacement as a new local-unsafe block, so we can try to index the new event-data.
	db.emitter.Emit(db.rootCtx, superevents.LocalUnsafeReceivedEvent{
		ChainID:        chainID,
		NewLocalUnsafe: replacement,
	})
	// The local-safe DB changed, so emit an event, so other sub-systems can react to the change.
	seals := result.Seals()
	db.emitter.Emit(db.rootCtx, superevents.LocalSafeUpdateEvent{
		ChainID:      chainID,
		NewLocalSafe: seals,
	})
	db.m.RecordLocalSafe(chainID, seals.Derived)
	// The event-DB will start indexing, and then unblock cross-safe update
	// of the new replaced block, via regular cross-safe update worker routine.
}
