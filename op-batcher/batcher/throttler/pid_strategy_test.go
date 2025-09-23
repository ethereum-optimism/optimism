package throttler

import (
	"math"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
)

// Test constants specific to PID strategy
const (
	TestPIDThreshold    = 700_000 // 700KB threshold for PID strategy tests
	TestPIDTxSize       = 4_500   // 4.5KB transaction size limit
	TestPIDBlockSize    = 20_000  // 20KB block size limit
	TestPIDAlwaysSize   = 115_000 // 115KB always block size
	TestPIDSampleTimeMs = 5       // 5ms sample time for testing
)

// Test PID configurations
var (
	TestPIDConfigBasic = config.PIDConfig{
		Kp:          0.3,                                    // Basic proportional gain
		Ki:          0.15,                                   // Basic integral gain
		Kd:          0.08,                                   // Basic derivative gain
		IntegralMax: 50.0,                                   // Integral windup protection
		OutputMax:   1.0,                                    // Maximum output
		SampleTime:  time.Millisecond * TestPIDSampleTimeMs, // Sample time
	}

	TestPIDConfigAggressive = config.PIDConfig{
		Kp:          0.8,                                    // Aggressive proportional gain
		Ki:          0.4,                                    // Aggressive integral gain
		Kd:          0.2,                                    // Aggressive derivative gain
		IntegralMax: 100.0,                                  // Higher integral limit
		OutputMax:   1.0,                                    // Maximum output
		SampleTime:  time.Millisecond * TestPIDSampleTimeMs, // Sample time
	}

	TestPIDConfigConservative = config.PIDConfig{
		Kp:          0.1,                                    // Conservative proportional gain
		Ki:          0.05,                                   // Conservative integral gain
		Kd:          0.02,                                   // Conservative derivative gain
		IntegralMax: 25.0,                                   // Lower integral limit
		OutputMax:   1.0,                                    // Maximum output
		SampleTime:  time.Millisecond * TestPIDSampleTimeMs, // Sample time
	}
)

func TestPIDStrategy_NewPIDStrategy(t *testing.T) {
	strategy := NewPIDStrategy(TestPIDThreshold, TestPIDConfigBasic)

	if strategy.threshold != TestPIDThreshold {
		t.Errorf("expected threshold %d, got %d", TestPIDThreshold, strategy.threshold)
	}

	if strategy.config.Kp != TestPIDConfigBasic.Kp {
		t.Errorf("expected Kp %f, got %f", TestPIDConfigBasic.Kp, strategy.config.Kp)
	}

	if strategy.config.Ki != TestPIDConfigBasic.Ki {
		t.Errorf("expected Ki %f, got %f", TestPIDConfigBasic.Ki, strategy.config.Ki)
	}

	if strategy.config.Kd != TestPIDConfigBasic.Kd {
		t.Errorf("expected Kd %f, got %f", TestPIDConfigBasic.Kd, strategy.config.Kd)
	}

	// Test initial state
	controllerType, intensity := strategy.Load()
	if controllerType != config.PIDControllerType {
		t.Errorf("expected controller type %s, got %s", config.PIDControllerType, controllerType)
	}

	if intensity != TestIntensityMin {
		t.Errorf("expected initial intensity %f, got %f", TestIntensityMin, intensity)
	}

	if strategy.initialized {
		t.Error("expected strategy to be uninitialized")
	}
}

func TestPIDStrategy_Update(t *testing.T) {
	strategy := NewPIDStrategy(TestPIDThreshold, TestPIDConfigBasic)

	t.Run("below threshold", func(t *testing.T) {
		intensity := strategy.Update(TestPIDThreshold / 2)

		if intensity != TestIntensityMin {
			t.Errorf("expected no throttling below threshold, got intensity %f", intensity)
		}
	})

	t.Run("first update above threshold", func(t *testing.T) {
		strategy.Reset()

		intensity := strategy.Update(TestPIDThreshold * 2)

		// First update should initialize the controller
		if intensity <= TestIntensityMin {
			t.Errorf("expected positive intensity above threshold, got %f", intensity)
		}
	})

	t.Run("subsequent updates", func(t *testing.T) {
		strategy.Reset()

		// First update to initialize
		time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
		intensity1 := strategy.Update(TestPIDThreshold * 2)

		// Second update after sample time
		time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
		intensity2 := strategy.Update(TestPIDThreshold * 3) // Higher error

		// Second update should respond to increased error
		if intensity2 <= intensity1 {
			t.Errorf("expected increased intensity with higher error: %f -> %f", intensity1, intensity2)
		}
	})

	t.Run("sample time enforcement", func(t *testing.T) {
		strategy.Reset()

		// First update
		time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
		intensity1 := strategy.Update(TestPIDThreshold * 2)

		// Second update immediately (within sample time)
		intensity2 := strategy.Update(TestPIDThreshold * 3)

		// Should return same result due to sample time
		if intensity2 != intensity1 {
			t.Errorf("expected same intensity within sample time: %f != %f", intensity1, intensity2)
		}
	})
}

