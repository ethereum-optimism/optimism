package rollup

import (
	"math"
	"testing"
)

type CalcGasPriceTestCase struct {
	name                     string
	avgGasPerSecondLastEpoch float64
	expectedNextGasPrice     float64
}

func returnConstFn(retVal float64) func() float64 {
	return func() float64 { return retVal }
}

func runCalcGasPriceTests(gp L2GasPricer, tcs []CalcGasPriceTestCase, t *testing.T) {
	for _, tc := range tcs {
		if tc.expectedNextGasPrice != gp.CalcNextGasPrice(tc.avgGasPerSecondLastEpoch) {
			t.Fatalf("failed on test: %s", tc.name)
		}
	}
}

func TestCalcGasPriceFarFromFloor(t *testing.T) {
	gp := L2GasPricer{
		curPrice:                 100,
		floorPrice:               1,
		getTargetGasPerSecond:    returnConstFn(10),
		maxPercentChangePerEpoch: 0.5,
	}
	tcs := []CalcGasPriceTestCase{
		// No change
		{
			name:                     "No change expected when already at target",
			avgGasPerSecondLastEpoch: 10,
			expectedNextGasPrice:     100,
		},
		// Price reduction
		{
			name:                     "Max % change bounds the reduction in price",
			avgGasPerSecondLastEpoch: 1,
			expectedNextGasPrice:     50,
		},
		{
			// We're half of our target, so reduce by half
			name:                     "Reduce fee by half if at 50% capacity",
			avgGasPerSecondLastEpoch: 5,
			expectedNextGasPrice:     50,
		},
		{
			name:                     "Reduce fee by 75% if at 75% capacity",
			avgGasPerSecondLastEpoch: 7.5,
			expectedNextGasPrice:     75,
		},
		// Price increase
		{
			name:                     "Max % change bounds the increase in price",
			avgGasPerSecondLastEpoch: 100,
			expectedNextGasPrice:     150,
		},
		{
			name:                     "Increase fee by 25% if at 125% capacity",
			avgGasPerSecondLastEpoch: 12.5,
			expectedNextGasPrice:     125,
		},
	}
	runCalcGasPriceTests(gp, tcs, t)
}

func TestCalcGasPriceAtFloor(t *testing.T) {
	gp := L2GasPricer{
		curPrice:                 100,
		floorPrice:               100,
		getTargetGasPerSecond:    returnConstFn(10),
		maxPercentChangePerEpoch: 0.5,
	}
	tcs := []CalcGasPriceTestCase{
		// No change
		{
			name:                     "No change expected when already at target",
			avgGasPerSecondLastEpoch: 10,
			expectedNextGasPrice:     100,
		},
		// Price reduction
		{
			name:                     "No change expected when at floorPrice",
			avgGasPerSecondLastEpoch: 1,
			expectedNextGasPrice:     100,
		},
		// Price increase
		{
			name:                     "Max % change bounds the increase in price",
			avgGasPerSecondLastEpoch: 100,
			expectedNextGasPrice:     150,
		},
	}
	runCalcGasPriceTests(gp, tcs, t)
}

func TestGasPricerUpdates(t *testing.T) {
	gp := L2GasPricer{
		curPrice:                 100,
		floorPrice:               100,
		getTargetGasPerSecond:    returnConstFn(10),
		maxPercentChangePerEpoch: 0.5,
	}
	gp.UpdateGasPrice(12.5)
	if gp.curPrice != 125 {
		t.Fatalf("gp.curPrice not updated correctly. Got: %v, expected: %v", gp.curPrice, 125)
	}
}

func TestGasPricerDynamicTarget(t *testing.T) {
	// In prod we will be committing to a gas per second schedule in order to
	// meter usage over time. This linear interpolation between a start time, end time,
	// start gas per second, and end gas per second is an example of how we can introduce
	// acceleration in our gas pricer
	startTimestamp := float64(0)
	startGasPerSecond := float64(10)
	endTimestamp := float64(100)
	endGasPerSecond := float64(100)

	// Helper function for calculating the current gas per second that we are targetting
	linearInterpolation := func(x float64, x1 float64, x2 float64, y1 float64, y2 float64) float64 {
		return math.Min(y1+((x-x1)/(x2-x1))*(y2-y1), endGasPerSecond)
	}

	mockTimestamp := float64(0) // start at timestamp 0
	// TargetGasPerSecond is dynamic based on the current "mocktimestamp"
	dynamicGetTarget := func() float64 {
		return linearInterpolation(mockTimestamp, startTimestamp, endTimestamp, startGasPerSecond, endGasPerSecond)
	}
	gp := L2GasPricer{
		curPrice:                 100,
		floorPrice:               1,
		getTargetGasPerSecond:    dynamicGetTarget,
		maxPercentChangePerEpoch: 0.5,
	}
	gasPerSecondDemanded := returnConstFn(15)
	for i := 0; i < 10; i += 1 {
		mockTimestamp = float64(i * 10)
		expectedPrice := math.Ceil(gp.curPrice * math.Max(0.5, gasPerSecondDemanded()/dynamicGetTarget()))
		gp.UpdateGasPrice(gasPerSecondDemanded())
		if gp.curPrice != expectedPrice {
			t.Fatalf("gp.curPrice not updated correctly. Got: %v expected: %v", gp.curPrice, expectedPrice)
		}
	}
}
