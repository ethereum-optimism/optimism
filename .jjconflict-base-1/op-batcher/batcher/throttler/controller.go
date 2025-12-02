package throttler

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/ethereum/go-ethereum/log"
)

// ThrottleController manages throttling using a pluggable strategy
type ThrottleController struct {
	mu            sync.RWMutex
	strategy      ThrottleStrategy
	config        ThrottleConfig
	currentParams atomic.Pointer[ThrottleParams]
}

func NewThrottleController(strategy ThrottleStrategy, config ThrottleConfig) *ThrottleController {
	controller := &ThrottleController{
		strategy: strategy,
		config:   config,
	}

	// Initialize with default params
	initialParams := &ThrottleParams{
		MaxTxSize:    0,
		MaxBlockSize: config.BlockSizeUpperLimit,
		Intensity:    0.0,
	}
	controller.currentParams.Store(initialParams)

	return controller
}

// Update updates the throttle parameters and returns the new params
func (tc *ThrottleController) Update(currentPendingBytes uint64) ThrottleParams {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	strategy := tc.strategy
	config := tc.config
	intensity := strategy.Update(currentPendingBytes)

	params := tc.intensityToParams(intensity, config)
	tc.currentParams.Store(&params)

	return params
}

// intensityToParams converts intensity to throttle parameters using common interpolation logic
func (tc *ThrottleController) intensityToParams(intensity float64, cfg ThrottleConfig) ThrottleParams {
	// Clamp intensity to 1.0 to prevent overflows, should never happen
	if intensity > 1.0 {
		log.Warn("throttler: intensity above maximum (will be clamped)", "intensity", intensity)
		intensity = 1.0
	}

	// If intensity is negative, set it to 0
	if intensity < 0 {
		log.Warn("throttler: intensity below minimum (will be clamped)", "intensity", intensity)
		intensity = 0
	}

	return ThrottleParams{
		MaxTxSize:    tc.intensityToTxSize(intensity, cfg),
		MaxBlockSize: tc.intensityToBlockSize(intensity, cfg),
		Intensity:    intensity,
	}
}

func (tc *ThrottleController) validateConfig(cfg ThrottleConfig) error {
	if cfg.BlockSizeLowerLimit > 0 && cfg.BlockSizeLowerLimit >= cfg.BlockSizeUpperLimit {
		log.Error("throttler: invalid block size limits",
			"blockSizeLowerLimit", cfg.BlockSizeLowerLimit,
			"blockSizeUpperLimit", cfg.BlockSizeUpperLimit,
			"controllerType", tc.GetType(),
		)
		return errors.New("throttler: invalid block size limits")
	}

	if cfg.TxSizeLowerLimit > 0 &&
		tc.GetType() != config.StepControllerType &&
		cfg.TxSizeLowerLimit >= cfg.TxSizeUpperLimit {
		log.Error("throttler: invalid tx size limits",
			"txSizeLowerLimit", cfg.TxSizeLowerLimit,
			"txSizeUpperLimit", cfg.TxSizeUpperLimit,
			"controllerType", tc.GetType(),
		)
		return errors.New("throttler: invalid tx size limits")
	}
	return nil
}

// intensityToBlockSize converts intensity in [0,1] to block size
func (tc *ThrottleController) intensityToBlockSize(intensity float64, cfg ThrottleConfig) uint64 {
	if cfg.BlockSizeLowerLimit == 0 {
		return 0
	}

	if intensity == 0 {
		return cfg.BlockSizeUpperLimit
	} else {
		switch tc.strategy.GetType() {
		case config.StepControllerType:
			return cfg.BlockSizeLowerLimit
		default:
			return uint64(float64(cfg.BlockSizeUpperLimit) - intensity*float64(cfg.BlockSizeUpperLimit-cfg.BlockSizeLowerLimit))
		}
	}

}

// intensityToTxSize converts intensity in [0,1] to tx size
func (tc *ThrottleController) intensityToTxSize(intensity float64, cfg ThrottleConfig) uint64 {
	if cfg.TxSizeLowerLimit == 0 {
		return 0
	}

	if intensity == 0 {
		return 0 // Transactions are not throttled at 0 intensity
	} else {
		switch tc.strategy.GetType() {
		case config.StepControllerType:
			return cfg.TxSizeLowerLimit
		default:
			return uint64(float64(cfg.TxSizeUpperLimit) - intensity*float64(cfg.TxSizeUpperLimit-cfg.TxSizeLowerLimit))
		}
	}
}

// Load returns the current controller type and parameters atomically
func (tc *ThrottleController) Load() (config.ThrottleControllerType, ThrottleParams) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	controllerType := tc.strategy.GetType()

	params := tc.currentParams.Load()
	if params == nil {
		return controllerType, ThrottleParams{}
	}

	return controllerType, *params
}

