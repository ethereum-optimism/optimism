package gasprices

import (
	"math"
	"testing"
)

type CalcGasPriceTestCase struct {
	name                     string
	avgGasPerSecondLastEpoch float64
	expectedNextGasPrice     uint64
}

func returnConstFn(retVal uint64) func() float64 {
	return func() float64 { return float64(retVal) }
}

func runCalcGasPriceTests(gp GasPricer, tcs []CalcGasPriceTestCase, t *testing.T) {
	for _, tc := range tcs {
		nextEpochGasPrice, err := gp.CalcNextEpochGasPrice(tc.avgGasPerSecondLastEpoch)
		if tc.expectedNextGasPrice != nextEpochGasPrice || err != nil {
			t.Fatalf("failed on test: %s", tc.name)
		}
	}
}

func TestCalcGasPriceFarFromFloor(t *testing.T) {
	gp := GasPricer{
		curPrice:              100,
		floorPrice:            1,
		getTargetGasPerSecond: returnConstFn(10),
		maxChangePerEpoch:     0.5,
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
	gp := GasPricer{
		curPrice:              100,
		floorPrice:            100,
		getTargetGasPerSecond: returnConstFn(10),
		maxChangePerEpoch:     0.5,
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
	gp := GasPricer{
		curPrice:              100,
		floorPrice:            100,
		getTargetGasPerSecond: returnConstFn(10),
		maxChangePerEpoch:     0.5,
	}
	_, err := gp.CompleteEpoch(12.5)
	if err != nil {
		t.Fatal(err)
	}
	if gp.curPrice != 125 {
		t.Fatalf("gp.curPrice not updated correctly. Got: %v, expected: %v", gp.curPrice, 125)
	}
}

func TestGetLinearInterpolationFn(t *testing.T) {
	mockTimestamp := float64(0) // start at timestamp 0
	// TargetGasPerSecond is dynamic based on the current "mocktimestamp"
	mockTimeNow := func() float64 {
		return mockTimestamp
	}
	l := GetLinearInterpolationFn(mockTimeNow, 0, 10, 0, 100)
	for expected := 0.0; expected < 100; expected += 10 {
		mockTimestamp = expected / 10 // To prove this is not identity function, divide by 10
		got := l()
		if got != expected {
			t.Fatalf("linear interpolation incorrect. Got: %v expected: %v", got, expected)
		}
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

	mockTimestamp := float64(0) // start at timestamp 0
	// TargetGasPerSecond is dynamic based on the current "mocktimestamp"
	mockTimeNow := func() float64 {
		return mockTimestamp
	}
	// TargetGasPerSecond is dynamic based on the current "mocktimestamp"
	dynamicGetTarget := GetLinearInterpolationFn(mockTimeNow, startTimestamp, endTimestamp, startGasPerSecond, endGasPerSecond)

	gp := GasPricer{
		curPrice:              100,
		floorPrice:            1,
		getTargetGasPerSecond: dynamicGetTarget,
		maxChangePerEpoch:     0.5,
	}
	gasPerSecondDemanded := returnConstFn(15)
	for i := 0; i < 10; i++ {
		mockTimestamp = float64(i * 10)
		expectedPrice := math.Ceil(float64(gp.curPrice) * math.Max(0.5, gasPerSecondDemanded()/dynamicGetTarget()))

		_, err := gp.CompleteEpoch(gasPerSecondDemanded())
		if err != nil {
			t.Fatal(err)
		}
		if gp.curPrice != uint64(expectedPrice) {
			t.Fatalf("gp.curPrice not updated correctly. Got: %v expected: %v", gp.curPrice, expectedPrice)
		}
	}
}
