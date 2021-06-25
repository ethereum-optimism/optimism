package gasprices

import (
	"fmt"
	"testing"
)

type MockEpoch struct {
	numBlocks uint64
}

// WIP
func TestUsageOfGasPriceUpdater(t *testing.T) {
	gasPerSecond := 3300000.0
	getGasTarget := func() float64 { return gasPerSecond }
	epochLengthSeconds := 10.0
	averageBlockGasLimit := 11000000.0
	// Based on our 10 second epoch, we are targetting this number of blocks per second
	numBlocksToTarget := (epochLengthSeconds * gasPerSecond) / averageBlockGasLimit
	fmt.Println("Number of target blocks: ", numBlocksToTarget)

	gasPricer := NewGasPricer(1, 1, getGasTarget, 10)

	curBlock := uint64(10)
	getLatestBlockNumber := func() (uint64, error) { return curBlock, nil }
	updateL2GasPrice := func(x float64) error {
		fmt.Println("new gas price:", x)
		return nil
	}

	// Example loop usage
	startBlock, _ := getLatestBlockNumber()
	gasUpdater := NewGasPriceUpdater(gasPricer, startBlock, 11000000, 10, getLatestBlockNumber, updateL2GasPrice)

	// In these mock epochs the gas price shold go up and then down again after the time has passed
	mockEpochs := []MockEpoch{
		// First jack up the price to show that it will grow over time
		MockEpoch{
			numBlocks: 10,
		},
		MockEpoch{
			numBlocks: 10,
		},
		MockEpoch{
			numBlocks: 10,
		},
		// Then stabilize around the GPS we want
		MockEpoch{
			numBlocks: 3,
		},
		MockEpoch{
			numBlocks: 3,
		},
		MockEpoch{
			numBlocks: 3,
		},
		MockEpoch{
			numBlocks: 3,
		},
		MockEpoch{
			numBlocks: 3,
		},
		MockEpoch{
			numBlocks: 3,
		},
		MockEpoch{
			numBlocks: 3,
		},
		MockEpoch{
			numBlocks: 3,
		},
		MockEpoch{
			numBlocks: 3,
		},
		// Then reduce the demand to show the fee goes back down to the floor
		MockEpoch{
			numBlocks: 1,
		},
		MockEpoch{
			numBlocks: 1,
		},
		MockEpoch{
			numBlocks: 1,
		},
		MockEpoch{
			numBlocks: 1,
		},
		MockEpoch{
			numBlocks: 1,
		},
		MockEpoch{
			numBlocks: 1,
		},
		MockEpoch{
			numBlocks: 1,
		},
	}
	loop := func(epoch MockEpoch) {
		curBlock += epoch.numBlocks
		gasUpdater.UpdateGasPrice()
	}
	for _, epoch := range mockEpochs {
		loop(epoch)
	}
}
