package throttler

import (
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
)

// Test configuration constants - Core throttle settings shared across all tests
const (
	// Primary throttle threshold: 1MB - this is the main decision point for when throttling begins
	TestThresholdBytes = 1_000_000 // 1MB threshold

	// Transaction and block size limits when throttling is active
	TestThrottleTxSize    = 5_000   // 5KB transaction size limit during throttling
	TestThrottleBlockSize = 21_000  // 21KB block size limit during throttling
	TestAlwaysBlockSize   = 130_000 // 130KB block size limit (always enforced)

	// Multiplier for gradual controllers (linear, quadratic) - defines max throttling point
	TestMaxThreshold = 2_000_000 // 2x threshold = maximum throttling point (2MB)
)

// Test load scenarios - All relative to TestThresholdBytes for easy understanding
const (
	TestLoadBelowThreshold    = TestThresholdBytes / 2                      // 500KB - 50% of threshold
	TestLoadAtThreshold       = TestThresholdBytes                          // 1MB - exactly at threshold
	TestLoadQuarterAbove      = TestThresholdBytes + TestThresholdBytes/4   // 1.25MB - 25% above threshold
	TestLoadHalfAbove         = TestThresholdBytes + TestThresholdBytes/2   // 1.5MB - 50% above threshold
	TestLoadThreeQuarterAbove = TestThresholdBytes + 3*TestThresholdBytes/4 // 1.75MB - 75% above threshold
	TestLoadDoubleThreshold   = TestThresholdBytes * 2                      // 2MB - 100% above threshold (max for 2x multiplier)
	TestLoadFarAbove          = TestThresholdBytes * 3                      // 3MB - far above threshold
	TestLoadBelowThresholdAlt = 800_000                                     // 800KB - alternative below threshold value
	TestLoadModerateAbove     = 1_200_000                                   // 1.2MB - moderate load above threshold
	TestLoadHighAbove         = 1_400_000                                   // 1.4MB - high load above threshold
)

// Test precision and validation constants
const (
	TestTolerance    = 0.001 // Tolerance for float comparisons
	TestIntensityMin = 0.0   // Minimum valid intensity
	TestIntensityMax = 1.0   // Maximum valid intensity
)

// PID controller test configuration
var (
	TestPIDConfig = config.PIDConfig{
		Kp:          0.2,                   // Proportional gain
		Ki:          0.1,                   // Integral gain
		Kd:          0.05,                  // Derivative gain
		IntegralMax: 100.0,                 // Maximum integral value (windup protection)
		OutputMax:   1.0,                   // Maximum output value
		SampleTime:  time.Millisecond * 10, // Minimum time between updates
	}

	TestPIDConfigResponsive = config.PIDConfig{
		Kp:          0.5, // More responsive proportional gain
		Ki:          0.2, // More responsive integral gain
		Kd:          0.1, // More responsive derivative gain
		IntegralMax: 100.0,
		OutputMax:   1.0,
		SampleTime:  time.Millisecond, // Faster sample time for responsive tests
	}
)

// Concurrency test constants
const (
	TestConcurrentGoroutines = 10
	TestConcurrentUpdates    = 100
	TestConcurrentLoadBase   = 500_000 // Base load for concurrent tests
	TestConcurrentLoadStep   = 100_000 // Load increment per goroutine
	TestConcurrentLoadInc    = 1_000   // Load increment per iteration
)

// Timing constants for PID controller tests
const (
	TestPIDSampleDelay  = time.Millisecond * 15 // Delay to ensure sample time passes
	TestPIDMicroDelay   = time.Microsecond * 10 // Small delay for concurrent PID tests
	TestPIDWindupRounds = 20                    // Number of rounds for windup protection test
)

// Common test variables - reused across multiple tests
var (
	// Standard controller configurations - reused across tests
	testStepStrategy = func(t *testing.T) *StepStrategy {
		return NewStepStrategy(TestThresholdBytes)
	}
	testLinearStrategy = func(t *testing.T) *LinearStrategy {
		return NewLinearStrategy(TestThresholdBytes, TestMaxThreshold, newTestLogger(t))
	}
	testQuadraticStrategy = func(t *testing.T) *QuadraticStrategy {
		return NewQuadraticStrategy(TestThresholdBytes, TestMaxThreshold, newTestLogger(t))
	}
	testPIDStrategy = func(t *testing.T) *PIDStrategy {
		return NewPIDStrategy(TestThresholdBytes, TestPIDConfig)
	}

	// Standard controllers - reused across tests
	testStepController = func(t *testing.T) *ThrottleController {
		return NewThrottleController(testStepStrategy(t), ThrottleConfig{})
	}
	testLinearController = func(t *testing.T) *ThrottleController {
		return NewThrottleController(testLinearStrategy(t), ThrottleConfig{})
	}
	testQuadraticController = func(t *testing.T) *ThrottleController {
		return NewThrottleController(testQuadraticStrategy(t), ThrottleConfig{})
	}
	testPIDController = func(t *testing.T) *ThrottleController {
		return NewThrottleController(testPIDStrategy(t), ThrottleConfig{})
	}

	// Test factory
	testFactory = func(t *testing.T) *ThrottleControllerFactory { return NewThrottleControllerFactory(newTestLogger(t)) }
)

