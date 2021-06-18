package gasprices

import (
	"fmt"
	"testing"
)

type MockEpoch struct {
	timestamp float64
	numBlocks uint64
}

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
	// In these mock epochs the gas price shold go up and then down again after the time has passed
	mockEpochs := []MockEpoch{
		MockEpoch{
			timestamp: 0,
			numBlocks: 5,
		},
		MockEpoch{
			timestamp: 0,
			numBlocks: 5,
		},
		MockEpoch{
			timestamp: 0,
			numBlocks: 5,
		},
		MockEpoch{
			timestamp: 0,
			numBlocks: 5,
		},
		MockEpoch{
			timestamp: 0,
			numBlocks: 5,
		},
		MockEpoch{
			timestamp: 50,
			numBlocks: 5,
		},
		MockEpoch{
			timestamp: 50,
			numBlocks: 5,
		},
		MockEpoch{
			timestamp: 50,
			numBlocks: 5,
		},
		MockEpoch{
			timestamp: 50,
			numBlocks: 5,
		},
	}
	loop := func(epoch MockEpoch) {
		mockTimestamp = epoch.timestamp
		curBlock += epoch.numBlocks
		gasUpdater.UpdateGasPrice()
		fmt.Println("gas price:", gasUpdater.gasPricer.curPrice)
	}
	for _, epoch := range mockEpochs {
		loop(epoch)
	}
}
