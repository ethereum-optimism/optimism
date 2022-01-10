package eth

import (
	"context"
	"fmt"
	"math/big"
)

// BlockLinkByNumber Retrieves the *currently* canonical block-hash at the given block-height, and the parent before it.
// The results of this should not be cached, or the cache needs to be reorg-aware.
type BlockLinkByNumber interface {
	BlockLinkByNumber(ctx context.Context, num uint64) (self BlockID, parent BlockID, err error)
}

type BlockLinkByNumberFn func(ctx context.Context, num uint64) (self BlockID, parent BlockID, err error)

func (fn BlockLinkByNumberFn) BlockLinkByNumber(ctx context.Context, num uint64) (self BlockID, parent BlockID, err error) {
	return fn(ctx, num)
}

// CanonicalChain presents the block-hashes by height by wrapping a header-source
// (useful due to lack of a direct JSON RPC endpoint)
func CanonicalChain(l1Src HeaderByNumberSource) BlockLinkByNumber {
	return BlockLinkByNumberFn(func(ctx context.Context, num uint64) (BlockID, BlockID, error) {
		header, err := l1Src.HeaderByNumber(ctx, big.NewInt(int64(num)))
		if err != nil {
			// w%: wrap the error, we still need to detect if a canonical block is not found, a.k.a. end of chain.
			return BlockID{}, BlockID{}, fmt.Errorf("failed to determine block-hash of height %d, could not get header: %w", num, err)
		}
		parentNum := num
		if parentNum > 0 {
			parentNum -= 1
		}
		return BlockID{Hash: header.Hash(), Number: num}, BlockID{Hash: header.ParentHash, Number: parentNum}, nil
	})
}
