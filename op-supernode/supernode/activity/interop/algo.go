package interop

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var (
	// ErrInitiatingMessageNotFound is returned when an executing message references
	// an initiating message that doesn't exist in the source chain's database.
	ErrInitiatingMessageNotFound = errors.New("initiating message not found")

	// ErrTimestampViolation is returned when an executing message references
	// an initiating message with a timestamp >= the executing message's timestamp.
	ErrTimestampViolation = errors.New("initiating message timestamp must be less than executing message timestamp")

	// ErrBlockMismatch is returned when the block in the logsDB doesn't match
	// the expected block from blocksAtTimestamp.
	ErrBlockMismatch = errors.New("block in logsDB does not match expected block")
)

// verifyInteropMessages validates all executing messages at the given timestamp.
// Returns a Result indicating whether all messages are valid or which chains have invalid blocks.
//
// For each chain:
// 1. Open the block from the logsDB and verify it matches blocksAtTimestamp
// 2. For each executing message in the block:
//   - Verify the initiating message exists in the source chain's logsDB
//   - Verify the initiating message timestamp < executing message timestamp
func (i *Interop) verifyInteropMessages(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error) {
	result := Result{
		Timestamp:    ts,
		L2Heads:      make(map[eth.ChainID]eth.BlockID),
		InvalidHeads: make(map[eth.ChainID]eth.BlockID),
	}

	for chainID, expectedBlock := range blocksAtTimestamp {
		db, ok := i.logsDBs[chainID]
		if !ok {
			// Skip chains that we don't have a logsDB for
			// This can happen if blocksAtTimestamp includes chains not registered with the interop activity
			continue
		}

		// Get the block from the logsDB
		blockRef, _, execMsgs, err := db.OpenBlock(expectedBlock.Number)
		if err != nil {
			return Result{}, fmt.Errorf("chain %s: failed to open block %d: %w", chainID, expectedBlock.Number, err)
		}

		// Verify the block hash matches what we expect
		if blockRef.Hash != expectedBlock.Hash {
			i.log.Warn("block hash mismatch",
				"chain", chainID,
				"expected", expectedBlock.Hash,
				"got", blockRef.Hash,
			)
			result.InvalidHeads[chainID] = expectedBlock
			result.L2Heads[chainID] = expectedBlock
			continue
		}

		// Verify each executing message
		blockValid := true
		for logIdx, execMsg := range execMsgs {
			if err := i.verifyExecutingMessage(chainID, blockRef.Time, logIdx, execMsg); err != nil {
				i.log.Warn("invalid executing message",
					"chain", chainID,
					"block", expectedBlock.Number,
					"logIdx", logIdx,
					"execMsg", execMsg,
					"err", err,
				)
				blockValid = false
				break
			}
		}

		result.L2Heads[chainID] = expectedBlock
		if !blockValid {
			result.InvalidHeads[chainID] = expectedBlock
		}
	}

	return result, nil
}

// verifyExecutingMessage verifies a single executing message by checking:
// 1. The initiating message exists in the source chain's database
// 2. The initiating message's timestamp is less than the executing block's timestamp
func (i *Interop) verifyExecutingMessage(executingChain eth.ChainID, executingTimestamp uint64, logIdx uint32, execMsg *types.ExecutingMessage) error {
	// Get the source chain's logsDB
	sourceDB, ok := i.logsDBs[execMsg.ChainID]
	if !ok {
		return fmt.Errorf("source chain %s not found: %w", execMsg.ChainID, ErrInitiatingMessageNotFound)
	}

	// Build the query for the initiating message
	query := types.ContainsQuery{
		BlockNum:  execMsg.BlockNum,
		LogIdx:    execMsg.LogIdx,
		Timestamp: execMsg.Timestamp,
		Checksum:  execMsg.Checksum,
	}

	// Check if the initiating message exists in the source chain's logsDB
	_, err := sourceDB.Contains(query)
	if err != nil {
		if errors.Is(err, types.ErrConflict) {
			return fmt.Errorf("chain %s block %d log %d: %w",
				execMsg.ChainID, execMsg.BlockNum, execMsg.LogIdx, ErrInitiatingMessageNotFound)
		}
		if errors.Is(err, types.ErrFuture) {
			// The source chain hasn't indexed this block yet - this shouldn't happen
			// since we process timestamps in order, but handle it gracefully
			return fmt.Errorf("chain %s block %d log %d: initiating message not yet indexed: %w",
				execMsg.ChainID, execMsg.BlockNum, execMsg.LogIdx, ErrInitiatingMessageNotFound)
		}
		return fmt.Errorf("chain %s: failed to check initiating message: %w", execMsg.ChainID, err)
	}

	// Verify timestamp ordering: initiating message timestamp must be < executing block timestamp
	if execMsg.Timestamp >= executingTimestamp {
		return fmt.Errorf("initiating timestamp %d >= executing timestamp %d: %w",
			execMsg.Timestamp, executingTimestamp, ErrTimestampViolation)
	}

	return nil
}
