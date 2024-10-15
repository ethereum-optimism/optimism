package db

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func (db *ChainsDB) AddLog(
	chain types.ChainID,
	logHash common.Hash,
	parentBlock eth.BlockID,
	logIdx uint32,
	execMsg *types.ExecutingMessage) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	logDB, ok := db.logDBs[chain]
	if !ok {
		return fmt.Errorf("cannot AddLog: %w: %v", ErrUnknownChain, chain)
	}
	return logDB.AddLog(logHash, parentBlock, logIdx, execMsg)
}

func (db *ChainsDB) SealBlock(chain types.ChainID, block eth.BlockRef) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	logDB, ok := db.logDBs[chain]
	if !ok {
		return fmt.Errorf("cannot SealBlock: %w: %v", ErrUnknownChain, chain)
	}
	err := logDB.SealBlock(block.ParentHash, block.ID(), block.Time)
	if err != nil {
		return fmt.Errorf("failed to seal block %v: %w", block, err)
	}
	return nil
}

func (db *ChainsDB) Rewind(chain types.ChainID, headBlockNum uint64) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	logDB, ok := db.logDBs[chain]
	if !ok {
		return fmt.Errorf("cannot Rewind: %w: %s", ErrUnknownChain, chain)
	}
	return logDB.Rewind(headBlockNum)
}

func (db *ChainsDB) UpdateLocalSafe(chain types.ChainID, derivedFrom eth.BlockRef, lastDerived eth.BlockRef) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	localDB, ok := db.localDBs[chain]
	if !ok {
		return fmt.Errorf("cannot UpdateLocalSafe: %w: %v", ErrUnknownChain, chain)
	}
	return localDB.AddDerived(derivedFrom, lastDerived)
}

func (db *ChainsDB) UpdateCrossUnsafe(chain types.ChainID, crossUnsafe types.BlockSeal) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if _, ok := db.crossUnsafe[chain]; !ok {
		return fmt.Errorf("cannot UpdateCrossUnsafe: %w: %s", ErrUnknownChain, chain)
	}
	db.crossUnsafe[chain] = crossUnsafe
	return nil
}

func (db *ChainsDB) UpdateCrossSafe(chain types.ChainID, l1View eth.BlockRef, lastCrossDerived eth.BlockRef) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	crossDB, ok := db.crossDBs[chain]
	if !ok {
		return fmt.Errorf("cannot UpdateCrossSafe: %w: %s", ErrUnknownChain, chain)
	}
	return crossDB.AddDerived(l1View, lastCrossDerived)
}

func (db *ChainsDB) UpdateFinalizedL1(finalized eth.BlockRef) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.finalizedL1.Number > finalized.Number {
		return fmt.Errorf("cannot rewind finalized L1 head from %s to %s", db.finalizedL1, finalized)
	}
	db.finalizedL1 = finalized
	return nil
}
