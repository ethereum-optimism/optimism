package status

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L1Tracker implements the L1Fetcher interface while proactively maintaining a reorg-aware cache
// of L1 block references by number. Populate the cache with the latest L1 block references.
type L1Tracker struct {
	derive.L1Fetcher
	cache *l1HeadBuffer
}

func NewL1Tracker(inner derive.L1Fetcher) *L1Tracker {
	return &L1Tracker{
		L1Fetcher: inner,
		cache:     newL1HeadBuffer(1000),
	}
}

func (st *L1Tracker) OnL1Unsafe(l1Unsafe eth.BlockRef) {
	st.cache.Insert(l1Unsafe)
}

func (l *L1Tracker) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	if ref, ok := l.cache.Get(num); ok {
		return ref, nil
	}

	return l.L1Fetcher.L1BlockRefByNumber(ctx, num)
}