func newTestLogger(t *testing.T) log.Logger {
	return testlog.Logger(t, log.LevelDebug)
}

// TestControllerFactory tests the factory pattern for creating different controller types
func TestControllerFactory(t *testing.T) {
	factory := testFactory(t)

	tests := []struct {
		name           string
		controllerType config.ThrottleControllerType
		pidConfig      *config.PIDConfig
		expectError    bool
	}{
		{
			name:           "step controller",
			controllerType: config.StepControllerType,
			pidConfig:      nil,
			expectError:    false,
		},
		{
			name:           "linear controller",
			controllerType: config.LinearControllerType,
			pidConfig:      nil,
			expectError:    false,
		},
		{
			name:           "quadratic controller",
			controllerType: config.QuadraticControllerType,
			pidConfig:      nil,
			expectError:    false,
		},
		{
			name:           "pid controller with config",
			controllerType: config.PIDControllerType,
			pidConfig:      &TestPIDConfig,
			expectError:    false,
		},
		{
			name:           "pid controller without config",
			controllerType: config.PIDControllerType,
			pidConfig:      nil,
			expectError:    true,
		},
		{
			name:           "empty controller type defaults to step",
			controllerType: "",
			pidConfig:      nil,
			expectError:    false,
		},
		{
			name:           "invalid controller type",
			controllerType: "invalid",
			pidConfig:      nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, err := factory.CreateController(
				tt.controllerType, config.ThrottleParams{
					Threshold:       TestThresholdBytes,
					MaxThreshold:    TestMaxThreshold,
					TxSize:          TestThrottleTxSize,
					BlockSize:       TestThrottleBlockSize,
					AlwaysBlockSize: TestAlwaysBlockSize,
				}, tt.pidConfig)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if controller == nil {
				t.Errorf("expected controller but got nil")
				return
			}

			// Verify the controller was created with the correct type
			expectedType := tt.controllerType
			if expectedType == "" {
				expectedType = config.StepControllerType // Default type
			}
			if controller.GetType() != expectedType {
				t.Errorf("expected type %s, got %s", expectedType, controller.GetType())
			}
		})
	}
}

// TestControllerAbstraction tests the controller abstraction layer
func TestControllerAbstraction(t *testing.T) {
	controllers := []struct {
		name       string
		controller *ThrottleController
		strategy   ThrottleStrategy
	}{
		{"step", testStepController(t), testStepStrategy(t)},
		{"linear", testLinearController(t), testLinearStrategy(t)},
		{"quadratic", testQuadraticController(t), testQuadraticStrategy(t)},
		{"pid", testPIDController(t), testPIDStrategy(t)},
	}

	for _, ctrl := range controllers {
		t.Run(ctrl.name, func(t *testing.T) {
			// Test that controller properly delegates to strategy
			controllerParams := ctrl.controller.Update(TestLoadHalfAbove)

			// Reset strategy to same state and test directly
			ctrl.strategy.Reset()
			if ctrl.name == "pid" {
				time.Sleep(TestPIDSampleDelay) // Allow sample time for PID
			}
			strategyParams := ctrl.strategy.Update(TestLoadHalfAbove)

			// Controller should produce same results as direct strategy call
			if controllerParams.Intensity != strategyParams {
				t.Errorf("controller/strategy intensity mismatch: %f != %f", controllerParams.Intensity, strategyParams)
			}

			// Test Load() method consistency
			controllerType, loadParams := ctrl.controller.Load()
			if controllerType != ctrl.strategy.GetType() {
				t.Errorf("Load() type mismatch: %s != %s", controllerType, ctrl.strategy.GetType())
			}
			if loadParams.Intensity != controllerParams.Intensity {
				t.Errorf("Load() intensity mismatch: %f != %f", loadParams.Intensity, controllerParams.Intensity)
			}

			// Test Reset() method
			ctrl.controller.Reset()
			resetType, resetParams := ctrl.controller.Load()
			if resetType != ctrl.strategy.GetType() {
				t.Errorf("Reset() type changed: %s != %s", resetType, ctrl.strategy.GetType())
			}
			if resetParams.Intensity != TestIntensityMin {
				t.Errorf("Reset() should return zero intensity, got %f", resetParams.Intensity)
			}
		})
	}
}

