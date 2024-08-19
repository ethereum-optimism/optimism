package db

import (
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	backendTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
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
	logDBs map[types.ChainID]LogStorage
	heads  HeadsStorage
}

func NewChainsDB(logDBs map[types.ChainID]LogStorage, heads HeadsStorage) *ChainsDB {
	return &ChainsDB{
		logDBs: logDBs,
		heads:  heads,
	}
}

// Resume prepares the chains db to resume recording events after a restart.
// It rewinds the database to the last block that is guaranteed to have been fully recorded to the database
// to ensure it can resume recording from the first log of the next block.
func (db *ChainsDB) Resume() error {
	for chain, logStore := range db.logDBs {
		if err := Resume(logStore); err != nil {
			return fmt.Errorf("failed to resume chain %v: %w", chain, err)
		}
	}
	return nil
}

// UpdateCrossSafeHeads updates the cross-heads of all chains
// this is an example of how to use the SafetyChecker to update the cross-heads
func (db *ChainsDB) UpdateCrossSafeHeads() error {
	checker := NewSafetyChecker(Safe, *db)
	return db.UpdateCrossHeads(checker)
}

// UpdateCrossHeadsForChain updates the cross-head for a single chain.
// the provided checker controls which heads are considered.
// TODO: we should invert control and have the underlying logDB call their own update
// for now, monolithic control is fine. There may be a stronger reason to refactor if the API needs it.
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
		safe := checker.Check(chainID, exec.BlockNum, exec.LogIdx, exec.Hash)
		if !safe {
			break
		}
		// if all is well, prepare the x-head update to this point
		xHead = i.Index()
	}

	// have the checker create an update to the x-head in question, and apply that update
	err = db.heads.Apply(checker.Update(chainID, xHead))
	if err != nil {
		return fmt.Errorf("failed to update cross-head for chain %v: %w", chainID, err)
	}
	return nil
}

// UpdateCrossSafeHeads updates the cross-heads of all chains
// based on the provided SafetyChecker. The SafetyChecker is used to determine
// the safety of each log entry in the database, and the cross-head associated with it.
func (db *ChainsDB) UpdateCrossHeads(checker SafetyChecker) error {
	currentHeads := db.heads.Current()
	for chainID := range currentHeads.Chains {
		if err := db.UpdateCrossHeadsForChain(chainID, checker); err != nil {
			return err
		}
	}
	return nil
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
