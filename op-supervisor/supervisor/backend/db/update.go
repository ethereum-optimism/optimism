package db

import (
	"fmt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
)

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

func (db *ChainsDB) SealBlock(chain types.ChainID, block eth.BlockRef) error {
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

func (db *ChainsDB) UpdateLocalSafe(chain types.ChainID, derivedFrom eth.BlockRef, lastDerived eth.BlockRef) error {
	localDB, ok := db.localDBs[chain]
	if !ok {
		return ErrUnknownChain
	}
	//TODO feed derived-from link into localDB.
	return nil
}

func (db *ChainsDB) UpdateFinalizedL1(finalized eth.BlockRef) error {
	if db.finalizedL1.Number > finalized.Number {
		return fmt.Errorf("cannot rewind finalized L1 head from %s to %s", db.finalizedL1, finalized)
	}
	db.finalizedL1 = finalized
	return nil
}
