package batcher

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/ethereum/go-ethereum/log"
)

// noopLogger creates a logger that discards all output for testing
func noopLogger() log.Logger {
	return log.Root()
}

// TestStepController tests the step controller behavior
func TestStepController(t *testing.T) {
	strategy := NewStepStrategy(1000000, 5000, 21000, 130000)
	controller := NewThrottleController(strategy)

	tests := []struct {
		name              string
		pendingBytes      uint64
		expectedIntensity float64
		expectedMaxTxSize uint64
		expectedMaxBlock  uint64
	}{
		{
			name:              "below threshold",
			pendingBytes:      500000,
			expectedIntensity: 0.0,
			expectedMaxTxSize: 0,
			expectedMaxBlock:  130000,
		},
		{
			name:              "at threshold",
			pendingBytes:      1000000,
			expectedIntensity: 0.0,
			expectedMaxTxSize: 0,
			expectedMaxBlock:  130000,
		},
		{
			name:              "above threshold",
			pendingBytes:      1500000,
			expectedIntensity: 1.0,
			expectedMaxTxSize: 5000,
			expectedMaxBlock:  21000,
		},
		{
			name:              "far above threshold",
			pendingBytes:      3000000,
			expectedIntensity: 1.0,
			expectedMaxTxSize: 5000,
			expectedMaxBlock:  21000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := controller.Update(tt.pendingBytes, 0)

			if params.Intensity != tt.expectedIntensity {
				t.Errorf("expected intensity %f, got %f", tt.expectedIntensity, params.Intensity)
			}
			if params.MaxTxSize != tt.expectedMaxTxSize {
				t.Errorf("expected MaxTxSize %d, got %d", tt.expectedMaxTxSize, params.MaxTxSize)
			}
			if params.MaxBlockSize != tt.expectedMaxBlock {
				t.Errorf("expected MaxBlockSize %d, got %d", tt.expectedMaxBlock, params.MaxBlockSize)
			}
		})
	}
}

// TestLinearController tests the linear controller behavior
func TestLinearController(t *testing.T) {
	strategy := NewLinearStrategy(1000000, 5000, 21000, 130000, 2, noopLogger())
	controller := NewThrottleController(strategy)

	tests := []struct {
		name              string
		pendingBytes      uint64
		expectedIntensity float64
		tolerance         float64
	}{
		{
			name:              "below threshold",
			pendingBytes:      500000,
			expectedIntensity: 0.0,
			tolerance:         0.001,
		},
		{
			name:              "at threshold",
			pendingBytes:      1000000,
			expectedIntensity: 0.0,
			tolerance:         0.001,
		},
		{
			name:              "25% above threshold",
			pendingBytes:      1250000,
			expectedIntensity: 0.25,
			tolerance:         0.001,
		},
		{
			name:              "50% above threshold",
			pendingBytes:      1500000,
			expectedIntensity: 0.5,
			tolerance:         0.001,
		},
		{
			name:              "100% above threshold (max)",
			pendingBytes:      2000000,
			expectedIntensity: 1.0,
			tolerance:         0.001,
		},
		{
			name:              "beyond max threshold",
			pendingBytes:      3000000,
			expectedIntensity: 1.0,
			tolerance:         0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := controller.Update(tt.pendingBytes, 0)

			if math.Abs(params.Intensity-tt.expectedIntensity) > tt.tolerance {
				t.Errorf("expected intensity %f ± %f, got %f",
					tt.expectedIntensity, tt.tolerance, params.Intensity)
			}

			// Verify block size scaling
			if tt.pendingBytes > 1000000 {
				expectedBlockSize := uint64(130000 - params.Intensity*float64(130000-21000))
				if params.MaxBlockSize != expectedBlockSize {
					t.Errorf("expected MaxBlockSize %d, got %d", expectedBlockSize, params.MaxBlockSize)
				}
			}
		})
	}
}

