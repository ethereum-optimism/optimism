package batcher

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/ethereum/go-ethereum/log"
)

// ThrottleParams holds the current throttling parameters
type ThrottleParams struct {
	MaxTxSize    uint64  // Maximum transaction size when throttling
	MaxBlockSize uint64  // Maximum block size when throttling
	Intensity    float64 // Throttling intensity (0.0 = no throttling, 1.0 = max throttling)
}

// ThrottleStrategy defines the interface for throttle strategies using the Strategy pattern
type ThrottleStrategy interface {
	// Update calculates new throttling parameters based on current pending bytes
	Update(currentPendingBytes, targetPendingBytes uint64) ThrottleParams
	// Reset resets the strategy state
	Reset()
	// GetType returns the strategy type
	GetType() config.ThrottleControllerType
	// Load returns the current throttle type and parameters atomically
	Load() (config.ThrottleControllerType, ThrottleParams)
}

// ThrottleController manages throttling using a pluggable strategy
type ThrottleController struct {
	mu            sync.RWMutex
	strategy      ThrottleStrategy
	currentParams atomic.Pointer[ThrottleParams]
}

func NewThrottleController(strategy ThrottleStrategy) *ThrottleController {
	controller := &ThrottleController{
		strategy: strategy,
	}

	// Initialize with default params
	initialParams := &ThrottleParams{
		MaxTxSize:    0,
		MaxBlockSize: 0,
		Intensity:    0.0,
	}
	controller.currentParams.Store(initialParams)

	return controller
}

// Update updates the throttle parameters and returns the new params
func (tc *ThrottleController) Update(currentPendingBytes, targetPendingBytes uint64) ThrottleParams {
	tc.mu.RLock()
	strategy := tc.strategy
	tc.mu.RUnlock()

	newParams := strategy.Update(currentPendingBytes, targetPendingBytes)
	tc.currentParams.Store(&newParams)

	return newParams
}

