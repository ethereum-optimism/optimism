package interop

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
)

// defaultMessageExpiryWindow is the default maximum age of an initiating message
// that can be executed. 7 days = 7 * 24 * 60 * 60 = 604800 seconds.
// The actual value used is read from the dependency set at construction time.
const defaultMessageExpiryWindow = 604800

var (
	// ErrUnknownChain is returned when an executing message references
	// a chain that is not registered with the interop activity.
	ErrUnknownChain = errors.New("unknown chain")

	// ErrTimestampViolation is returned when an executing message references
	// an initiating message with a timestamp > the executing message's timestamp.
	ErrTimestampViolation = errors.New("initiating message timestamp must not be greater than executing message timestamp")

	// ErrMessageExpired is returned when an executing message references
	// an initiating message that has expired (older than the message expiry window).
	ErrMessageExpired = errors.New("initiating message has expired")

	// ErrExecutedTooEarly is returned when an executing message is in the executing chain's
	// pre-activation or activation block.
	ErrExecutedTooEarly = errors.New("interop is not active for at least one block on the executing chain")

	// ErrInitiatedTooEarly is returned when an executing message references an initiating
	// message in the initiating chain's pre-activation or activation block.
	ErrInitiatedTooEarly = errors.New("interop is not active for at least one block on the initiating chain")
)

type blockPerChain = map[eth.ChainID]eth.BlockID

// l1Inclusion returns the latest L1 block from the l1Heads snapshot. l1Heads must be the
// per-chain snapshot captured atomically with blocksAtTimestamp in observeRound; re-reading
// it here would race with L2 reorgs between observation and verification.
func (i *Interop) l1Inclusion(blocksAtTimestamp blockPerChain, l1Heads blockPerChain) (eth.BlockID, error) {
	l1Inclusion := eth.BlockID{}
	for chainID := range blocksAtTimestamp {
		if _, ok := i.chains[chainID]; !ok {
			continue
		}
		l1Block, ok := l1Heads[chainID]
		if !ok {
			return eth.BlockID{}, fmt.Errorf("chain %s: missing L1 inclusion in observation snapshot", chainID)
		}
		if l1Block.Number >= l1Inclusion.Number {
			l1Inclusion = l1Block
		}
	}
	return l1Inclusion, nil
}

// verifyInteropMessages validates all executing messages at the given timestamp.
// Returns a Result indicating whether all messages are valid or which chains have invalid blocks.
//
// For each chain:
// 1. Open the block from the logsDB and verify it matches blocksAtTimestamp
// 2. For each executing message in the block:
//   - Verify the initiating message exists in the source chain's logsDB
//   - Verify the initiating message timestamp <= executing message timestamp
//   - Verify the initiating message hasn't expired (within message expiry window)
func (i *Interop) verifyInteropMessages(ts uint64, blocksAtTimestamp blockPerChain, l1Heads blockPerChain, view *frontierVerificationView) (Result, error) {
	result := Result{
		Timestamp:    ts,
		L2Heads:      make(blockPerChain),
		InvalidHeads: make(map[eth.ChainID]InvalidHead),
	}

	if l1Inclusion, err := i.l1Inclusion(blocksAtTimestamp, l1Heads); err != nil {
		return Result{}, err
	} else {
		result.L1Inclusion = l1Inclusion
	}

	for chainID, expectedBlock := range blocksAtTimestamp {
		var (
			blockRef eth.BlockRef
			execMsgs map[uint32]*messages.ExecutingMessage
			err      error
		)
		if frontierBlock, ok := view.block(chainID); ok {
			blockRef = frontierBlock.ref
			execMsgs = frontierBlock.execMsgs
		} else {
			db, ok := i.logsDBs[chainID]
			if !ok {
				// Skip chains that we don't have a logsDB for
				// This can happen if blocksAtTimestamp includes chains not registered with the interop activity
				continue
			}

			blockRef, _, execMsgs, err = db.OpenBlock(expectedBlock.Number)
			if err != nil {
				return Result{}, fmt.Errorf("chain %s: failed to open block %d: %w", chainID, expectedBlock.Number, err)
			}
		}

		// Verify the block hash matches what we expect
		if blockRef.Hash != expectedBlock.Hash {
			i.log.Warn("block hash mismatch",
				"chain", chainID,
				"expected", expectedBlock.Hash,
				"got", blockRef.Hash,
			)
			invalid, err := i.newInvalidHead(chainID, expectedBlock)
			if err != nil {
				return Result{}, fmt.Errorf("chain %s: %w", chainID, err)
			}
			result.InvalidHeads[chainID] = invalid
			result.L2Heads[chainID] = expectedBlock
			continue
		}

		// Verify each executing message
		blockValid := true
		for logIdx, execMsg := range execMsgs {
			err := i.verifyExecutingMessage(chainID, blockRef.Time, logIdx, execMsg, view)
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
			invalid, err := i.newInvalidHead(chainID, expectedBlock)
			if err != nil {
				return Result{}, fmt.Errorf("chain %s: %w", chainID, err)
			}
			result.InvalidHeads[chainID] = invalid
		}
	}

	return result, nil
}

