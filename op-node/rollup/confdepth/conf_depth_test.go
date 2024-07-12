package confdepth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type confTest struct {
	name      string
	head      uint64
	req       uint64
	depth     uint64
	pass      bool
	fromCache bool
}

func mockL1BlockRef(num uint64) eth.L1BlockRef {
	return eth.L1BlockRef{Number: num, Hash: common.Hash{byte(num)}, ParentHash: common.Hash{byte(num - 1)}}
}

func (ct *confTest) Run(t *testing.T) {
	l1Fetcher := &testutils.MockL1Source{}
	var l1Head eth.L1BlockRef
	if ct.head != 0 {
		l1Head = mockL1BlockRef(ct.head)
	}
	l1HeadGetter := func() eth.L1BlockRef { return l1Head }

	cd := NewConfDepth(ct.depth, l1HeadGetter, l1Fetcher)
	if ct.pass && !ct.fromCache {
		// no calls to the l1Fetcher are made if the confirmation depth of the request is not met
		l1Fetcher.ExpectL1BlockRefByNumber(ct.req, mockL1BlockRef(ct.req), nil)
	}
	out, err := cd.L1BlockRefByNumber(context.Background(), ct.req)
	l1Fetcher.AssertExpectations(t)
	if ct.pass {
		require.NoError(t, err)
		require.Equal(t, out, mockL1BlockRef(ct.req))
	} else {
		require.Equal(t, ethereum.NotFound, err)
	}
}

func TestConfDepth(t *testing.T) {
	// note: we're not testing overflows.
	// If a request is large enough to overflow the conf depth check, it's not returning anything anyway.
	testCases := []confTest{
		{name: "zero conf future", head: 4, req: 5, depth: 0, pass: true},
		{name: "zero conf present", head: 4, req: 4, depth: 0, pass: true, fromCache: true},
		{name: "zero conf past", head: 4, req: 3, depth: 0, pass: true},
		{name: "one conf future", head: 4, req: 5, depth: 1, pass: false},
		{name: "one conf present", head: 4, req: 4, depth: 1, pass: false},
		{name: "one conf past", head: 4, req: 3, depth: 1, pass: true},
		{name: "two conf future", head: 4, req: 5, depth: 2, pass: false},
		{name: "two conf present", head: 4, req: 4, depth: 2, pass: false},
		{name: "two conf not like 1", head: 4, req: 3, depth: 2, pass: false},
		{name: "two conf pass", head: 4, req: 2, depth: 2, pass: true},
		{name: "easy pass", head: 100, req: 20, depth: 5, pass: true},
		{name: "genesis case", head: 0, req: 0, depth: 4, pass: true},
		{name: "no L1 state", req: 10, depth: 4, pass: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.Run)
	}
}

func TestConfDepthCachingReorgs(t *testing.T) {
	l1Fetcher := &testutils.MockL1Source{}

	l1Head := mockL1BlockRef(100)
	l1HeadGetter := func() eth.L1BlockRef { return l1Head }

	cd := NewConfDepth(0, l1HeadGetter, l1Fetcher)

	l1Fetcher.ExpectL1BlockRefByNumber(99, mockL1BlockRef(99), nil)
	out, err := cd.L1BlockRefByNumber(context.Background(), 99)
	require.NoError(t, err)
	require.Equal(t, out, mockL1BlockRef(99))
	l1Fetcher.AssertExpectations(t)

	// from cache
	out, err = cd.L1BlockRefByNumber(context.Background(), 100)
	require.NoError(t, err)
	require.Equal(t, out, mockL1BlockRef(100))

	l1Head = mockL1BlockRef(101)

	// from cache
	out, err = cd.L1BlockRefByNumber(context.Background(), 101)
	require.NoError(t, err)
	require.Equal(t, out, mockL1BlockRef(101))

	l1Head = mockL1BlockRef(102)

	// from cache
	out, err = cd.L1BlockRefByNumber(context.Background(), 102)
	require.NoError(t, err)
	require.Equal(t, out, mockL1BlockRef(102))

	// trigger a reorg of block 101, invalidating the following cache elements
	l1Head = mockL1BlockRef(101)
	l1Head.Hash = common.Hash{0xde, 0xad, 0xbe, 0xef}

	l1Fetcher.ExpectL1BlockRefByNumber(102, mockL1BlockRef(102), nil)
	out, err = cd.L1BlockRefByNumber(context.Background(), 102)
	require.NoError(t, err)
	require.Equal(t, out, mockL1BlockRef(102))
	l1Fetcher.AssertExpectations(t)

	// block 100 is still in the cache
	out, err = cd.L1BlockRefByNumber(context.Background(), 100)
	require.NoError(t, err)
	require.Equal(t, out, mockL1BlockRef(100))

	// head jumps ahead, invalidating the entire cache
	l1Head = mockL1BlockRef(200)

	l1Fetcher.ExpectL1BlockRefByNumber(100, mockL1BlockRef(100), nil)
	out, err = cd.L1BlockRefByNumber(context.Background(), 100)
	require.NoError(t, err)
	require.Equal(t, out, mockL1BlockRef(100))
	l1Fetcher.AssertExpectations(t)
}

func TestConfDepthCachingEviction(t *testing.T) {
	l1Fetcher := &testutils.MockL1Source{}

	l1Head := mockL1BlockRef(100)
	l1HeadGetter := func() eth.L1BlockRef { return l1Head }

	cd := NewConfDepth(0, l1HeadGetter, l1Fetcher)

	// insert 1001 elements into the cache, resulting in the oldest entry being evicted
	for idx := 1000; idx <= 2000; idx++ {
		l1Head = mockL1BlockRef(uint64(idx))

		// request current head from cache
		out, err := cd.L1BlockRefByNumber(context.Background(), uint64(idx))
		require.NoError(t, err)
		require.Equal(t, out, mockL1BlockRef(uint64(idx)))
	}

	l1Fetcher.ExpectL1BlockRefByNumber(1000, mockL1BlockRef(1000), nil)
	out, err := cd.L1BlockRefByNumber(context.Background(), 1000)
	require.NoError(t, err)
	require.Equal(t, out, mockL1BlockRef(1000))
	l1Fetcher.AssertExpectations(t)

	for idx := 1001; idx <= 2000; idx++ {
		// these elements are still in the cache
		out, err = cd.L1BlockRefByNumber(context.Background(), uint64(idx))
		require.NoError(t, err)
		require.Equal(t, out, mockL1BlockRef(uint64(idx)))
	}
}
