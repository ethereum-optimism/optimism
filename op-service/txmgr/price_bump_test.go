package txmgr

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type priceBumpTest struct {
	prevGasTip  int64
	prevBaseFee int64
	newGasTip   int64
	newBaseFee  int64
	expectedTip int64
	expectedFC  int64
	isBlobTx    bool
}

func (tc *priceBumpTest) run(t *testing.T) {
	prevFC := calcGasFeeCap(big.NewInt(tc.prevBaseFee), big.NewInt(tc.prevGasTip))
	lgr := testlog.Logger(t, log.LevelCrit)

	tip, fc := updateFees(big.NewInt(tc.prevGasTip), prevFC, big.NewInt(tc.newGasTip), big.NewInt(tc.newBaseFee), tc.isBlobTx, lgr)

	require.Equal(t, tc.expectedTip, tip.Int64(), "tip must be as expected")
	require.Equal(t, tc.expectedFC, fc.Int64(), "fee cap must be as expected")
}

func TestUpdateFees(t *testing.T) {
	require.Equal(t, int64(10), priceBump, "test must be updated if priceBump is adjusted")
	tests := []priceBumpTest{
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 90, newBaseFee: 900,
			expectedTip: 110, expectedFC: 2310,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 90, newBaseFee: 900,
			expectedTip: 200, expectedFC: 4200,
			isBlobTx: true,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 101, newBaseFee: 1000,
			expectedTip: 110, expectedFC: 2310,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 101, newBaseFee: 1000,
			expectedTip: 200, expectedFC: 4200,
			isBlobTx: true,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 100, newBaseFee: 1001,
			expectedTip: 110, expectedFC: 2310,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 100, newBaseFee: 1001,
			expectedTip: 200, expectedFC: 4200,
			isBlobTx: true,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 101, newBaseFee: 900,
			expectedTip: 110, expectedFC: 2310,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 101, newBaseFee: 900,
			expectedTip: 200, expectedFC: 4200,
			isBlobTx: true,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 90, newBaseFee: 1010,
			expectedTip: 110, expectedFC: 2310,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 90, newBaseFee: 1010,
			expectedTip: 200, expectedFC: 4200,
			isBlobTx: true,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 101, newBaseFee: 2000,
			expectedTip: 110, expectedFC: 4110,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 101, newBaseFee: 3000,
			expectedTip: 200, expectedFC: 6200,
			isBlobTx: true,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 120, newBaseFee: 900,
			expectedTip: 120, expectedFC: 2310,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 220, newBaseFee: 900,
			expectedTip: 220, expectedFC: 4200,
			isBlobTx: true,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 120, newBaseFee: 1100,
			expectedTip: 120, expectedFC: 2320,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 220, newBaseFee: 2000,
			expectedTip: 220, expectedFC: 4220,
			isBlobTx: true,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 120, newBaseFee: 1140,
			expectedTip: 120, expectedFC: 2400,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 220, newBaseFee: 2040,
			expectedTip: 220, expectedFC: 4300,
			isBlobTx: true,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 120, newBaseFee: 1200,
			expectedTip: 120, expectedFC: 2520,
		},
		{
			prevGasTip: 100, prevBaseFee: 1000,
			newGasTip: 220, newBaseFee: 2100,
			expectedTip: 220, expectedFC: 4420,
			isBlobTx: true,
		},
	}
	for i, test := range tests {
		i := i
		test := test
		t.Run(fmt.Sprint(i), test.run)
	}
}