// TestQuadraticController tests the quadratic controller behavior
func TestQuadraticController(t *testing.T) {
	strategy := NewQuadraticStrategy(1000000, 5000, 21000, 130000, 2, noopLogger())
	controller := NewThrottleController(strategy)

	tests := []struct {
		name              string
		pendingBytes      uint64
		expectedIntensity float64
		tolerance         float64
	}{
		{
			name:              "below threshold",
			pendingBytes:      500000,
			expectedIntensity: 0.0,
			tolerance:         0.001,
		},
		{
			name:              "25% above threshold",
			pendingBytes:      1250000,
			expectedIntensity: 0.0625, // (0.25)^2
			tolerance:         0.001,
		},
		{
			name:              "50% above threshold",
			pendingBytes:      1500000,
			expectedIntensity: 0.25, // (0.5)^2
			tolerance:         0.001,
		},
		{
			name:              "75% above threshold",
			pendingBytes:      1750000,
			expectedIntensity: 0.5625, // (0.75)^2
			tolerance:         0.001,
		},
		{
			name:              "100% above threshold (max)",
			pendingBytes:      2000000,
			expectedIntensity: 1.0,
			tolerance:         0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := controller.Update(tt.pendingBytes, 0)

			if math.Abs(params.Intensity-tt.expectedIntensity) > tt.tolerance {
				t.Errorf("expected intensity %f ± %f, got %f",
					tt.expectedIntensity, tt.tolerance, params.Intensity)
			}
		})
	}
}

// TestPIDController tests the PID controller behavior
func TestPIDController(t *testing.T) {
	config := config.PIDConfig{
		Kp:          0.2,
		Ki:          0.1,
		Kd:          0.05,
		IntegralMax: 100.0,
		OutputMax:   1.0,
		SampleTime:  time.Millisecond * 10,
	}

	strategy := NewPIDStrategy(1000000, 5000, 21000, 130000, config)
	controller := NewThrottleController(strategy)

	t.Run("below threshold", func(t *testing.T) {
		params := controller.Update(800000, 1000000)
		if params.Intensity != 0.0 {
			t.Errorf("expected no throttling below threshold, got intensity %f", params.Intensity)
		}
	})

	t.Run("above threshold with delay", func(t *testing.T) {
		// First call above threshold
		time.Sleep(time.Millisecond * 15) // Ensure sample time passes
		params1 := controller.Update(1200000, 1000000)

		if params1.Intensity <= 0 {
			t.Errorf("expected positive intensity above threshold, got %f", params1.Intensity)
		}

		// Second call with higher load
		time.Sleep(time.Millisecond * 15)
		params2 := controller.Update(1400000, 1000000)

		if params2.Intensity <= params1.Intensity {
			t.Errorf("expected increasing intensity with higher error, got %f -> %f",
				params1.Intensity, params2.Intensity)
		}
	})

	t.Run("integral windup protection", func(t *testing.T) {
		controller.Reset()

		// Simulate sustained high error to test integral windup
		for i := 0; i < 20; i++ {
			time.Sleep(time.Millisecond * 15)
			controller.Update(2000000, 1000000) // Large error
		}

		// Controller should still be stable (not have runaway integral)
		params := controller.Update(2000000, 1000000)
		if params.Intensity > config.OutputMax {
			t.Errorf("intensity exceeded OutputMax: %f > %f", params.Intensity, config.OutputMax)
		}
	})

	t.Run("reset functionality", func(t *testing.T) {
		// Build up some state
		time.Sleep(time.Millisecond * 15)
		controller.Update(1500000, 1000000)

		// Reset and verify clean state
		controller.Reset()

		// Should behave like first call again
		time.Sleep(time.Millisecond * 15)
		params := controller.Update(800000, 1000000)
		if params.Intensity != 0.0 {
			t.Errorf("expected clean state after reset, got intensity %f", params.Intensity)
		}
	})
}

