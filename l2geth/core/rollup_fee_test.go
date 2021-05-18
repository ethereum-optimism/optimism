package core

import (
	"math/big"
	"testing"
)

var feeTests = map[string]struct {
	dataLen        int
	gasUsed        uint64
	dataPrice      uint64
	executionPrice uint64
	maxGasLimit    uint64
}{
	"simple":               {10000000, 10, 20, 30, 10_000_000},
	"zero gas used":        {10000, 0, 20, 30, 10_000_000},
	"zero data price":      {10000, 0, 0, 30, 10_000_000},
	"zero execution price": {10000, 0, 0, 0, 10_000_000},
}

func TestCalculateRollupFee(t *testing.T) {
	for name, tt := range feeTests {
		t.Run(name, func(t *testing.T) {
			data := make([]byte, 0, tt.dataLen)
			gasUsed, dataPrice := new(big.Int).SetUint64(tt.gasUsed), new(big.Int).SetUint64(tt.dataPrice)
			fee := CalculateRollupFee(data, gasUsed, dataPrice, new(big.Int).SetUint64(tt.executionPrice))

			zeroes, ones := zeroesAndOnes(data)
			zeroesCost := zeroes * 4
			onesCost := ones * 16
			dataCost := zeroesCost + onesCost + overhead.Uint64()
			l1Fee := dataCost * tt.dataPrice

			executionFee := tt.executionPrice * tt.gasUsed
			fee1 := l1Fee * executionFee
			fee2 := tt.maxGasLimit * gasUsed.Uint64()
			expectedFee := fee1 + fee2

			if fee != expectedFee {
				t.Errorf("rollup fee check failed: expected %d, got %d", expectedFee, fee)
			}
		})
	}
}
