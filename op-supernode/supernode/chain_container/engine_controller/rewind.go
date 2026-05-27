package engine_controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// fcuRetryDelay is the delay between FCU retry attempts when the head has not yet
	// converged to the expected value. This gives the execution layer time to flush
	// internal caches between forkchoice updates.
	fcuRetryDelay = 500 * time.Millisecond
	// maxFCUAttempts is the maximum number of times to retry an FCU before giving up.
	maxFCUAttempts = 20
)

var (
	ErrRewindNilTarget                  = errors.New("rewind requires a non-nil target payload")
	ErrRewindTargetMismatch             = errors.New("rewind target payload inconsistent with rollup config")
	ErrRewindComputeTargetsFailed       = errors.New("failed to compute rewind targets")
	ErrRewindInsertSyntheticFailed      = errors.New("failed to insert synthetic payload")
	ErrRewindSyntheticPayloadRejected   = errors.New("synthetic payload rejected by engine")
	ErrRewindReinsertCanonicalFailed    = errors.New("failed to re-insert canonical payload")
	ErrRewindCanonicalPayloadRejected   = errors.New("canonical payload rejected by engine on re-insert")
	ErrRewindFCUSyntheticFailed         = errors.New("failed to FCU to synthetic block")
	ErrRewindFCUTargetFailed            = errors.New("failed to FCU to target block")
	ErrRewindFCURejected                = errors.New("forkchoice update rejected by engine")
	ErrRewindTimestampToBlockConversion = errors.New("failed to convert timestamp to block number")
	ErrRewindOverFinalizedHead          = errors.New("cannot rewind over finalized head")
	ErrRewindFCUHeadMismatch            = errors.New("FCU head did not converge to expected value")
	ErrRewindCurrentUnsafeFailed        = errors.New("failed to get current unsafe block")
)