// TestPIDControllerWithMetrics tests PID controller with metrics integration
func TestPIDControllerWithMetrics(t *testing.T) {
	config := config.PIDConfig{
		Kp:          0.2,
		Ki:          0.1,
		Kd:          0.05,
		IntegralMax: 100.0,
		OutputMax:   1.0,
		SampleTime:  time.Millisecond * 10,
	}

	strategy := NewPIDStrategy(1000000, 5000, 21000, 130000, config)
	controller := NewThrottleController(strategy)

	// Mock metrics
	metrics := &mockMetrics{}
	strategy.SetMetrics(metrics)

	// Test with load above threshold - first call to initialize
	time.Sleep(time.Millisecond * 15)
	params := controller.Update(1500000, 1000000)

	// Debug output to understand what's happening
	t.Logf("First update - Intensity: %f, Error: %f, ResponseTime: %v",
		params.Intensity, metrics.GetLastError(), metrics.GetResponseTime())

	// The first call initializes the controller, subsequent calls should show response
	time.Sleep(time.Millisecond * 15)
	params = controller.Update(1500000, 1000000)

	t.Logf("Second update - Intensity: %f, Error: %f, ResponseTime: %v",
		params.Intensity, metrics.GetLastError(), metrics.GetResponseTime())

	if params.Intensity <= 0 {
		t.Errorf("expected positive intensity for load above threshold, got %f", params.Intensity)
	}

	// Check metrics were recorded - note that we record the raw error, not normalized
	if metrics.GetLastError() <= 0 {
		t.Errorf("expected positive error for load above target, got %f", metrics.GetLastError())
	}

	if metrics.GetResponseTime() <= 0 {
		t.Errorf("expected positive response time to be recorded, got %v", metrics.GetResponseTime())
	}
}

// TestControllerFactory tests the factory pattern
func TestControllerFactory(t *testing.T) {
	factory := NewThrottleControllerFactory(noopLogger())

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
			pidConfig: &config.PIDConfig{
				Kp: 0.2, Ki: 0.1, Kd: 0.05,
				IntegralMax: 100.0, OutputMax: 1.0, SampleTime: time.Second,
			},
			expectError: false,
		},
		{
			name:           "pid controller without config",
			controllerType: config.PIDControllerType,
			pidConfig:      nil,
			expectError:    true,
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
				tt.controllerType, 1000000, 5000, 21000, 130000, 2, tt.pidConfig)

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

			if controller.GetType() != tt.controllerType {
				t.Errorf("expected type %s, got %s", tt.controllerType, controller.GetType())
			}
		})
	}
}

// TestControllerConcurrency tests thread safety
func TestControllerConcurrency(t *testing.T) {
	controllers := []struct {
		name       string
		controller *ThrottleController
	}{
		{"step", NewThrottleController(NewStepStrategy(1000000, 5000, 21000, 130000))},
		{"linear", NewThrottleController(NewLinearStrategy(1000000, 5000, 21000, 130000, 2, noopLogger()))},
		{"quadratic", NewThrottleController(NewQuadraticStrategy(1000000, 5000, 21000, 130000, 2, noopLogger()))},
		{"pid", NewThrottleController(NewPIDStrategy(1000000, 5000, 21000, 130000, config.PIDConfig{
			Kp: 0.2, Ki: 0.1, Kd: 0.05,
			IntegralMax: 100.0, OutputMax: 1.0, SampleTime: time.Millisecond,
		}))},
	}

	for _, ctrl := range controllers {
		t.Run(ctrl.name, func(t *testing.T) {
			const numGoroutines = 10
			const numUpdates = 100

			var wg sync.WaitGroup
			wg.Add(numGoroutines)

			// Run concurrent updates
			for i := 0; i < numGoroutines; i++ {
				go func(id int) {
					defer wg.Done()
					for j := 0; j < numUpdates; j++ {
						// Vary the load based on goroutine ID and iteration
						load := uint64(500000 + (id * 100000) + (j * 1000))
						params := ctrl.controller.Update(load, 0)

						// Basic sanity checks
						if params.Intensity < 0 || params.Intensity > 1 {
							t.Errorf("invalid intensity: %f", params.Intensity)
						}

						// Small delay for PID controller
						if ctrl.name == "pid" {
							time.Sleep(time.Microsecond * 10)
						}
					}
				}(i)
			}

			wg.Wait()
		})
	}
}

