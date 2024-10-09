package db

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var ErrUnknownChain = errors.New("unknown chain")

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

type LocalDerivedFromStorage interface {
	AddDerived(derivedFrom eth.BlockRef, derived eth.BlockRef) error
}

type CrossDerivedFromStorage interface {
	LocalDerivedFromStorage
	// This will start to differ with reorg support
}

var _ LogStorage = (*logs.DB)(nil)

// ChainsDB is a database that stores logs and derived-from data for multiple chains.
// it implements the ChainsStorage interface.
type ChainsDB struct {
	// RW mutex:
	// Read = chains can be read / mutated.
	// Write = set of chains is changing.
	mu sync.RWMutex

	// unsafe info: the sequence of block seals and events
	logDBs map[types.ChainID]LogStorage

	// cross-unsafe: how far we have processed the unsafe data.
	// TODO: not initialized yet. Should just set it to the last known cross-safe block.
	crossUnsafe map[types.ChainID]types.HeadPointer

	// local-safe: index of what we optimistically know about L2 blocks being derived from L1
	localDBs map[types.ChainID]LocalDerivedFromStorage

	// cross-safe: index of L2 blocks we know to only have cross-L2 valid dependencies
	crossDBs map[types.ChainID]CrossDerivedFromStorage

	// finalized: the L1 finality progress. This can be translated into what may be considered as finalized in L2.
	// TODO: not initialized yet. Should just wait for a new signal of it.
	finalizedL1 eth.L1BlockRef

	logger log.Logger
}

func NewChainsDB(logDBs map[types.ChainID]LogStorage, l log.Logger) *ChainsDB {
	return &ChainsDB{
		logDBs: logDBs,
		logger: l,
	}
}

func (db *ChainsDB) AddLogDB(chain types.ChainID, logDB LogStorage) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.logDBs[chain] != nil {
		log.Warn("overwriting existing logDB for chain", "chain", chain)
	}
	db.logDBs[chain] = logDB
}

// ResumeFromLastSealedBlock prepares the chains db to resume recording events after a restart.
// It rewinds the database to the last block that is guaranteed to have been fully recorded to the database,
// to ensure it can resume recording from the first log of the next block.
func (db *ChainsDB) ResumeFromLastSealedBlock() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

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

func (db *ChainsDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	var combined error
	for id, logDB := range db.logDBs {
		if err := logDB.Close(); err != nil {
			combined = errors.Join(combined, fmt.Errorf("failed to close log db for chain %v: %w", id, err))
		}
	}
	return combined
}
