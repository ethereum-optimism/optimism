package sync

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var _ L1Chain = (*testutils.FakeChainSource)(nil)
var _ L2Chain = (*testutils.FakeChainSource)(nil)

// generateFakeL2 creates a fake L2 chain with the following conditions:
// - The L2 chain is based off of the L1 chain
// - The actual L1 chain is the New L1 chain
// - Both heads are at the tip of their respective chains
func (c *syncStartTestCase) generateFakeL2(t *testing.T) (*testutils.FakeChainSource, eth.L2BlockRef, rollup.Genesis) {
	log := testlog.Logger(t, log.LvlError)
	chain := testutils.NewFakeChainSource([]string{c.L1, c.NewL1}, []string{c.L2}, int(c.GenesisL1Num), log)
	chain.SetL2Head(len(c.L2) - 1)
	genesis := testutils.FakeGenesis(c.GenesisL1, c.GenesisL2, int(c.GenesisL1Num))
	head, err := chain.L2BlockRefByNumber(context.Background(), nil)
	require.Nil(t, err)
	chain.ReorgL1()
	for i := 0; i < len(c.NewL1)-1; i++ {
		chain.AdvanceL1()
	}
	return chain, head, genesis

}

type syncStartTestCase struct {
	Name string

	L1    string // L1 Chain prior to a re-org or other change
	L2    string // L2 Chain that follows from L1Chain
	NewL1 string // New L1 chain

	GenesisL1    rune
	GenesisL1Num uint64
	GenesisL2    rune

	SeqWindowSize uint64
	SafeL2Head    rune
	UnsafeL2Head  rune
	ExpectedErr   error
}

func refToRune(r eth.BlockID) rune {
	return rune(r.Hash.Bytes()[0])
}

func (c *syncStartTestCase) Run(t *testing.T) {
	chain, l2Head, genesis := c.generateFakeL2(t)

	unsafeL2Head, safeHead, err := FindL2Heads(context.Background(), l2Head, c.SeqWindowSize, chain, chain, &genesis)

	if c.ExpectedErr != nil {
		require.Error(t, err, "Expecting an error in this test case")
		require.ErrorIs(t, c.ExpectedErr, err, "Unexpected error")
	} else {

		require.NoError(t, err)
		expectedUnsafeHead := refToRune(unsafeL2Head.ID())
		require.Equal(t, string(c.UnsafeL2Head), string(expectedUnsafeHead), "Unsafe L2 Head not equal")

		expectedSafeHead := refToRune(safeHead.ID())
		require.Equal(t, string(c.SafeL2Head), string(expectedSafeHead), "Safe L2 Head not equal")
	}
}

