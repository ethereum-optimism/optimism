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
