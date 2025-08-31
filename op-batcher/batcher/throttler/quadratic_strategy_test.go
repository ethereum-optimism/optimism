package throttler

import (
	"math"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/stretchr/testify/require"
)

// Test constants specific to quadratic strategy
const (
	TestQuadraticThreshold    = 600_000                                                           // 600KB threshold for quadratic strategy tests
	TestQuadraticTxSize       = 3_500                                                             // 3.5KB transaction size limit
	TestQuadraticBlockSize    = 16_000                                                            // 16KB block size limit
	TestQuadraticAlwaysSize   = 110_000                                                           // 110KB always block size
	TestQuadraticMultiplier   = 2.0                                                               // 3x threshold multiplier
	TestQuadraticMaxThreshold = uint64(float64(TestQuadraticThreshold) * TestQuadraticMultiplier) // 1.8MB max threshold
)

func TestQuadraticStrategy_NewQuadraticStrategy(t *testing.T) {
	strategy := NewQuadraticStrategy(TestQuadraticThreshold, TestQuadraticMaxThreshold, newTestLogger(t))

	if strategy.lowerThreshold != TestQuadraticThreshold {
		t.Errorf("expected threshold %d, got %d", TestQuadraticThreshold, strategy.lowerThreshold)
	}

	if strategy.upperThreshold != TestQuadraticMaxThreshold {
		t.Errorf("expected maxThreshold %d, got %d", TestQuadraticMaxThreshold, strategy.upperThreshold)
	}

	// Test initial state
	controllerType, intensity := strategy.Load()
	if controllerType != config.QuadraticControllerType {
		t.Errorf("expected controller type %s, got %s", config.QuadraticControllerType, controllerType)
	}

	if intensity != TestIntensityMin {
		t.Errorf("expected initial intensity %f, got %f", TestIntensityMin, intensity)
	}
}

