package testutils

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandomBlockWithoutTxs(t *testing.T) {
	block, receipts := RandomBlock(rand.New(rand.NewSource(1)), 0)
	require.Empty(t, block.Transactions())
	require.Empty(t, receipts)
	require.Zero(t, block.GasUsed())
}
