package disburser_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/go/teleportr/drivers/disburser"
	"github.com/stretchr/testify/require"
)

func uint64Ptr(x uint64) *uint64 {
	return &x
}

type filterStartBlockNumberTestCase struct {
	name                string
	params              disburser.FilterStartBlockNumberParams
	expStartBlockNumber uint64
}

// TestFindFilterStartBlockNumber exhaustively tests the behavior of
// FindFilterStartBlockNumber and its edge cases.
func TestFindFilterStartBlockNumber(t *testing.T) {
	tests := []filterStartBlockNumberTestCase{
		// Deploy number should be returned if LastProcessedBlockNumber is nil.
		{
			name: "init returns deploy block number",
			params: disburser.FilterStartBlockNumberParams{
				BlockNumber:              10,
				NumConfirmations:         5,
				DeployBlockNumber:        42,
				LastProcessedBlockNumber: nil,
			},
			expStartBlockNumber: 42,
		},
		// Deploy number should be returned if the deploy number is still in our
		// confirmation window.
		{
			name: "conf lookback before deploy number",
			params: disburser.FilterStartBlockNumberParams{
				BlockNumber:              43,
				NumConfirmations:         5,
				DeployBlockNumber:        42,
				LastProcessedBlockNumber: uint64Ptr(43),
			},
			expStartBlockNumber: 42,
		},
		// Deploy number should be returned if the deploy number is still in our
		// confirmation window.
		{
			name: "conf lookback before deploy number",
			params: disburser.FilterStartBlockNumberParams{
				BlockNumber:              43,
				NumConfirmations:         44,
				DeployBlockNumber:        42,
				LastProcessedBlockNumber: uint64Ptr(43),
			},
			expStartBlockNumber: 42,
		},
		// If our confirmation window is ahead of the last deposit + 1, expect
		// last deposit + 1.
		{
			name: "conf lookback gt last deposit plus one",
			params: disburser.FilterStartBlockNumberParams{
				BlockNumber:              100,
				NumConfirmations:         5,
				DeployBlockNumber:        42,
				LastProcessedBlockNumber: uint64Ptr(43),
			},
			expStartBlockNumber: 44,
		},
		// If our confirmation window is equal to last deposit + 1, expect last
		// deposit + 1.
		{
			name: "conf lookback eq last deposit plus one",
			params: disburser.FilterStartBlockNumberParams{
				BlockNumber:              48,
				NumConfirmations:         5,
				DeployBlockNumber:        42,
				LastProcessedBlockNumber: uint64Ptr(43),
			},
			expStartBlockNumber: 44,
		},
		// If our confirmation window starts before last deposit + 1, expect
		// block number - num confs + 1.
		{
			name: "conf lookback lt last deposit plus one",
			params: disburser.FilterStartBlockNumberParams{
				BlockNumber:              47,
				NumConfirmations:         5,
				DeployBlockNumber:        42,
				LastProcessedBlockNumber: uint64Ptr(43),
			},
			expStartBlockNumber: 43,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testFindFilterStartBlockNumber(t, test)
		})
	}
}

func testFindFilterStartBlockNumber(
	t *testing.T,
	test filterStartBlockNumberTestCase,
) {

	startBlockNumber := disburser.FindFilterStartBlockNumber(test.params)
	require.Equal(t, test.expStartBlockNumber, startBlockNumber)
}
