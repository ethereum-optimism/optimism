package engine_controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrRewindTargetBlockNotFound        = errors.New("failed to get target block at timestamp")
	ErrRewindComputeTargetsFailed       = errors.New("failed to compute rewind targets")
	ErrRewindInsertSyntheticFailed      = errors.New("failed to insert synthetic payload")
	ErrRewindSyntheticPayloadRejected   = errors.New("synthetic payload rejected by engine")
	ErrRewindFCUSyntheticFailed         = errors.New("failed to FCU to synthetic block")
	ErrRewindFCUTargetFailed            = errors.New("failed to FCU to target block")
	ErrRewindVerificationFailed         = errors.New("rewind state verification failed")
	ErrRewindFCURejected                = errors.New("forkchoice update rejected by engine")
	ErrRewindTimestampToBlockConversion = errors.New("failed to convert timestamp to block number")
	ErrRewindPayloadNotFound            = errors.New("failed to get payload for block")
	ErrRewindOverFinalizedHead          = errors.New("cannot rewind over finalized head")
)

// RewindToTimestamp rewinds the L2 execution layer to the block at or before the given timestamp.
//
// The rewind is performed in two steps:
//  1. Insert a synthetic block (modified fee recipient) and FCU to it, which triggers a reorg
//     that orphans all blocks after the target.
//  2. FCU back to the original target block, completing the rewind.
//
// TODO: in future, we could push the implementation into the engine itself which would reduce the
// number of RPC calls required and remove the need for the synthetic block to be inserted.
func (e *simpleEngineController) RewindToTimestamp(ctx context.Context, timestamp uint64) error {
	if e.l2 == nil {
		return ErrNoEngineClient
	}

	// Step 0: infer the target block:
	// [n-1,parent] <-- [n,target] <-- [m>n,unsafe]
	targetBlock, err := e.blockAtTimestamp(ctx, timestamp)
	if err != nil {
		return fmt.Errorf("%w %d: %w", ErrRewindTargetBlockNotFound, timestamp, err)
	}

	// Step 1: Insert a synthetic block (modified fee recipient) which
	// is built on the parent of the target block:
	// [n-1,parent] <-- [n,target] <--...<-- [m>n,unsafe]
	//
	//                 [n,synthetic]
	syntheticBlockHash, err := e.insertSyntheticPayload(ctx, targetBlock.Number)
	if err != nil {
		return err
	}

	// Step 2: compute rewind targets for safe and finalized heads, ensuring they do not go forwards:
	targetSafeBlock, targetFinalizedBlock, err := e.computeRewindTargets(ctx, targetBlock)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRewindComputeTargetsFailed, err)
	}

	// Step 3: FCU to the synthetic block to trigger a reorg, removing the target block
	// from the canonical chain.
	// We use the parent hash of the target block as the safe and finalized block
	// in the FCU since these are guaranteed to be in the canonical chain of the synthetic block.
	// [n-1,parent]   [n,target]
	//      |\
	//       \_______ [n,synthetic,unsafe]
	parentHash := targetBlock.ParentHash
	if err := e.forkchoiceUpdate(ctx, syntheticBlockHash, parentHash, parentHash); err != nil {
		return fmt.Errorf("%w: %w", ErrRewindFCUSyntheticFailed, err)
	}
	e.log.Info("executed FCU to synthetic block", "syntheticHead", syntheticBlockHash, "safe", parentHash, "finalized", parentHash)

	// Step 4: FCU to the actual target block
	// [n-1,parent] <-- [n,target, unsafe]
	//
	//                  [n,synthetic]
	if err := e.forkchoiceUpdate(ctx, targetBlock.Hash, targetSafeBlock.Hash, targetFinalizedBlock.Hash); err != nil {
		return fmt.Errorf("%w: %w", ErrRewindFCUTargetFailed, err)
	}
	e.log.Info("executed FCU to target block", "head", targetBlock.Hash, "safe", targetSafeBlock.Hash, "finalized", targetFinalizedBlock.Hash)

	// Step 5: Verify the rewind state
	if err := e.verifyRewindState(ctx, targetBlock, targetSafeBlock, targetFinalizedBlock); err != nil {
		return fmt.Errorf("%w: %w", ErrRewindVerificationFailed, err)
	}
	return nil
}

// computeRewindTargets determines the safe and finalized block targets for the rewind.
// Safe and finalized are clamped to not move forward (only backward or stay the same).
func (e *simpleEngineController) computeRewindTargets(ctx context.Context, targetBlock eth.L2BlockRef) (safe, finalized eth.L2BlockRef, err error) {
	currentSafe, err := e.l2.L2BlockRefByLabel(ctx, eth.Safe)
	if err != nil {
		return eth.L2BlockRef{}, eth.L2BlockRef{}, fmt.Errorf("failed to get current safe block: %w", err)
	}

	currentFinalized, err := e.l2.L2BlockRefByLabel(ctx, eth.Finalized)
	if err != nil {
		return eth.L2BlockRef{}, eth.L2BlockRef{}, fmt.Errorf("failed to get current finalized block: %w", err)
	}

	if targetBlock.Number < currentFinalized.Number {
		return eth.L2BlockRef{}, eth.L2BlockRef{}, ErrRewindOverFinalizedHead
	}

	return earliest(currentSafe, targetBlock), earliest(currentFinalized, targetBlock), nil
}

