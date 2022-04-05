package sync

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type fakeChainSource struct {
	L1 []eth.L1BlockRef
	L2 map[common.Hash]eth.L2BlockRef
}

func (m *fakeChainSource) L1HeadBlockRef(ctx context.Context) (eth.L1BlockRef, error) {
	return m.L1[len(m.L1)-1], nil
}

func (m *fakeChainSource) L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error) {
	n := int(number)
	if n >= len(m.L1) {
		return eth.L1BlockRef{}, ethereum.NotFound
	}
	return m.L1[n], nil
}

func (m *fakeChainSource) L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error) {
	ref, ok := m.L2[l2Hash]
	if !ok {
		return eth.L2BlockRef{}, ethereum.NotFound
	}
	return ref, nil
}

var _ L1Chain = (*fakeChainSource)(nil)
var _ L2Chain = (*fakeChainSource)(nil)

func fakeID(id rune, num uint64) eth.BlockID {
	var h common.Hash
	copy(h[:], string(id))
	return eth.BlockID{Hash: h, Number: uint64(num)}
}

func fakeL1Block(self rune, parent rune, num uint64) eth.L1BlockRef {
	var parentID eth.BlockID
	if num != 0 {
		parentID = fakeID(parent, num-1)
	}
	id := fakeID(self, num)
	return eth.L1BlockRef{Hash: id.Hash, Number: id.Number, ParentHash: parentID.Hash}
}

func fakeL2Block(self rune, parent rune, l1parent eth.BlockID, num uint64) eth.L2BlockRef {
	var parentID eth.BlockID
	if num != 0 {
		parentID = fakeID(parent, num-1)
	}
	id := fakeID(self, num)
	return eth.L2BlockRef{Hash: id.Hash, Number: id.Number, ParentHash: parentID.Hash, L1Origin: l1parent}
}

func (c *syncStartTestCase) generateFakeL2() (*fakeChainSource, eth.L2BlockRef, rollup.Genesis) {
	var l1 []eth.L1BlockRef
	var newl1 []eth.L1BlockRef
	var prevID rune
	for i, id := range c.L1 {
		l1 = append(l1, fakeL1Block(id, prevID, uint64(i)))
		// fmt.Printf("%v\t%v\n", l1[i].Self, l1[i].Parent)
		prevID = id
	}
	prevID = rune(0)
	for i, id := range c.NewL1 {
		newl1 = append(newl1, fakeL1Block(id, prevID, uint64(i)))
		// fmt.Printf("%v\t%v\n", newl1[i].Self, newl1[i].Parent)
		prevID = id
	}

	prevID = rune(0)
	var head eth.L2BlockRef
	m := make(map[common.Hash]eth.L2BlockRef)
	for i, id := range c.L2 {
		b := fakeL2Block(id, prevID, l1[i+int(c.GenesisL1Num)].ID(), uint64(i)+c.GenesisL2Num)
		m[b.Hash] = b
		// fmt.Printf("%v\t%v\t%v\n", b.Self, b.Parent, b.L1Origin)
		prevID = id
		head = b
	}
	genesis := rollup.Genesis{
		L1: fakeID(c.GenesisL1, c.GenesisL1Num),
		L2: fakeID(c.GenesisL2, c.GenesisL2Num),
	}
	return &fakeChainSource{L1: newl1, L2: m}, head, genesis

}

type syncStartTestCase struct {
	Name string

	L1        string // L1 Chain prior to a re-org or other change
	L2        string // L2 Chain that follows from L1Chain
	NewL1     string // New L1 chain
	ReorgBase rune   // Highest L1 block in the pre and post re-org L1 chian

	GenesisL1    rune
	GenesisL1Num uint64
	GenesisL2    rune
	GenesisL2Num uint64

	SeqWindowSize uint64
	SafeL2Head    rune
	UnsafeL2Head  rune
	ExpectedErr   error
}

func refToRune(r eth.BlockID) rune {
	return rune(r.Hash.Bytes()[0])
}

func (c *syncStartTestCase) Run(t *testing.T) {
	msr, l2Head, genesis := c.generateFakeL2()

	unsafeL2Head, safeHead, err := FindL2Heads(context.TODO(), l2Head, c.SeqWindowSize, msr, msr, &genesis)

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
