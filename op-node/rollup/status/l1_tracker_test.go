package status

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func mockL1BlockRef(num uint64) eth.L1BlockRef {
	return eth.L1BlockRef{Number: num, Hash: common.Hash{byte(num)}, ParentHash: common.Hash{byte(num - 1)}}
}

func newL1HeadEvent(l1Tracker *L1Tracker, head eth.L1BlockRef) {
	l1Tracker.OnEvent(L1UnsafeEvent{
		L1Unsafe: head,
	})
}

func TestCachingReorgs(t *testing.T) {
	ctx := context.Background()
	l1Fetcher := &testutils.MockL1Source{}
	l1Tracker := NewL1Tracker(l1Fetcher)

	// no blocks added to cache yet
	l1Head := mockL1BlockRef(99)
	l1Fetcher.ExpectL1BlockRefByNumber(99, l1Head, nil)
	ret, err := l1Tracker.L1BlockRefByNumber(ctx, 99)
	require.NoError(t, err)
	require.Equal(t, l1Head, ret)
	l1Fetcher.AssertExpectations(t)

	// from cache
	l1Head = mockL1BlockRef(100)
	newL1HeadEvent(l1Tracker, l1Head)
	ret, err = l1Tracker.L1BlockRefByNumber(ctx, 100)
	require.NoError(t, err)
	require.Equal(t, l1Head, ret)

	// from cache
	l1Head = mockL1BlockRef(101)
	newL1HeadEvent(l1Tracker, l1Head)
	ret, err = l1Tracker.L1BlockRefByNumber(ctx, 101)
	require.NoError(t, err)
	require.Equal(t, l1Head, ret)

	// from cache
	l1Head = mockL1BlockRef(102)
	newL1HeadEvent(l1Tracker, l1Head)
	ret, err = l1Tracker.L1BlockRefByNumber(ctx, 102)
	require.NoError(t, err)
	require.Equal(t, l1Head, ret)

	// trigger a reorg of block 101, invalidating the following cache elements (102)
	l1Head = mockL1BlockRef(101)
	l1Head.Hash = common.Hash{0xde, 0xad, 0xbe, 0xef}
	newL1HeadEvent(l1Tracker, l1Head)
	ret, err = l1Tracker.L1BlockRefByNumber(ctx, 101)
	require.NoError(t, err)
	require.Equal(t, l1Head, ret)

	// confirm that 102 has been removed
	l1Fetcher.ExpectL1BlockRefByNumber(102, mockL1BlockRef(102), nil)
	ret, err = l1Tracker.L1BlockRefByNumber(ctx, 102)
	require.NoError(t, err)
	require.Equal(t, mockL1BlockRef(102), ret)
	l1Fetcher.AssertExpectations(t)

	// append a new block 102 based on the new 101
	parentHash := l1Head.Hash
	l1Head = mockL1BlockRef(102)
	l1Head.ParentHash = parentHash
	newL1HeadEvent(l1Tracker, l1Head)
	ret, err = l1Tracker.L1BlockRefByNumber(ctx, 102)
	require.NoError(t, err)
	require.Equal(t, l1Head, ret)

	// receive a new 102 block built on the original 101, invalidating the entire cache
	l1Head = mockL1BlockRef(102)
	newL1HeadEvent(l1Tracker, l1Head)
	ret, err = l1Tracker.L1BlockRefByNumber(ctx, 102)
	require.NoError(t, err)
	require.Equal(t, l1Head, ret)

	// confirm that the cache contains no 101
	l1Fetcher.ExpectL1BlockRefByNumber(101, mockL1BlockRef(101), nil)
	ret, err = l1Tracker.L1BlockRefByNumber(ctx, 101)
	require.NoError(t, err)
	require.Equal(t, mockL1BlockRef(101), ret)
	l1Fetcher.AssertExpectations(t)

	// head jumps ahead from 102->104, invalidating the entire cache
	l1Head = mockL1BlockRef(104)
	newL1HeadEvent(l1Tracker, l1Head)
	ret, err = l1Tracker.L1BlockRefByNumber(ctx, 104)
	require.NoError(t, err)
	require.Equal(t, mockL1BlockRef(104), ret)
	l1Fetcher.AssertExpectations(t)

	// confirm that the cache contains no 102
	l1Fetcher.ExpectL1BlockRefByNumber(102, mockL1BlockRef(102), nil)
	ret, err = l1Tracker.L1BlockRefByNumber(ctx, 102)
	require.NoError(t, err)
	require.Equal(t, mockL1BlockRef(102), ret)
	l1Fetcher.AssertExpectations(t)
}

func TestCacheEviction(t *testing.T) {
	ctx := context.Background()
	l1Fetcher := &testutils.MockL1Source{}
	l1Tracker := NewL1Tracker(l1Fetcher)

	// insert 1000 elements into the cache
	for idx := 1000; idx < 2000; idx++ {
		l1Head := mockL1BlockRef(uint64(idx))
		newL1HeadEvent(l1Tracker, l1Head)
	}

	// request each element from cache
	for idx := 1000; idx < 2000; idx++ {
		ret, err := l1Tracker.L1BlockRefByNumber(ctx, uint64(idx))
		require.NoError(t, err)
		require.Equal(t, mockL1BlockRef(uint64(idx)), ret)
	}

	// insert 1001st element, removing the first
	l1Head := mockL1BlockRef(2000)
	newL1HeadEvent(l1Tracker, l1Head)

	// request first element, which now requires a live fetch instead
	l1Fetcher.ExpectL1BlockRefByNumber(1000, mockL1BlockRef(1000), nil)
	ret, err := l1Tracker.L1BlockRefByNumber(ctx, 1000)
	require.NoError(t, err)
	require.Equal(t, mockL1BlockRef(1000), ret)
}
