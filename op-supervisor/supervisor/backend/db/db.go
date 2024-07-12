package db

import (
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-service/eth"
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
}

type HeadsStorage interface {
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