// SetStrategy changes the throttle strategy at runtime
func (tc *ThrottleController) SetStrategy(strategy ThrottleStrategy, resetParams ThrottleParams) {
	tc.mu.Lock()
	tc.strategy = strategy
	tc.mu.Unlock()

	tc.currentParams.Store(&resetParams)
}

// Reset resets the current strategy state
func (tc *ThrottleController) Reset() {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	strategy := tc.strategy
	config := tc.config

	// Call strategy reset without holding the controller lock
	strategy.Reset()

	// Reset to default parameters
	resetParams := ThrottleParams{
		MaxTxSize:    0,
		MaxBlockSize: config.BlockSizeUpperLimit,
		Intensity:    0.0,
	}
	tc.currentParams.Store(&resetParams)
}

// GetType returns the current strategy type
func (tc *ThrottleController) GetType() config.ThrottleControllerType {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.strategy.GetType()
}

// GetPIDStrategy returns the PID strategy if the current strategy is PID, otherwise returns nil
func (tc *ThrottleController) GetPIDStrategy() *PIDStrategy {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	if pidStrategy, ok := tc.strategy.(*PIDStrategy); ok {
		return pidStrategy
	}
	return nil
}

// ThrottleControllerFactory creates throttle controllers based on configuration
type ThrottleControllerFactory struct {
	log log.Logger
}

func NewThrottleControllerFactory(log log.Logger) *ThrottleControllerFactory {
	return &ThrottleControllerFactory{log: log}
}

func (f *ThrottleControllerFactory) CreateController(
	controllerType config.ThrottleControllerType,
	throttleParams config.ThrottleParams,
	pidConfig *config.PIDConfig,
) (*ThrottleController, error) {
	var strategy ThrottleStrategy

	throttleConfig := ThrottleConfig{
		TxSizeLowerLimit:    throttleParams.TxSizeLowerLimit,
		TxSizeUpperLimit:    throttleParams.TxSizeUpperLimit,
		BlockSizeLowerLimit: throttleParams.BlockSizeLowerLimit,
		BlockSizeUpperLimit: throttleParams.BlockSizeUpperLimit,
	}

	// Default to step controller if no type is specified
	if controllerType == "" {
		controllerType = config.StepControllerType
	}

	switch controllerType {
	case config.StepControllerType:
		strategy = NewStepStrategy(throttleParams.LowerThreshold)
	case config.LinearControllerType:
		strategy = NewLinearStrategy(throttleParams.LowerThreshold, throttleParams.UpperThreshold, f.log)
	case config.QuadraticControllerType:
		strategy = NewQuadraticStrategy(throttleParams.LowerThreshold, throttleParams.UpperThreshold, f.log)
	case config.PIDControllerType:
		log.Warn("EXPERIMENTAL FEATURE")
		log.Warn("PID controller is an EXPERIMENTAL feature that should only be used by experts. PID controller requires deep understanding of control theory and careful tuning. Improper configuration can lead to system instability or poor performance. Use with extreme caution in production environments.")
		if pidConfig == nil {
			return nil, fmt.Errorf("PID configuration is required for PID controller")
		}

		// Validate PID configuration parameters
		if pidConfig.Kp < 0 {
			return nil, fmt.Errorf("PID Kp gain must be non-negative, got %f", pidConfig.Kp)
		}
		if pidConfig.Ki < 0 {
			return nil, fmt.Errorf("PID Ki gain must be non-negative, got %f", pidConfig.Ki)
		}
		if pidConfig.Kd < 0 {
			return nil, fmt.Errorf("PID Kd gain must be non-negative, got %f", pidConfig.Kd)
		}
		if pidConfig.IntegralMax <= 0 {
			return nil, fmt.Errorf("PID IntegralMax must be positive, got %f", pidConfig.IntegralMax)
		}
		if pidConfig.OutputMax <= 0 || pidConfig.OutputMax > 1 {
			return nil, fmt.Errorf("PID OutputMax must be between 0 and 1, got %f", pidConfig.OutputMax)
		}
		if pidConfig.SampleTime <= 0 {
			return nil, fmt.Errorf("PID SampleTime must be positive, got %v", pidConfig.SampleTime)
		}

		strategy = NewPIDStrategy(throttleParams.LowerThreshold, *pidConfig)
	default:
		return nil, fmt.Errorf("unsupported throttle controller type: %s", controllerType)
	}

	newController := NewThrottleController(strategy, throttleConfig)
	err := newController.validateConfig(throttleConfig)
	if err != nil {
		return nil, err
	}

	return newController, nil
}
