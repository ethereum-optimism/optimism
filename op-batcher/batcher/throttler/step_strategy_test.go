package throttler

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
)

// Test constants specific to step strategy
const (
	TestStepThreshold  = 500_000 // 500KB threshold for step strategy tests
	TestStepTxSize     = 3_000   // 3KB transaction size limit
	TestStepBlockSize  = 15_000  // 15KB block size limit
	TestStepAlwaysSize = 100_000 // 100KB always block size
)

func TestStepStrategy_NewStepStrategy(t *testing.T) {
	strategy := NewStepStrategy(TestStepThreshold)

	if strategy.threshold != TestStepThreshold {
		t.Errorf("expected threshold %d, got %d", TestStepThreshold, strategy.threshold)
	}

	// Test initial state
	controllerType, intensity := strategy.Load()
	if controllerType != config.StepControllerType {
		t.Errorf("expected controller type %s, got %s", config.StepControllerType, controllerType)
	}

	if intensity != TestIntensityMin {
		t.Errorf("expected initial intensity %f, got %f", TestIntensityMin, intensity)
	}
}

func TestStepStrategy_Update(t *testing.T) {
	strategy := NewStepStrategy(TestStepThreshold)

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
			pendingBytes:      TestStepThreshold / 2,
			targetBytes:       0,
			expectedIntensity: TestIntensityMin,
		},
		{
			name:              "exactly at threshold",
			pendingBytes:      TestStepThreshold,
			targetBytes:       0,
			expectedIntensity: TestIntensityMin,
		},
		{
			name:              "just above threshold",
			pendingBytes:      TestStepThreshold + 1,
			targetBytes:       0,
			expectedIntensity: TestIntensityMax,
		},
		{
			name:              "far above threshold",
			pendingBytes:      TestStepThreshold * 10,
			targetBytes:       0,
			expectedIntensity: TestIntensityMax,
		},
		{
			name:              "with target bytes ignored",
			pendingBytes:      TestStepThreshold + 1000,
			targetBytes:       TestStepThreshold * 2, // Target bytes should be ignored in step strategy
			expectedIntensity: TestIntensityMax,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intensity := strategy.Update(tt.pendingBytes)

			if intensity != tt.expectedIntensity {
				t.Errorf("expected intensity %f, got %f", tt.expectedIntensity, intensity)
			}
		})
	}
}

func TestStepStrategy_Load(t *testing.T) {
	strategy := NewStepStrategy(TestStepThreshold)

	// Test initial load
	controllerType, _ := strategy.Load()
	if controllerType != config.StepControllerType {
		t.Errorf("expected controller type %s, got %s", config.StepControllerType, controllerType)
	}

	// Test load consistency after update
	updateIntensity := strategy.Update(TestStepThreshold * 2)

	loadType, loadIntensity := strategy.Load()
	if loadType != config.StepControllerType {
		t.Errorf("expected controller type %s after update, got %s", config.StepControllerType, loadType)
	}

	if loadIntensity != updateIntensity {
		t.Errorf("Load() returned different intensity than Update(): %f vs %f", loadIntensity, updateIntensity)
	}
}

func TestStepStrategy_GetType(t *testing.T) {
	strategy := NewStepStrategy(TestStepThreshold)

	if strategy.GetType() != config.StepControllerType {
		t.Errorf("expected GetType() to return %s, got %s", config.StepControllerType, strategy.GetType())
	}
}

func TestStepStrategy_Reset(t *testing.T) {
	strategy := NewStepStrategy(TestStepThreshold)

	// Update to build some state
	strategy.Update(TestStepThreshold * 2)

	// Reset should restore to initial state
	strategy.Reset()

	controllerType, intensity := strategy.Load()
	if controllerType != config.StepControllerType {
		t.Errorf("expected controller type %s after reset, got %s", config.StepControllerType, controllerType)
	}

	if intensity != TestIntensityMin {
		t.Errorf("expected intensity %f after reset, got %f", TestIntensityMin, intensity)
	}
}

func TestStepStrategy_BoundaryConditions(t *testing.T) {
	strategy := NewStepStrategy(TestStepThreshold)

	// Test boundary at threshold
	belowIntensity := strategy.Update(TestStepThreshold)
	aboveIntensity := strategy.Update(TestStepThreshold + 1)

	if belowIntensity != TestIntensityMin {
		t.Errorf("expected no throttling at threshold, got intensity %f", belowIntensity)
	}

	if aboveIntensity != TestIntensityMax {
		t.Errorf("expected full throttling just above threshold, got intensity %f", aboveIntensity)
	}

	// Test with zero threshold
	zeroThresholdStrategy := NewStepStrategy(0)
	intensity := zeroThresholdStrategy.Update(1)

	if intensity != TestIntensityMax {
		t.Errorf("expected immediate throttling with zero threshold, got intensity %f", intensity)
	}
}

func TestStepStrategy_ConcurrentAccess(t *testing.T) {
	strategy := NewStepStrategy(TestStepThreshold)

	// Test that strategy can handle concurrent access without panics
	done := make(chan bool, 2)

	// Writer goroutine
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			load := uint64(TestStepThreshold + i*1000)
			strategy.Update(load)
		}
	}()

	// Reader goroutine
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			strategy.Load()
		}
	}()

	// Wait for both goroutines to complete
	<-done
	<-done
}
