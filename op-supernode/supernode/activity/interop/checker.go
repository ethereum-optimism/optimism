package interop

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// l1ByNumberSource provides L1 block lookups by number for consistency checking.
type l1ByNumberSource interface {
	L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error)
}

// l1ConsistencyChecker verifies that a set of L1 block IDs all belong to
// the same L1 fork.
type l1ConsistencyChecker interface {
	SameL1Chain(ctx context.Context, heads []eth.BlockID) (bool, error)
}

// l1ByNumberChecker is the production l1ConsistencyChecker: it checks each
// head against the canonical L1 chain fetched from an l1ByNumberSource.
type l1ByNumberChecker struct {
	l1 l1ByNumberSource
}

func newL1ConsistencyChecker(l1 l1ByNumberSource) l1ConsistencyChecker {
	if l1 == nil {
		return nil
	}
	return &l1ByNumberChecker{l1: l1}
}

// SameL1Chain returns true if all non-zero heads belong to the same canonical L1 chain.
func (c *l1ByNumberChecker) SameL1Chain(ctx context.Context, heads []eth.BlockID) (bool, error) {
	for _, head := range heads {
		if head == (eth.BlockID{}) {
			continue
		}
		canonical, err := c.l1.L1BlockRefByNumber(ctx, head.Number)
		if err != nil {
			return false, err
		}
		if canonical.Hash != head.Hash {
			return false, nil
		}
	}
	return true, nil
}
