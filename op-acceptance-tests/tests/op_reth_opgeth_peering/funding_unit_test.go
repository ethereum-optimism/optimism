package op_reth_opgeth_peering

import (
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
		require.Equal(t, tc.levels, fundingLevels(tc.count), "count %d", tc.count)
		levels := fundingLevels(tc.count)
		require.LessOrEqual(t, levels[0], fanOut, "the funder pays the first round alone")
		for i := 1; i < len(levels); i++ {
			require.GreaterOrEqual(t, levels[i-1]*fanOut, levels[i], "round %d must have enough payers", i)
		}
	}
}

func TestFundingAmountsCoverWhatEachRoundPaysOut(t *testing.T) {
	leaf := eth.GWei(1_000)
	amounts := fundingAmounts([]int{2, 17, 257}, leaf)
	require.Equal(t, leaf, amounts[2])
	require.Equal(t, leaf.Add(fundingTxCost).Mul(fanOut), amounts[1])
	require.Equal(t, amounts[1].Add(fundingTxCost).Mul(fanOut), amounts[0])
}
