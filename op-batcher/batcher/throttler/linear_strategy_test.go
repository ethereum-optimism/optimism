package throttler

import (
	"math"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/stretchr/testify/require"
)

// Test constants specific to linear strategy
const (
	TestLinearThreshold    = 600_000                                                     // 600KB threshold for linear strategy tests
	TestLinearTxSize       = 3_500                                                       // 3.5KB transaction size limit
	TestLinearBlockSize    = 16_000                                                      // 16KB block size limit
	TestLinearAlwaysSize   = 110_000                                                     // 110KB always block size
	TestLinearMultiplier   = 2.0                                                         // 3x threshold multiplier
	TestLinearMaxThreshold = uint64(float64(TestLinearThreshold) * TestLinearMultiplier) // 1.8MB max threshold
)

func TestLinearStrategy_NewLinearStrategy(t *testing.T) {
	strategy := NewLinearStrategy(TestLinearThreshold, TestLinearMaxThreshold, newTestLogger(t))

	if strategy.lowerThreshold != TestLinearThreshold {
		t.Errorf("expected threshold %d, got %d", TestLinearThreshold, strategy.lowerThreshold)
	}

	if strategy.upperThreshold != TestLinearMaxThreshold {
		t.Errorf("expected maxThreshold %d, got %d", TestLinearMaxThreshold, strategy.upperThreshold)
	}

	// Test initial state
	controllerType, intensity := strategy.Load()
	if controllerType != config.LinearControllerType {
		t.Errorf("expected controller type %s, got %s", config.LinearControllerType, controllerType)
	}

	if intensity != TestIntensityMin {
		t.Errorf("expected initial intensity %f, got %f", TestIntensityMin, intensity)
	}
}

func TestLinearStrategy_Update(t *testing.T) {
	strategy := NewLinearStrategy(TestLinearThreshold, TestLinearMaxThreshold, newTestLogger(t))

	tests := []struct {
		name              string
		pendingBytes      uint64
		targetBytes       uint64
		expectedIntensity float64
	}{
		{
			name:              "zero load",
			pendingBytes:      0,
			targetBytes:       0,
			expectedIntensity: TestIntensityMin,
		},
		{
			name:              "below threshold",
			pendingBytes:      TestLinearThreshold / 2,
			targetBytes:       0,
			expectedIntensity: TestIntensityMin,
		},
		{
			name:              "exactly at threshold",
			pendingBytes:      TestLinearThreshold,
			targetBytes:       0,
			expectedIntensity: TestIntensityMin,
		},
		{
			name:              "25% above threshold",
			pendingBytes:      TestLinearThreshold + TestLinearThreshold/4,
			targetBytes:       0,
			expectedIntensity: 0.25,
		},
		{
			name:              "50% above threshold",
			pendingBytes:      TestLinearThreshold + TestLinearThreshold/2,
			targetBytes:       0,
			expectedIntensity: 0.50,
		},
		{
			name:              "75% above threshold",
			pendingBytes:      TestLinearThreshold + 3*TestLinearThreshold/4,
			targetBytes:       0,
			expectedIntensity: 0.75,
		},
		{
			name:              "100% above threshold (max)",
			pendingBytes:      TestLinearMaxThreshold,
			targetBytes:       0,
			expectedIntensity: TestIntensityMax,
		},
		{
			name:              "beyond max threshold",
			pendingBytes:      TestLinearMaxThreshold * 2,
			targetBytes:       0,
			expectedIntensity: TestIntensityMax,
		},
		{
			name:              "with target bytes ignored",
			pendingBytes:      TestLinearThreshold + TestLinearThreshold/2,
			targetBytes:       TestLinearThreshold * 10, // Target bytes should be ignored
			expectedIntensity: 0.50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intensity := strategy.Update(tt.pendingBytes)

			if math.Abs(intensity-tt.expectedIntensity) > TestTolerance {
				t.Errorf("expected intensity %f ± %f, got %f", tt.expectedIntensity, TestTolerance, intensity)
			}
		})
	}
}