// insertSyntheticPayload creates and inserts a synthetic block derived from the block at the given number.
// The synthetic block has a modified fee recipient to produce a different block hash.
// Returns the hash of the synthetic block.
func (e *simpleEngineController) insertSyntheticPayload(ctx context.Context, blockNumber uint64) (common.Hash, error) {
	envelope, err := e.l2.PayloadByNumber(ctx, blockNumber)
	if err != nil || envelope == nil || envelope.ExecutionPayload == nil {
		return common.Hash{}, fmt.Errorf("failed to get payload for block %d: %w, err: %w", blockNumber, ErrRewindPayloadNotFound, err)
	}

	// Deep clone the envelope and payload
	newEnvelope := *envelope
	newPayload := *(envelope.ExecutionPayload)
	newEnvelope.ExecutionPayload = &newPayload

	// Modify the cloned payload to create a synthetic block with a different hash
	newPayload.FeeRecipient = common.MaxAddress
	syntheticHash, _ := newEnvelope.CheckBlockHash() // ignore "ok" since we know it won't match
	newPayload.BlockHash = syntheticHash

	e.log.Info("inserting synthetic payload", "blockNumber", blockNumber, "parentHash", newPayload.ParentHash, "syntheticHash", syntheticHash)
	status, err := e.l2.NewPayload(ctx, &newPayload, envelope.ParentBeaconBlockRoot)
	if err != nil {
		return common.Hash{}, fmt.Errorf("%w: %w", ErrRewindInsertSyntheticFailed, err)
	}
	if status.Status != eth.ExecutionValid {
		validationErr := ""
		if status.ValidationError != nil {
			validationErr = *status.ValidationError
		}
		return common.Hash{}, fmt.Errorf("%w: status=%s validationError=%q blockNumber=%d parentHash=%s syntheticHash=%s",
			ErrRewindSyntheticPayloadRejected, status.Status, validationErr, blockNumber, newPayload.ParentHash, syntheticHash)
	}

	return syntheticHash, nil
}

// verifyRewindState checks that the engine state matches the expected targets after a rewind.
func (e *simpleEngineController) verifyRewindState(ctx context.Context, targetUnsafe, targetSafe, targetFinalized eth.L2BlockRef) error {
	unsafe, err := e.l2.L2BlockRefByLabel(ctx, eth.Unsafe)
	if err != nil {
		return fmt.Errorf("failed to verify unsafe block: %w", err)
	}
	if unsafe.Number != targetUnsafe.Number {
		return fmt.Errorf("unexpected unsafe block number: got %d, want %d", unsafe.Number, targetUnsafe.Number)
	}
	if unsafe.Hash != targetUnsafe.Hash {
		return fmt.Errorf("unexpected unsafe block hash: got %s, want %s", unsafe.Hash, targetUnsafe.Hash)
	}

	safe, err := e.l2.L2BlockRefByLabel(ctx, eth.Safe)
	if err != nil {
		return fmt.Errorf("failed to verify safe block: %w", err)
	}
	if safe.Number != targetSafe.Number {
		return fmt.Errorf("unexpected safe block number: got %d, want %d", safe.Number, targetSafe.Number)
	}
	if safe.Hash != targetSafe.Hash {
		return fmt.Errorf("unexpected safe block hash: got %s, want %s", safe.Hash, targetSafe.Hash)
	}

	finalized, err := e.l2.L2BlockRefByLabel(ctx, eth.Finalized)
	if err != nil {
		return fmt.Errorf("failed to verify finalized block: %w", err)
	}
	if finalized.Number != targetFinalized.Number {
		return fmt.Errorf("unexpected finalized block number: got %d, want %d", finalized.Number, targetFinalized.Number)
	}
	if finalized.Hash != targetFinalized.Hash {
		return fmt.Errorf("unexpected finalized block hash: got %s, want %s", finalized.Hash, targetFinalized.Hash)
	}

	return nil
}

// forkchoiceUpdate sends a forkchoice update to the engine and validates the response.
func (e *simpleEngineController) forkchoiceUpdate(ctx context.Context, head, safe, finalized common.Hash) error {
	fcs := eth.ForkchoiceState{
		HeadBlockHash:      head,
		SafeBlockHash:      safe,
		FinalizedBlockHash: finalized,
	}
	res, err := e.l2.ForkchoiceUpdate(ctx, &fcs, nil)
	if err != nil {
		return err
	}
	if res.PayloadStatus.Status != eth.ExecutionValid {
		validationErr := ""
		if res.PayloadStatus.ValidationError != nil {
			validationErr = *res.PayloadStatus.ValidationError
		}
		return fmt.Errorf("%w: status=%s validationError=%q head=%s safe=%s finalized=%s",
			ErrRewindFCURejected, res.PayloadStatus.Status, validationErr, head, safe, finalized)
	}
	return nil
}

func earliest(a, b eth.L2BlockRef) eth.L2BlockRef {
	if a.Number < b.Number {
		return a
	}
	return b
}
