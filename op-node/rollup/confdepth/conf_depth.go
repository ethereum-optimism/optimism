package confdepth

import (
	"context"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// confDepth is an util that wraps the L1 input fetcher used in the pipeline,
// and hides the part of the L1 chain with insufficient confirmations.
//
// At 0 depth the l1 head is completely ignored.
//
// confDepth also caches the L1 head block references to avoid unnecessary fetches,
// by storing a sequence of up to 1000 blocks with a valid chain of parent hashes.
type confDepth struct {
	// everything fetched by hash is trusted already, so we implement those by embedding the fetcher
	derive.L1Fetcher
	l1Head           func() eth.L1BlockRef
	depth            uint64
	l1Cache          map[uint64]eth.L1BlockRef
	oldestCacheEntry uint64
	newestCacheEntry uint64
}

func NewConfDepth(depth uint64, l1Head func() eth.L1BlockRef, fetcher derive.L1Fetcher) *confDepth {
	return &confDepth{L1Fetcher: fetcher, l1Head: l1Head, depth: depth, l1Cache: make(map[uint64]eth.L1BlockRef)}
}

// L1BlockRefByNumber is used for L1 traversal and for finding a safe common point between the L2 engine and L1 chain.
// Any block numbers that are within confirmation depth of the L1 head are mocked to be "not found",
// effectively hiding the uncertain part of the L1 chain.
func (c *confDepth) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	// Don't apply the conf depth if l1Head is empty (as it is during the startup case before the l1State is initialized).
	l1Head := c.l1Head()
	if l1Head == (eth.L1BlockRef{}) {
		return c.L1Fetcher.L1BlockRefByNumber(ctx, num)
	}

	// First, check if the L1 head is in the cache.
	// If the hash doesn't match the one in the cache, we have a reorg and need to remove all entries after the new head.
	if ref, ok := c.l1Cache[l1Head.Number]; ok {
		if ref.Hash != l1Head.Hash {
			// Reorg detected, invalidate all entries after the new head.
			for idx := l1Head.Number + 1; idx <= c.newestCacheEntry; idx++ {
				delete(c.l1Cache, idx)
			}
			c.l1Cache[l1Head.Number] = l1Head
			c.newestCacheEntry = l1Head.Number
		}
	} else if ref, ok := c.l1Cache[l1Head.Number-1]; ok && ref.Hash == l1Head.ParentHash {
		// Parent hash matches, so we can safely add the new head to the cache.
		c.l1Cache[l1Head.Number] = l1Head
		c.newestCacheEntry = l1Head.Number
		if len(c.l1Cache) > 1000 {
			delete(c.l1Cache, c.oldestCacheEntry)
			c.oldestCacheEntry++
		}
	} else {
		// Parent not found or doesn't match, so invalidate the entire cache.
		c.l1Cache = make(map[uint64]eth.L1BlockRef)
		c.l1Cache[l1Head.Number] = l1Head
		c.oldestCacheEntry = l1Head.Number
		c.newestCacheEntry = l1Head.Number
	}

	if num == 0 || c.depth == 0 || num+c.depth <= l1Head.Number {
		// Attempt to retrieve from the cache first, falling back to a live fetch
		if ref, ok := c.l1Cache[num]; ok {
			return ref, nil
		}

		return c.L1Fetcher.L1BlockRefByNumber(ctx, num)
	}
	return eth.L1BlockRef{}, ethereum.NotFound
}

var _ derive.L1Fetcher = (*confDepth)(nil)
