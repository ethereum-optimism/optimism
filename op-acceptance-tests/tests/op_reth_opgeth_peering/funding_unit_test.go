package op_reth_opgeth_peering

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestFundingLevelsEndWithTheCountAndKeepEveryRoundWithinTheFanOut(t *testing.T) {
	for _, tc := range []struct {
		count  int
		levels []int
	}{
		{count: 1, levels: []int{1}},
		{count: 16, levels: []int{16}},
		{count: 17, levels: []int{2, 17}},
		{count: 256, levels: []int{16, 256}},
		{count: 257, levels: []int{2, 17, 257}},
		{count: 4112, levels: []int{2, 17, 257, 4112}},
	} {
		t.Run(fmt.Sprintf("%d_accounts", tc.count), func(t *testing.T) {
			require.Equal(t, tc.levels, fundingLevels(tc.count))
		})
	}
}

func TestFundingAmountsCoverWhatEachRoundPaysOut(t *testing.T) {
	// At a 100 gwei fee cap a transfer costs 2,100,000 gwei; a payer of 16 accounts at 1,000
	// gwei each needs 16 x 2,101,000 gwei, and its own payer 16 x (33,616,000 + 2,100,000) gwei.
	amounts := fundingAmounts([]int{2, 17, 257}, eth.GWei(1_000))
	require.Equal(t, []eth.ETH{eth.GWei(571_456_000), eth.GWei(33_616_000), eth.GWei(1_000)}, amounts)
}
