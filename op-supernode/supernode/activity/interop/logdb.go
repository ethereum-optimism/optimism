package interop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// LogsDB is the interface for interacting with a chain's logs database.
// *logs.DB implements this interface.
type LogsDB interface {
	// LatestSealedBlock returns the latest sealed block ID, or false if no blocks are sealed.
	LatestSealedBlock() (eth.BlockID, bool)
	// FindSealedBlock returns the block seal for the given block number.
	FindSealedBlock(number uint64) (types.BlockSeal, error)
	// OpenBlock returns the block reference, log count, and executing messages for a block.
	OpenBlock(blockNum uint64) (ref eth.BlockRef, logCount uint32, execMsgs map[uint32]*types.ExecutingMessage, err error)
	// Contains checks if an initiating message exists in the database.
	// Returns the block seal if found, or an error (ErrConflict if not found, ErrFuture if not yet indexed).
	Contains(query types.ContainsQuery) (types.BlockSeal, error)
	// AddLog adds a log entry to the database.
	AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *types.ExecutingMessage) error
	// SealBlock seals a block in the database.
	SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error
	// Close closes the database.
	Close() error
}

// Compile-time check that *logs.DB implements LogsDB.
var _ LogsDB = (*logs.DB)(nil)

// noopLogsDBMetrics implements the logs.Metrics interface with no-op methods.
type noopLogsDBMetrics struct{}

func (n *noopLogsDBMetrics) RecordDBEntryCount(kind string, count int64) {}
func (n *noopLogsDBMetrics) RecordDBSearchEntriesRead(count int64)       {}

// openLogsDB opens a logs.DB for the given chain in the data directory.
func openLogsDB(logger log.Logger, chainID eth.ChainID, dataDir string) (LogsDB, error) {
	chainDir := filepath.Join(dataDir, fmt.Sprintf("chain-%s", chainID))
	if err := os.MkdirAll(chainDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chain directory: %w", err)
	}

	dbPath := filepath.Join(chainDir, "logs.db")
	db, err := logs.NewFromFile(logger, &noopLogsDBMetrics{}, chainID, dbPath, true)
	if err != nil {
		return nil, fmt.Errorf("failed to open logs DB for chain %s: %w", chainID, err)
	}

	logger.Info("Initialized logs DB", "chain", chainID, "path", dbPath)
	return db, nil
}

var (
	// ErrPreviousTimestampNotSealed is returned when loadLogs is called but the
	// previous timestamp has not been sealed in the logsDB.
	ErrPreviousTimestampNotSealed = errors.New("previous timestamp not sealed in logsDB")

	// ErrParentHashMismatch is returned when the block's parent hash does not match
	// the hash of the last sealed block in the logsDB.
	ErrParentHashMismatch = errors.New("block parent hash does not match logsDB")
)

// loadLogs loads and persists logs for the given timestamp for all chains.
// The previous timestamp MUST already be sealed in the database; if not, an error is returned.
// For the activation timestamp (first timestamp), the logsDB must be empty.
func (i *Interop) loadLogs(ts uint64) error {
	for chainID, chain := range i.chains {
		db := i.logsDBs[chainID]

		// Verify the previous timestamp is sealed (or DB is empty for activation timestamp)
		// Returns the hash of the previous sealed block, or nil if DB is empty
		prevHash, err := i.verifyPreviousTimestampSealed(chainID, db, ts)
		if err != nil {
			return err
		}

		// Get the block at timestamp ts
		block, err := chain.BlockAtTimestamp(i.ctx, ts, eth.Safe)
		if err != nil {
			return fmt.Errorf("chain %s: failed to get block at timestamp %d: %w", chainID, ts, err)
		}

		// Check if this block is already sealed in the logsDB
		// This happens when no new block exists at this timestamp - we get the same block
		// that was already sealed at a previous timestamp
		latestSealed, hasSealed := db.LatestSealedBlock()
		if hasSealed && latestSealed.Hash == block.Hash {
			// Block is already sealed, nothing new to process for this chain
			i.log.Debug("block already sealed, skipping",
				"chain", chainID,
				"block", block.Number,
				"timestamp", ts,
			)
			continue
		}

		// Fetch receipts for the block
		blockInfo, receipts, err := chain.FetchReceipts(i.ctx, block.ID())
		if err != nil {
			return fmt.Errorf("chain %s: failed to fetch receipts for block %d: %w", chainID, block.Number, err)
		}

		// Verify chain continuity: block's parent must match the last sealed block
		if prevHash != nil && blockInfo.ParentHash() != *prevHash {
			return fmt.Errorf("chain %s: block %d parent hash %s does not match logsDB hash %s: %w",
				chainID, block.Number, blockInfo.ParentHash(), *prevHash, ErrParentHashMismatch)
		}

		// Process logs and seal the block
		if err := i.processBlockLogs(db, blockInfo, receipts); err != nil {
			return fmt.Errorf("chain %s: failed to process block logs for block %d: %w", chainID, block.Number, err)
		}

		i.log.Debug("loaded logs for chain",
			"chain", chainID,
			"block", block.Number,
			"timestamp", ts,
		)
	}

	return nil
}

