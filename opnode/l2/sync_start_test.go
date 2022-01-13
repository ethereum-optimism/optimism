package l2

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

type l2Block struct {
	Self   eth.BlockID
	FromL1 eth.BlockID
}

type mockSyncReference struct {
	L2 []l2Block
	L1 []eth.BlockID
}

func (m *mockSyncReference) RefByL1Num(ctx context.Context, l1Num uint64) (self eth.BlockID, parent eth.BlockID, err error) {
	if l1Num >= uint64(len(m.L1)) {
		err = ethereum.NotFound
		return
	}
	self = m.L1[l1Num]
	if l1Num > 0 {
		parent = m.L1[l1Num-1]
	}
	return
}

func (m *mockSyncReference) RefByL2Num(ctx context.Context, l2Num *big.Int, genesis *Genesis) (refL1 eth.BlockID, refL2 eth.BlockID, parentL2 common.Hash, err error) {
	if len(m.L2) == 0 {
		panic("bad test, no l2 chain")
	}
	i := uint64(len(m.L2) - 1)
	if l2Num != nil {
		i = l2Num.Uint64()
	}
	head := m.L2[i]
	refL1 = head.FromL1
	refL2 = head.Self
	if i > 0 {
		parentL2 = m.L2[i-1].Self.Hash
	}
	return
}

func (m *mockSyncReference) RefByL2Hash(ctx context.Context, l2Hash common.Hash, genesis *Genesis) (refL1 eth.BlockID, refL2 eth.BlockID, parentL2 common.Hash, err error) {
	for i, bl := range m.L2 {
		if bl.Self.Hash == l2Hash {
			return m.RefByL2Num(ctx, big.NewInt(int64(i)), genesis)
		}
	}
	err = ethereum.NotFound
	return
}

var _ SyncReference = (*mockSyncReference)(nil)

func mockID(id rune, num uint64) eth.BlockID {
	var h common.Hash
	copy(h[:], string(id))
	return eth.BlockID{Hash: h, Number: uint64(num)}
}

func chainL1(ids string) (out []eth.BlockID) {
	for i, id := range ids {
		out = append(out, mockID(id, uint64(i)))
	}
	return
}

func chainL2(l1 []eth.BlockID, ids string) (out []l2Block) {
	for i, id := range ids {
		out = append(out, l2Block{
			Self:   mockID(id, uint64(i)),
			FromL1: l1[i],
		})
	}
	return
}

type syncStartTestCase struct {
	Name string

	OffsetL2 uint64
	EngineL1 string
	EngineL2 string
	ActualL1 string

	GenesisL2 rune
	GenesisL1 rune

	ExpectedNextRefL1 rune
	ExpectedRefL2     rune

	ExpectedErr error
}

func (c *syncStartTestCase) Run(t *testing.T) {
	engL1 := chainL1(c.EngineL1)
	engL2 := chainL2(engL1[c.OffsetL2:], c.EngineL2)
	actL1 := chainL1(c.ActualL1)

	msr := &mockSyncReference{
		L2: engL2,
		L1: actL1,
	}

	genesis := &Genesis{
		L1: mockID(c.GenesisL1, c.OffsetL2),
		L2: mockID(c.GenesisL2, 0),
	}

	expectedNextRefL1Num := ^uint64(0)
	for i, id := range c.ActualL1 {
		if id == c.ExpectedNextRefL1 {
			expectedNextRefL1Num = uint64(i)
		}
	}
	if c.ExpectedNextRefL1 == 0 {
		expectedNextRefL1Num = uint64(0)
	}
	expectedNextRefL1 := mockID(c.ExpectedNextRefL1, expectedNextRefL1Num)

	expectedRefL2Num := ^uint64(0)
	for i, id := range c.EngineL2 {
		if id == c.ExpectedRefL2 {
			expectedRefL2Num = uint64(i)
		}
	}
	if c.ExpectedRefL2 == 0 {
		expectedRefL2Num = uint64(0)
	}
	expectedRefL2 := mockID(c.ExpectedRefL2, expectedRefL2Num)

	nextRefL1, refL2, err := FindSyncStart(context.Background(), msr, genesis)
	if c.ExpectedErr != nil {
		assert.Error(t, err, "got next L1 %s (%d), onto L2: %s (%d)", nextRefL1.Hash[:1], nextRefL1.Number, refL2.Hash[:1], refL2.Number)
		assert.ErrorIs(t, err, c.ExpectedErr)
	} else {
		assert.NoError(t, err)
		assert.Equal(t, expectedNextRefL1, nextRefL1, "expected %s (nr %d) but got %s (nr %d)", expectedNextRefL1.Hash[:1], expectedNextRefL1.Number, nextRefL1.Hash[:1], nextRefL1.Number)
		assert.Equal(t, expectedRefL2, refL2, "expected %s (nr %d) but got %s (nr %d)", expectedRefL2.Hash[:1], expectedRefL2.Number, refL2.Hash[:1], refL2.Number)
	}
}