// TestControllerBehaviorComparison compares all controllers with same inputs
func TestControllerBehaviorComparison(t *testing.T) {
	controllers := map[string]*ThrottleController{
		"step":      NewThrottleController(NewStepStrategy(1000000, 5000, 21000, 130000)),
		"linear":    NewThrottleController(NewLinearStrategy(1000000, 5000, 21000, 130000, 2, noopLogger())),
		"quadratic": NewThrottleController(NewQuadraticStrategy(1000000, 5000, 21000, 130000, 2, noopLogger())),
		"pid": NewThrottleController(NewPIDStrategy(1000000, 5000, 21000, 130000, config.PIDConfig{
			Kp: 0.2, Ki: 0.1, Kd: 0.05,
			IntegralMax: 100.0, OutputMax: 1.0, SampleTime: time.Millisecond,
		})),
	}

	loadScenarios := []uint64{500000, 1000000, 1250000, 1500000, 2000000}

	results := make(map[string][]ThrottleParams)

	for name, controller := range controllers {
		results[name] = make([]ThrottleParams, len(loadScenarios))
		for i, load := range loadScenarios {
			if name == "pid" && i > 0 {
				time.Sleep(time.Millisecond * 2) // Allow sample time
			}
			results[name][i] = controller.Update(load, 0)
		}
	}

	// Verify expected relationships
	t.Run("below threshold all zero", func(t *testing.T) {
		for name, params := range results {
			if params[0].Intensity != 0.0 { // 500000 < 1000000
				t.Errorf("%s controller should have zero intensity below threshold, got %f",
					name, params[0].Intensity)
			}
		}
	})

	t.Run("at threshold all zero", func(t *testing.T) {
		for name, params := range results {
			if params[1].Intensity != 0.0 { // 1000000 == 1000000
				t.Errorf("%s controller should have zero intensity at threshold, got %f",
					name, params[1].Intensity)
			}
		}
	})

	t.Run("step vs gradual scaling", func(t *testing.T) {
		stepIntensity := results["step"][2].Intensity // 1250000
		linearIntensity := results["linear"][2].Intensity
		quadIntensity := results["quadratic"][2].Intensity

		if stepIntensity != 1.0 {
			t.Errorf("step controller should be 1.0 above threshold, got %f", stepIntensity)
		}

		if linearIntensity >= stepIntensity {
			t.Errorf("linear should be less than step at moderate load: %f >= %f",
				linearIntensity, stepIntensity)
		}

		if quadIntensity >= linearIntensity {
			t.Errorf("quadratic should be less than linear at moderate load: %f >= %f",
				quadIntensity, linearIntensity)
		}
	})

	// Log comparison for manual inspection
	t.Logf("Controller Comparison:")
	t.Logf("%-10s %-8s %-8s %-8s %-8s", "Load", "Step", "Linear", "Quad", "PID")
	for i, load := range loadScenarios {
		t.Logf("%-10d %-8.3f %-8.3f %-8.3f %-8.3f",
			load,
			results["step"][i].Intensity,
			results["linear"][i].Intensity,
			results["quadratic"][i].Intensity,
			results["pid"][i].Intensity)
	}
}

