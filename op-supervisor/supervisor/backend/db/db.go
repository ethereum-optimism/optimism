package db

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/safety"
	backendTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var (
	ErrUnknownChain = errors.New("unknown chain")
)

type LogStorage interface {
	io.Closer

	AddLog(logHash backendTypes.TruncatedHash, parentBlock eth.BlockID,
		logIdx uint32, execMsg *backendTypes.ExecutingMessage) error

	SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error

	Rewind(newHeadBlockNum uint64) error

	LatestSealedBlockNum() (n uint64, ok bool)

	// FindSealedBlock finds the requested block, to check if it exists,
	// returning the next index after it where things continue from.
	// returns ErrFuture if the block is too new to be able to tell
	// returns ErrDifferent if the known block does not match
	FindSealedBlock(block eth.BlockID) (nextEntry entrydb.EntryIdx, err error)

	IteratorStartingAt(sealedNum uint64, logIndex uint32) (logs.Iterator, error)

	// returns ErrConflict if the log does not match the canonical chain.
	// returns ErrFuture if the log is out of reach.
	// returns nil if the log is known and matches the canonical chain.
	Contains(blockNum uint64, logIdx uint32, logHash backendTypes.TruncatedHash) (nextIndex entrydb.EntryIdx, err error)
}

var _ LogStorage = (*logs.DB)(nil)

type HeadsStorage interface {
	CrossUnsafe(id types.ChainID) heads.HeadPointer
	CrossSafe(id types.ChainID) heads.HeadPointer
	CrossFinalized(id types.ChainID) heads.HeadPointer
	LocalUnsafe(id types.ChainID) heads.HeadPointer
	LocalSafe(id types.ChainID) heads.HeadPointer
	LocalFinalized(id types.ChainID) heads.HeadPointer

	UpdateCrossUnsafe(id types.ChainID, pointer heads.HeadPointer) error
	UpdateCrossSafe(id types.ChainID, pointer heads.HeadPointer) error
	UpdateCrossFinalized(id types.ChainID, pointer heads.HeadPointer) error

	UpdateLocalUnsafe(id types.ChainID, pointer heads.HeadPointer) error
	UpdateLocalSafe(id types.ChainID, pointer heads.HeadPointer) error
	UpdateLocalFinalized(id types.ChainID, pointer heads.HeadPointer) error
}

// ChainsDB is a database that stores logs and heads for multiple chains.
// it implements the ChainsStorage interface.
type ChainsDB struct {
	logDBs           map[types.ChainID]LogStorage
	heads            HeadsStorage
	safetyIndex      safety.SafetyIndex
	maintenanceReady chan struct{}
	logger           log.Logger
}

func NewChainsDB(logDBs map[types.ChainID]LogStorage, heads HeadsStorage, l log.Logger) *ChainsDB {
	ret := &ChainsDB{
		logDBs:           logDBs,
		heads:            heads,
		logger:           l,
		maintenanceReady: make(chan struct{}, 1),
	}
	ret.safetyIndex = safety.NewRecentSafetyIndex(l, ret)
	return ret
}

func (db *ChainsDB) AddLogDB(chain types.ChainID, logDB LogStorage) {
	if db.logDBs[chain] != nil {
		log.Warn("overwriting existing logDB for chain", "chain", chain)
	}
	db.logDBs[chain] = logDB
}

func (db *ChainsDB) IteratorStartingAt(chain types.ChainID, sealedNum uint64, logIndex uint32) (logs.Iterator, error) {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return nil, fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	return logDB.IteratorStartingAt(sealedNum, logIndex)
}

// ResumeFromLastSealedBlock prepares the chains db to resume recording events after a restart.
// It rewinds the database to the last block that is guaranteed to have been fully recorded to the database,
// to ensure it can resume recording from the first log of the next block.
func (db *ChainsDB) ResumeFromLastSealedBlock() error {
	for chain, logStore := range db.logDBs {
		headNum, ok := logStore.LatestSealedBlockNum()
		if !ok {
			// db must be empty, nothing to rewind to
			db.logger.Info("Resuming, but found no DB contents", "chain", chain)
			continue
		}
		db.logger.Info("Resuming, starting from last sealed block", "head", headNum)
		if err := logStore.Rewind(headNum); err != nil {
			return fmt.Errorf("failed to rewind chain %s to sealed block %d", chain, headNum)
		}
	}
	return nil
}

