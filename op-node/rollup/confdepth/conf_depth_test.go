package confdepth

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type confTest struct {
	name  string
	head  uint64
	req   uint64
	depth uint64
	pass  bool
}

func mockL1BlockRef(num uint64) eth.L1BlockRef {
	return eth.L1BlockRef{Number: num, Hash: common.Hash{byte(num)}}
}

func (ct *confTest) Run(t *testing.T) {
	l1Fetcher := &testutils.MockL1Source{}
	var l1Head eth.L1BlockRef
	if ct.head != 0 {
		l1Head = mockL1BlockRef(ct.head)
	}
	l1HeadGetter := func() eth.L1BlockRef { return l1Head }

	cd := NewConfDepth(ct.depth, l1HeadGetter, l1Fetcher)
	if ct.pass {
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
		{name: "zero conf present", head: 4, req: 4, depth: 0, pass: true},
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

type labeledConfTest struct {
	name    string
	label   eth.BlockLabel
	tracked uint64 // number of the tracked labeled block, 0 if not tracked yet
	fetched uint64 // number returned when the labeled block is fetched, 0 if no fetch is expected
	req     uint64
	depth   uint64
	pass    bool
}

func (ct *labeledConfTest) Run(t *testing.T) {
	l1Fetcher := &testutils.MockL1Source{}
	var tracked eth.L1BlockRef
	if ct.tracked != 0 {
		tracked = mockL1BlockRef(ct.tracked)
	}
	if ct.fetched != 0 {
		l1Fetcher.ExpectL1BlockRefByLabel(ct.label, mockL1BlockRef(ct.fetched), nil)
	}
	if ct.pass {
		l1Fetcher.ExpectL1BlockRefByNumber(ct.req, mockL1BlockRef(ct.req), nil)
	}

	cd := NewConfDepthFromLabel(ct.depth, ct.label, func() eth.L1BlockRef { return tracked }, l1Fetcher)
	out, err := cd.L1BlockRefByNumber(context.Background(), ct.req)
	l1Fetcher.AssertExpectations(t)
	if ct.pass {
		require.NoError(t, err)
		require.Equal(t, mockL1BlockRef(ct.req), out)
	} else {
		require.Equal(t, ethereum.NotFound, err)
	}
}

func TestConfDepthFromLabel(t *testing.T) {
	testCases := []labeledConfTest{
		{name: "safe zero conf at safe", label: eth.Safe, tracked: 10, req: 10, depth: 0, pass: true},
		{name: "safe zero conf past safe", label: eth.Safe, tracked: 10, req: 11, depth: 0, pass: false},
		{name: "safe one conf", label: eth.Safe, tracked: 10, req: 9, depth: 1, pass: true},
		{name: "safe one conf at safe", label: eth.Safe, tracked: 10, req: 10, depth: 1, pass: false},
		{name: "finalized two conf", label: eth.Finalized, tracked: 10, req: 8, depth: 2, pass: true},
		{name: "finalized two conf too new", label: eth.Finalized, tracked: 10, req: 9, depth: 2, pass: false},
		{name: "safe not tracked fetches", label: eth.Safe, fetched: 10, req: 9, depth: 1, pass: true},
		{name: "safe not tracked fetches and hides", label: eth.Safe, fetched: 10, req: 11, depth: 0, pass: false},
		{name: "safe genesis", label: eth.Safe, tracked: 10, req: 0, depth: 4, pass: true},
		{name: "empty label is latest", label: "", req: 10, depth: 4, pass: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.Run)
	}
}

func TestConfDepthFromLabelFetchError(t *testing.T) {
	l1Fetcher := &testutils.MockL1Source{}
	fetchErr := errors.New("rpc down")
	l1Fetcher.ExpectL1BlockRefByLabel(eth.Safe, eth.L1BlockRef{}, fetchErr)

	cd := NewConfDepthFromLabel(1, eth.Safe, func() eth.L1BlockRef { return eth.L1BlockRef{} }, l1Fetcher)
	_, err := cd.L1BlockRefByNumber(context.Background(), 5)
	l1Fetcher.AssertExpectations(t)
	require.ErrorIs(t, err, fetchErr)
}
