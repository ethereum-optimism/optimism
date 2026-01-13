package filter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_findBlockByTimestamp(t *testing.T) {
	// Sanity check: binary search finds the correct block
	blocks := map[uint64]uint64{
		1: 100,
		2: 200,
		3: 300,
	}
	fetcher := func(ctx context.Context, blockNum uint64) (uint64, error) {
		return blocks[blockNum], nil
	}

	result, err := findBlockByTimestamp(context.Background(), 150, 3, fetcher)
	require.NoError(t, err)
	require.Equal(t, uint64(2), result) // First block with timestamp >= 150
}