// verifyPreviousTimestampSealed checks that the logsDB is ready to process timestamp ts.
// For the activation timestamp, the logsDB must be empty and returns nil for the previous hash.
// For subsequent timestamps, the latest sealed block must have timestamp < ts.
// (Note: not ts-1 exactly, because blocks may span multiple timestamps when block time > 1)
// Returns the hash of the previous sealed block, or nil if the DB is empty (activation timestamp case).
func (i *Interop) verifyPreviousTimestampSealed(chainID eth.ChainID, db LogsDB, ts uint64) (*common.Hash, error) {
	latestBlock, hasBlocks := db.LatestSealedBlock()

	// For the activation timestamp, the logsDB should be empty
	if ts == i.activationTimestamp {
		if hasBlocks {
			return nil, fmt.Errorf("chain %s: logsDB should be empty for activation timestamp %d, but has block %d: %w",
				chainID, ts, latestBlock.Number, ErrPreviousTimestampNotSealed)
		}
		return nil, nil
	}

	// if there are no blocks but we are not verifying the activation timestamp, return an error
	if !hasBlocks {
		return nil, fmt.Errorf("chain %s: logsDB is empty but expected blocks before timestamp %d: %w",
			chainID, ts, ErrPreviousTimestampNotSealed)
	}

	// get the actual seal from the database
	seal, err := db.FindSealedBlock(latestBlock.Number)
	if err != nil {
		return nil, fmt.Errorf("chain %s: failed to find sealed block %d: %w", chainID, latestBlock.Number, err)
	}

	// The latest sealed block must be from before the current timestamp.
	// It doesn't have to be exactly ts-1 because blocks may span multiple timestamps
	// (e.g., with block time = 2: block at ts=100, no block at ts=101, block at ts=102)
	if seal.Timestamp >= ts {
		return nil, fmt.Errorf("chain %s: latest sealed block timestamp %d is not before current timestamp %d: %w",
			chainID, seal.Timestamp, ts, ErrPreviousTimestampNotSealed)
	}

	return &latestBlock.Hash, nil
}

// processBlockLogs processes the receipts for a block and stores the logs in the database.
func (i *Interop) processBlockLogs(db LogsDB, blockInfo eth.BlockInfo, receipts gethTypes.Receipts) error {
	blockNum := blockInfo.NumberU64()
	blockID := eth.BlockID{Hash: blockInfo.Hash(), Number: blockNum}
	parentHash := blockInfo.ParentHash()

	parentBlock := eth.BlockID{Hash: parentHash, Number: blockNum - 1}
	if blockNum == 0 {
		parentBlock = eth.BlockID{}
	}

	var logIndex uint32
	for _, receipt := range receipts {
		for _, l := range receipt.Logs {
			logHash := processors.LogToLogHash(l)

			// Decode executing message if present (nil if not an executing message)
			execMsg, _ := processors.DecodeExecutingMessageLog(l)

			if err := db.AddLog(logHash, parentBlock, logIndex, execMsg); err != nil {
				return fmt.Errorf("failed to add log %d: %w", logIndex, err)
			}
			logIndex++
		}
	}

	// Seal the block
	if err := db.SealBlock(parentHash, blockID, blockInfo.Time()); err != nil {
		return fmt.Errorf("failed to seal block: %w", err)
	}

	return nil
}