// Load returns the current controller type and parameters atomically
func (tc *ThrottleController) Load() (config.ThrottleControllerType, ThrottleParams) {
	tc.mu.RLock()
	controllerType := tc.strategy.GetType()
	tc.mu.RUnlock()

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
	strategy := tc.strategy
	tc.mu.RUnlock()

	strategy.Reset()
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

// StepStrategy implements binary on/off throttling (existing behavior)
type StepStrategy struct {
	threshold         uint64
	throttleTxSize    uint64
	throttleBlockSize uint64
	alwaysBlockSize   uint64
}

func NewStepStrategy(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize uint64) *StepStrategy {
	return &StepStrategy{
		threshold:         threshold,
		throttleTxSize:    throttleTxSize,
		throttleBlockSize: throttleBlockSize,
		alwaysBlockSize:   alwaysBlockSize,
	}
}

func (s *StepStrategy) Update(currentPendingBytes, targetPendingBytes uint64) ThrottleParams {
	maxBlockSize := s.alwaysBlockSize
	var maxTxSize uint64 = 0
	var intensity float64 = 0.0

	if currentPendingBytes > s.threshold {
		maxTxSize = s.throttleTxSize
		if s.throttleBlockSize != 0 && (maxBlockSize == 0 || s.throttleBlockSize < maxBlockSize) {
			maxBlockSize = s.throttleBlockSize
		}
		intensity = 1.0
	}

	return ThrottleParams{
		MaxTxSize:    maxTxSize,
		MaxBlockSize: maxBlockSize,
		Intensity:    intensity,
	}
}

func (s *StepStrategy) Reset() {
	// No state to reset for step strategy
}

func (s *StepStrategy) GetType() config.ThrottleControllerType {
	return config.StepControllerType
}

func (s *StepStrategy) Load() (config.ThrottleControllerType, ThrottleParams) {
	return s.GetType(), ThrottleParams{}
}

// LinearStrategy implements linear throttling based on pending bytes
type LinearStrategy struct {
	threshold         uint64
	maxThreshold      uint64 // Point at which maximum throttling is applied
	throttleTxSize    uint64
	throttleBlockSize uint64
	alwaysBlockSize   uint64
}

func NewLinearStrategy(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize uint64, multiplier float64, log log.Logger) *LinearStrategy {
	// Set max threshold to multiplier * base threshold for linear scaling
	maxThreshold := threshold * uint64(multiplier)
	// Ensure maxThreshold is always greater than threshold to prevent division by zero
	if maxThreshold <= threshold {
		maxThreshold = threshold + 1
		log.Warn("maxThreshold is less than or equal to threshold, setting maxThreshold to threshold + 1", "threshold", threshold, "multiplier", multiplier, "maxThreshold", maxThreshold)
	}
	return &LinearStrategy{
		threshold:         threshold,
		maxThreshold:      maxThreshold,
		throttleTxSize:    throttleTxSize,
		throttleBlockSize: throttleBlockSize,
		alwaysBlockSize:   alwaysBlockSize,
	}
}

func (l *LinearStrategy) Update(currentPendingBytes, targetPendingBytes uint64) ThrottleParams {
	maxBlockSize := l.alwaysBlockSize
	var maxTxSize uint64 = 0
	var intensity float64 = 0.0

	if currentPendingBytes > l.threshold {
		// Linear scaling from threshold to maxThreshold
		if currentPendingBytes >= l.maxThreshold {
			intensity = 1.0
		} else {
			// Linear interpolation
			intensity = float64(currentPendingBytes-l.threshold) / float64(l.maxThreshold-l.threshold)
		}

		// Apply intensity to tx size throttling
		if intensity > 0 {
			maxTxSize = l.throttleTxSize
		}

		// Apply intensity to block size throttling
		if l.throttleBlockSize != 0 {
			// Interpolate between alwaysBlockSize and throttleBlockSize
			if maxBlockSize == 0 || l.throttleBlockSize < maxBlockSize {
				targetBlockSize := l.throttleBlockSize
				if maxBlockSize > 0 {
					// Linear interpolation between always and throttle block sizes
					targetBlockSize = uint64(float64(maxBlockSize) - intensity*float64(maxBlockSize-l.throttleBlockSize))
				}
				maxBlockSize = targetBlockSize
			}
		}
	}

	return ThrottleParams{
		MaxTxSize:    maxTxSize,
		MaxBlockSize: maxBlockSize,
		Intensity:    intensity,
	}
}

func (l *LinearStrategy) Reset() {
	// No state to reset for linear strategy
}

func (l *LinearStrategy) GetType() config.ThrottleControllerType {
	return config.LinearControllerType
}

func (l *LinearStrategy) Load() (config.ThrottleControllerType, ThrottleParams) {
	return l.GetType(), ThrottleParams{}
}

// QuadraticStrategy implements quadratic throttling for more aggressive scaling
type QuadraticStrategy struct {
	threshold         uint64
	maxThreshold      uint64
	throttleTxSize    uint64
	throttleBlockSize uint64
	alwaysBlockSize   uint64
}

func NewQuadraticStrategy(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize uint64, multiplier float64, log log.Logger) *QuadraticStrategy {
	maxThreshold := threshold * uint64(multiplier)
	// Ensure maxThreshold is always greater than threshold to prevent division by zero
	if maxThreshold <= threshold {
		maxThreshold = threshold + 1
		log.Warn("maxThreshold is less than or equal to threshold, setting maxThreshold to threshold + 1", "threshold", threshold, "multiplier", multiplier, "maxThreshold", maxThreshold)
	}
	return &QuadraticStrategy{
		threshold:         threshold,
		maxThreshold:      maxThreshold,
		throttleTxSize:    throttleTxSize,
		throttleBlockSize: throttleBlockSize,
		alwaysBlockSize:   alwaysBlockSize,
	}
}

func (q *QuadraticStrategy) Update(currentPendingBytes, targetPendingBytes uint64) ThrottleParams {
	maxBlockSize := q.alwaysBlockSize
	var maxTxSize uint64 = 0
	var intensity float64 = 0.0

	if currentPendingBytes > q.threshold {
		// Quadratic scaling from threshold to maxThreshold
		if currentPendingBytes >= q.maxThreshold {
			intensity = 1.0
		} else {
			// Quadratic interpolation (x^2 curve for more aggressive throttling)
			linear := float64(currentPendingBytes-q.threshold) / float64(q.maxThreshold-q.threshold)
			intensity = linear * linear // Quadratic scaling
		}

		// Apply intensity to tx size throttling
		if intensity > 0 {
			maxTxSize = q.throttleTxSize
		}

		// Apply intensity to block size throttling
		if q.throttleBlockSize != 0 {
			if maxBlockSize == 0 || q.throttleBlockSize < maxBlockSize {
				targetBlockSize := q.throttleBlockSize
				if maxBlockSize > 0 {
					targetBlockSize = uint64(float64(maxBlockSize) - intensity*float64(maxBlockSize-q.throttleBlockSize))
				}
				maxBlockSize = targetBlockSize
			}
		}
	}

	return ThrottleParams{
		MaxTxSize:    maxTxSize,
		MaxBlockSize: maxBlockSize,
		Intensity:    intensity,
	}
}

func (q *QuadraticStrategy) Reset() {
	// No state to reset for quadratic strategy
}

func (q *QuadraticStrategy) GetType() config.ThrottleControllerType {
	return config.QuadraticControllerType
}

func (q *QuadraticStrategy) Load() (config.ThrottleControllerType, ThrottleParams) {
	return q.GetType(), ThrottleParams{}
}

// PIDStrategy implements PID-based throttling
type PIDStrategy struct {
	config            config.PIDConfig
	threshold         uint64
	throttleTxSize    uint64
	throttleBlockSize uint64
	alwaysBlockSize   uint64

	mu             sync.Mutex
	lastError      float64
	integral       float64
	lastUpdateTime time.Time
	initialized    bool

	// Optional metrics interface for detailed PID metrics
	metrics interface {
		RecordThrottleControllerState(error, integral, derivative float64)
		RecordThrottleResponseTime(time.Duration)
	}
}

func NewPIDStrategy(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize uint64, config config.PIDConfig) *PIDStrategy {
	return &PIDStrategy{
		config:            config,
		threshold:         threshold,
		throttleTxSize:    throttleTxSize,
		throttleBlockSize: throttleBlockSize,
		alwaysBlockSize:   alwaysBlockSize,
	}
}

func (p *PIDStrategy) SetMetrics(metrics interface {
	RecordThrottleControllerState(error, integral, derivative float64)
	RecordThrottleResponseTime(time.Duration)
}) {
	p.metrics = metrics
}

func (p *PIDStrategy) Update(currentPendingBytes, targetPendingBytes uint64) ThrottleParams {
	startTime := time.Now()
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	if !p.initialized {
		p.lastUpdateTime = now
		p.initialized = true
		p.lastError = 0
		p.integral = 0
	}

	// Check if enough time has passed since last update
	dt := now.Sub(p.lastUpdateTime)
	if dt < p.config.SampleTime {
		intensity := p.calculateCurrentIntensity()
		return p.buildThrottleParams(intensity)
	}

	p.lastUpdateTime = now

	if targetPendingBytes == 0 {
		targetPendingBytes = p.threshold
	}

	// Calculate error (positive when above target)
	pendingBytesError := float64(int64(currentPendingBytes) - int64(targetPendingBytes))

	// Only apply PID control if we're above the base threshold
	var intensity float64 = 0.0
	if currentPendingBytes > p.threshold {
		// Normalize error by threshold to get a reasonable scale
		normalizedError := pendingBytesError / float64(p.threshold)

		proportional := p.config.Kp * normalizedError

		// Update integral term with windup protection
		p.integral += normalizedError * dt.Seconds()
		if p.integral > p.config.IntegralMax {
			p.integral = p.config.IntegralMax
		} else if p.integral < -p.config.IntegralMax {
			p.integral = -p.config.IntegralMax
		}
		integralTerm := p.config.Ki * p.integral

		// Calculate derivative term
		derivative := (normalizedError - p.lastError) / dt.Seconds()
		derivativeTerm := p.config.Kd * derivative

		// Combine PID terms
		pidOutput := proportional + integralTerm + derivativeTerm

		// Clamp output to valid range [0, OutputMax]
		intensity = math.Max(0, math.Min(p.config.OutputMax, pidOutput))

		p.lastError = normalizedError

		if p.metrics != nil {
			p.metrics.RecordThrottleControllerState(pendingBytesError, p.integral, derivative)
			p.metrics.RecordThrottleResponseTime(time.Since(startTime))
		}
	} else {
		// Below threshold - reset integral term to prevent windup
		p.integral = 0
		p.lastError = 0
	}

	return p.buildThrottleParams(intensity)
}

func (p *PIDStrategy) calculateCurrentIntensity() float64 {
	return math.Max(0, math.Min(1, p.config.Kp*p.lastError))
}

func (p *PIDStrategy) buildThrottleParams(intensity float64) ThrottleParams {
	maxBlockSize := p.alwaysBlockSize
	var maxTxSize uint64 = 0

	if intensity > 0 {
		maxTxSize = p.throttleTxSize

		if p.throttleBlockSize != 0 {
			if maxBlockSize == 0 || p.throttleBlockSize < maxBlockSize {
				targetBlockSize := p.throttleBlockSize
				if maxBlockSize > 0 {
					targetBlockSize = uint64(float64(maxBlockSize) - intensity*float64(maxBlockSize-p.throttleBlockSize))
				}
				maxBlockSize = targetBlockSize
			}
		}
	}

	return ThrottleParams{
		MaxTxSize:    maxTxSize,
		MaxBlockSize: maxBlockSize,
		Intensity:    math.Max(0, math.Min(1, intensity)),
	}
}

func (p *PIDStrategy) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastError = 0
	p.integral = 0
	p.initialized = false
}

