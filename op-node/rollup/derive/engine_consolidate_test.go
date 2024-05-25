package derive

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestWithdrawalsMatch(t *testing.T) {
	tests := []struct {
		attrs       *types.Withdrawals
		block       *types.Withdrawals
		shouldMatch bool
	}{
		{
			attrs:       nil,
			block:       nil,
			shouldMatch: true,
		},
		{
			attrs:       &types.Withdrawals{},
			block:       nil,
			shouldMatch: false,
		},
		{
			attrs:       nil,
			block:       &types.Withdrawals{},
			shouldMatch: false,
		},
		{
			attrs:       &types.Withdrawals{},
			block:       &types.Withdrawals{},
			shouldMatch: true,
		},
		{
			attrs: &types.Withdrawals{
				{
					Index: 1,
				},
			},
			block:       &types.Withdrawals{},
			shouldMatch: false,
		},
		{
			attrs: &types.Withdrawals{
				{
					Index: 1,
				},
			},
			block: &types.Withdrawals{
				{
					Index: 2,
				},
			},
			shouldMatch: false,
		},
	}

	for _, test := range tests {
		err := checkWithdrawalsMatch(test.attrs, test.block)

		if test.shouldMatch {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}