// TestControllerStrategySwapping tests changing strategies at runtime
func TestControllerStrategySwapping(t *testing.T) {
	// Start with step controller
	stepStrategy := testStepStrategy(t)
	controller := NewThrottleController(stepStrategy, ThrottleConfig{})

	// Test initial behavior
	params := controller.Update(TestLoadHalfAbove)
	if params.Intensity != TestIntensityMax {
		t.Errorf("expected step controller intensity %f, got %f", TestIntensityMax, params.Intensity)
	}

	// Switch to quadratic controller
	resetParams := ThrottleParams{MaxTxSize: 0, MaxBlockSize: TestAlwaysBlockSize, Intensity: 0.0}
	controller.SetStrategy(testQuadraticStrategy(t), resetParams)

	// Test new behavior
	params = controller.Update(TestLoadHalfAbove)
	expectedQuadraticIntensity := 0.25
	if params.Intensity != expectedQuadraticIntensity {
		t.Errorf("expected quadratic controller intensity %f, got %f", expectedQuadraticIntensity, params.Intensity)
	}

	// Verify Load() method returns correct parameters after switch
	controllerType, loadedParams := controller.Load()
	if controllerType != config.QuadraticControllerType {
		t.Errorf("expected controller type %s, got %s", config.QuadraticControllerType, controllerType)
	}
	if loadedParams.Intensity != params.Intensity {
		t.Errorf("expected loaded intensity %f, got %f", params.Intensity, loadedParams.Intensity)
	}
}

// TestControllerTypeConsistency tests that controller types are reported consistently
func TestControllerTypeConsistency(t *testing.T) {
	factory := testFactory(t)

	testCases := []struct {
		controllerType config.ThrottleControllerType
		pidConfig      *config.PIDConfig
	}{
		{config.StepControllerType, nil},
		{config.LinearControllerType, nil},
		{config.QuadraticControllerType, nil},
		{config.PIDControllerType, &TestPIDConfig},
	}

	for _, tc := range testCases {
		t.Run(string(tc.controllerType), func(t *testing.T) {
			controller, err := factory.CreateController(
				tc.controllerType, config.ThrottleParams{
					Threshold:       TestThresholdBytes,
					MaxThreshold:    TestMaxThreshold,
					TxSize:          TestThrottleTxSize,
					BlockSize:       TestThrottleBlockSize,
					AlwaysBlockSize: TestAlwaysBlockSize,
				}, tc.pidConfig)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check consistency across different methods
			if controller.GetType() != tc.controllerType {
				t.Errorf("GetType() returned %s, expected %s", controller.GetType(), tc.controllerType)
			}

			loadType, _ := controller.Load()
			if loadType != tc.controllerType {
				t.Errorf("Load() returned type %s, expected %s", loadType, tc.controllerType)
			}
		})
	}
}

// Mock metrics implementation for testing
type mockMetrics struct {
	lastError      float64
	lastIntegral   float64
	lastDerivative float64
	responseTime   time.Duration
	mu             sync.RWMutex
}

func (m *mockMetrics) RecordThrottleControllerState(error, integral, derivative float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastError = error
	m.lastIntegral = integral
	m.lastDerivative = derivative
}

func (m *mockMetrics) RecordThrottleResponseTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responseTime = duration
}

func (m *mockMetrics) GetLastError() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastError
}

func (m *mockMetrics) GetResponseTime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.responseTime
}

