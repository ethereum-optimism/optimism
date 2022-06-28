package gasprices

import (
	"math/big"
	"testing"
)

type MockEpoch struct {
	numBlocks   uint64
	repeatCount uint64
	postHook    func(prevGasPrice uint64, gasPriceUpdater *GasPriceUpdater)
}

// Return a gas pricer that targets 3 blocks per epoch & 10% max change per epoch.
func makeTestGasPricerAndUpdater(curPrice uint64) (*GasPricer, *GasPriceUpdater, func(uint64), error) {
	gpsTarget := 990000.3
	getGasTarget := func() float64 { return gpsTarget }
	epochLengthSeconds := uint64(10)
	averageBlockGasLimit := uint64(11000000)
	// Based on our 10 second epoch, we are targeting 3 blocks per epoch.
	gasPricer, err := NewGasPricer(curPrice, 1, getGasTarget, 10)
	if err != nil {
		return nil, nil, nil, err
	}

	curBlock := uint64(10)
	incrementCurrentBlock := func(newBlockNum uint64) { curBlock += newBlockNum }
	getLatestBlockNumber := func() (uint64, error) { return curBlock, nil }
	updateL2GasPrice := func(x uint64) error {
		return nil
	}

	// This is paramaterized based on 3 blocks per epoch, where each uses
	// the average block gas limit plus an additional bit of gas added
	getGasUsedByBlockFn := func(number *big.Int) (uint64, error) {
		return averageBlockGasLimit*3/epochLengthSeconds + 1, nil
	}

	startBlock, _ := getLatestBlockNumber()
	gasUpdater, err := NewGasPriceUpdater(
		gasPricer,
		startBlock,
		averageBlockGasLimit,
		epochLengthSeconds,
		getLatestBlockNumber,
		getGasUsedByBlockFn,
		updateL2GasPrice,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	return gasPricer, gasUpdater, incrementCurrentBlock, nil
}

func TestUpdateGasPriceCallsUpdateL2GasPriceFn(t *testing.T) {
	_, gasUpdater, incrementCurrentBlock, err := makeTestGasPricerAndUpdater(1)
	if err != nil {
		t.Fatal(err)
	}
	wasCalled := false
	gasUpdater.updateL2GasPriceFn = func(gasPrice uint64) error {
		wasCalled = true
		return nil
	}
	incrementCurrentBlock(3)
	if err := gasUpdater.UpdateGasPrice(); err != nil {
		t.Fatal(err)
	}
	if wasCalled != true {
		t.Fatalf("Expected updateL2GasPrice to be called.")
	}
}

func TestUpdateGasPriceCorrectlyUpdatesAZeroBlockEpoch(t *testing.T) {
	gasPricer, gasUpdater, _, err := makeTestGasPricerAndUpdater(100)
	if err != nil {
		t.Fatal(err)
	}
	gasPriceBefore := gasPricer.curPrice
	gasPriceAfter := gasPricer.curPrice
	gasUpdater.updateL2GasPriceFn = func(gasPrice uint64) error {
		gasPriceAfter = gasPrice
		return nil
	}
	if err := gasUpdater.UpdateGasPrice(); err != nil {
		t.Fatal(err)
	}
	if gasPriceBefore < gasPriceAfter {
		t.Fatalf("Expected gasPrice to go down because we had fewer than 3 blocks in the epoch.")
	}
}

func TestUpdateGasPriceFailsIfBlockNumberGoesBackwards(t *testing.T) {
	_, gasUpdater, _, err := makeTestGasPricerAndUpdater(1)
	if err != nil {
		t.Fatal(err)
	}
	gasUpdater.epochStartBlockNumber = 10
	gasUpdater.getLatestBlockNumberFn = func() (uint64, error) { return 0, nil }
	err = gasUpdater.UpdateGasPrice()
	if err == nil {
		t.Fatalf("Expected UpdateGasPrice to fail when block number goes backwards.")
	}
}

func TestUsageOfGasPriceUpdater(t *testing.T) {
	_, gasUpdater, incrementCurrentBlock, err := makeTestGasPricerAndUpdater(1000)
	if err != nil {
		t.Fatal(err)
	}
	// In these mock epochs the gas price shold go up and then down again after the time has passed
	mockEpochs := []MockEpoch{
		// First jack up the price to show that it will grow over time
		MockEpoch{
			numBlocks:   10,
			repeatCount: 3,
			// Make sure the gas price is increasing
			postHook: func(prevGasPrice uint64, gasPriceUpdater *GasPriceUpdater) {
				curPrice := gasPriceUpdater.gasPricer.curPrice
				if prevGasPrice >= curPrice {
					t.Fatalf("Expected gas price to increase. Got %d, was %d", curPrice, prevGasPrice)
				}
			},
		},
		// Then stabilize around the GPS we want
		MockEpoch{
			numBlocks:   3,
			repeatCount: 5,
			postHook:    func(prevGasPrice uint64, gasPriceUpdater *GasPriceUpdater) {},
		},
		MockEpoch{
			numBlocks:   3,
			repeatCount: 0,
			postHook: func(prevGasPrice uint64, gasPriceUpdater *GasPriceUpdater) {
				curPrice := gasPriceUpdater.gasPricer.curPrice
				if prevGasPrice != curPrice {
					t.Fatalf("Expected gas price to stablize. Got %d, was %d", curPrice, prevGasPrice)
				}

				targetGps := gasPriceUpdater.gasPricer.getTargetGasPerSecond()
				averageGps := gasPriceUpdater.gasPricer.avgGasPerSecondLastEpoch
				if targetGps != averageGps {
					t.Fatalf("Average gas/second (%f) did not converge to target (%f)",
						averageGps, targetGps)
				}
			},
		},
		// Then reduce the demand to show the fee goes back down to the floor
		MockEpoch{
			numBlocks:   1,
			repeatCount: 5,
			postHook: func(prevGasPrice uint64, gasPriceUpdater *GasPriceUpdater) {
				curPrice := gasPriceUpdater.gasPricer.curPrice
				if prevGasPrice <= curPrice && curPrice != gasPriceUpdater.gasPricer.floorPrice {
					t.Fatalf("Expected gas price either reduce or be at the floor.")
				}
			},
		},
	}
	loop := func(epoch MockEpoch) {
		prevGasPrice := gasUpdater.gasPricer.curPrice
		incrementCurrentBlock(epoch.numBlocks)
		err = gasUpdater.UpdateGasPrice()
		if err != nil {
			t.Fatal(err)
		}
		epoch.postHook(prevGasPrice, gasUpdater)
	}
	for _, epoch := range mockEpochs {
		for i := 0; i < int(epoch.repeatCount)+1; i++ {
			loop(epoch)
		}
	}
}
