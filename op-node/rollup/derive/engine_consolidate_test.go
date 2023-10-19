package derive

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func TestWithdrawalsMatch(t *testing.T) {
	tests := []struct {
		attrs       *eth.Withdrawals
		block       *eth.Withdrawals
		shouldMatch bool
	}{
		{
			attrs:       nil,
			block:       nil,
			shouldMatch: true,
		},
		{
			attrs:       &eth.Withdrawals{},
			block:       nil,
			shouldMatch: false,
		},
		{
			attrs:       nil,
			block:       &eth.Withdrawals{},
			shouldMatch: false,
		},
		{
			attrs:       &eth.Withdrawals{},
			block:       &eth.Withdrawals{},
			shouldMatch: true,
		},
		{
			attrs: &eth.Withdrawals{
				{
					Index: 1,
				},
			},
			block:       &eth.Withdrawals{},
			shouldMatch: false,
		},
		{
			attrs: &eth.Withdrawals{
				{
					Index: 1,
				},
			},
			block: &eth.Withdrawals{
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
