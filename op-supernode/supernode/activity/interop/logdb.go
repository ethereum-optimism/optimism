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
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/reads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// LogsDB is the interface for interacting with a chain's logs database.
// *logs.DB implements this interface.
type LogsDB interface {
	// LatestSealedBlock returns the latest sealed block ID, or false if no blocks are sealed.
	LatestSealedBlock() (eth.BlockID, bool)
	// FirstSealedBlock returns the first block seal in the DB.
	FirstSealedBlock() (types.BlockSeal, error)
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
	// Rewind removes all blocks after newHead from the database.
	Rewind(inv reads.Invalidator, newHead eth.BlockID) error
	// Clear removes all data from the database.
	Clear(inv reads.Invalidator) error
	// Close closes the database.
	Close() error
}

// Compile-time check that *logs.DB implements LogsDB.
var _ LogsDB = (*logs.DB)(nil)

// noopLogsDBMetrics implements the logs.Metrics interface with no-op methods.
type noopLogsDBMetrics struct{}

func (n *noopLogsDBMetrics) RecordDBEntryCount(kind string, count int64) {}
func (n *noopLogsDBMetrics) RecordDBSearchEntriesRead(count int64)       {}

// noopInvalidator implements reads.Invalidator as a no-op.
// Used for rewind operations where we don't need cache invalidation.
// noopInvalidator is a stub needed to use the logs.DB.Rewind method.
// read-handle invalidation is not currently used
type noopInvalidator struct{}

func (n *noopInvalidator) TryInvalidate(rule reads.InvalidationRule) (release func(), err error) {
	return func() {}, nil
}

var _ reads.Invalidator = (*noopInvalidator)(nil)

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
	// ErrPreviousTimestampNotSealed is returned when a logsDB write path needs a
	// previous timestamp that has not been sealed yet.
	ErrPreviousTimestampNotSealed = errors.New("previous timestamp not sealed in logsDB")

	// ErrParentHashMismatch is returned when the block's parent hash does not match
	// the hash of the last sealed block in the logsDB.
	ErrParentHashMismatch = errors.New("block parent hash does not match logsDB")

	// ErrStaleLogsDB is returned when the logsDB has data for a different block
	// at the same height (e.g., after a chain reorg). The caller should repair
	// the logsDB by trimming to the verified frontier and retrying.
	ErrStaleLogsDB = errors.New("logsDB has stale block data from a reorg")
)

// persistFrontierLogs persists the exact accepted frontier blocks for a
// timestamp. Frontier logs are only written here, during DecisionAdvance
// transition apply — verification itself is read-only.
func (i *Interop) persistFrontierLogs(ts uint64, blocksAtTS map[eth.ChainID]eth.BlockID) error {
	for chainID, blockID := range blocksAtTS {
		chain, ok := i.chains[chainID]
		if !ok {
			continue
		}
		db := i.logsDBs[chainID]

		latestBlock, hasBlocks, err := i.verifyCanAddTimestamp(chainID, db, ts, chain.BlockTime())
		if err != nil {
			return err
		}

		blockInfo, receipts, err := chain.FetchReceipts(i.ctx, blockID)
		if err != nil {
			return fmt.Errorf("chain %s: failed to fetch receipts for block %v: %w", chainID, blockID, err)
		}

		if hasBlocks {
			if latestBlock.Number > blockID.Number {
				seal, err := db.FindSealedBlock(blockID.Number)
				if err == nil && seal.Hash == blockID.Hash {
					continue
				}
				return fmt.Errorf("chain %s: logsDB has stale data at height %d: %w",
					chainID, blockID.Number, ErrStaleLogsDB)
			}
			if latestBlock.Number == blockID.Number {
				if latestBlock.Hash == blockID.Hash {
					continue
				}
				return fmt.Errorf("chain %s: logsDB has block %s at height %d, expected %s: %w",
					chainID, latestBlock.Hash, latestBlock.Number, blockID.Hash, ErrStaleLogsDB)
			}

			if blockInfo.ParentHash() != latestBlock.Hash {
				return fmt.Errorf("chain %s: block %d parent hash %s does not match logsDB last sealed block hash %s: %w",
					chainID, blockID.Number, blockInfo.ParentHash(), latestBlock.Hash, ErrParentHashMismatch)
			}
		}

		isFirstBlock := !hasBlocks
		if err := i.processBlockLogs(db, blockInfo, receipts, isFirstBlock); err != nil {
			return fmt.Errorf("chain %s: failed to process block logs for block %d: %w", chainID, blockID.Number, err)
		}
	}

	return nil
}

func (i *Interop) verifyCanAddTimestamp(chainID eth.ChainID, db LogsDB, ts uint64, blockTime uint64) (eth.BlockID, bool, error) {
	latestBlock, hasBlocks := db.LatestSealedBlock()

	// If no blocks in DB:
	// - At activation timestamp: OK, proceed to load the first block
	// - Not at activation timestamp: ERROR, we're missing data
	if !hasBlocks {
		if ts == i.activationTimestamp {
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

	// If the gap is less than a block time, we can still append the timestamp to the database.
	// This is expected for chains whose block time is greater than one second, since the
	// interop timestamp may legitimately fall between consecutive L2 blocks.
	if gap < blockTime {
		i.log.Debug("verifyCanAddTimestamp: timestamp falls between L2 blocks for this chain; this can be expected for chains with block times greater than one second",
			"chain", chainID,
			"timestamp", ts,
			"block time", blockTime,
			"gap", gap,
		)
	}

	return latestBlock, hasBlocks, nil
}

// processBlockLogs processes the receipts for a block and stores the logs in the database.
// If isFirstBlock is true, this is the first block being added to the logsDB (at activation timestamp),
// and we first seal a "virtual parent" block so that logs have a sealed block to reference.
// This allows the logsDB to start at any block number, not just genesis.
func (i *Interop) processBlockLogs(db LogsDB, blockInfo eth.BlockInfo, receipts gethTypes.Receipts, isFirstBlock bool) error {
	blockNum := blockInfo.NumberU64()
	blockID := eth.BlockID{Hash: blockInfo.Hash(), Number: blockNum}
	parentHash := blockInfo.ParentHash()

	parentBlock := eth.BlockID{Hash: parentHash, Number: blockNum - 1}
	sealParentHash := parentHash

	// For the first block in the logsDB (activation block), we need to first seal
	// a virtual parent block so that logs have a sealed block to reference.
	// When the DB is empty, SealBlock allows any block to be added without parent validation.
	if isFirstBlock && blockNum > 0 {
		// Seal the parent as a "virtual genesis" - this works because DB is empty
		if err := db.SealBlock(common.Hash{}, parentBlock, blockInfo.Time()); err != nil {
			return fmt.Errorf("failed to seal virtual parent for first block: %w", err)
		}
		// parentBlock stays as-is (references the now-sealed parent)
		// sealParentHash stays as parentHash
	} else if blockNum == 0 {
		// Actual genesis block - no parent, no logs allowed
		parentBlock = eth.BlockID{}
		sealParentHash = common.Hash{}
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

	if err := db.SealBlock(sealParentHash, blockID, blockInfo.Time()); err != nil {
		return fmt.Errorf("failed to seal block: %w", err)
	}

	return nil
}
