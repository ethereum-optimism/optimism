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
	LastCheckpointBehind(entrydb.EntryIdx) (*logs.Iterator, error)
}

type HeadsStorage interface {
	Current() *heads.Heads
	Apply(op heads.Operation) error
}

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

// UpdateCrossSafeHeads updates the cross-heads of all chains
// based on the provided SafetyChecker. The SafetyChecker is used to determine
// the safety of each log entry in the database, and the cross-head associated with it.
// TODO: rather than make this monolithic across all chains, this should be broken up
// allowing each chain to update on its own routine
func (db *ChainsDB) UpdateCrossHeads(checker SafetyChecker) error {
	currentHeads := db.heads.Current()
	for chainID := range currentHeads.Chains {
		// start with the xsafe head of the chain
		xHead := checker.CrossHeadForChain(chainID)
		// rewind the index to the last checkpoint and get the iterator
		i, err := db.logDBs[chainID].LastCheckpointBehind(xHead)
		if err != nil {
			return fmt.Errorf("failed to rewind cross-safe head for chain %v: %w", chainID, err)
		}
		// play forward from this checkpoint, advancing the cross-safe head as far as possible
		for {
			_, _, _, err := i.NextLog()
			if err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf("failed to read next log for chain %v: %w", chainID, err)
			}
			// if we've advanced past the local safety threshold, stop
			if i.Index() > checker.LocalHeadForChain(chainID) {
				break
			}
			// all non-executing messages are safe to advance
			// executing messages are safe to advance once checked
			em, err := i.ExecMessage()
			if err != nil {
				return fmt.Errorf("failed to get executing message for chain %v: %w", chainID, err)
			} else if em != (backendTypes.ExecutingMessage{}) {
				// if there is an executing message, check it
				chainID := types.ChainIDFromUInt64(uint64(em.Chain))
				safe := checker.Check(chainID, em.BlockNum, em.LogIdx, em.Hash)
				if !safe {
					break
				}
			}
			// record the current index, as it is safe to advance to this point
			xHead = i.Index()
		}
		// have the checker create an update to the x-head in question, and apply that update
		err = db.heads.Apply(checker.Update(chainID, xHead))
		if err != nil {
			return fmt.Errorf("failed to update cross-head for chain %v: %w", chainID, err)
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
