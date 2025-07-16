package syncnode

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/binary"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
)

// resetTracker manages a bisection between consistent and inconsistent blocks
// and is used to prepare a reset request to be handled by a managed node.
type resetTracker struct {
	a eth.BlockID
	z eth.BlockID

	log     log.Logger
	backend resetBackend
}

type resetBackend interface {
	BlockIDByNumber(ctx context.Context, n uint64) (eth.BlockID, error)
	IsLocalSafe(ctx context.Context, block eth.BlockID) error

	L2BlockRefByNumber(ctx context.Context, n uint64) (eth.L2BlockRef, error)
	L1BlockIDByNumber(ctx context.Context, n uint64) (eth.BlockID, error)
	LocalUnsafe(ctx context.Context) (eth.BlockID, error)
}

// init initializes the reset tracker with
// empty start and end of range, and no reset in progress
func newResetTracker(logger log.Logger, b resetBackend) *resetTracker {
	return &resetTracker{
		log:     logger,
		backend: b,
	}
}

type resetTarget struct {
	Target     eth.BlockID
	PreInterop bool
}

// FindResetTarget initializes the reset tracker
// and starts the bisection process at the given block
// which will lead to a reset request
func (t *resetTracker) FindResetTarget(ctx context.Context, a, z eth.BlockID) (resetTarget, error) {
	t.log.Info("beginning reset", "a", a, "z", z)
	t.a = a
	t.z = z

	nodeCtx, nCancel := context.WithTimeout(ctx, nodeTimeout)
	defer nCancel()

	// before starting bisection, check if z is already consistent (i.e. the node is ahead but otherwise consistent)
	nodeZ, err := t.backend.BlockIDByNumber(nodeCtx, t.z.Number)
	// if z is already consistent, we can skip the bisection
	// and move straight to a targeted reset
	if err == nil && nodeZ == t.z {
		return resetTarget{Target: t.z}, nil
	}

	// before starting bisection, check if a is inconsistent (i.e. the node has no common reference point)
	// if the first block in the range can't be found or is inconsistent, we initiate a pre-Interop reset
	nodeA, err := t.backend.BlockIDByNumber(nodeCtx, t.a.Number)
	if errors.Is(err, ethereum.NotFound) {
		t.log.Debug("start of range is not known to node, returning pre-Interop reset target", "a", t.a)
		return resetTarget{PreInterop: true}, nil
	} else if err != nil {
		return resetTarget{}, fmt.Errorf("failed to query start block: %w", err)
	} else if nodeA != t.a {
		t.log.Debug("start of range mismatch between node and supervisor, returning pre-Interop reset target", "a", t.a)
		return resetTarget{PreInterop: true}, nil
	}

	// repeatedly bisect the range until the last consistent block is found
	for {
		// covers both cases where a+1 == z and a == z
		if t.a.Number+1 >= t.z.Number {
			t.log.Debug("reset target converged. Resetting to start of range", "a", t.a, "z", t.z)
			return resetTarget{Target: t.a}, nil
		}
		err := t.bisect(ctx)
		if err != nil {
			return resetTarget{}, fmt.Errorf("failed to bisect range [%s, %s]: %w", t.a, t.z, err)
		}
	}
}

// bisect halves the search range of the ongoing reset to narrow down
// where the reset will target. It bisects the range and constrains either
// the start or the end of the range, based on the consistency of the midpoint
// with the logs db.
func (t *resetTracker) bisect(ctx context.Context) error {
	internalCtx, iCancel := context.WithTimeout(ctx, internalTimeout)
	defer iCancel()
	nodeCtx, nCancel := context.WithTimeout(ctx, nodeTimeout)
	defer nCancel()

	// attempt to get the block at the midpoint of the range
	i := (t.a.Number + t.z.Number) / 2
	nodeI, err := t.backend.BlockIDByNumber(nodeCtx, i)

	// if the block is not known to the node, it is defacto inconsistent
	if errors.Is(err, ethereum.NotFound) {
		t.log.Debug("midpoint of range is not known to node. pulling back end of range", "i", i)
		t.z = eth.BlockID{Number: i}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to query midpoint block number %d: %w", i, err)
	}

	// Check if the block at i is consistent with the local-safe DB,
	if err = t.backend.IsLocalSafe(internalCtx, nodeI); errors.Is(err, types.ErrFuture) || errors.Is(err, types.ErrConflict) {
		// TODO: do we need to add more sentinel errors here?
		// TODO(#16026): could gracefully exit on block-replacement (no need to reset what is already being built replacement for)
		t.log.Debug("midpoint of range is inconsistent. pulling back end of range", "i", i)
		t.z = nodeI
	} else if err != nil {
		return fmt.Errorf("failed to check if midpoint %d is local safe: %w", i, err)
	} else {
		t.log.Debug("midpoint of range is consistent. pushing up start of range", "i", i)
		t.a = nodeI
	}
	return nil
}