// StartCrossHeadMaintenance starts a background process that maintains the cross-heads of the chains
// for now it does not prevent multiple instances of this process from running
func (db *ChainsDB) StartCrossHeadMaintenance(ctx context.Context) {
	go func() {
		db.logger.Info("cross-head maintenance loop started")
		// run the maintenance loop every 1 seconds for now
		ticker := time.NewTicker(time.Second * 1)
		for {
			select {
			case <-ctx.Done():
				db.logger.Warn("context cancelled, stopping maintenance loop")
				return
			case <-ticker.C:
				db.logger.Debug("regular maintenance requested")
				db.RequestMaintenance()
			case <-db.maintenanceReady:
				db.logger.Debug("running maintenance")
				if err := db.updateAllHeads(); err != nil {
					db.logger.Error("failed to update cross-heads", "err", err)
				}
			}
		}
	}()
}

// Check calls the underlying logDB to determine if the given log entry is safe with respect to the checker's criteria.
func (db *ChainsDB) Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash backendTypes.TruncatedHash) (backendTypes.TruncatedHash, error) {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return backendTypes.TruncatedHash{}, fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	_, err := logDB.Contains(blockNum, logIdx, logHash)
	if err != nil {
		return backendTypes.TruncatedHash{}, err
	}
	// TODO: need to get the actual block hash for this log entry
	return backendTypes.TruncatedHash{}, nil
}

// RequestMaintenance requests that the maintenance loop update the cross-heads
// it does not block if maintenance is already scheduled
func (db *ChainsDB) RequestMaintenance() {
	select {
	case db.maintenanceReady <- struct{}{}:
		return
	default:
		return
	}
}

// updateAllHeads updates the cross-heads of all safety levels
// it is called by the maintenance loop
func (db *ChainsDB) updateAllHeads() error {
	// create three safety checkers, one for each safety level
	unsafeChecker := NewSafetyChecker(Unsafe, db)
	safeChecker := NewSafetyChecker(Safe, db)
	finalizedChecker := NewSafetyChecker(Finalized, db)
	for _, checker := range []SafetyChecker{
		unsafeChecker,
		safeChecker,
		finalizedChecker} {
		if err := db.UpdateCrossHeads(checker); err != nil {
			return fmt.Errorf("failed to update cross-heads for safety level %s: %w", checker, err)
		}
	}
	return nil
}

