package throttler

import (
	"github.com/ethereum-optimism/optimism/op-batcher/config"
)

// ThrottleParams holds the current throttling parameters
type ThrottleParams struct {
	MaxTxSize    uint64 // Maximum transaction size when throttling
	MaxBlockSize uint64 // Maximum block size when throttling
	// Intensity represents the throttling intensity as a normalized value between 0.0 and 1.0
	// 0.0 = no throttling applied, DA parameters for the block builder are set to their largest values
	// 1.0 = maximum throttling applied, DA parameters for the block builder are set to their smallest values
	// Values between 0.0 and 1.0 represent proportional throttling levels
	Intensity float64
}

func (p ThrottleParams) IsThrottling() bool {
	return p.Intensity > 0
}

// ThrottleStrategy defines the interface for throttle strategies using the Strategy pattern
// Strategies now only calculate intensity; the controller handles interpolation to final parameters
type ThrottleStrategy interface {
	// Update calculates new throttling intensity based on current pending bytes
	// Returns intensity value between 0.0 and 1.0
	Update(currentPendingBytes uint64) float64
	// Reset resets the strategy state
	Reset()
	// GetType returns the strategy type
	GetType() config.ThrottleControllerType
	// Load returns the current throttle type and intensity atomically
	Load() (config.ThrottleControllerType, float64)
}

// ThrottleConfig holds the configuration parameters for throttling
type ThrottleConfig struct {
	TxSizeLowerLimit    uint64
	TxSizeUpperLimit    uint64
	BlockSizeLowerLimit uint64
	BlockSizeUpperLimit uint64
}
