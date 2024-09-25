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
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var (
	ErrUnknownChain = errors.New("unknown chain")
)

type LogStorage interface {
	io.Closer

	AddLog(logHash common.Hash, parentBlock eth.BlockID,
		logIdx uint32, execMsg *types.ExecutingMessage) error

	SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error

	Rewind(newHeadBlockNum uint64) error

	LatestSealedBlockNum() (n uint64, ok bool)

	// FindSealedBlock finds the requested block, to check if it exists,
	// returning the next index after it where things continue from.
	// returns ErrFuture if the block is too new to be able to tell
	// returns ErrDifferent if the known block does not match
	FindSealedBlock(block eth.BlockID) (nextEntry entrydb.EntryIdx, err error)

	IteratorStartingAt(i entrydb.EntryIdx) (logs.Iterator, error)

	// returns ErrConflict if the log does not match the canonical chain.
	// returns ErrFuture if the log is out of reach.
	// returns nil if the log is known and matches the canonical chain.
	Contains(blockNum uint64, logIdx uint32, logHash common.Hash) (nextIndex entrydb.EntryIdx, err error)
}

var _ LogStorage = (*logs.DB)(nil)

type HeadsStorage interface {
	Current() *heads.Heads
	Apply(op heads.Operation) error
}

// ChainsDB is a database that stores logs and heads for multiple chains.
// it implements the ChainsStorage interface.
type ChainsDB struct {
	logDBs           map[types.ChainID]LogStorage
	heads            HeadsStorage
	maintenanceReady chan struct{}
	logger           log.Logger
}

func NewChainsDB(logDBs map[types.ChainID]LogStorage, heads HeadsStorage, l log.Logger) *ChainsDB {
	return &ChainsDB{
		logDBs:           logDBs,
		heads:            heads,
		logger:           l,
		maintenanceReady: make(chan struct{}, 1),
	}
}

func (db *ChainsDB) AddLogDB(chain types.ChainID, logDB LogStorage) {
	if db.logDBs[chain] != nil {
		log.Warn("overwriting existing logDB for chain", "chain", chain)
	}
	db.logDBs[chain] = logDB
}

// ResumeFromLastSealedBlock prepares the chains db to resume recording events after a restart.
// It rewinds the database to the last block that is guaranteed to have been fully recorded to the database,
// to ensure it can resume recording from the first log of the next block.
func (db *ChainsDB) ResumeFromLastSealedBlock() error {
	for chain, logStore := range db.logDBs {
		headNum, ok := logStore.LatestSealedBlockNum()
		if ok {
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
func (db *ChainsDB) Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash common.Hash) (entrydb.EntryIdx, error) {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return 0, fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	return logDB.Contains(blockNum, logIdx, logHash)
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
			return fmt.Errorf("failed to update cross-heads for safety level %v: %w", checker.Name(), err)
		}
	}
	return nil
}

// UpdateCrossHeadsForChain updates the cross-head for a single chain.
// the provided checker controls which heads are considered.
func (db *ChainsDB) UpdateCrossHeadsForChain(chainID types.ChainID, checker SafetyChecker) error {
	// start with the xsafe head of the chain
	xHead := checker.CrossHeadForChain(chainID)
	// advance as far as the local head
	localHead := checker.LocalHeadForChain(chainID)
	// get an iterator for the last checkpoint behind the x-head
	iter, err := db.logDBs[chainID].IteratorStartingAt(xHead)
	if err != nil {
		return fmt.Errorf("failed to rewind cross-safe head for chain %v: %w", chainID, err)
	}
	// track if we updated the cross-head
	updated := false
	// advance the logDB through all executing messages we can
	// this loop will break:
	// - when we reach the local head
	// - when we reach a message that is not safe
	// - if an error occurs
	for {
		if err := iter.NextExecMsg(); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("failed to read next executing message for chain %v: %w", chainID, err)
		}
		// if we would exceed the local head, then abort
		if iter.NextIndex() > localHead {
			xHead = localHead // clip to local head
			updated = localHead != xHead
			break
		}
		exec := iter.ExecMessage()
		if exec == nil {
			panic("expected executing message after traversing to one without error")
		}
		// use the checker to determine if this message is safe
		safe := checker.Check(
			types.ChainIDFromUInt64(uint64(exec.Chain)),
			exec.BlockNum,
			exec.LogIdx,
			exec.Hash)
		if !safe {
			break
		}
		// if all is well, prepare the x-head update to this point
		xHead = iter.NextIndex()
		updated = true
	}
	// have the checker create an update to the x-head in question, and apply that update
	err = db.heads.Apply(checker.Update(chainID, xHead))
	if err != nil {
		return fmt.Errorf("failed to update cross-head for chain %v: %w", chainID, err)
	}
	// if any chain was updated, we can trigger a maintenance request
	// this allows for the maintenance loop to handle cascading updates
	// instead of waiting for the next scheduled update
	if updated {
		db.logger.Info("Promoting cross-head", "head", xHead, "safety-level", checker.SafetyLevel())
		db.RequestMaintenance()
	} else {
		db.logger.Info("No cross-head update", "head", xHead, "safety-level", checker.SafetyLevel())
	}
	return nil
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

func (db *ChainsDB) SealBlock(chain types.ChainID, parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	return logDB.SealBlock(parentHash, block, timestamp)
}

func (db *ChainsDB) AddLog(chain types.ChainID, logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *types.ExecutingMessage) error {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	return logDB.AddLog(logHash, parentBlock, logIdx, execMsg)
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
