package core

import (
	"math/big"
	"testing"
)

var feeTests = map[string]struct {
	dataLen        int
	gasUsed        uint64
	dataPrice      int64
	executionPrice int64
}{
	"simple":               {10000, 10, 20, 30},
	"zero gas used":        {10000, 0, 20, 30},
	"zero data price":      {10000, 0, 0, 30},
	"zero execution price": {10000, 0, 0, 0},
}

func TestCalculateRollupFee(t *testing.T) {
	for name, tt := range feeTests {
		t.Run(name, func(t *testing.T) {
			data := make([]byte, 0, tt.dataLen)
			fee := CalculateRollupFee(data, tt.gasUsed, big.NewInt(tt.dataPrice), big.NewInt(tt.executionPrice))

			dataFee := uint64((ROLLUP_BASE_TX_SIZE + len(data)) * int(tt.dataPrice))
			executionFee := uint64(tt.executionPrice) * tt.gasUsed
			expectedFee := dataFee + executionFee
			if fee.Cmp(big.NewInt(int64(expectedFee))) != 0 {
				t.Errorf("rollup fee check failed: expected %d, got %s", expectedFee, fee.String())
			}
		})
	}
}
