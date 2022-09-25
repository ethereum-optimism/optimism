package derive

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestChannelOutAddBlock(t *testing.T) {
	cout, err := NewChannelOut()
	require.NoError(t, err)

	t.Run("returns err if first tx is not an l1info tx", func(t *testing.T) {
		header := &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(100)}
		block := types.NewBlockWithHeader(header).WithBody(
			[]*types.Transaction{
				types.NewTx(&types.DynamicFeeTx{}),
			},
			nil,
		)
		err := cout.AddBlock(block)
		require.Error(t, err)
		require.Equal(t, ErrNotDepositTx, err)
	})
}