// TestIntensityToParams tests the intensityToParams function that converts intensity to ThrottleParams
func TestIntensityToParams(t *testing.T) {
	testConfig := ThrottleConfig{
		Threshold:         TestThresholdBytes,
		ThrottleTxSize:    TestThrottleTxSize,
		ThrottleBlockSize: TestThrottleBlockSize,
		AlwaysBlockSize:   TestAlwaysBlockSize,
	}

	controller := NewThrottleController(testStepStrategy(t), testConfig)

	tests := []struct {
		name                 string
		intensity            float64
		expectedMaxTxSize    uint64
		expectedMaxBlockSize uint64
		expectedIntensity    float64
	}{
		{
			name:                 "zero intensity",
			intensity:            0.0,
			expectedMaxTxSize:    0,
			expectedMaxBlockSize: TestAlwaysBlockSize,
			expectedIntensity:    0.0,
		},
		{
			name:                 "minimum positive intensity",
			intensity:            0.001,
			expectedMaxTxSize:    TestThrottleTxSize,
			expectedMaxBlockSize: TestAlwaysBlockSize - uint64(0.001*float64(TestAlwaysBlockSize-TestThrottleBlockSize)), // Interpolated value
			expectedIntensity:    0.001,
		},
		{
			name:                 "half intensity",
			intensity:            0.5,
			expectedMaxTxSize:    TestThrottleTxSize,
			expectedMaxBlockSize: TestAlwaysBlockSize - uint64(0.5*float64(TestAlwaysBlockSize-TestThrottleBlockSize)), // Interpolated value
			expectedIntensity:    0.5,
		},
		{
			name:                 "maximum intensity",
			intensity:            1.0,
			expectedMaxTxSize:    TestThrottleTxSize,
			expectedMaxBlockSize: TestThrottleBlockSize,
			expectedIntensity:    1.0,
		},
		{
			name:                 "intensity above maximum (should be clamped)",
			intensity:            1.5,
			expectedMaxTxSize:    TestThrottleTxSize,
			expectedMaxBlockSize: TestThrottleBlockSize,
			expectedIntensity:    1.0,
		},
		{
			name:                 "negative intensity",
			intensity:            -0.5,
			expectedMaxTxSize:    0,
			expectedMaxBlockSize: TestAlwaysBlockSize,
			expectedIntensity:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := controller.intensityToParams(tt.intensity, testConfig)

			if params.MaxTxSize != tt.expectedMaxTxSize {
				t.Errorf("expected MaxTxSize %d, got %d", tt.expectedMaxTxSize, params.MaxTxSize)
			}

			if params.MaxBlockSize != tt.expectedMaxBlockSize {
				t.Errorf("expected MaxBlockSize %d, got %d", tt.expectedMaxBlockSize, params.MaxBlockSize)
			}

			if params.Intensity != tt.expectedIntensity {
				t.Errorf("expected Intensity %f, got %f", tt.expectedIntensity, params.Intensity)
			}
		})
	}
}

// TestIntensityToParamsBlockSizeInterpolation tests block size interpolation when ThrottleBlockSize is less than AlwaysBlockSize
func TestIntensityToParamsBlockSizeInterpolation(t *testing.T) {
	testConfig := ThrottleConfig{
		Threshold:         TestThresholdBytes,
		ThrottleTxSize:    TestThrottleTxSize,
		ThrottleBlockSize: 50_000,  // 50KB throttle block size
		AlwaysBlockSize:   100_000, // 100KB always block size
	}

	controller := NewThrottleController(testStepStrategy(t), testConfig)

	tests := []struct {
		name                 string
		intensity            float64
		expectedMaxBlockSize uint64
		tolerance            uint64
	}{
		{
			name:                 "zero intensity - always block size",
			intensity:            0.0,
			expectedMaxBlockSize: 100_000,
			tolerance:            0,
		},
		{
			name:                 "25% intensity - 75% of way to throttle size",
			intensity:            0.25,
			expectedMaxBlockSize: 87_500, // 100_000 - 0.25 * (100_000 - 50_000)
			tolerance:            100,
		},
		{
			name:                 "50% intensity - 50% of way to throttle size",
			intensity:            0.5,
			expectedMaxBlockSize: 75_000, // 100_000 - 0.5 * (100_000 - 50_000)
			tolerance:            100,
		},
		{
			name:                 "75% intensity - 25% of way to throttle size",
			intensity:            0.75,
			expectedMaxBlockSize: 62_500, // 100_000 - 0.75 * (100_000 - 50_000)
			tolerance:            100,
		},
		{
			name:                 "100% intensity - throttle block size",
			intensity:            1.0,
			expectedMaxBlockSize: 50_000,
			tolerance:            0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := controller.intensityToParams(tt.intensity, testConfig)

			if params.MaxBlockSize > tt.expectedMaxBlockSize+tt.tolerance ||
				params.MaxBlockSize < tt.expectedMaxBlockSize-tt.tolerance {
				t.Errorf("expected MaxBlockSize %d ± %d, got %d",
					tt.expectedMaxBlockSize, tt.tolerance, params.MaxBlockSize)
			}

			if params.Intensity != tt.intensity {
				t.Errorf("expected Intensity %f, got %f", tt.intensity, params.Intensity)
			}
		})
	}
}

