package db

import (
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/safety"
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

	IteratorStartingAt(sealedNum uint64, logsSince uint32) (logs.Iterator, error)

	// returns ErrConflict if the log does not match the canonical chain.
	// returns ErrFuture if the log is out of reach.
	// returns nil if the log is known and matches the canonical chain.
	Contains(blockNum uint64, logIdx uint32, logHash common.Hash) (nextIndex entrydb.EntryIdx, err error)
}

var _ LogStorage = (*logs.DB)(nil)

// ChainsDB is a database that stores logs and heads for multiple chains.
// it implements the ChainsStorage interface.
type ChainsDB struct {
	logDBs      map[types.ChainID]LogStorage
	safetyIndex safety.SafetyIndex
	logger      log.Logger
}

func NewChainsDB(logDBs map[types.ChainID]LogStorage, l log.Logger) *ChainsDB {
	ret := &ChainsDB{
		logDBs: logDBs,
		logger: l,
	}
	ret.safetyIndex = safety.NewSafetyIndex(l, ret)
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

// Check calls the underlying logDB to determine if the given log entry is safe with respect to the checker's criteria.
func (db *ChainsDB) Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash common.Hash) (common.Hash, error) {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return common.Hash{}, fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	_, err := logDB.Contains(blockNum, logIdx, logHash)
	if err != nil {
		return common.Hash{}, err
	}
	// TODO(#11693): need to get the actual block hash for this log entry for reorg detection
	return common.Hash{}, nil
}

// Safest returns the strongest safety level that can be guaranteed for the given log entry.
// it assumes the log entry has already been checked and is valid, this funcion only checks safety levels.
func (db *ChainsDB) Safest(chainID types.ChainID, blockNum uint64, index uint32) (safest types.SafetyLevel) {
	safest = types.LocalUnsafe
	if crossUnsafe, err := db.safetyIndex.CrossUnsafeL2(chainID); err == nil && crossUnsafe.WithinRange(blockNum, index) {
		safest = types.CrossUnsafe
	}
	if localSafe, err := db.safetyIndex.LocalSafeL2(chainID); err == nil && localSafe.WithinRange(blockNum, index) {
		safest = types.LocalSafe
	}
	if crossSafe, err := db.safetyIndex.LocalSafeL2(chainID); err == nil && crossSafe.WithinRange(blockNum, index) {
		safest = types.CrossSafe
	}
	if finalized, err := db.safetyIndex.FinalizedL2(chainID); err == nil {
		if finalized.Number >= blockNum {
			safest = types.Finalized
		}
	}
	return
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
	logHash common.Hash,
	parentBlock eth.BlockID,
	logIdx uint32,
	execMsg *types.ExecutingMessage) error {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	return logDB.AddLog(logHash, parentBlock, logIdx, execMsg)
}

func (db *ChainsDB) SealBlock(
	chain types.ChainID,
	block eth.BlockRef) error {
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