func TestPIDStrategy_GetType(t *testing.T) {
	strategy := NewPIDStrategy(TestPIDThreshold, TestPIDConfigBasic)

	if strategy.GetType() != config.PIDControllerType {
		t.Errorf("expected GetType() to return %s, got %s", config.PIDControllerType, strategy.GetType())
	}
}

func TestPIDStrategy_Reset(t *testing.T) {
	strategy := NewPIDStrategy(TestPIDThreshold, TestPIDConfigBasic)

	// Build up some state
	time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
	strategy.Update(TestPIDThreshold * 2)

	// Reset should clear state
	strategy.Reset()

	if strategy.initialized {
		t.Error("expected strategy to be uninitialized after reset")
	}

	if strategy.lastError != 0 {
		t.Errorf("expected lastError to be 0 after reset, got %f", strategy.lastError)
	}

	if strategy.integral != 0 {
		t.Errorf("expected integral to be 0 after reset, got %f", strategy.integral)
	}

	controllerType, intensity := strategy.Load()
	if controllerType != config.PIDControllerType {
		t.Errorf("expected controller type %s after reset, got %s", config.PIDControllerType, controllerType)
	}

	if intensity != TestIntensityMin {
		t.Errorf("expected intensity %f after reset, got %f", TestIntensityMin, intensity)
	}
}

func TestPIDStrategy_IntegralWindup(t *testing.T) {
	strategy := NewPIDStrategy(TestPIDThreshold, TestPIDConfigBasic)

	// Simulate sustained high error to test integral windup protection
	for i := 0; i < 20; i++ {
		time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
		strategy.Update(TestPIDThreshold * 10) // Very high error
	}

	// Integral should be clamped to IntegralMax
	if math.Abs(strategy.integral) > TestPIDConfigBasic.IntegralMax {
		t.Errorf("integral exceeded IntegralMax: %f > %f", math.Abs(strategy.integral), TestPIDConfigBasic.IntegralMax)
	}

	// Output should be clamped to OutputMax
	intensity := strategy.Update(TestPIDThreshold * 10)
	if intensity > TestPIDConfigBasic.OutputMax {
		t.Errorf("intensity exceeded OutputMax: %f > %f", intensity, TestPIDConfigBasic.OutputMax)
	}
}

func TestPIDStrategy_ResponseToErrorDecrease(t *testing.T) {
	strategy := NewPIDStrategy(TestPIDThreshold, TestPIDConfigBasic)

	// Build up intensity with high error
	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
		strategy.Update(TestPIDThreshold * 5)
	}

	// Get current intensity
	time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
	highErrorIntensity := strategy.Update(TestPIDThreshold * 5)

	// Reduce error significantly
	time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
	lowErrorIntensity := strategy.Update(TestPIDThreshold + TestPIDThreshold/10)

	// Intensity should decrease with reduced error
	if lowErrorIntensity >= highErrorIntensity {
		t.Errorf("expected reduced intensity with lower error: %f >= %f", lowErrorIntensity, highErrorIntensity)
	}
}

func TestPIDStrategy_DifferentConfigurations(t *testing.T) {
	configs := []struct {
		name   string
		config config.PIDConfig
	}{
		{"basic", TestPIDConfigBasic},
		{"aggressive", TestPIDConfigAggressive},
		{"conservative", TestPIDConfigConservative},
	}

	for _, cfg := range configs {
		t.Run(cfg.name, func(t *testing.T) {
			strategy := NewPIDStrategy(TestPIDThreshold, cfg.config)

			// Test response to same error
			time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
			intensity := strategy.Update(TestPIDThreshold * 2)

			// Should respond to error above threshold
			if intensity <= TestIntensityMin {
				t.Errorf("expected positive intensity for %s config, got %f", cfg.name, intensity)
			}

			// Should not exceed limits
			if intensity > cfg.config.OutputMax {
				t.Errorf("intensity exceeded OutputMax for %s config: %f > %f", cfg.name, intensity, cfg.config.OutputMax)
			}
		})
	}
}

