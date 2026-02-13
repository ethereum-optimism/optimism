package sync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var _ L1Chain = (*testutils.FakeChainSource)(nil)
var _ L2Chain = (*testutils.FakeChainSource)(nil)

// generateFakeL2 creates a fake L2 chain with the following conditions:
// - The L2 chain is based off of the L1 chain
// - The actual L1 chain is the New L1 chain
// - Both heads are at the tip of their respective chains
func (c *syncStartTestCase) generateFakeL2(t *testing.T) (*testutils.FakeChainSource, rollup.Genesis) {
	t.Helper()
	log := testlog.Logger(t, log.LevelError)
	chain := testutils.NewFakeChainSource([]string{c.L1, c.NewL1}, []string{c.L2}, int(c.GenesisL1Num), log)
	chain.SetL2Head(len(c.L2) - 1)
	genesis := testutils.FakeGenesis(c.GenesisL1, c.GenesisL2, c.GenesisL1Num)
	chain.ReorgL1()
	for i := 0; i < len(c.NewL1)-1; i++ {
		chain.AdvanceL1()
	}
	return chain, genesis

}

func runeToHash(id rune) common.Hash {
	var h common.Hash
	copy(h[:], string(id))
	return h
}

type syncStartTestCase struct {
	Name string

	L1    string // L1 Chain prior to a re-org or other change
	L2    string // L2 Chain that follows from L1Chain
	NewL1 string // New L1 chain

	PreFinalizedL2 rune
	PreSafeL2      rune

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
	chain, genesis := c.generateFakeL2(t)
	chain.SetL2Finalized(runeToHash(c.PreFinalizedL2))
	chain.SetL2Safe(runeToHash(c.PreSafeL2))

	cfg := &rollup.Config{
		Genesis:       genesis,
		SeqWindowSize: c.SeqWindowSize,
	}
	lgr := log.NewLogger(log.DiscardHandler())
	result, err := FindL2Heads(context.Background(), cfg, chain, chain, lgr, &Config{})
	if c.ExpectedErr != nil {
		require.ErrorIs(t, err, c.ExpectedErr, "expected error")
		return
	} else {
		require.NoError(t, err, "expected no error")
	}

	gotUnsafeHead := refToRune(result.Unsafe.ID())
	require.Equal(t, string(c.UnsafeL2Head), string(gotUnsafeHead), "Unsafe L2 Head not equal")

	gotSafeHead := refToRune(result.Safe.ID())
	require.Equal(t, string(c.SafeL2Head), string(gotSafeHead), "Safe L2 Head not equal")
}