// TestLoadSpikeResponse tests how controllers respond to sudden load changes
func TestLoadSpikeResponse(t *testing.T) {
	controllers := map[string]*ThrottleController{
		"step":      NewThrottleController(NewStepStrategy(1000000, 5000, 21000, 130000)),
		"linear":    NewThrottleController(NewLinearStrategy(1000000, 5000, 21000, 130000, 2, noopLogger())),
		"quadratic": NewThrottleController(NewQuadraticStrategy(1000000, 5000, 21000, 130000, 2, noopLogger())),
		"pid": NewThrottleController(NewPIDStrategy(1000000, 5000, 21000, 130000, config.PIDConfig{
			Kp: 0.5, Ki: 0.2, Kd: 0.1, // More responsive for this test
			IntegralMax: 100.0, OutputMax: 1.0, SampleTime: time.Millisecond,
		})),
	}

	// Simulate: low -> high -> low load pattern
	loadSequence := []uint64{800000, 2000000, 800000}

	for name, controller := range controllers {
		t.Run(name, func(t *testing.T) {
			var intensities []float64

			for i, load := range loadSequence {
				if name == "pid" && i > 0 {
					time.Sleep(time.Millisecond * 2)
				}
				params := controller.Update(load, 0)
				intensities = append(intensities, params.Intensity)
			}

			// Verify response pattern
			if intensities[0] != 0.0 {
				t.Errorf("expected no throttling at low load, got %.3f", intensities[0])
			}

			if intensities[1] <= 0.0 {
				t.Errorf("expected throttling at high load, got %.3f", intensities[1])
			}

			// For non-step controllers, intensity should decrease when load drops
			if name != "step" && intensities[2] >= intensities[1] {
				t.Errorf("expected reduced throttling when load decreases: %.3f -> %.3f",
					intensities[1], intensities[2])
			}

			t.Logf("%s response: %.3f -> %.3f -> %.3f",
				name, intensities[0], intensities[1], intensities[2])
		})
	}
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	t.Run("zero threshold", func(t *testing.T) {
		controller := NewThrottleController(NewStepStrategy(0, 5000, 21000, 130000))
		params := controller.Update(1000, 0)

		// Should throttle immediately since any load > 0 threshold
		if params.Intensity != 1.0 {
			t.Errorf("expected full throttling with zero threshold, got %f", params.Intensity)
		}
	})

	t.Run("zero throttle sizes", func(t *testing.T) {
		controller := NewThrottleController(NewStepStrategy(1000000, 0, 0, 0))
		params := controller.Update(2000000, 0)

		if params.MaxTxSize != 0 {
			t.Errorf("expected zero MaxTxSize, got %d", params.MaxTxSize)
		}
		if params.MaxBlockSize != 0 {
			t.Errorf("expected zero MaxBlockSize, got %d", params.MaxBlockSize)
		}
	})

	t.Run("pid with zero gains", func(t *testing.T) {
		config := config.PIDConfig{
			Kp: 0, Ki: 0, Kd: 0,
			IntegralMax: 100.0, OutputMax: 1.0, SampleTime: time.Millisecond,
		}
		controller := NewThrottleController(NewPIDStrategy(1000000, 5000, 21000, 130000, config))

		time.Sleep(time.Millisecond * 2)
		params := controller.Update(2000000, 1000000)

		// With zero gains, should have minimal response
		if params.Intensity > 0.1 {
			t.Errorf("expected minimal response with zero gains, got %f", params.Intensity)
		}
	})

	t.Run("very high sample time", func(t *testing.T) {
		config := config.PIDConfig{
			Kp: 0.2, Ki: 0.1, Kd: 0.05,
			IntegralMax: 100.0, OutputMax: 1.0, SampleTime: time.Hour, // Very high
		}
		controller := NewThrottleController(NewPIDStrategy(1000000, 5000, 21000, 130000, config))

		// Multiple quick calls should return same result due to sample time
		params1 := controller.Update(2000000, 1000000)
		params2 := controller.Update(2000000, 1000000)

		if params1.Intensity != params2.Intensity {
			t.Errorf("expected same intensity due to sample time: %f vs %f",
				params1.Intensity, params2.Intensity)
		}
	})
}