func (p *PIDStrategy) GetType() config.ThrottleControllerType {
	return config.PIDControllerType
}

func (p *PIDStrategy) Load() (config.ThrottleControllerType, ThrottleParams) {
	return p.GetType(), ThrottleParams{}
}

// ThrottleControllerFactory creates throttle controllers based on configuration
type ThrottleControllerFactory struct {
	log log.Logger
}

func NewThrottleControllerFactory(log log.Logger) *ThrottleControllerFactory {
	return &ThrottleControllerFactory{
		log: log,
	}
}

func (f *ThrottleControllerFactory) CreateController(
	controllerType config.ThrottleControllerType,
	threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize uint64,
	thresholdMultiplier float64,
	pidConfig *config.PIDConfig,
) (*ThrottleController, error) {
	var strategy ThrottleStrategy

	switch controllerType {
	case config.StepControllerType:
		strategy = NewStepStrategy(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize)
	case config.LinearControllerType:
		strategy = NewLinearStrategy(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize, thresholdMultiplier, f.log)
	case config.QuadraticControllerType:
		strategy = NewQuadraticStrategy(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize, thresholdMultiplier, f.log)
	case config.PIDControllerType:
		if pidConfig == nil {
			return nil, fmt.Errorf("PID configuration required for PID controller")
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

		strategy = NewPIDStrategy(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize, *pidConfig)
	default:
		return nil, fmt.Errorf("unknown throttle controller type: %s", controllerType)
	}

	return NewThrottleController(strategy), nil
}
