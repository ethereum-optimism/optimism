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
// The confirmations are counted from the L1 block with the configured label (latest by default).
// At 0 depth from the latest block, the l1 head is completely ignored.
type confDepth struct {
	// everything fetched by hash is trusted already, so we implement those by embedding the fetcher
	derive.L1Fetcher
	l1Ref func() eth.L1BlockRef
	label eth.BlockLabel
	depth uint64
}

func NewConfDepth(depth uint64, l1Head func() eth.L1BlockRef, fetcher derive.L1Fetcher) *confDepth {
	return NewConfDepthFromLabel(depth, eth.Unsafe, l1Head, fetcher)
}

// NewConfDepthFromLabel counts the confirmation depth from the L1 block with the given label,
// as returned by l1Ref. An empty label is treated as latest.
func NewConfDepthFromLabel(depth uint64, label eth.BlockLabel, l1Ref func() eth.L1BlockRef, fetcher derive.L1Fetcher) *confDepth {
	if label == "" {
		label = eth.Unsafe
	}
	return &confDepth{L1Fetcher: fetcher, l1Ref: l1Ref, label: label, depth: depth}
}

// L1BlockRefByNumber is used for L1 traversal and for finding a safe common point between the L2 engine and L1 chain.
// Any block numbers that are within confirmation depth of the labeled L1 block are mocked to be "not found",
// effectively hiding the uncertain part of the L1 chain.
func (c *confDepth) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	fromHead := c.label == eth.Unsafe
	l1Ref := c.l1Ref()
	if l1Ref == (eth.L1BlockRef{}) {
		// Don't apply the conf depth if l1Head is empty (as it is during the startup case before the l1State is initialized).
		if fromHead {
			return c.L1Fetcher.L1BlockRefByNumber(ctx, num)
		}
		// The safe and finalized blocks are polled less often than the head, so they may not be tracked yet.
		// Fetch the labeled block instead of lifting the confirmation limit.
		var err error
		l1Ref, err = c.L1Fetcher.L1BlockRefByLabel(ctx, c.label)
		if err != nil {
			return eth.L1BlockRef{}, err
		}
	}
	if num == 0 || (fromHead && c.depth == 0) || num+c.depth <= l1Ref.Number {
		return c.L1Fetcher.L1BlockRefByNumber(ctx, num)
	}
	return eth.L1BlockRef{}, ethereum.NotFound
}

var _ derive.L1Fetcher = (*confDepth)(nil)