func TestFindSyncStart(t *testing.T) {
	testCases := []syncStartTestCase{
		{
			Name:          "already synced",
			GenesisL1Num:  0,
			L1:            "ab",
			L2:            "AB",
			NewL1:         "ab",
			GenesisL1:     'a',
			GenesisL2:     'A',
			UnsafeL2Head:  'B',
			SeqWindowSize: 2,
			SafeL2Head:    'A',
			ExpectedErr:   nil,
		},
		{
			Name:          "small reorg long chain",
			GenesisL1Num:  0,
			L1:            "abcdefgh",
			L2:            "ABCDEFGH",
			NewL1:         "abcdefgx",
			GenesisL1:     'a',
			GenesisL2:     'A',
			UnsafeL2Head:  'G',
			SeqWindowSize: 2,
			SafeL2Head:    'F',
			ExpectedErr:   nil,
		},
		{
			Name:          "L1 Chain ahead",
			GenesisL1Num:  0,
			L1:            "abcde",
			L2:            "ABCD",
			NewL1:         "abcde",
			GenesisL1:     'a',
			GenesisL2:     'A',
			UnsafeL2Head:  'D',
			SeqWindowSize: 3,
			SafeL2Head:    'B',
			ExpectedErr:   nil,
		},
		{
			Name:          "L2 Chain ahead after reorg",
			GenesisL1Num:  0,
			L1:            "abxyz",
			L2:            "ABXYZ",
			NewL1:         "abx",
			GenesisL1:     'a',
			GenesisL2:     'A',
			UnsafeL2Head:  'Z',
			SeqWindowSize: 2,
			SafeL2Head:    'B',
			ExpectedErr:   nil,
		},
		{
			Name:          "genesis",
			GenesisL1Num:  0,
			L1:            "a",
			L2:            "A",
			NewL1:         "a",
			GenesisL1:     'a',
			GenesisL2:     'A',
			UnsafeL2Head:  'A',
			SeqWindowSize: 2,
			SafeL2Head:    'A',
			ExpectedErr:   nil,
		},
		{
			Name:          "reorg one step back",
			GenesisL1Num:  0,
			L1:            "abcd",
			L2:            "ABCD",
			NewL1:         "abcx",
			GenesisL1:     'a',
			GenesisL2:     'A',
			UnsafeL2Head:  'C',
			SeqWindowSize: 3,
			SafeL2Head:    'A',
			ExpectedErr:   nil,
		},
		{
			Name:          "reorg two steps back",
			GenesisL1Num:  0,
			L1:            "abc",
			L2:            "ABC",
			NewL1:         "axy",
			GenesisL1:     'a',
			GenesisL2:     'A',
			UnsafeL2Head:  'A',
			SeqWindowSize: 2,
			SafeL2Head:    'A',
			ExpectedErr:   nil,
		},
		{
			Name:          "reorg three steps back",
			GenesisL1Num:  0,
			L1:            "abcdef",
			L2:            "ABCDEF",
			NewL1:         "abcxyz",
			GenesisL1:     'a',
			GenesisL2:     'A',
			UnsafeL2Head:  'C',
			SeqWindowSize: 2,
			SafeL2Head:    'B',
			ExpectedErr:   nil,
		},
		{
			Name:         "unexpected L1 chain",
			GenesisL1Num: 0,
			L1:           "abcdef",
			L2:           "ABCDEF",
			NewL1:        "xyzwio",
			GenesisL1:    'a',
			GenesisL2:    'A',
			UnsafeL2Head: 0,
			ExpectedErr:  WrongChainErr,
		},
		{
			Name:         "unexpected L2 chain",
			GenesisL1Num: 0,
			L1:           "abcdef",
			L2:           "ABCDEF",
			NewL1:        "xyzwio",
			GenesisL1:    'a',
			GenesisL2:    'X',
			UnsafeL2Head: 0,
			ExpectedErr:  WrongChainErr,
		},
		{
			Name:          "offset L2 genesis",
			GenesisL1Num:  3,
			L1:            "abcdef",
			L2:            "DEF",
			NewL1:         "abcdef",
			GenesisL1:     'd',
			GenesisL2:     'D',
			UnsafeL2Head:  'F',
			SeqWindowSize: 2,
			SafeL2Head:    'E',
			ExpectedErr:   nil,
		},
		{
			Name:          "offset L2 genesis reorg",
			GenesisL1Num:  3,
			L1:            "abcdefgh",
			L2:            "DEFGH",
			NewL1:         "abcdxyzw",
			GenesisL1:     'd',
			GenesisL2:     'D',
			UnsafeL2Head:  'D',
			SeqWindowSize: 2,
			SafeL2Head:    'D',
			ExpectedErr:   nil,
		},
		{
			Name:         "reorg past offset genesis",
			GenesisL1Num: 3,
			L1:           "abcdefgh",
			L2:           "DEFGH",
			NewL1:        "abxyzwio",
			GenesisL1:    'd',
			GenesisL2:    'D',
			UnsafeL2Head: 0,
			ExpectedErr:  WrongChainErr,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, testCase.Run)
	}
}