// isL1OriginValid compares l1 origin info from the node and compares with l1 block fetched from l1 accessor db
func (t *resetTracker) isL1OriginValid(ctx context.Context, blockNum uint64) (eth.L2BlockRef, error) {
	current, err := t.backend.L2BlockRefByNumber(ctx, blockNum)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	// Check if L1Origin has been reorged
	l1Blk, err := t.backend.L1BlockIDByNumber(ctx, current.L1Origin.Number)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	if l1Blk.Hash != current.L1Origin.Hash {
		t.log.Debug("L1Origin field is invalid/outdated, so block is invalid and should be reorged", "currentNumber", current.Number, "currentL1Origin", current, "newL1Origin", l1Blk)
		return eth.L2BlockRef{}, nil
	}
	t.log.Trace("L1Origin field points to canonical L1 block, so block is valid", "blocknum", blockNum, "l1Blk", l1Blk)
	return current, nil
}

// FindResetUnsafeHeadTarget searches and returns the latest valid unsafe block of the L2 chain
// starting from lSafe and checking until the latest unsafe block.
func (t *resetTracker) FindResetUnsafeHeadTarget(ctx context.Context, lSafe eth.BlockID) (eth.BlockID, error) {
	latestlUnsafe, err := t.backend.LocalUnsafe(ctx)
	if err != nil {
		t.log.Error("failed to get last local unsafe block. cancelling reset", "err", err)
		return eth.BlockID{}, nil
	}
	t.log.Info("Searching for latest valid local unsafe", "latestlUnsafe", latestlUnsafe, "lSafe", lSafe)

	target := lSafe.Number
	targetDiff := int(latestlUnsafe.Number - target)
	if targetDiff > 0 {
		// Binary search to find and return the last valid block for idx in [0, targetDiff)
		// We don't check validity of `target`, `target` is not in the search space, it is checked
		// in the walkback loop section below if necessary.

		// Search space:
		// ------------------------------------------------------------------------------------------
		// target.Number |  idx=0      idx=1      idx=2     ...  idx = targetDiff-1 = latestUnsafe   |
		// false         |  t/f        t/f        t/f       ...  t/f                                 |
		// ------------------------------------------------------------------------------------------
		idx, valid, err := binary.SearchL(targetDiff, func(i int) (bool, eth.L2BlockRef, error) {
			block, err := t.isL1OriginValid(ctx, target+1+uint64(i))
			return block != (eth.L2BlockRef{}), block, err
		})
		if err != nil {
			return eth.BlockID{}, err
		}
		if idx != -1 {
			t.log.Info("Found last valid block with binary search", "valid", valid)
			return valid.ID(), nil
		} else {
			t.log.Info("All blocks checked by binary search are invalid between target and latestUnsafe")
		}
	} else if targetDiff < 0 {
		t.log.Warn("Latest unsafe block is older than target, using latest unsafe for search")
		target = latestlUnsafe.Number
	}

	// In the following walkback loop, the following two cases are covered:
	// 1. targetDiff == 0 or targetDiff < 0 (i.e. target == latestUnsafe), or
	// 2. all blocks checked by binary search were invalid, so we have to go from `target` backwards indefinitely
	//    until we find a valid block
	for n := target; ; n-- {
		if n == target-1 {
			t.log.Warn("No valid unsafe block found up to target, searching further")
		}
		valid, err := t.isL1OriginValid(ctx, n)
		if err != nil {
			return eth.BlockID{}, err
		}
		if valid != (eth.L2BlockRef{}) {
			t.log.Info("Found last valid block", "valid", valid)
			return valid.ID(), nil
		}
	}
}