func TestFindSyncStart(t *testing.T) {
	testCases := []syncStartTestCase{
		{
			Name:              "happy extend",
			OffsetL2:          0,
			EngineL1:          "ab",
			EngineL2:          "AB",
			ActualL1:          "abc",
			GenesisL1:         'a',
			GenesisL2:         'A',
			ExpectedNextRefL1: 'c',
			ExpectedRefL2:     'B',
			ExpectedErr:       nil,
		},
		{
			Name:              "extend one at a time",
			OffsetL2:          0,
			EngineL1:          "ab",
			EngineL2:          "AB",
			ActualL1:          "abcdef",
			GenesisL1:         'a',
			GenesisL2:         'A',
			ExpectedNextRefL1: 'c',
			ExpectedRefL2:     'B',
			ExpectedErr:       nil,
		},
		{
			Name:              "already synced",
			OffsetL2:          0,
			EngineL1:          "abcde",
			EngineL2:          "ABCDE",
			ActualL1:          "abcde",
			GenesisL1:         'a',
			GenesisL2:         'A',
			ExpectedNextRefL1: 'e',
			ExpectedRefL2:     'D',
			ExpectedErr:       nil,
		},
		{
			Name:              "genesis",
			OffsetL2:          0,
			EngineL1:          "a",
			EngineL2:          "A",
			ActualL1:          "a",
			GenesisL1:         'a',
			GenesisL2:         'A',
			ExpectedNextRefL1: 'a',
			ExpectedRefL2:     0, // actual zero hash before genesis hash
			ExpectedErr:       nil,
		},
		{
			Name:              "reorg two steps back",
			OffsetL2:          0,
			EngineL1:          "abc",
			EngineL2:          "ABC",
			ActualL1:          "axy",
			GenesisL1:         'a',
			GenesisL2:         'A',
			ExpectedNextRefL1: 'x',
			ExpectedRefL2:     'A',
			ExpectedErr:       nil,
		},
		{
			Name:              "Orphan block",
			OffsetL2:          0,
			EngineL1:          "abcd",
			EngineL2:          "ABCD",
			ActualL1:          "abcx",
			GenesisL1:         'a',
			GenesisL2:         'A',
			ExpectedNextRefL1: 'x',
			ExpectedRefL2:     'C',
			ExpectedErr:       nil,
		},
		{
			Name:              "L2 chain ahead",
			OffsetL2:          0,
			EngineL1:          "abcdef",
			EngineL2:          "ABCDEF",
			ActualL1:          "abc",
			GenesisL1:         'a',
			GenesisL2:         'A',
			ExpectedNextRefL1: 0,
			ExpectedRefL2:     0,
			ExpectedErr:       ethereum.NotFound,
		},
		{
			Name:              "L2 chain ahead reorg",
			OffsetL2:          0,
			EngineL1:          "abcdef",
			EngineL2:          "ABCDEF",
			ActualL1:          "abcx",
			GenesisL1:         'a',
			GenesisL2:         'A',
			ExpectedNextRefL1: 'x',
			ExpectedRefL2:     'C',
			ExpectedErr:       nil,
		},
		{
			Name:              "unexpected L1 chain",
			OffsetL2:          0,
			EngineL1:          "abcdef",
			EngineL2:          "ABCDEF",
			ActualL1:          "xyz",
			GenesisL1:         'a',
			GenesisL2:         'A',
			ExpectedNextRefL1: 0,
			ExpectedRefL2:     0,
			ExpectedErr:       WrongChainErr,
		},
		{
			Name:              "unexpected L2 chain",
			OffsetL2:          0,
			EngineL1:          "abcdef",
			EngineL2:          "ABCDEF",
			ActualL1:          "xyz",
			GenesisL1:         'a',
			GenesisL2:         'X',
			ExpectedNextRefL1: 0,
			ExpectedRefL2:     0,
			ExpectedErr:       WrongChainErr,
		},
		// TODO more test cases
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, testCase.Run)
	}
}
