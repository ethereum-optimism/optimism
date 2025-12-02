package match

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum/go-ethereum/common"
)

// WithArchive matches the first archive node. This matcher makes two assumptions:
//
//  1. Non-archive nodes only store the last N blocks worth of historical balances, while archive
//     nodes store all of them.
//  2. The chain has at least N+1 blocks.
//
// Either assumption being false could result in false positives or false negatives. Note that
// there is also a race condition where assumption (2) becomes true after the function returns.
func WithArchive(ctx context.Context) stack.Matcher[stack.L2ELNodeID, stack.L2ELNode] {
	return MatchElemFn[stack.L2ELNodeID, stack.L2ELNode](func(elem stack.L2ELNode) bool {
		if _, err := elem.L2EthClient().BlockRefByNumber(ctx, 1); err != nil {
			// The devnet is fresh. This is almost guaranteed to be a devnet created by sysgo,
			// which always uses archive mode.
			return true
		}
		// Use block 1 since EL clients may keep the genesis block when not in archive mode.
		_, err := elem.L2EthClient().BalanceAt(ctx, common.Address{}, big.NewInt(1))
		return err == nil
	})
}