// TestIntensityToParamsEdgeCases tests edge cases for the intensityToParams function
func TestIntensityToParamsEdgeCases(t *testing.T) {
	t.Run("zero throttle block size", func(t *testing.T) {
		testConfig := ThrottleConfig{
			Threshold:         TestThresholdBytes,
			ThrottleTxSize:    TestThrottleTxSize,
			ThrottleBlockSize: 0,
			AlwaysBlockSize:   TestAlwaysBlockSize,
		}

		controller := NewThrottleController(testStepStrategy(t), testConfig)
		params := controller.intensityToParams(0.5, testConfig)

		if params.MaxBlockSize != TestAlwaysBlockSize {
			t.Errorf("expected MaxBlockSize %d with zero throttle block size, got %d",
				TestAlwaysBlockSize, params.MaxBlockSize)
		}
	})

	t.Run("throttle block size greater than always block size", func(t *testing.T) {
		testConfig := ThrottleConfig{
			Threshold:         TestThresholdBytes,
			ThrottleTxSize:    TestThrottleTxSize,
			ThrottleBlockSize: TestAlwaysBlockSize + 50_000, // Greater than always size
			AlwaysBlockSize:   TestAlwaysBlockSize,
		}

		controller := NewThrottleController(testStepStrategy(t), testConfig)
		params := controller.intensityToParams(0.5, testConfig)

		// Should use always block size when throttle block size is greater
		if params.MaxBlockSize != TestAlwaysBlockSize {
			t.Errorf("expected MaxBlockSize %d when throttle > always, got %d",
				TestAlwaysBlockSize, params.MaxBlockSize)
		}
	})

	t.Run("zero always block size", func(t *testing.T) {
		testConfig := ThrottleConfig{
			Threshold:         TestThresholdBytes,
			ThrottleTxSize:    TestThrottleTxSize,
			ThrottleBlockSize: TestThrottleBlockSize,
			AlwaysBlockSize:   0,
		}

		controller := NewThrottleController(testStepStrategy(t), testConfig)
		params := controller.intensityToParams(0.5, testConfig)

		if params.MaxBlockSize != TestThrottleBlockSize {
			t.Errorf("expected MaxBlockSize %d with zero always block size, got %d",
				TestThrottleBlockSize, params.MaxBlockSize)
		}
	})
}

// TestIntensityToParamsConsistency tests that intensityToParams produces consistent results
func TestIntensityToParamsConsistency(t *testing.T) {
	testConfig := ThrottleConfig{
		Threshold:         TestThresholdBytes,
		ThrottleTxSize:    TestThrottleTxSize,
		ThrottleBlockSize: TestThrottleBlockSize,
		AlwaysBlockSize:   TestAlwaysBlockSize,
	}

	controller := NewThrottleController(testStepStrategy(t), testConfig)

	// Test that calling intensityToParams multiple times with same input produces same output
	intensity := 0.7
	params1 := controller.intensityToParams(intensity, testConfig)
	params2 := controller.intensityToParams(intensity, testConfig)

	if params1.MaxTxSize != params2.MaxTxSize {
		t.Errorf("inconsistent MaxTxSize: %d != %d", params1.MaxTxSize, params2.MaxTxSize)
	}

	if params1.MaxBlockSize != params2.MaxBlockSize {
		t.Errorf("inconsistent MaxBlockSize: %d != %d", params1.MaxBlockSize, params2.MaxBlockSize)
	}

	if params1.Intensity != params2.Intensity {
		t.Errorf("inconsistent Intensity: %f != %f", params1.Intensity, params2.Intensity)
	}
}

// TestIntensityToParamsThreadSafety tests that intensityToParams is thread-safe
func TestIntensityToParamsThreadSafety(t *testing.T) {
	testConfig := ThrottleConfig{
		Threshold:         TestThresholdBytes,
		ThrottleTxSize:    TestThrottleTxSize,
		ThrottleBlockSize: TestThrottleBlockSize,
		AlwaysBlockSize:   TestAlwaysBlockSize,
	}

	controller := NewThrottleController(testStepStrategy(t), testConfig)

	// Run multiple goroutines calling intensityToParams concurrently
	const numGoroutines = 10
	const numCalls = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineId int) {
			defer wg.Done()

			intensity := float64(goroutineId) / float64(numGoroutines) // Different intensity per goroutine

			for j := 0; j < numCalls; j++ {
				params := controller.intensityToParams(intensity, testConfig)

				// Verify the params are reasonable
				if params.Intensity != intensity {
					t.Errorf("goroutine %d call %d: expected intensity %f, got %f",
						goroutineId, j, intensity, params.Intensity)
				}

				if intensity > 0 && params.MaxTxSize != TestThrottleTxSize {
					t.Errorf("goroutine %d call %d: expected MaxTxSize %d, got %d",
						goroutineId, j, TestThrottleTxSize, params.MaxTxSize)
				}
			}
		}(i)
	}

	wg.Wait()
}
