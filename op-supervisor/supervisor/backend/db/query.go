package db

import (
	"fmt"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func (db *ChainsDB) FindSealedBlock(chain types.ChainID, block eth.BlockID) (nextEntry entrydb.EntryIdx, err error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

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
	db.mu.RLock()
	defer db.mu.RUnlock()

	logDB, knownChain := db.logDBs[chain]
	if !knownChain {
		return 0, false
	}
	return logDB.LatestSealedBlockNum()
}

func (db *ChainsDB) UnsafeView(chainID types.ChainID, unsafe types.ReferenceView) (heads.HeadPointer, heads.HeadPointer, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	eventsDB, ok := db.logDBs[chainID]
	if !ok {
		return heads.HeadPointer{}, heads.HeadPointer{}, ErrUnknownChain
	}
	// TODO fetch cross-unsafe
	return heads.HeadPointer{}, heads.HeadPointer{}, nil
}

func (db *ChainsDB) SafeView(chainID types.ChainID, safe types.ReferenceView) (heads.HeadPointer, heads.HeadPointer, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	localDB, ok := db.localDBs[chainID]
	if !ok {
		return heads.HeadPointer{}, heads.HeadPointer{}, ErrUnknownChain
	}
	// TODO tip of localDB = local safe head
	crossDB, ok := db.crossDBs[chainID]
	if !ok {
		return heads.HeadPointer{}, heads.HeadPointer{}, ErrUnknownChain
	}
	// TODO tip of crossDB = cross safe head
	return heads.HeadPointer{}, heads.HeadPointer{}, nil
}

func (db *ChainsDB) Finalized(chainID types.ChainID) (eth.BlockID, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// TODO
}

func (db *ChainsDB) DerivedFrom(chainID types.ChainID, derived eth.BlockID) (derivedFrom eth.BlockID, err error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	localDB, ok := db.localDBs[chainID]
	if !ok {
		return eth.BlockRef{}, ErrUnknownChain
	}
	localDB.DerivedFrom()
}

// Check calls the underlying logDB to determine if the given log entry exists at the given location.
// If the block-seal of the block that includes the log is known, it is returned. It is fully zeroed otherwise, if the block is in-progress.
func (db *ChainsDB) Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash common.Hash) (includedIn eth.BlockID, err error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	logDB, ok := db.logDBs[chain]
	if !ok {
		return eth.BlockID{}, fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	_, err := logDB.Contains(blockNum, logIdx, logHash)
	if err != nil {
		return eth.BlockID{}, err
	}
	// TODO fix this for cross-safe to work
	return eth.BlockID{}, nil
}

// Safest returns the strongest safety level that can be guaranteed for the given log entry.
// it assumes the log entry has already been checked and is valid, this funcion only checks safety levels.
func (db *ChainsDB) Safest(chainID types.ChainID, blockNum uint64, index uint32) (safest types.SafetyLevel) {
	db.mu.RLock()
	defer db.mu.RUnlock()

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

func (db *ChainsDB) IteratorStartingAt(chain types.ChainID, sealedNum uint64, logIndex uint32) (logs.Iterator, error) {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return nil, fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	return logDB.IteratorStartingAt(sealedNum, logIndex)
}