// verifyExecutingMessage verifies a single executing message by checking:
//  1. The initiating message exists in the source chain's database
//  2. The initiating message's timestamp is not greater than the executing block's timestamp
//  3. The initiating message hasn't expired (timestamp + messageExpiryWindow >= executing timestamp)
//  4. Neither the executing block nor the initiating block falls in its chain's interop
//     activation block (interop must be active for at least one full block on both sides)
func (i *Interop) verifyExecutingMessage(executingChain eth.ChainID, executingTimestamp uint64, logIdx uint32, execMsg *messages.ExecutingMessage, view *frontierVerificationView) error {
	// Get the source chain's logsDB
	sourceDB, ok := i.logsDBs[execMsg.ChainID]
	if !ok {
		return fmt.Errorf("source chain %s not found: %w", execMsg.ChainID, ErrUnknownChain)
	}

	// Activation invariant: interop must be active for at least one full block on
	// both the executing chain and the initiating chain. Matches kona.
	execChain, ok := i.chains[executingChain]
	if !ok {
		return fmt.Errorf("executing chain %s not registered: %w", executingChain, ErrUnknownChain)
	}
	if executingTimestamp < i.activationTimestamp+execChain.BlockTime() {
		return fmt.Errorf("executing chain %s timestamp %d < activation %d + blockTime: %w",
			executingChain, executingTimestamp, i.activationTimestamp, ErrExecutedTooEarly)
	}

	initChain, ok := i.chains[execMsg.ChainID]
	if !ok {
		return fmt.Errorf("initiating chain %s not registered: %w", execMsg.ChainID, ErrUnknownChain)
	}
	if execMsg.Timestamp < i.activationTimestamp+initChain.BlockTime() {
		return fmt.Errorf("initiating chain %s timestamp %d < activation %d + blockTime: %w",
			execMsg.ChainID, execMsg.Timestamp, i.activationTimestamp, ErrInitiatedTooEarly)
	}

	// Verify timestamp ordering: initiating message timestamp must be <= executing block timestamp.
	if execMsg.Timestamp > executingTimestamp {
		return fmt.Errorf("initiating timestamp %d > executing timestamp %d: %w",
			execMsg.Timestamp, executingTimestamp, ErrTimestampViolation)
	}

	// Verify the message hasn't expired: initiating timestamp + messageExpiryWindow must be >= executing timestamp
	if execMsg.Timestamp+i.messageExpiryWindow < executingTimestamp {
		return fmt.Errorf("initiating timestamp %d + expiry %d < executing timestamp %d: %w",
			execMsg.Timestamp, i.messageExpiryWindow, executingTimestamp, ErrMessageExpired)
	}

	// Build the query for the initiating message
	query := messages.ContainsQuery{
		BlockNum:  execMsg.BlockNum,
		LogIdx:    execMsg.LogIdx,
		Timestamp: execMsg.Timestamp,
		Checksum:  execMsg.Checksum,
	}

	// Same-timestamp dependencies may live in the current frontier view rather
	// than accepted-history logsDB.
	if execMsg.Timestamp == executingTimestamp {
		if _, ok := view.contains(execMsg.ChainID, query); ok {
			return nil
		}
	}

	// Check if the initiating message exists in the source chain's logsDB
	_, err := sourceDB.Contains(query)
	return err
}
