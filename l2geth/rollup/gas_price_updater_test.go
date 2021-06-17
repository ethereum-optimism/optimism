package rollup

import (
	"fmt"
	"testing"
)

// WIP
func TestUsageOfGasPriceUpdater(t *testing.T) {
	startTimestamp := float64(0)
	startGasPerSecond := float64(10)
	endTimestamp := float64(100)
	endGasPerSecond := float64(100)
	mockTimestamp := float64(0) // start at timestamp 0
	mockTimeNow := func() float64 {
		return mockTimestamp
	}
	getGasTarget := GetLinearInterpolationFn(mockTimeNow, startTimestamp, endTimestamp, startGasPerSecond, endGasPerSecond)

	gasPricer := NewGasPricer(1, 1, getGasTarget, 10)

	curBlock := uint64(10)
	getLatestBlockNumber := func() (uint64, error) { return curBlock, nil }
	updateL2GasPrice := func(x float64) error { return nil }

	// Example loop usage
	startBlock, _ := getLatestBlockNumber()
	gasUpdater := NewGasPriceUpdater(gasPricer, startBlock, 1, getLatestBlockNumber, updateL2GasPrice)
	fmt.Println("First gas price:", gasUpdater.gasPricer.curPrice)
	curBlock += 100
	gasUpdater.UpdateGasPrice()
	fmt.Println("Second gas price:", gasUpdater.gasPricer.curPrice)
}
