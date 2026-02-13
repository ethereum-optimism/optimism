package derive

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestL1TraversalNext tests that the `Next` function only returns
// a block reference once and then properly returns io.EOF afterwards
func TestL1TraversalNext(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	a := testutils.RandomBlockRef(rng)
	// Load up the initial state with a reset
	l1Cfg := eth.SystemConfig{
		BatcherAddr: testutils.RandomAddress(rng),
		Overhead:    [32]byte{42},
		Scalar:      [32]byte{69},
	}
	sysCfgAddr := testutils.RandomAddress(rng)
	cfg := &rollup.Config{
		Genesis:               rollup.Genesis{SystemConfig: l1Cfg},
		L1SystemConfigAddress: sysCfgAddr,
	}
	tr := NewL1Traversal(testlog.Logger(t, log.LevelError), cfg, nil)

	_ = tr.Reset(context.Background(), a, l1Cfg)

	// First call should always succeed
	ref, err := tr.NextL1Block(context.Background())
	require.Nil(t, err)
	require.Equal(t, a, ref)

	// Subsequent calls should return io.EOF
	ref, err = tr.NextL1Block(context.Background())
	require.Equal(t, eth.L1BlockRef{}, ref)
	require.Equal(t, io.EOF, err)

	ref, err = tr.NextL1Block(context.Background())
	require.Equal(t, eth.L1BlockRef{}, ref)
	require.Equal(t, io.EOF, err)
}

// TestL1TraversalAdvance tests that the `Advance` function properly
// handles different error cases and returns the expected block ref
// if there is no error.
func TestL1TraversalAdvance(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	a := testutils.RandomBlockRef(rng)
	b := testutils.NextRandomRef(rng, a)
	// x is at the same height as b but does not extend `a`
	x := testutils.RandomBlockRef(rng)
	x.Number = b.Number
	sysCfgAddr := testutils.RandomAddress(rng)

	tests := []struct {
		name         string
		startBlock   eth.L1BlockRef
		nextBlock    eth.L1BlockRef
		initialL1Cfg eth.SystemConfig
		l1Receipts   []*types.Receipt
		fetcherErr   error
		expectedErr  error
	}{
		{
			name:       "simple extension",
			startBlock: a,
			nextBlock:  b,
			initialL1Cfg: eth.SystemConfig{
				BatcherAddr: common.Address{11},
				Overhead:    [32]byte{22},
				Scalar:      [32]byte{33},
			},
			l1Receipts:  []*types.Receipt{},
			fetcherErr:  nil,
			expectedErr: nil,
		},
		{
			name:        "reorg",
			startBlock:  a,
			nextBlock:   x,
			fetcherErr:  nil,
			expectedErr: ErrReset,
		},
		{
			name:        "not found",
			startBlock:  a,
			nextBlock:   eth.L1BlockRef{},
			fetcherErr:  ethereum.NotFound,
			expectedErr: io.EOF,
		},
		{
			name:        "temporary error",
			startBlock:  a,
			nextBlock:   eth.L1BlockRef{},
			fetcherErr:  errors.New("interrupted connection"),
			expectedErr: ErrTemporary,
		},
		// TODO: add tests that cover the receipts to config data updates
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			src := &testutils.MockL1Source{}
			src.ExpectL1BlockRefByNumber(test.startBlock.Number+1, test.nextBlock, test.fetcherErr)
			info := &testutils.MockBlockInfo{
				InfoHash:       test.nextBlock.Hash,
				InfoParentHash: test.nextBlock.ParentHash,
				InfoNum:        test.nextBlock.Number,
				InfoTime:       test.nextBlock.Time,
				// TODO: don't need full L1 info in receipts fetching API maybe?
			}
			if test.l1Receipts != nil {
				src.ExpectFetchReceipts(test.nextBlock.Hash, info, test.l1Receipts, nil)
			}

			cfg := &rollup.Config{
				Genesis:               rollup.Genesis{SystemConfig: test.initialL1Cfg},
				L1SystemConfigAddress: sysCfgAddr,
			}
			tr := NewL1Traversal(testlog.Logger(t, log.LevelError), cfg, src)
			// Load up the initial state with a reset
			_ = tr.Reset(context.Background(), test.startBlock, test.initialL1Cfg)

			// Advance it + assert output
			err := tr.AdvanceL1Block(context.Background())
			require.ErrorIs(t, err, test.expectedErr)

			if test.expectedErr == nil {
				ref, err := tr.NextL1Block(context.Background())
				require.Nil(t, err)
				require.Equal(t, test.nextBlock, ref)
			}

			src.AssertExpectations(t)
		})
	}

}
