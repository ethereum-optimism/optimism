package txmgr

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type priceBumpTest struct {
	prevGasTip  int64
	prevBasefee int64
	newGasTip   int64
	newBasefee  int64
	expectedTip int64
	expectedFC  int64
}

func (tc *priceBumpTest) run(t *testing.T) {
	prevFC := CalcGasFeeCap(big.NewInt(tc.prevBasefee), big.NewInt(tc.prevGasTip))
	lgr := testlog.Logger(t, log.LvlCrit)

	tip, fc := updateFees(big.NewInt(tc.prevGasTip), prevFC, big.NewInt(tc.newGasTip), big.NewInt(tc.newBasefee), lgr)

	require.Equal(t, tc.expectedTip, tip.Int64(), "tip must be as expected")
	require.Equal(t, tc.expectedFC, fc.Int64(), "fee cap must be as expected")
}

func TestUpdateFees(t *testing.T) {
	tests := []priceBumpTest{
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 90, newBasefee: 900,
			expectedTip: 100, expectedFC: 2100,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 101, newBasefee: 1000,
			expectedTip: 115, expectedFC: 2415,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 100, newBasefee: 1001,
			expectedTip: 115, expectedFC: 2415,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 101, newBasefee: 900,
			expectedTip: 115, expectedFC: 2415,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 90, newBasefee: 1010,
			expectedTip: 115, expectedFC: 2415,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 101, newBasefee: 2000,
			expectedTip: 115, expectedFC: 4115,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 120, newBasefee: 900,
			expectedTip: 120, expectedFC: 2415,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 120, newBasefee: 1100,
			expectedTip: 120, expectedFC: 2415,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 120, newBasefee: 1140,
			expectedTip: 120, expectedFC: 2415,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 120, newBasefee: 1200,
			expectedTip: 120, expectedFC: 2520,
		},
	}
	for i, test := range tests {
		i := i
		test := test
		t.Run(fmt.Sprint(i), test.run)
	}
}
