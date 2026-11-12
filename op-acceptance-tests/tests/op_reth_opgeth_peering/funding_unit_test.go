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
		{count: 8192, levels: []int{2, 32, 512, 8192}},
	} {
		t.Run(fmt.Sprintf("%d_accounts", tc.count), func(t *testing.T) {
			require.Equal(t, tc.levels, fundingLevels(tc.count))
		})
	}
}

func TestFundingAmountsCoverWhatEachRoundPaysOut(t *testing.T) {
	// At a 100 gwei fee cap and a 100,000 gwei L1 data fee allowance a transfer costs 2,200,000
	// gwei; a payer of 16 accounts at 1,000 gwei each needs 16 x 2,201,000 gwei, and its own payer
	// 16 x (35,216,000 + 2,200,000) gwei.
	amounts := fundingAmounts([]int{2, 17, 257}, eth.GWei(1_000))
	require.Equal(t, []eth.ETH{eth.GWei(598_656_000), eth.GWei(35_216_000), eth.GWei(1_000)}, amounts)
}

func TestFundingRoundAttemptsGrowWithTheBlocksTheRoundFills(t *testing.T) {
	const blockGasLimit = 60_000_000 // 2857 transfers per block
	for _, tc := range []struct {
		name      string
		transfers int

		attempts int
	}{
		{name: "one_transfer", transfers: 1, attempts: fundingRoundBaseAttempts + fundingRoundAttemptsPerBlock},
		{name: "one_full_block", transfers: 2857, attempts: fundingRoundBaseAttempts + fundingRoundAttemptsPerBlock},
		{name: "one_over_a_full_block", transfers: 2858, attempts: fundingRoundBaseAttempts + 2*fundingRoundAttemptsPerBlock},
		{name: "burst_last_round", transfers: 8192, attempts: fundingRoundBaseAttempts + 3*fundingRoundAttemptsPerBlock},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.attempts, fundingRoundAttempts(tc.transfers, blockGasLimit))
		})
	}
}