// TestParameterValidation tests parameter validation and error cases
func TestParameterValidation(t *testing.T) {
	factory := NewThrottleControllerFactory(noopLogger())

	t.Run("invalid pid config", func(t *testing.T) {
		invalidConfigs := []*config.PIDConfig{
			nil, // No config provided
			{Kp: -1, Ki: 0.1, Kd: 0.05, IntegralMax: 100, OutputMax: 1, SampleTime: time.Second},   // Negative Kp
			{Kp: 0.1, Ki: -1, Kd: 0.05, IntegralMax: 100, OutputMax: 1, SampleTime: time.Second},   // Negative Ki
			{Kp: 0.1, Ki: 0.1, Kd: -1, IntegralMax: 100, OutputMax: 1, SampleTime: time.Second},    // Negative Kd
			{Kp: 0.1, Ki: 0.1, Kd: 0.05, IntegralMax: -100, OutputMax: 1, SampleTime: time.Second}, // Negative IntegralMax
			{Kp: 0.1, Ki: 0.1, Kd: 0.05, IntegralMax: 100, OutputMax: -1, SampleTime: time.Second}, // Negative OutputMax
			{Kp: 0.1, Ki: 0.1, Kd: 0.05, IntegralMax: 100, OutputMax: 2, SampleTime: time.Second},  // OutputMax > 1
			{Kp: 0.1, Ki: 0.1, Kd: 0.05, IntegralMax: 100, OutputMax: 1, SampleTime: -time.Second}, // Negative SampleTime
		}

		for i, invalidConfig := range invalidConfigs {
			_, err := factory.CreateController(config.PIDControllerType, 1000000, 5000, 21000, 130000, 2, invalidConfig)
			if err == nil {
				t.Errorf("config %d: expected error for invalid PID config but got none", i)
			}
		}
	})
}

// Benchmark tests for performance
func BenchmarkControllerUpdates(b *testing.B) {
	controllers := []struct {
		name       string
		controller *ThrottleController
	}{
		{"Step", NewThrottleController(NewStepStrategy(1000000, 5000, 21000, 130000))},
		{"Linear", NewThrottleController(NewLinearStrategy(1000000, 5000, 21000, 130000, 2, noopLogger()))},
		{"Quadratic", NewThrottleController(NewQuadraticStrategy(1000000, 5000, 21000, 130000, 2, noopLogger()))},
		{"PID", NewThrottleController(NewPIDStrategy(1000000, 5000, 21000, 130000, config.PIDConfig{
			Kp: 0.2, Ki: 0.1, Kd: 0.05,
			IntegralMax: 100.0, OutputMax: 1.0, SampleTime: time.Microsecond,
		}))},
	}

	for _, ctrl := range controllers {
		b.Run(ctrl.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				load := uint64(500000 + (i%10)*100000) // Vary load
				ctrl.controller.Update(load, 0)
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

// TestPIDMetricsSetup tests that PID metrics are properly configured during controller setup
func TestPIDMetricsSetup(t *testing.T) {
	factory := NewThrottleControllerFactory(noopLogger())

	pidConfig := &config.PIDConfig{
		Kp: 0.2, Ki: 0.1, Kd: 0.05,
		IntegralMax: 100.0, OutputMax: 1.0, SampleTime: time.Millisecond * 10,
	}

	// Create a PID controller
	controller, err := factory.CreateController(
		config.PIDControllerType, 1000000, 5000, 21000, 130000, 2, pidConfig)
	if err != nil {
		t.Fatalf("Failed to create PID controller: %v", err)
	}

	// Verify we can access the PID strategy
	pidStrategy := controller.GetPIDStrategy()
	if pidStrategy == nil {
		t.Fatal("Expected PID strategy but got nil")
	}

	// Setup mock metrics
	metrics := &mockMetrics{}
	pidStrategy.SetMetrics(metrics)

	// Trigger two updates to generate metrics (first call initializes, second generates metrics)
	time.Sleep(time.Millisecond * 15)   // Ensure sample time passes
	controller.Update(1500000, 1000000) // First call initializes

	time.Sleep(time.Millisecond * 15)             // Ensure sample time passes
	params := controller.Update(1500000, 1000000) // Second call generates metrics

	if params.Intensity <= 0 {
		t.Errorf("Expected positive intensity for load above threshold, got %f", params.Intensity)
	}

	// Verify metrics were recorded
	if metrics.GetLastError() <= 0 {
		t.Errorf("Expected metrics to be recorded but got error %f", metrics.GetLastError())
	}

	if metrics.GetResponseTime() <= 0 {
		t.Errorf("Expected response time to be recorded but got %v", metrics.GetResponseTime())
	}

	t.Logf("PID metrics successfully recorded - Error: %f, ResponseTime: %v",
		metrics.GetLastError(), metrics.GetResponseTime())
}
