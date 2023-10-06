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
	prevBasefee int64
	newGasTip   int64
	newBasefee  int64
	expectedTip int64
	expectedFC  int64
}

func (tc *priceBumpTest) run(t *testing.T) {
	prevFC := calcGasFeeCap(big.NewInt(tc.prevBasefee), big.NewInt(tc.prevGasTip))
	lgr := testlog.Logger(t, log.LvlCrit)

	tip, fc := updateFees(big.NewInt(tc.prevGasTip), prevFC, big.NewInt(tc.newGasTip), big.NewInt(tc.newBasefee), lgr)

	require.Equal(t, tc.expectedTip, tip.Int64(), "tip must be as expected")
	require.Equal(t, tc.expectedFC, fc.Int64(), "fee cap must be as expected")
}

func TestUpdateFees(t *testing.T) {
	require.Equal(t, int64(10), priceBump, "test must be updated if priceBump is adjusted")
	tests := []priceBumpTest{
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 90, newBasefee: 900,
			expectedTip: 110, expectedFC: 2310,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 101, newBasefee: 1000,
			expectedTip: 110, expectedFC: 2310,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 100, newBasefee: 1001,
			expectedTip: 110, expectedFC: 2310,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 101, newBasefee: 900,
			expectedTip: 110, expectedFC: 2310,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 90, newBasefee: 1010,
			expectedTip: 110, expectedFC: 2310,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 101, newBasefee: 2000,
			expectedTip: 110, expectedFC: 4110,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 120, newBasefee: 900,
			expectedTip: 120, expectedFC: 2310,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 120, newBasefee: 1100,
			expectedTip: 120, expectedFC: 2320,
		},
		{
			prevGasTip: 100, prevBasefee: 1000,
			newGasTip: 120, newBasefee: 1140,
			expectedTip: 120, expectedFC: 2400,
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