// Rewind rewinds the L2 execution layer to the supplied target block.
//
// The target payload is authoritative — it must be loaded from durable storage (the supernode
// WAL operation record) by the caller. The engine controller does not consult the live EL to
// discover the target, because after a crash mid-rewind the EL's canonical chain at the target
// height may be a synthetic block from the previous attempt rather than the original.
//
// The rewind is performed by:
//  1. Inserting a synthetic block (modified extra data) sharing the target's parent and FCU-ing
//     to it. This makes the original target block non-canonical in the EL's view.
//  2. Re-inserting the supplied target payload via engine_newPayload. This guarantees the EL
//     has the block durably stored before we ask it to become the head — pruners (op-reth in
//     particular) may have removed the block while it sat non-canonical after step 1.
//  3. FCU-ing to the target block, which restores it as the canonical head.
//
// If the unsafe head already matches the target hash the call is a no-op.
//
// TODO: in future, we could push the implementation into the engine itself which would reduce
// the number of RPC calls required and remove the need for the synthetic block to be inserted.
func (e *simpleEngineController) Rewind(ctx context.Context, target *eth.ExecutionPayloadEnvelope) error {
	if e.l2 == nil {
		return ErrNoEngineClient
	}
	if target == nil || target.ExecutionPayload == nil {
		return ErrRewindNilTarget
	}

	payload := target.ExecutionPayload
	targetNumber := uint64(payload.BlockNumber)
	targetHash := payload.BlockHash

	if e.rollup != nil {
		expectedNumber, err := e.rollup.TargetBlockNumber(uint64(payload.Timestamp))
		if err != nil {
			return fmt.Errorf("%w: %w", ErrRewindTimestampToBlockConversion, err)
		}
		if expectedNumber != targetNumber {
			return fmt.Errorf("%w: payload number=%d timestamp=%d (rollup expects %d)",
				ErrRewindTargetMismatch, targetNumber, uint64(payload.Timestamp), expectedNumber)
		}
	}

	targetRef := eth.L2BlockRef{
		Hash:       targetHash,
		Number:     targetNumber,
		ParentHash: payload.ParentHash,
		Time:       uint64(payload.Timestamp),
	}

	unsafe, err := e.l2.L2BlockRefByLabel(ctx, eth.Unsafe)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRewindCurrentUnsafeFailed, err)
	}
	// No-op if the chain is already at or behind the target. The chain state may have moved
	// between when the WAL'd target was captured and now, so the unsafe head may legitimately
	// be below the recorded target.
	if unsafe.Hash == targetHash || unsafe.Number < targetNumber {
		e.log.Info("rewind skipped: chain already at or behind target",
			"unsafeHash", unsafe.Hash, "unsafeNumber", unsafe.Number,
			"targetHash", targetHash, "targetNumber", targetNumber)
		return nil
	}

	// Step 1: Insert a synthetic block (modified extra data) which is built on the parent of
	// the target block:
	// [n-1,parent] <-- [n,target] <--...<-- [m>n,unsafe]
	//
	//                 [n,synthetic]
	syntheticBlockHash, err := e.insertSyntheticPayload(ctx, target)
	if err != nil {
		return err
	}

	// Step 2: compute rewind targets for safe and finalized heads, ensuring they do not move
	// forward.
	targetSafeBlock, targetFinalizedBlock, err := e.computeRewindTargets(ctx, targetRef)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRewindComputeTargetsFailed, err)
	}

	// Step 3: FCU to the synthetic block to trigger a reorg, removing the target block from
	// the canonical chain. The parent of the target block is used as safe and finalized in
	// the FCU since it is guaranteed to be in the canonical chain of the synthetic block.
	// [n-1,parent]   [n,target]
	//      |\
	//       \_______ [n,synthetic,unsafe]
	parentHash := targetRef.ParentHash
	if err := e.forkchoiceUpdateWithRetry(ctx, syntheticBlockHash, parentHash, parentHash); err != nil {
		return fmt.Errorf("%w: %w", ErrRewindFCUSyntheticFailed, err)
	}
	e.log.Info("executed FCU to synthetic block", "syntheticHead", syntheticBlockHash, "safe", parentHash, "finalized", parentHash)

	// Step 4: re-insert the canonical target payload via engine_newPayload. The previous FCU
	// to the synthetic block made the target block non-canonical, which means the EL is free
	// to prune it (op-reth's pruner does so eagerly). Calling NewPayload again forces the EL
	// to re-import the block before we ask it to become the head in step 5.
	if err := e.reinsertCanonicalPayload(ctx, target); err != nil {
		return err
	}

	// Step 5: FCU to the actual target block.
	// [n-1,parent] <-- [n,target, unsafe]
	//
	//                  [n,synthetic]
	if err := e.forkchoiceUpdateWithRetry(ctx, targetHash, targetSafeBlock.Hash, targetFinalizedBlock.Hash); err != nil {
		return fmt.Errorf("%w: %w", ErrRewindFCUTargetFailed, err)
	}
	e.log.Info("executed FCU to target block", "head", targetHash, "safe", targetSafeBlock.Hash, "finalized", targetFinalizedBlock.Hash)

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

// insertSyntheticPayload derives a synthetic block from the supplied canonical envelope and
// submits it via engine_newPayload. The synthetic block shares the canonical block's parent
// but has modified ExtraData to produce a different block hash.
// Returns the hash of the synthetic block.
func (e *simpleEngineController) insertSyntheticPayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) (common.Hash, error) {
	// Deep clone the envelope and payload so we can mutate fields without affecting the
	// canonical envelope, which is reused by reinsertCanonicalPayload later in the rewind.
	newEnvelope := *envelope
	newPayload := *(envelope.ExecutionPayload)
	newEnvelope.ExecutionPayload = &newPayload

	// Modify ExtraData to produce a different block hash without affecting the state root.
	// We must only change header fields that are not accessible via EVM opcodes. Fields
	// that are EVM-accessible (e.g. coinbase, timestamp, prevrandao) influence execution
	// and would cause the recomputed state root to diverge from the one in the payload,
	// causing the engine to reject it. ExtraData has no EVM opcode and is safe to modify.
	extra := make([]byte, len(newPayload.ExtraData))
	copy(extra, newPayload.ExtraData)
	if len(extra) == 0 {
		extra = []byte{0x00}
	} else {
		extra[len(extra)-1] ^= 0xff
	}
	newPayload.ExtraData = extra
	syntheticHash, _ := newEnvelope.CheckBlockHash() // ignore "ok" since we know it won't match
	newPayload.BlockHash = syntheticHash

	e.log.Info("inserting synthetic payload",
		"blockNumber", uint64(newPayload.BlockNumber), "parentHash", newPayload.ParentHash, "syntheticHash", syntheticHash)
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
			ErrRewindSyntheticPayloadRejected, status.Status, validationErr, uint64(newPayload.BlockNumber), newPayload.ParentHash, syntheticHash)
	}

	return syntheticHash, nil
}