// UpdateCrossHeadsForChain updates the cross-head for a single chain.
// the provided checker controls which heads are considered.
func (db *ChainsDB) UpdateCrossHeadsForChain(chainID types.ChainID, checker SafetyChecker) error {
	// start with the xsafe head of the chain
	xHead := checker.CrossHead(chainID)
	// advance as far as the local head
	localHead := checker.LocalHead(chainID)
	// get an iterator for the next item
	iter, err := db.logDBs[chainID].IteratorStartingAt(xHead.LastSealedBlockNum, xHead.LogsSince)
	if err != nil {
		return fmt.Errorf("failed to open iterator at sealed block %d logsSince %d for chain %v: %w",
			xHead.LastSealedBlockNum, xHead.LogsSince, chainID, err)
	}
	// track if we updated the cross-head
	updated := false
	// advance the logDB through all executing messages we can
	// this loop will break:
	// - when we reach the local head
	// - when we reach a message that is not safe
	// - if an error occurs
	for {
		if err := iter.NextInitMsg(); errors.Is(err, logs.ErrFuture) {
			// We ran out of events, but there can still be empty blocks.
			// Take the last block we've processed, and try to update the x-head with it.
			sealedBlockHash, sealedBlockNum, ok := iter.SealedBlock()
			if !ok {
				break
			}
			// We can only drop the logsSince value to 0 if the block is not seen.
			if sealedBlockNum > xHead.LastSealedBlockNum {
				// if we would exceed the local head, then abort
				if !localHead.WithinRange(sealedBlockNum, 0) {
					break
				}
				xHead = heads.HeadPointer{
					LastSealedBlockHash: sealedBlockHash,
					LastSealedBlockNum:  sealedBlockNum,
					LogsSince:           0,
				}
				updated = true
			}
			break
		} else if err != nil {
			return fmt.Errorf("failed to read next executing message for chain %v: %w", chainID, err)
		}

		sealedBlockHash, sealedBlockNum, ok := iter.SealedBlock()
		if !ok {
			break
		}
		_, logIdx, ok := iter.InitMessage()
		if !ok {
			break
		}
		// if we would exceed the local head, then abort
		if !localHead.WithinRange(sealedBlockNum, logIdx) {
			break
		}

		// Check the executing message, if any
		exec := iter.ExecMessage()
		if exec != nil {
			// Use the checker to determine if this message exists in the canonical chain,
			// within the view of the checker's safety level
			if err := checker.CheckCross(
				types.ChainIDFromUInt64(uint64(exec.Chain)),
				exec.BlockNum,
				exec.LogIdx,
				exec.Hash); err != nil {
				if errors.Is(err, logs.ErrConflict) {
					db.logger.Error("Bad executing message!", "err", err)
				} else if errors.Is(err, logs.ErrFuture) {
					db.logger.Warn("Executing message references future message", "err", err)
				} else {
					db.logger.Error("Failed to check executing message")
				}
				break
			}
		}
		// if all is well, prepare the x-head update to this point
		xHead = heads.HeadPointer{
			LastSealedBlockHash: sealedBlockHash,
			LastSealedBlockNum:  sealedBlockNum,
			LogsSince:           logIdx + 1,
		}
		updated = true
	}
	// if any chain was updated, we can trigger a maintenance request
	// this allows for the maintenance loop to handle cascading updates
	// instead of waiting for the next scheduled update
	if updated {
		db.logger.Info("Promoting cross-head", "chain", chainID, "head", xHead, "safety-level", checker.CrossSafetyLevel())
		err = checker.UpdateCross(chainID, xHead)
		if err != nil {
			return fmt.Errorf("failed to update cross-head for chain %v: %w", chainID, err)
		}
		db.RequestMaintenance()
	} else {
		db.logger.Debug("No cross-head update", "chain", chainID, "head", xHead, "safety-level", checker.CrossSafetyLevel())
	}
	return nil
}

func (db *ChainsDB) Heads() HeadsStorage {
	return db.heads
}

// UpdateCrossHeads updates the cross-heads of all chains
// based on the provided SafetyChecker. The SafetyChecker is used to determine
// the safety of each log entry in the database, and the cross-head associated with it.
func (db *ChainsDB) UpdateCrossHeads(checker SafetyChecker) error {
	for chainID := range db.logDBs {
		err := db.UpdateCrossHeadsForChain(chainID, checker)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *ChainsDB) FindSealedBlock(chain types.ChainID, block eth.BlockID) (nextEntry entrydb.EntryIdx, err error) {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return 0, fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	return logDB.FindSealedBlock(block)
}

// LatestBlockNum returns the latest fully-sealed block number that has been recorded to the logs db
// for the given chain. It does not contain safety guarantees.
// The block number might not be available (empty database, or non-existent chain).
func (db *ChainsDB) LatestBlockNum(chain types.ChainID) (num uint64, ok bool) {
	logDB, knownChain := db.logDBs[chain]
	if !knownChain {
		return 0, false
	}
	return logDB.LatestSealedBlockNum()
}

func (db *ChainsDB) AddLog(
	chain types.ChainID,
	logHash backendTypes.TruncatedHash,
	parentBlock eth.BlockID,
	logIdx uint32,
	execMsg *backendTypes.ExecutingMessage) error {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	return logDB.AddLog(logHash, parentBlock, logIdx, execMsg)
}

func (db *ChainsDB) SealBlock(
	chain types.ChainID,
	block eth.L2BlockRef) error {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	err := logDB.SealBlock(block.ParentHash, block.ID(), block.Time)
	if err != nil {
		return fmt.Errorf("failed to seal block %v: %w", block, err)
	}
	err = db.safetyIndex.UpdateLocalUnsafe(chain, block)
	if err != nil {
		return fmt.Errorf("failed to update local-unsafe: %w", err)
	}
	return nil
}

func (db *ChainsDB) Rewind(chain types.ChainID, headBlockNum uint64) error {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	return logDB.Rewind(headBlockNum)
}

func (db *ChainsDB) Close() error {
	var combined error
	for id, logDB := range db.logDBs {
		if err := logDB.Close(); err != nil {
			combined = errors.Join(combined, fmt.Errorf("failed to close log db for chain %v: %w", id, err))
		}
	}
	return combined
}