func TestQuadraticStrategy_Update(t *testing.T) {
	strategy := NewQuadraticStrategy(TestQuadraticThreshold, TestQuadraticMaxThreshold, newTestLogger(t))

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
			pendingBytes:      TestQuadraticThreshold / 2,
			targetBytes:       0,
			expectedIntensity: TestIntensityMin,
		},
		{
			name:              "exactly at threshold",
			pendingBytes:      TestQuadraticThreshold,
			targetBytes:       0,
			expectedIntensity: TestIntensityMin,
		},
		{
			name:              "25% above threshold",
			pendingBytes:      TestQuadraticThreshold + TestQuadraticThreshold/4,
			targetBytes:       0,
			expectedIntensity: 0.0625, // (0.25)^2
		},
		{
			name:              "50% above threshold",
			pendingBytes:      TestQuadraticThreshold + TestQuadraticThreshold/2,
			targetBytes:       0,
			expectedIntensity: 0.25, // (0.5)^2
		},
		{
			name:              "75% above threshold",
			pendingBytes:      TestQuadraticThreshold + 3*TestQuadraticThreshold/4,
			targetBytes:       0,
			expectedIntensity: 0.5625, // (0.75)^2
		},
		{
			name:              "100% above threshold (max)",
			pendingBytes:      TestQuadraticMaxThreshold,
			targetBytes:       0,
			expectedIntensity: TestIntensityMax,
		},
		{
			name:              "beyond max threshold",
			pendingBytes:      TestQuadraticMaxThreshold * 2,
			targetBytes:       0,
			expectedIntensity: TestIntensityMax,
		},
		{
			name:              "with target bytes ignored",
			pendingBytes:      TestQuadraticThreshold + TestQuadraticThreshold/2,
			targetBytes:       TestQuadraticThreshold * 10, // Target bytes should be ignored
			expectedIntensity: 0.25,                        // (0.5)^2
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

func TestQuadraticStrategy_QuadraticScaling(t *testing.T) {
	strategy := NewQuadraticStrategy(TestQuadraticThreshold, TestQuadraticMaxThreshold, newTestLogger(t))

	// Test that intensity scales quadratically between threshold and maxThreshold
	testPoints := []struct {
		linearRatio       float64
		expectedIntensity float64
	}{
		{0.0, 0.0},     // At threshold
		{0.25, 0.0625}, // 25% of way to max -> (0.25)^2
		{0.5, 0.25},    // 50% of way to max -> (0.5)^2
		{0.75, 0.5625}, // 75% of way to max -> (0.75)^2
		{1.0, 1.0},     // At max threshold -> (1.0)^2
	}

	for _, tp := range testPoints {
		t.Run("", func(t *testing.T) {
			// Calculate load based on linear ratio
			load := TestQuadraticThreshold + uint64(tp.linearRatio*float64(TestQuadraticMaxThreshold-TestQuadraticThreshold))
			intensity := strategy.Update(load)

			if math.Abs(intensity-tp.expectedIntensity) > TestTolerance {
				t.Errorf("linear ratio %.2f: expected intensity %f, got %f", tp.linearRatio, tp.expectedIntensity, intensity)
			}
		})
	}
}

func TestQuadraticStrategy_GetType(t *testing.T) {
	strategy := NewQuadraticStrategy(TestQuadraticThreshold, TestQuadraticMaxThreshold, newTestLogger(t))

	if strategy.GetType() != config.QuadraticControllerType {
		t.Errorf("expected GetType() to return %s, got %s", config.QuadraticControllerType, strategy.GetType())
	}
}

func TestQuadraticStrategy_Reset(t *testing.T) {
	strategy := NewQuadraticStrategy(TestQuadraticThreshold, TestQuadraticMaxThreshold, newTestLogger(t))

	// Update to build some state
	strategy.Update(TestQuadraticMaxThreshold)

	// Reset should restore to initial state
	strategy.Reset()

	controllerType, intensity := strategy.Load()
	if controllerType != config.QuadraticControllerType {
		t.Errorf("expected controller type %s after reset, got %s", config.QuadraticControllerType, controllerType)
	}

	if intensity != TestIntensityMin {
		t.Errorf("expected intensity %f after reset, got %f", TestIntensityMin, intensity)
	}
}

func TestQuadraticStrategy_EdgeCases(t *testing.T) {
	t.Run("max threshold less than threshold", func(t *testing.T) {
		require.Panics(t, func() {
			// Test when multiplier results in maxThreshold <= threshold
			NewQuadraticStrategy(TestQuadraticThreshold, 0, newTestLogger(t))
		})
	})

	t.Run("very large multiplier", func(t *testing.T) {
		strategy := NewQuadraticStrategy(TestLinearThreshold, TestLinearThreshold*2000, newTestLogger(t))

		// Even with very large multiplier, should work correctly
		intensity := strategy.Update(TestLinearThreshold * 2)

		// Should be very low intensity due to large range and linear scaling
		if intensity > 0.05 {
			t.Errorf("expected very low intensity with large multiplier, got %f", intensity)
		}
	})
}

func TestQuadraticStrategy_Load(t *testing.T) {
	strategy := NewQuadraticStrategy(TestQuadraticThreshold, TestQuadraticMaxThreshold, newTestLogger(t))

	// Test load consistency after update
	updateIntensity := strategy.Update(TestQuadraticThreshold + TestQuadraticThreshold/2)

	loadType, loadIntensity := strategy.Load()
	if loadType != config.QuadraticControllerType {
		t.Errorf("expected controller type %s after update, got %s", config.QuadraticControllerType, loadType)
	}

	if math.Abs(loadIntensity-updateIntensity) > TestTolerance {
		t.Errorf("Load() returned different intensity than Update(): %f vs %f", loadIntensity, updateIntensity)
	}
}

func TestQuadraticStrategy_IntensityProgression(t *testing.T) {
	strategy := NewQuadraticStrategy(TestQuadraticThreshold, TestQuadraticMaxThreshold, newTestLogger(t))

	// Test that intensity increases properly as load increases
	loads := []uint64{
		TestQuadraticThreshold + TestQuadraticThreshold/8,   // 12.5% above threshold
		TestQuadraticThreshold + TestQuadraticThreshold/4,   // 25% above threshold
		TestQuadraticThreshold + TestQuadraticThreshold/2,   // 50% above threshold
		TestQuadraticThreshold + 3*TestQuadraticThreshold/4, // 75% above threshold
		TestQuadraticMaxThreshold,                           // 100% above threshold
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
