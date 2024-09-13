package db

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	backendTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrUnknownChain = errors.New("unknown chain")
)

type LogStorage interface {
	io.Closer
	AddLog(logHash backendTypes.TruncatedHash, block eth.BlockID, timestamp uint64, logIdx uint32, execMsg *backendTypes.ExecutingMessage) error
	Rewind(newHeadBlockNum uint64) error
	LatestBlockNum() uint64
	ClosestBlockInfo(blockNum uint64) (uint64, backendTypes.TruncatedHash, error)
	ClosestBlockIterator(blockNum uint64) (logs.Iterator, error)
	Contains(blockNum uint64, logIdx uint32, loghash backendTypes.TruncatedHash) (bool, entrydb.EntryIdx, error)
	LastCheckpointBehind(entrydb.EntryIdx) (logs.Iterator, error)
	NextExecutingMessage(logs.Iterator) (backendTypes.ExecutingMessage, error)
}

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
}

func NewChainsDB(logDBs map[types.ChainID]LogStorage, heads HeadsStorage) *ChainsDB {
	return &ChainsDB{
		logDBs: logDBs,
		heads:  heads,
	}
}

func (db *ChainsDB) AddLogDB(chain types.ChainID, logDB LogStorage) {
	if db.logDBs[chain] != nil {
		log.Warn("overwriting existing logDB for chain", "chain", chain)
	}
	db.logDBs[chain] = logDB
}

// Resume prepares the chains db to resume recording events after a restart.
// It rewinds the database to the last block that is guaranteed to have been fully recorded to the database
// to ensure it can resume recording from the first log of the next block.
// TODO(#11793): we can rename this to something more descriptive like "PrepareWithRollback"
func (db *ChainsDB) Resume() error {
	for chain, logStore := range db.logDBs {
		if err := Resume(logStore); err != nil {
			return fmt.Errorf("failed to resume chain %v: %w", chain, err)
		}
	}
	return nil
}

// StartCrossHeadMaintenance starts a background process that maintains the cross-heads of the chains
// for now it does not prevent multiple instances of this process from running
func (db *ChainsDB) StartCrossHeadMaintenance(ctx context.Context) {
	go func() {
		// run the maintenance loop every 10 seconds for now
		ticker := time.NewTicker(time.Second * 10)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				db.RequestMaintenance()
			case <-db.maintenanceReady:
				if err := db.updateAllHeads(); err != nil {
					log.Error("failed to update cross-heads", "err", err)
				}
			}
		}
	}()
}

// Check calls the underlying logDB to determine if the given log entry is safe with respect to the checker's criteria.
func (db *ChainsDB) Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash backendTypes.TruncatedHash) (bool, entrydb.EntryIdx, error) {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return false, 0, fmt.Errorf("%w: %v", ErrUnknownChain, chain)
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
	i, err := db.logDBs[chainID].LastCheckpointBehind(xHead)
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
		exec, err := db.logDBs[chainID].NextExecutingMessage(i)
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("failed to read next executing message for chain %v: %w", chainID, err)
		}
		// if we are now beyond the local head, stop
		if i.Index() > localHead {
			break
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
		xHead = i.Index()
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
		db.RequestMaintenance()
	}
	return nil
}

// UpdateCrossHeads updates the cross-heads of all chains
// based on the provided SafetyChecker. The SafetyChecker is used to determine
// the safety of each log entry in the database, and the cross-head associated with it.
func (db *ChainsDB) UpdateCrossHeads(checker SafetyChecker) error {
	currentHeads := db.heads.Current()
	for chainID := range currentHeads.Chains {
		err := db.UpdateCrossHeadsForChain(chainID, checker)
		if err != nil {
			return err
		}
	}
	return nil
}

// LastLogInBlock scans through the logs of the given chain starting from the given block number,
// and returns the index of the last log entry in that block.
func (db *ChainsDB) LastLogInBlock(chain types.ChainID, blockNum uint64) (entrydb.EntryIdx, error) {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return 0, fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	iter, err := logDB.ClosestBlockIterator(blockNum)
	if err != nil {
		return 0, fmt.Errorf("failed to get block iterator for chain %v: %w", chain, err)
	}
	ret := entrydb.EntryIdx(0)
	// scan through using the iterator until the block number exceeds the target
	for {
		bn, index, _, err := iter.NextLog()
		// if we have reached the end of the database, stop
		if err == io.EOF {
			break
		}
		// all other errors are fatal
		if err != nil {
			return 0, fmt.Errorf("failed to read next log entry for chain %v: %w", chain, err)
		}
		// if we are now beyond the target block, stop withour updating the return value
		if bn > blockNum {
			break
		}
		// only update the return value if the block number is the same
		// it is possible the iterator started before the target block, or that the target block is not in the db
		if bn == blockNum {
			ret = entrydb.EntryIdx(index)
		}
	}
	// if we never found the block, return an error
	if ret == 0 {
		return 0, fmt.Errorf("block %v not found in chain %v", blockNum, chain)
	}
	return ret, nil
}

// LatestBlockNum returns the latest block number that has been recorded to the logs db
// for the given chain. It does not contain safety guarantees.
func (db *ChainsDB) LatestBlockNum(chain types.ChainID) uint64 {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return 0
	}
	return logDB.LatestBlockNum()
}

func (db *ChainsDB) AddLog(chain types.ChainID, logHash backendTypes.TruncatedHash, block eth.BlockID, timestamp uint64, logIdx uint32, execMsg *backendTypes.ExecutingMessage) error {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	return logDB.AddLog(logHash, block, timestamp, logIdx, execMsg)
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
