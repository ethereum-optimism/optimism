package syncnode

import (
	"context"
	"errors"
	"fmt"

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
