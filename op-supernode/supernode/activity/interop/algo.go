package interop

import (
	"errors"
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// ExpiryTime is the maximum age of an initiating message that can be executed.
// Messages older than this are considered expired and invalid.
// 7 days = 7 * 24 * 60 * 60 = 604800 seconds
const ExpiryTime = 604800

var (
	// ErrUnknownChain is returned when an executing message references
	// a chain that is not registered with the interop activity.
	ErrUnknownChain = errors.New("unknown chain")

	// ErrTimestampViolation is returned when an executing message references
	// an initiating message with a timestamp > the executing message's timestamp.
	ErrTimestampViolation = errors.New("initiating message timestamp must not be greater than executing message timestamp")

	// ErrMessageExpired is returned when an executing message references
	// an initiating message that has expired (older than ExpiryTime).
	ErrMessageExpired = errors.New("initiating message has expired")
)

// verifyInteropMessages validates all executing messages at the given timestamp.
// Returns a Result indicating whether all messages are valid or which chains have invalid blocks.
//
// For each chain:
// 1. Open the block from the logsDB and verify it matches blocksAtTimestamp
// 2. For each executing message in the block:
//   - Verify the initiating message exists in the source chain's logsDB
//   - Verify the initiating message timestamp <= executing message timestamp
//   - Verify the initiating message hasn't expired (within ExpiryTime)
func (i *Interop) verifyInteropMessages(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error) {
	result := Result{
		Timestamp:    ts,
		L2Heads:      make(map[eth.ChainID]eth.BlockID),
		InvalidHeads: make(map[eth.ChainID]eth.BlockID),
	}

	// Compute L1Inclusion: the earliest L1 block such that all L2 blocks at the
	// supplied timestamp were derived
	// from a source at or before that L1 block.
	earliestL1Inclusion := eth.BlockID{
		Number: math.MaxUint64,
	}
	for chainID := range blocksAtTimestamp {
		chain, ok := i.chains[chainID]
		if !ok {
			continue
		}
		_, l1Block, err := chain.OptimisticAt(i.ctx, ts)
		if err != nil {
			i.log.Error("failed to get L1 inclusion for L2 block", "chainID", chainID, "timestamp", ts, "err", err)
			return Result{}, fmt.Errorf("chain %s: failed to get L1 inclusion: %w", chainID, err)
		}
		if l1Block.Number < earliestL1Inclusion.Number {
			earliestL1Inclusion = l1Block
		}
	}
	if earliestL1Inclusion.Number == math.MaxUint64 {
		return Result{}, fmt.Errorf("no L1 inclusion found for timestamp %d", ts)
	}
	result.L1Inclusion = earliestL1Inclusion

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
			// OpenBlock fails for the first block in the DB because it tries to find the parent.
			// Handle this by checking if this is the first sealed block and using FirstSealedBlock instead.
			if errors.Is(err, types.ErrSkipped) {
				firstBlock, firstErr := db.FirstSealedBlock()
				if firstErr != nil {
					return Result{}, fmt.Errorf("chain %s: failed to open block %d and failed to get first block: %w", chainID, expectedBlock.Number, err)
				}
				if firstBlock.Number == expectedBlock.Number {
					// This is the first block in the logsDB. Use FirstSealedBlock info.
					// The first block has no executing messages (since we can't verify them without prior data).
					if firstBlock.Hash != expectedBlock.Hash {
						i.log.Warn("first block hash mismatch",
							"chain", chainID,
							"expected", expectedBlock.Hash,
							"got", firstBlock.Hash,
						)
						result.InvalidHeads[chainID] = expectedBlock
					}
					result.L2Heads[chainID] = expectedBlock
					continue
				}
			}
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
			err := i.verifyExecutingMessage(chainID, blockRef.Time, logIdx, execMsg)
			if err != nil {
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
//  1. The initiating message exists in the source chain's database
//  2. The initiating message's timestamp is not greater than the executing block's timestamp
//  3. The initiating message hasn't expired (timestamp + ExpiryTime >= executing timestamp)
func (i *Interop) verifyExecutingMessage(executingChain eth.ChainID, executingTimestamp uint64, logIdx uint32, execMsg *types.ExecutingMessage) error {
	// Get the source chain's logsDB
	sourceDB, ok := i.logsDBs[execMsg.ChainID]
	if !ok {
		return fmt.Errorf("source chain %s not found: %w", execMsg.ChainID, ErrUnknownChain)
	}

	// Verify timestamp ordering: initiating message timestamp must be <= executing block timestamp.
	if execMsg.Timestamp > executingTimestamp {
		return fmt.Errorf("initiating timestamp %d > executing timestamp %d: %w",
			execMsg.Timestamp, executingTimestamp, ErrTimestampViolation)
	}

	// Verify the message hasn't expired: initiating timestamp + ExpiryTime must be >= executing timestamp
	if execMsg.Timestamp+ExpiryTime < executingTimestamp {
		return fmt.Errorf("initiating timestamp %d + expiry %d < executing timestamp %d: %w",
			execMsg.Timestamp, ExpiryTime, executingTimestamp, ErrMessageExpired)
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
	return err
}
