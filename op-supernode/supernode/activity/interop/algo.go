package interop

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// noopLogsDBMetrics implements the logs.Metrics interface with no-op methods.
type noopLogsDBMetrics struct{}

func (n *noopLogsDBMetrics) RecordDBEntryCount(kind string, count int64) {}
func (n *noopLogsDBMetrics) RecordDBSearchEntriesRead(count int64)      {}

// openLogsDB opens a logs.DB for the given chain in the data directory.
func openLogsDB(logger log.Logger, chainID eth.ChainID, dataDir string) (*logs.DB, error) {
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

// IncludedMessage wraps an executing message with its inclusion context.
// The ExecutingMessage contains the initiating message's data (source chain),
// while InclusionBlockNum/Timestamp indicate when it was executed (this chain).
type IncludedMessage struct {
	*types.ExecutingMessage
	InclusionBlockNum  uint64
	InclusionTimestamp uint64
}

// =============================================================================
// Interop Algorithm Functions (stubbed - to be implemented)
// =============================================================================

// loadLogs loads and persists logs up to the given timestamp for all chains.
// The previous timestamp is assumed to already be downloaded and verified.
func (i *Interop) loadLogs(ts uint64) error {
	// TODO(#18743): Implement log loading
	// For each chain:
	// 1. Determine the block range from last processed to the block at timestamp
	// 2. Fetch receipts for each block
	// 3. Process logs and store in the chain's logsDB
	return nil
}

// verifyInteropMessages validates all executing messages at the given timestamp.
// Returns a Result indicating whether all messages are valid or which chains have invalid blocks.
func (i *Interop) verifyInteropMessages(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error) {
	// TODO(#18743): Implement message verification
	// For now, return all blocks as valid (stub behavior)
	result := Result{Timestamp: ts, L2Heads: make(map[eth.ChainID]eth.BlockID)}
	for _, chain := range i.chains {
		blockID := blocksAtTimestamp[chain.ID()]
		result.L2Heads[chain.ID()] = blockID
	}
	return result, nil
}

// invalidateBlock handles an invalid block by notifying the chain to reorg.
func (i *Interop) invalidateBlock(chainID eth.ChainID, blockID eth.BlockID) error {
	// TODO(#18944): Implement block invalidation
	// This should trigger the chain container to reorg away from the invalid block
	i.log.Warn("invalidateBlock called but not implemented", "chainID", chainID, "blockID", blockID)
	return nil
}