func TestLinearStrategy_LinearScaling(t *testing.T) {
	strategy := NewLinearStrategy(TestLinearThreshold, TestLinearMaxThreshold, newTestLogger(t))

	// Test that intensity scales linearly between threshold and maxThreshold
	testPoints := []struct {
		linearRatio       float64
		expectedIntensity float64
	}{
		{0.0, 0.0},   // At threshold
		{0.25, 0.25}, // 25% of way to max -> linear 0.25
		{0.5, 0.5},   // 50% of way to max -> linear 0.5
		{0.75, 0.75}, // 75% of way to max -> linear 0.75
		{1.0, 1.0},   // At max threshold -> linear 1.0
	}

	for _, tp := range testPoints {
		t.Run("", func(t *testing.T) {
			// Calculate load based on linear ratio
			load := TestLinearThreshold + uint64(tp.linearRatio*float64(TestLinearMaxThreshold-TestLinearThreshold))
			intensity := strategy.Update(load)

			if math.Abs(intensity-tp.expectedIntensity) > TestTolerance {
				t.Errorf("linear ratio %.2f: expected intensity %f, got %f", tp.linearRatio, tp.expectedIntensity, intensity)
			}
		})
	}
}

func TestLinearStrategy_GetType(t *testing.T) {
	strategy := NewLinearStrategy(TestLinearThreshold, TestLinearMaxThreshold, newTestLogger(t))

	if strategy.GetType() != config.LinearControllerType {
		t.Errorf("expected GetType() to return %s, got %s", config.LinearControllerType, strategy.GetType())
	}
}

func TestLinearStrategy_Reset(t *testing.T) {
	strategy := NewLinearStrategy(TestLinearThreshold, TestLinearMaxThreshold, newTestLogger(t))

	// Update to build some state
	strategy.Update(TestLinearMaxThreshold)

	// Reset should restore to initial state
	strategy.Reset()

	controllerType, intensity := strategy.Load()
	if controllerType != config.LinearControllerType {
		t.Errorf("expected controller type %s after reset, got %s", config.LinearControllerType, controllerType)
	}

	if intensity != TestIntensityMin {
		t.Errorf("expected intensity %f after reset, got %f", TestIntensityMin, intensity)
	}
}

func TestLinearStrategy_EdgeCases(t *testing.T) {
	t.Run("max threshold less than threshold", func(t *testing.T) {

		require.Panics(t, func() {
			// Test when multiplier results in maxThreshold <= threshold
			NewLinearStrategy(TestLinearThreshold, 0, newTestLogger(t))
		})

	})

	t.Run("very large multiplier", func(t *testing.T) {
		strategy := NewLinearStrategy(TestLinearThreshold, TestLinearThreshold*2000, newTestLogger(t))

		// Even with very large multiplier, should work correctly
		intensity := strategy.Update(TestLinearThreshold * 2)

		// Should be very low intensity due to large range and linear scaling
		if intensity > 0.05 {
			t.Errorf("expected very low intensity with large multiplier, got %f", intensity)
		}
	})
}

func TestLinearStrategy_Load(t *testing.T) {
	strategy := NewLinearStrategy(TestLinearThreshold, TestLinearMaxThreshold, newTestLogger(t))

	// Test load consistency after update
	updateIntensity := strategy.Update(TestLinearThreshold + TestLinearThreshold/2)

	loadType, loadIntensity := strategy.Load()
	if loadType != config.LinearControllerType {
		t.Errorf("expected controller type %s after update, got %s", config.LinearControllerType, loadType)
	}

	if math.Abs(loadIntensity-updateIntensity) > TestTolerance {
		t.Errorf("Load() returned different intensity than Update(): %f vs %f", loadIntensity, updateIntensity)
	}
}

func TestLinearStrategy_IntensityProgression(t *testing.T) {
	strategy := NewLinearStrategy(TestLinearThreshold, TestLinearMaxThreshold, newTestLogger(t))

	// Test that intensity increases properly as load increases
	loads := []uint64{
		TestLinearThreshold + TestLinearThreshold/8,   // 12.5% above threshold
		TestLinearThreshold + TestLinearThreshold/4,   // 25% above threshold
		TestLinearThreshold + TestLinearThreshold/2,   // 50% above threshold
		TestLinearThreshold + 3*TestLinearThreshold/4, // 75% above threshold
		TestLinearMaxThreshold,                        // 100% above threshold
	}

	var previousIntensity float64

	for i, load := range loads {
		t.Run("", func(t *testing.T) {
			intensity := strategy.Update(load)

			// Intensity should increase with load
			if i > 0 && intensity <= previousIntensity {
				t.Errorf("test %d: intensity should increase with load, got %f <= %f", i, intensity, previousIntensity)
			}

			// Should be within valid range
			if intensity < TestIntensityMin || intensity > TestIntensityMax {
				t.Errorf("test %d: intensity out of range: %f", i, intensity)
			}

			previousIntensity = intensity
		})
	}
}
