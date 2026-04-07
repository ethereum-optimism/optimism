package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// DerivedBlockSteps returns floor(offsetDerived / blockTime) L2 block steps to walk back from the EL-sync tip.
// blockTime is rollup L2 block time in seconds. A zero or negative offset yields 0; zero block time yields 0.
func DerivedBlockSteps(offsetDerived time.Duration, blockTime uint64) uint64 {
	if offsetDerived <= 0 || blockTime == 0 {
		return 0
	}
	sec := uint64(offsetDerived / time.Second)
	return sec / blockTime
}

// L2AncestorByN walks parent links from tip, at most n times, clamping at genesisL2 (inclusive).
func L2AncestorByN(ctx context.Context, l2 L2Chain, genesisL2 eth.BlockID, tip eth.L2BlockRef, n uint64) (eth.L2BlockRef, error) {
	cur := tip
	for i := uint64(0); i < n; i++ {
		if cur.Number <= genesisL2.Number {
			return cur, nil
		}
		parent, err := l2.L2BlockRefByHash(ctx, cur.ParentHash)
		if err != nil {
			return eth.L2BlockRef{}, fmt.Errorf("parent of L2 block %s: %w", cur, err)
		}
		cur = parent
	}
	return cur, nil
}
