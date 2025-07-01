package batcher

import (
	"fmt"
	"math"
	"sync"
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

// ThrottleController defines the interface for throttle controllers
type ThrottleController interface {
	// Update calculates new throttling parameters based on current pending bytes
	Update(currentPendingBytes, targetPendingBytes uint64) ThrottleParams
	// Reset resets the controller state
	Reset()
	// GetType returns the controller type
	GetType() config.ThrottleControllerType
}

// StepController implements binary on/off throttling (existing behavior)
type StepController struct {
	threshold         uint64
	throttleTxSize    uint64
	throttleBlockSize uint64
	alwaysBlockSize   uint64
}

func NewStepController(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize uint64) *StepController {
	return &StepController{
		threshold:         threshold,
		throttleTxSize:    throttleTxSize,
		throttleBlockSize: throttleBlockSize,
		alwaysBlockSize:   alwaysBlockSize,
	}
}

func (s *StepController) Update(currentPendingBytes, targetPendingBytes uint64) ThrottleParams {
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

func (s *StepController) Reset() {
	// No state to reset for step controller
}

func (s *StepController) GetType() config.ThrottleControllerType {
	return config.StepControllerType
}

// LinearController implements linear throttling based on pending bytes
type LinearController struct {
	threshold         uint64
	maxThreshold      uint64 // Point at which maximum throttling is applied
	throttleTxSize    uint64
	throttleBlockSize uint64
	alwaysBlockSize   uint64
}

func NewLinearController(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize uint64, multiplier float64, log log.Logger) *LinearController {
	// Set max threshold to multiplier * base threshold for linear scaling
	maxThreshold := threshold * uint64(multiplier)
	// Ensure maxThreshold is always greater than threshold to prevent division by zero
	if maxThreshold <= threshold {
		maxThreshold = threshold + 1
		log.Warn("maxThreshold is less than or equal to threshold, setting maxThreshold to threshold + 1", "threshold", threshold, "multiplier", multiplier, "maxThreshold", maxThreshold)
	}
	return &LinearController{
		threshold:         threshold,
		maxThreshold:      maxThreshold,
		throttleTxSize:    throttleTxSize,
		throttleBlockSize: throttleBlockSize,
		alwaysBlockSize:   alwaysBlockSize,
	}
}

func (l *LinearController) Update(currentPendingBytes, targetPendingBytes uint64) ThrottleParams {
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

func (l *LinearController) Reset() {
	// No state to reset for linear controller
}

func (l *LinearController) GetType() config.ThrottleControllerType {
	return config.LinearControllerType
}

// QuadraticController implements quadratic throttling for more aggressive scaling
type QuadraticController struct {
	threshold         uint64
	maxThreshold      uint64
	throttleTxSize    uint64
	throttleBlockSize uint64
	alwaysBlockSize   uint64
}

func NewQuadraticController(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize uint64, multiplier float64, log log.Logger) *QuadraticController {
	maxThreshold := threshold * uint64(multiplier)
	// Ensure maxThreshold is always greater than threshold to prevent division by zero
	if maxThreshold <= threshold {
		maxThreshold = threshold + 1
		log.Warn("maxThreshold is less than or equal to threshold, setting maxThreshold to threshold + 1", "threshold", threshold, "multiplier", multiplier, "maxThreshold", maxThreshold)
	}
	return &QuadraticController{
		threshold:         threshold,
		maxThreshold:      maxThreshold,
		throttleTxSize:    throttleTxSize,
		throttleBlockSize: throttleBlockSize,
		alwaysBlockSize:   alwaysBlockSize,
	}
}

func (q *QuadraticController) Update(currentPendingBytes, targetPendingBytes uint64) ThrottleParams {
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

func (q *QuadraticController) Reset() {
	// No state to reset for quadratic controller
}

func (q *QuadraticController) GetType() config.ThrottleControllerType {
	return config.QuadraticControllerType
}

// PIDController implements PID-based throttling
type PIDController struct {
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

func NewPIDController(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize uint64, config config.PIDConfig) *PIDController {
	return &PIDController{
		config:            config,
		threshold:         threshold,
		throttleTxSize:    throttleTxSize,
		throttleBlockSize: throttleBlockSize,
		alwaysBlockSize:   alwaysBlockSize,
	}
}

func (p *PIDController) SetMetrics(metrics interface {
	RecordThrottleControllerState(error, integral, derivative float64)
	RecordThrottleResponseTime(time.Duration)
}) {
	p.metrics = metrics
}

func (p *PIDController) Update(currentPendingBytes, targetPendingBytes uint64) ThrottleParams {
	startTime := time.Now()
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	// Initialize on first call
	if !p.initialized {
		p.lastUpdateTime = now
		p.initialized = true
		p.lastError = 0
		p.integral = 0
	}

	// Check if enough time has passed since last update
	dt := now.Sub(p.lastUpdateTime)
	if dt < p.config.SampleTime {
		// Return current state if not enough time has passed
		intensity := p.calculateCurrentIntensity()
		return p.buildThrottleParams(intensity)
	}

	p.lastUpdateTime = now

	// Use threshold as target if no explicit target provided
	if targetPendingBytes == 0 {
		targetPendingBytes = p.threshold
	}

	// Calculate error (positive when above target)
	error := float64(int64(currentPendingBytes) - int64(targetPendingBytes))

	// Only apply PID control if we're above the base threshold
	var intensity float64 = 0.0
	if currentPendingBytes > p.threshold {
		// Proportional term
		proportional := p.config.Kp * error

		// Integral term with windup protection
		p.integral += error * dt.Seconds()
		if p.integral > p.config.IntegralMax {
			p.integral = p.config.IntegralMax
		} else if p.integral < -p.config.IntegralMax {
			p.integral = -p.config.IntegralMax
		}
		integralTerm := p.config.Ki * p.integral

		// Derivative term
		derivative := 0.0
		if dt.Seconds() > 0 {
			derivative = (error - p.lastError) / dt.Seconds()
		}
		derivativeTerm := p.config.Kd * derivative

		// Calculate PID output
		output := proportional + integralTerm + derivativeTerm

		// Normalize output to [0, 1] range
		normalizedOutput := output / float64(p.threshold)
		if normalizedOutput > p.config.OutputMax {
			normalizedOutput = p.config.OutputMax
		} else if normalizedOutput < 0 {
			normalizedOutput = 0
		}

		intensity = normalizedOutput
		p.lastError = error

		// Record PID-specific metrics if available
		if p.metrics != nil {
			p.metrics.RecordThrottleControllerState(error, p.integral, derivative)
			p.metrics.RecordThrottleResponseTime(time.Since(startTime))
		}
	} else {
		// Reset integral when below threshold
		p.integral = 0
		p.lastError = 0
	}

	return p.buildThrottleParams(intensity)
}

func (p *PIDController) calculateCurrentIntensity() float64 {
	// This is a simplified calculation for when we haven't updated recently
	// In a real implementation, you might want to store the last calculated intensity
	return 0.0
}

func (p *PIDController) buildThrottleParams(intensity float64) ThrottleParams {
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

func (p *PIDController) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastError = 0
	p.integral = 0
	p.initialized = false
}

func (p *PIDController) GetType() config.ThrottleControllerType {
	return config.PIDControllerType
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
) (ThrottleController, error) {
	switch controllerType {
	case config.StepControllerType:
		return NewStepController(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize), nil
	case config.LinearControllerType:
		return NewLinearController(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize, thresholdMultiplier, f.log), nil
	case config.QuadraticControllerType:
		return NewQuadraticController(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize, thresholdMultiplier, f.log), nil
	case config.PIDControllerType:
		if pidConfig == nil {
			return nil, fmt.Errorf("PID configuration required for PID controller")
		}
		return NewPIDController(threshold, throttleTxSize, throttleBlockSize, alwaysBlockSize, *pidConfig), nil
	default:
		return nil, fmt.Errorf("unknown throttle controller type: %s", controllerType)
	}
}
