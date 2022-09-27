package derive

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
)

// TestL1TraversalNext tests that the `Next` function only returns
// a block reference once and then properly returns io.EOF afterwards
func TestL1TraversalNext(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	a := testutils.RandomBlockRef(rng)

	tr := NewL1Traversal(testlog.Logger(t, log.LvlError), nil)
	// Load up the initial state with a reset
	_ = tr.Reset(context.Background(), a)

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

	tests := []struct {
		name        string
		startBlock  eth.L1BlockRef
		nextBlock   eth.L1BlockRef
		fetcherErr  error
		expectedErr error
	}{
		{
			name:        "simple extension",
			startBlock:  a,
			nextBlock:   b,
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			src := &testutils.MockL1Source{}
			src.ExpectL1BlockRefByNumber(test.startBlock.Number+1, test.nextBlock, test.fetcherErr)

			tr := NewL1Traversal(testlog.Logger(t, log.LvlError), src)
			// Load up the initial state with a reset
			_ = tr.Reset(context.Background(), test.startBlock)

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