func TestPIDStrategy_Load(t *testing.T) {
	strategy := NewPIDStrategy(TestPIDThreshold, TestPIDConfigBasic)

	// Test load consistency after update
	time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
	updateIntensity := strategy.Update(TestPIDThreshold * 2)

	loadType, loadIntensity := strategy.Load()
	if loadType != config.PIDControllerType {
		t.Errorf("expected controller type %s after update, got %s", config.PIDControllerType, loadType)
	}

	if math.Abs(loadIntensity-updateIntensity) > TestTolerance {
		t.Errorf("Load() returned different intensity than Update(): %f vs %f", loadIntensity, updateIntensity)
	}
}

func TestPIDStrategy_SetMetrics(t *testing.T) {
	strategy := NewPIDStrategy(TestPIDThreshold, TestPIDConfigBasic)

	// Setup mock metrics
	metrics := &mockMetrics{}
	strategy.SetMetrics(metrics)

	// Trigger updates to generate metrics
	time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
	strategy.Update(TestPIDThreshold * 2) // Initialize

	time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
	strategy.Update(TestPIDThreshold * 2) // Generate metrics

	// Check that metrics were recorded
	if metrics.GetLastError() <= 0 {
		t.Errorf("expected metrics to record positive error, got %f", metrics.GetLastError())
	}

	if metrics.GetResponseTime() <= 0 {
		t.Errorf("expected metrics to record positive response time, got %v", metrics.GetResponseTime())
	}
}

func TestPIDStrategy_ErrorCalculation(t *testing.T) {
	strategy := NewPIDStrategy(TestPIDThreshold, TestPIDConfigBasic)

	testCases := []struct {
		name         string
		pendingBytes uint64
		targetBytes  uint64
		expectError  bool
	}{
		{
			name:         "no error at threshold",
			pendingBytes: TestPIDThreshold,
			targetBytes:  TestPIDThreshold,
			expectError:  false,
		},
		{
			name:         "error above threshold",
			pendingBytes: TestPIDThreshold * 2,
			targetBytes:  TestPIDThreshold,
			expectError:  true,
		},
		{
			name:         "no error below threshold",
			pendingBytes: TestPIDThreshold / 2,
			targetBytes:  TestPIDThreshold,
			expectError:  false,
		},
		{
			name:         "error with different target",
			pendingBytes: TestPIDThreshold * 2,
			targetBytes:  TestPIDThreshold / 2,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			strategy.Reset()

			time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
			intensity := strategy.Update(tc.pendingBytes)

			if tc.expectError {
				if intensity <= TestIntensityMin {
					t.Errorf("expected positive intensity for error case, got %f", intensity)
				}
			} else {
				if intensity != TestIntensityMin {
					t.Errorf("expected zero intensity for no error case, got %f", intensity)
				}
			}
		})
	}
}

func TestPIDStrategy_DerivativeComponent(t *testing.T) {
	// Use config with higher derivative gain to test derivative component
	derivativeConfig := config.PIDConfig{
		Kp:          0.1,  // Low proportional
		Ki:          0.01, // Low integral
		Kd:          0.5,  // High derivative
		IntegralMax: 50.0,
		OutputMax:   1.0,
		SampleTime:  time.Millisecond * TestPIDSampleTimeMs,
	}

	strategy := NewPIDStrategy(TestPIDThreshold, derivativeConfig)

	// Initialize with moderate error
	time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
	intensity1 := strategy.Update(TestPIDThreshold * 2)

	// Increase error rapidly (should trigger derivative response)
	time.Sleep(time.Millisecond * TestPIDSampleTimeMs * 2)
	intensity2 := strategy.Update(TestPIDThreshold * 5)

	// With high derivative gain, should respond strongly to rapid error change
	if intensity2 <= intensity1 {
		t.Errorf("expected derivative component to increase response: %f <= %f", intensity2, intensity1)
	}
}