// reinsertCanonicalPayload re-submits the canonical target envelope via engine_newPayload.
// It exists to guarantee the EL has the canonical block durably stored after the synthetic
// FCU has made it non-canonical (and therefore potentially prune-eligible) but before we
// FCU back to it.
func (e *simpleEngineController) reinsertCanonicalPayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
	payload := envelope.ExecutionPayload
	e.log.Info("re-inserting canonical payload",
		"blockNumber", uint64(payload.BlockNumber), "hash", payload.BlockHash, "parentHash", payload.ParentHash)
	status, err := e.l2.NewPayload(ctx, payload, envelope.ParentBeaconBlockRoot)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRewindReinsertCanonicalFailed, err)
	}
	if status.Status != eth.ExecutionValid {
		validationErr := ""
		if status.ValidationError != nil {
			validationErr = *status.ValidationError
		}
		return fmt.Errorf("%w: status=%s validationError=%q blockNumber=%d hash=%s",
			ErrRewindCanonicalPayloadRejected, status.Status, validationErr, uint64(payload.BlockNumber), payload.BlockHash)
	}
	return nil
}

// verifyRewindState checks that the engine's unsafe, safe, and finalized heads match the
// expected block hashes. Hash equality is the authoritative check — if the hash matches,
// the block number is correct by definition.
func (e *simpleEngineController) verifyRewindState(ctx context.Context, targetUnsafe, targetSafe, targetFinalized common.Hash) error {
	unsafe, err := e.l2.L2BlockRefByLabel(ctx, eth.Unsafe)
	if err != nil {
		return fmt.Errorf("failed to verify unsafe block: %w", err)
	}
	if unsafe.Hash != targetUnsafe {
		return fmt.Errorf("unexpected unsafe block hash: got %s, want %s", unsafe.Hash, targetUnsafe)
	}

	safe, err := e.l2.L2BlockRefByLabel(ctx, eth.Safe)
	if err != nil {
		return fmt.Errorf("failed to verify safe block: %w", err)
	}
	if safe.Hash != targetSafe {
		return fmt.Errorf("unexpected safe block hash: got %s, want %s", safe.Hash, targetSafe)
	}

	finalized, err := e.l2.L2BlockRefByLabel(ctx, eth.Finalized)
	if err != nil {
		return fmt.Errorf("failed to verify finalized block: %w", err)
	}
	if finalized.Hash != targetFinalized {
		return fmt.Errorf("unexpected finalized block hash: got %s, want %s", finalized.Hash, targetFinalized)
	}

	return nil
}

// forkchoiceUpdateWithRetry sends a forkchoice update and then verifies that the engine state
// matches the expected values. If the state hasn't converged (e.g. due to an execution layer
// race condition like reth#23205), it sleeps and retries the FCU up to maxFCUAttempts.
// TODO(#19772): track whether this workaround is going to be permanent or temporary.
func (e *simpleEngineController) forkchoiceUpdateWithRetry(ctx context.Context, head, safe, finalized common.Hash) error {
	for attempt := 1; attempt <= maxFCUAttempts; attempt++ {
		if err := e.forkchoiceUpdate(ctx, head, safe, finalized); err != nil {
			return err
		}
		if err := e.verifyRewindState(ctx, head, safe, finalized); err == nil {
			return nil
		} else if attempt == maxFCUAttempts {
			return fmt.Errorf("%w after %d attempts: %w", ErrRewindFCUHeadMismatch, maxFCUAttempts, err)
		} else {
			e.log.Warn("FCU state not yet converged, retrying", "attempt", attempt, "expectedHead", head, "err", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(fcuRetryDelay):
			}
		}
	}
	return nil // unreachable
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
