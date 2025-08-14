package throttler

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
)

func TestUnlimitedStrategy_NewUnlimitedStrategy(t *testing.T) {
	strategy := NewUnlimitedStrategy()

	controllerType, intensity := strategy.Load()
	if controllerType != config.UnlimitedControllerType {
		t.Errorf("expected controller type %s, got %s", config.UnlimitedControllerType, controllerType)
	}

	if intensity != TestIntensityMin {
		t.Errorf("expected initial intensity %f, got %f", TestIntensityMin, intensity)
	}
}

func TestUnlimitedStrategy_Update_NoThrottling(t *testing.T) {
	strategy := NewUnlimitedStrategy()

	tests := []struct {
		name         string
		pendingBytes uint64
	}{
		{"zero load", 0},
		{"small load", 1},
		{"at lower threshold", TestLowerThresholdBytes},
		{"above lower threshold", TestLoadHighAbove},
		{"far above", TestLoadFarAbove},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intensity := strategy.Update(tt.pendingBytes)
			if intensity != TestIntensityMin {
				t.Errorf("expected intensity %f, got %f", TestIntensityMin, intensity)
			}
		})
	}
}

func TestUnlimitedStrategy_Load(t *testing.T) {
	strategy := NewUnlimitedStrategy()

	// After any updates, Load should still report unlimited with zero intensity
	_ = strategy.Update(TestLoadHighAbove)
	controllerType, intensity := strategy.Load()

	if controllerType != config.UnlimitedControllerType {
		t.Errorf("expected controller type %s, got %s", config.UnlimitedControllerType, controllerType)
	}
	if intensity != TestIntensityMin {
		t.Errorf("expected intensity %f, got %f", TestIntensityMin, intensity)
	}
}

func TestUnlimitedStrategy_GetType(t *testing.T) {
	strategy := NewUnlimitedStrategy()

	if strategy.GetType() != config.UnlimitedControllerType {
		t.Errorf("expected GetType() to return %s, got %s", config.UnlimitedControllerType, strategy.GetType())
	}
}

func TestUnlimitedStrategy_Reset_NoOp(t *testing.T) {
	strategy := NewUnlimitedStrategy()

	// Build some activity and then reset
	_ = strategy.Update(TestLoadFarAbove)
	strategy.Reset()

	controllerType, intensity := strategy.Load()
	if controllerType != config.UnlimitedControllerType {
		t.Errorf("expected controller type %s after reset, got %s", config.UnlimitedControllerType, controllerType)
	}
	if intensity != TestIntensityMin {
		t.Errorf("expected intensity %f after reset, got %f", TestIntensityMin, intensity)
	}
}

func TestUnlimitedStrategy_ControllerIntegration(t *testing.T) {
	strategy := NewUnlimitedStrategy()
	controller := NewThrottleController(strategy, testThrottleConfig)

	params := controller.Update(TestLoadHighAbove)
	if params.Intensity != TestIntensityMin {
		t.Errorf("expected controller intensity %f, got %f", TestIntensityMin, params.Intensity)
	}
	if params.MaxTxSize != 0 {
		t.Errorf("expected MaxTxSize 0 for unlimited strategy, got %d", params.MaxTxSize)
	}
	if params.MaxBlockSize != 0 {
		t.Errorf("expected MaxBlockSize 0 for unlimited strategy, got %d", params.MaxBlockSize)
	}
}
