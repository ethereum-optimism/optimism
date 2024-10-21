package db

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/fromda"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type LogStorage interface {
	io.Closer

	AddLog(logHash common.Hash, parentBlock eth.BlockID,
		logIdx uint32, execMsg *types.ExecutingMessage) error

	SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error

	Rewind(newHeadBlockNum uint64) error

	LatestSealedBlockNum() (n uint64, ok bool)

	// FindSealedBlock finds the requested block by number, to check if it exists,
	// returning the block seal if it was found.
	// returns ErrFuture if the block is too new to be able to tell.
	FindSealedBlock(number uint64) (block types.BlockSeal, err error)

	IteratorStartingAt(sealedNum uint64, logsSince uint32) (logs.Iterator, error)

	// Contains returns no error iff the specified logHash is recorded in the specified blockNum and logIdx.
	// If the log is out of reach, then ErrFuture is returned.
	// If the log is determined to conflict with the canonical chain, then ErrConflict is returned.
	// logIdx is the index of the log in the array of all logs in the block.
	// This can be used to check the validity of cross-chain interop events.
	// The block-seal of the blockNum block, that the log was included in, is returned.
	// This seal may be fully zeroed, without error, if the block isn't fully known yet.
	Contains(blockNum uint64, logIdx uint32, logHash common.Hash) (includedIn types.BlockSeal, err error)

	// OpenBlock accumulates the ExecutingMessage events for a block and returns them
	OpenBlock(blockNum uint64) (ref eth.BlockRef, logCount uint32, execMsgs map[uint32]*types.ExecutingMessage, err error)
}

type LocalDerivedFromStorage interface {
	First() (derivedFrom types.BlockSeal, derived types.BlockSeal, err error)
	Latest() (derivedFrom types.BlockSeal, derived types.BlockSeal, err error)
	AddDerived(derivedFrom eth.BlockRef, derived eth.BlockRef) error
	LastDerivedAt(derivedFrom eth.BlockID) (derived types.BlockSeal, err error)
	DerivedFrom(derived eth.BlockID) (derivedFrom types.BlockSeal, err error)
	FirstAfter(derivedFrom, derived eth.BlockID) (nextDerivedFrom, nextDerived types.BlockSeal, err error)
	NextDerivedFrom(derivedFrom eth.BlockID) (nextDerivedFrom types.BlockSeal, err error)
	NextDerived(derived eth.BlockID) (derivedFrom types.BlockSeal, nextDerived types.BlockSeal, err error)
	PreviousDerivedFrom(derivedFrom eth.BlockID) (prevDerivedFrom types.BlockSeal, err error)
	PreviousDerived(derived eth.BlockID) (prevDerived types.BlockSeal, err error)
}

var _ LocalDerivedFromStorage = (*fromda.DB)(nil)

type CrossDerivedFromStorage interface {
	LocalDerivedFromStorage
	// This will start to differ with reorg support
}

var _ LogStorage = (*logs.DB)(nil)

// ChainsDB is a database that stores logs and derived-from data for multiple chains.
// it implements the LogStorage interface, as well as several DB interfaces needed by the cross package.
type ChainsDB struct {
	// RW mutex:
	// Read = chains can be read / mutated.
	// Write = set of chains is changing.
	mu sync.RWMutex

	// unsafe info: the sequence of block seals and events
	logDBs map[types.ChainID]LogStorage

	// cross-unsafe: how far we have processed the unsafe data.
	// If present but set to a zeroed value the cross-unsafe will fallback to cross-safe.
	crossUnsafe map[types.ChainID]types.BlockSeal

	// local-safe: index of what we optimistically know about L2 blocks being derived from L1
	localDBs map[types.ChainID]LocalDerivedFromStorage

	// cross-safe: index of L2 blocks we know to only have cross-L2 valid dependencies
	crossDBs map[types.ChainID]CrossDerivedFromStorage

	// finalized: the L1 finality progress. This can be translated into what may be considered as finalized in L2.
	// It is initially zeroed, and the L2 finality query will return
	// an error until it has this L1 finality to work with.
	finalizedL1 eth.L1BlockRef

	// depSet is the dependency set, used to determine what may be tracked,
	// what is missing, and to provide it to DB users.
	depSet depset.DependencySet

	logger log.Logger
}

func NewChainsDB(l log.Logger, depSet depset.DependencySet) *ChainsDB {
	return &ChainsDB{
		logDBs:      make(map[types.ChainID]LogStorage),
		logger:      l,
		localDBs:    make(map[types.ChainID]LocalDerivedFromStorage),
		crossDBs:    make(map[types.ChainID]CrossDerivedFromStorage),
		crossUnsafe: make(map[types.ChainID]types.BlockSeal),
		depSet:      depSet,
	}
}

func (db *ChainsDB) AddLogDB(chainID types.ChainID, logDB LogStorage) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, ok := db.logDBs[chainID]; ok {
		db.logger.Warn("overwriting existing log DB for chain", "chain", chainID)
	}

	db.logDBs[chainID] = logDB
}

func (db *ChainsDB) AddLocalDerivedFromDB(chainID types.ChainID, dfDB LocalDerivedFromStorage) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, ok := db.localDBs[chainID]; ok {
		db.logger.Warn("overwriting existing local derived-from DB for chain", "chain", chainID)
	}

	db.localDBs[chainID] = dfDB
}

func (db *ChainsDB) AddCrossDerivedFromDB(chainID types.ChainID, dfDB CrossDerivedFromStorage) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, ok := db.crossDBs[chainID]; ok {
		db.logger.Warn("overwriting existing cross derived-from DB for chain", "chain", chainID)
	}

	db.crossDBs[chainID] = dfDB
}

func (db *ChainsDB) AddCrossUnsafeTracker(chainID types.ChainID) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, ok := db.crossUnsafe[chainID]; ok {
		db.logger.Warn("overwriting existing cross-unsafe tracker for chain", "chain", chainID)
	}
	db.crossUnsafe[chainID] = types.BlockSeal{}
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

func (db *ChainsDB) DependencySet() depset.DependencySet {
	return db.depSet
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