func TestFindSyncStart(t *testing.T) {
	testCases := []syncStartTestCase{
		{
			Name:           "already synced",
			GenesisL1Num:   0,
			L1:             "ab",
			L2:             "AB",
			NewL1:          "ab",
			PreFinalizedL2: 'A',
			PreSafeL2:      'A',
			GenesisL1:      'a',
			GenesisL2:      'A',
			UnsafeL2Head:   'B',
			SeqWindowSize:  2,
			SafeL2Head:     'A',
			ExpectedErr:    nil,
		},
		{
			Name:           "already synced with safe head after genesis",
			GenesisL1Num:   0,
			L1:             "abcdefghijkj",
			L2:             "ABCDEFGHIJKJ",
			NewL1:          "abcdefghijkj",
			PreFinalizedL2: 'B',
			PreSafeL2:      'D',
			GenesisL1:      'a',
			GenesisL2:      'A',
			UnsafeL2Head:   'J',
			SeqWindowSize:  2,
			// Important this steps back at least one safe block so the safedb is sent the latest safe head
			// again - we may be resetting because the safedb failed to write the previous entry
			SafeL2Head:  'C',
			ExpectedErr: nil,
		},
		{
			Name:           "small reorg long chain",
			GenesisL1Num:   0,
			L1:             "abcdefgh",
			L2:             "ABCDEFGH",
			NewL1:          "abcdefgx",
			PreFinalizedL2: 'B',
			PreSafeL2:      'H',
			GenesisL1:      'a',
			GenesisL2:      'A',
			UnsafeL2Head:   'G',
			SeqWindowSize:  2,
			SafeL2Head:     'C',
			ExpectedErr:    nil,
		},
		{
			Name:           "L1 Chain ahead",
			GenesisL1Num:   0,
			L1:             "abcdef",
			L2:             "ABCDE",
			NewL1:          "abcdef",
			PreFinalizedL2: 'A',
			PreSafeL2:      'D',
			GenesisL1:      'a',
			GenesisL2:      'A',
			UnsafeL2Head:   'E',
			SeqWindowSize:  2,
			SafeL2Head:     'A',
			ExpectedErr:    nil,
		},
		{
			Name:           "L2 Chain ahead after reorg",
			GenesisL1Num:   0,
			L1:             "abcxyz",
			L2:             "ABCXYZ",
			NewL1:          "abcx",
			PreFinalizedL2: 'B',
			PreSafeL2:      'X',
			GenesisL1:      'a',
			GenesisL2:      'A',
			UnsafeL2Head:   'Z',
			SeqWindowSize:  2,
			SafeL2Head:     'B',
			ExpectedErr:    nil,
		},
		{
			Name:           "genesis",
			GenesisL1Num:   0,
			L1:             "a",
			L2:             "A",
			NewL1:          "a",
			PreFinalizedL2: 'A',
			PreSafeL2:      'A',
			GenesisL1:      'a',
			GenesisL2:      'A',
			UnsafeL2Head:   'A',
			SeqWindowSize:  2,
			SafeL2Head:     'A',
			ExpectedErr:    nil,
		},
		{
			Name:           "reorg one step back",
			GenesisL1Num:   0,
			L1:             "abcdefg",
			L2:             "ABCDEFG",
			NewL1:          "abcdefx",
			PreFinalizedL2: 'A',
			PreSafeL2:      'E',
			GenesisL1:      'a',
			GenesisL2:      'A',
			UnsafeL2Head:   'F',
			SeqWindowSize:  3,
			SafeL2Head:     'A',
			ExpectedErr:    nil,
		},
		{
			Name:           "reorg two steps back, clip genesis and finalized",
			GenesisL1Num:   0,
			L1:             "abc",
			L2:             "ABC",
			PreFinalizedL2: 'A',
			PreSafeL2:      'B',
			NewL1:          "axy",
			GenesisL1:      'a',
			GenesisL2:      'A',
			UnsafeL2Head:   'A',
			SeqWindowSize:  2,
			SafeL2Head:     'A',
			ExpectedErr:    nil,
		},
		{
			Name:           "reorg three steps back",
			GenesisL1Num:   0,
			L1:             "abcdefgh",
			L2:             "ABCDEFGH",
			NewL1:          "abcdexyz",
			PreFinalizedL2: 'A',
			PreSafeL2:      'D',
			GenesisL1:      'a',
			GenesisL2:      'A',
			UnsafeL2Head:   'E',
			SeqWindowSize:  2,
			SafeL2Head:     'A',
			ExpectedErr:    nil,
		},
		{
			Name:           "unexpected L1 chain",
			GenesisL1Num:   0,
			L1:             "abcdef",
			L2:             "ABCDEF",
			NewL1:          "xyzwio",
			PreFinalizedL2: 'A',
			PreSafeL2:      'B',
			GenesisL1:      'a',
			GenesisL2:      'A',
			UnsafeL2Head:   0,
			SeqWindowSize:  2,
			ExpectedErr:    WrongChainErr,
		},
		{
			Name:           "unexpected L2 chain",
			GenesisL1Num:   0,
			L1:             "abcdef",
			L2:             "ABCDEF",
			NewL1:          "xyzwio",
			PreFinalizedL2: 'A',
			PreSafeL2:      'B',
			GenesisL1:      'a',
			GenesisL2:      'X',
			UnsafeL2Head:   0,
			SeqWindowSize:  2,
			ExpectedErr:    WrongChainErr,
		},
		{
			Name:           "offset L2 genesis",
			GenesisL1Num:   3,
			L1:             "abcdefghi",
			L2:             "DEFGHI",
			NewL1:          "abcdefghi",
			PreFinalizedL2: 'E',
			PreSafeL2:      'H',
			GenesisL1:      'd',
			GenesisL2:      'D',
			UnsafeL2Head:   'I',
			SeqWindowSize:  2,
			SafeL2Head:     'E',
			ExpectedErr:    nil,
		},
		{
			Name:           "offset L2 genesis reorg",
			GenesisL1Num:   3,
			L1:             "abcdefgh",
			L2:             "DEFGH",
			NewL1:          "abcdxyzw",
			PreFinalizedL2: 'D',
			PreSafeL2:      'D',
			GenesisL1:      'd',
			GenesisL2:      'D',
			UnsafeL2Head:   'D',
			SeqWindowSize:  2,
			SafeL2Head:     'D',
			ExpectedErr:    nil,
		},
		{
			Name:           "reorg past offset genesis",
			GenesisL1Num:   3,
			L1:             "abcdefgh",
			L2:             "DEFGH",
			NewL1:          "abxyzwio",
			PreFinalizedL2: 'D',
			PreSafeL2:      'D',
			GenesisL1:      'd',
			GenesisL2:      'D',
			UnsafeL2Head:   0,
			SeqWindowSize:  2,
			SafeL2Head:     'D',
			ExpectedErr:    WrongChainErr,
		},
		{
			// FindL2Heads() keeps walking back to safe head after finding canonical unsafe head
			// TooDeepReorgErr must not be raised
			Name:           "long traverse to safe head",
			GenesisL1Num:   0,
			L1:             "abcdefgh",
			L2:             "ABCDEFGH",
			NewL1:          "abcdefgx",
			PreFinalizedL2: 'B',
			PreSafeL2:      'B',
			GenesisL1:      'a',
			GenesisL2:      'A',
			UnsafeL2Head:   'G',
			SeqWindowSize:  1,
			SafeL2Head:     'B',
			ExpectedErr:    nil,
		},
		{
			// L2 reorg is too deep
			Name:           "reorg too deep",
			GenesisL1Num:   0,
			L1:             "abcdefgh",
			L2:             "ABCDEFGH",
			NewL1:          "abijklmn",
			PreFinalizedL2: 'B',
			PreSafeL2:      'B',
			GenesisL1:      'a',
			GenesisL2:      'A',
			SeqWindowSize:  1,
			ExpectedErr:    TooDeepReorgErr,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, testCase.Run)
	}
}
