package db

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func (db *ChainsDB) FindSealedBlock(chain types.ChainID, number uint64) (seal types.BlockSeal, err error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	logDB, ok := db.logDBs[chain]
	if !ok {
		return types.BlockSeal{}, fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	return logDB.FindSealedBlock(number)
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

func (db *ChainsDB) LocalUnsafe(chainID types.ChainID) (types.BlockSeal, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	eventsDB, ok := db.logDBs[chainID]
	if !ok {
		return types.BlockSeal{}, ErrUnknownChain
	}
	n, ok := eventsDB.LatestSealedBlockNum()
	if !ok {
		return types.BlockSeal{}, entrydb.ErrFuture
	}
	return eventsDB.FindSealedBlock(n)
}

func (db *ChainsDB) CrossUnsafe(chainID types.ChainID) (types.BlockSeal, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	result, ok := db.crossUnsafe[chainID]
	if !ok {
		return types.BlockSeal{}, ErrUnknownChain
	}
	return result, nil
}

func (db *ChainsDB) LocalSafe(chainID types.ChainID) (derivedFrom eth.BlockRef, derived eth.BlockRef, err error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	localDB, ok := db.localDBs[chainID]
	if !ok {
		return eth.BlockRef{}, eth.BlockRef{}, ErrUnknownChain
	}
	return localDB.Last()
}

func (db *ChainsDB) CrossSafe(chainID types.ChainID) (derivedFrom eth.BlockRef, derived eth.BlockRef, err error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	crossDB, ok := db.crossDBs[chainID]
	if !ok {
		return eth.BlockRef{}, eth.BlockRef{}, ErrUnknownChain
	}
	return crossDB.Last()
}

func (db *ChainsDB) Finalized(chainID types.ChainID) (eth.BlockID, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	finalizedL1 := db.finalizedL1
	if finalizedL1 == (eth.L1BlockRef{}) {
		return eth.BlockID{}, errors.New("no finalized L1 signal, cannot determine L2 finality yet")
	}
	derived, err := db.LastDerivedFrom(chainID, finalizedL1.ID())
	if err != nil {
		return eth.BlockID{}, errors.New("could not find what was last derived from the finalized L1 block")
	}
	return derived, nil
}

func (db *ChainsDB) LastDerivedFrom(chainID types.ChainID, derivedFrom eth.BlockID) (derived eth.BlockID, err error) {
	crossDB, ok := db.crossDBs[chainID]
	if !ok {
		return eth.BlockID{}, ErrUnknownChain
	}
	return crossDB.LastDerived(derivedFrom)
}

func (db *ChainsDB) DerivedFrom(chainID types.ChainID, derived eth.BlockID) (derivedFrom eth.BlockID, err error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	localDB, ok := db.localDBs[chainID]
	if !ok {
		return eth.BlockID{}, ErrUnknownChain
	}
	return localDB.DerivedFrom(derived)
}

// Check calls the underlying logDB to determine if the given log entry exists at the given location.
// If the block-seal of the block that includes the log is known, it is returned. It is fully zeroed otherwise, if the block is in-progress.
func (db *ChainsDB) Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash common.Hash) (includedIn types.BlockSeal, err error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	logDB, ok := db.logDBs[chain]
	if !ok {
		return types.BlockSeal{}, fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	return logDB.Contains(blockNum, logIdx, logHash)
}

// Safest returns the strongest safety level that can be guaranteed for the given log entry.
// it assumes the log entry has already been checked and is valid, this function only checks safety levels.
// Cross-safety levels are all considered to be more safe than any form of local-safety.
func (db *ChainsDB) Safest(chainID types.ChainID, blockNum uint64, index uint32) (safest types.SafetyLevel, err error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if finalized, err := db.Finalized(chainID); err == nil {
		if finalized.Number >= blockNum {
			return types.Finalized, nil
		}
	}
	_, crossSafe, err := db.CrossSafe(chainID)
	if err != nil {
		return types.Invalid, err
	}
	if crossSafe.Number >= blockNum {
		return types.CrossSafe, nil
	}
	crossUnsafe, err := db.CrossUnsafe(chainID)
	if err != nil {
		return types.Invalid, err
	}
	// TODO(#12425): API: "index" for in-progress block building shouldn't be exposed from DB.
	//  For now we're not counting anything cross-safe until the block is sealed.
	if blockNum <= crossUnsafe.Number {
		return types.CrossUnsafe, nil
	}
	_, localSafe, err := db.LocalSafe(chainID)
	if err != nil {
		return types.Invalid, err
	}
	if blockNum <= localSafe.Number {
		return types.LocalSafe, nil
	}
	return types.LocalUnsafe, nil
}

func (db *ChainsDB) IteratorStartingAt(chain types.ChainID, sealedNum uint64, logIndex uint32) (logs.Iterator, error) {
	logDB, ok := db.logDBs[chain]
	if !ok {
		return nil, fmt.Errorf("%w: %v", ErrUnknownChain, chain)
	}
	return logDB.IteratorStartingAt(sealedNum, logIndex)
}
