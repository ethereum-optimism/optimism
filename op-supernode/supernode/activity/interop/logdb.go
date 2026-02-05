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
// For the activation timestamp, we skip loading since the logsDB may not support non-sequential
// block sealing (e.g., when interop activates after genesis).
func (i *Interop) loadLogs(ts uint64) error {
	// At activation timestamp, skip loading logs.
	// The logsDB requires sequential block processing from genesis, which isn't possible
	// when interop activates after genesis. Verification is also skipped at activation.
	if ts == i.activationTimestamp {
		i.log.Info("at activation timestamp, skipping logsDB loading", "timestamp", ts)
		return nil
	}

	for chainID, chain := range i.chains {
		db := i.logsDBs[chainID]

		// Verify the previous timestamp is sealed (or DB is empty for activation timestamp)
		// Returns the hash of the previous sealed block, or nil if DB is empty
		latestBlock, hasBlocks, err := i.verifyCanAddTimestamp(chainID, db, ts, chain.BlockTime())
		if err != nil {
			return err
		}

		// Get the block at timestamp ts
		block, err := chain.BlockAtTimestamp(i.ctx, ts, eth.Safe)
		if err != nil {
			return fmt.Errorf("chain %s: failed to get block at timestamp %d: %w", chainID, ts, err)
		}

		// Fetch receipts for the block
		blockInfo, receipts, err := chain.FetchReceipts(i.ctx, block.ID())
		if err != nil {
			return fmt.Errorf("chain %s: failed to fetch receipts for block %d: %w", chainID, block.Number, err)
		}

		// if the database has blocks, check if we can skip or need to verify continuity
		if hasBlocks {
			// if the latest block is the same or beyond the block we are loading, skip loading
			if latestBlock.Number >= block.Number {
				continue
			}

			// Verify chain continuity: block's parent must match the last sealed block
			if blockInfo.ParentHash() != latestBlock.Hash {
				return fmt.Errorf("chain %s: block %d parent hash %s does not match logsDB last sealed block hash %s: %w",
					chainID, block.Number, blockInfo.ParentHash(), latestBlock.Hash, ErrParentHashMismatch)
			}
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

func (i *Interop) verifyCanAddTimestamp(chainID eth.ChainID, db LogsDB, ts uint64, blockTime uint64) (eth.BlockID, bool, error) {
	latestBlock, hasBlocks := db.LatestSealedBlock()

	// If no blocks in DB:
	// - At activation timestamp: OK (we'll skip loading at activation anyway)
	// - At activation+blockTime (first real timestamp after activation): OK, this is effectively the first timestamp
	// - Otherwise: ERROR, we're missing data
	if !hasBlocks {
		// Allow empty DB at activation (we skip loading) or at activation+blockTime (first real timestamp)
		if ts == i.activationTimestamp || ts == i.activationTimestamp+blockTime {
			return eth.BlockID{}, hasBlocks, nil
		}
		return eth.BlockID{}, hasBlocks, fmt.Errorf("chain %s: logsDB is empty but expected blocks before timestamp %d: %w",
			chainID, ts, ErrPreviousTimestampNotSealed)
	}

	// DB has blocks - fall through to normal timestamp checks below
	// This handles the case where we restart at activation timestamp but the logsDB already has data

	// determine the timestamp of the last sealed block
	seal, err := db.FindSealedBlock(latestBlock.Number)
	if err != nil {
		return eth.BlockID{}, hasBlocks, fmt.Errorf("chain %s: failed to find sealed block %d: %w", chainID, latestBlock.Number, err)
	}

	// if the last sealed block is already after the timestamp in question, return success
	if seal.Timestamp > ts {
		return latestBlock, hasBlocks, nil
	}

	gap := ts - seal.Timestamp

	// if there is more than a block time of gap, we cannot append the timestamp to the database
	if gap > blockTime {
		return eth.BlockID{}, hasBlocks, fmt.Errorf("chain %s: the prior block timestamp %d (%d minus block time %d) is not sealed (last sealed block timestamp: %d): %w",
			chainID, ts-blockTime, ts, blockTime, seal.Timestamp, ErrPreviousTimestampNotSealed)
	}

	// if the gap is less than a block time, we can append the timestamp to the database, but warn the caller
	if gap < blockTime {
		i.log.Warn("verifyCanAddTimestamp: requested for timestamp which is not a multiple of block time",
			"chain", chainID,
			"timestamp", ts,
			"block time", blockTime,
			"gap", gap,
		)
	}

	return latestBlock, hasBlocks, nil
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
