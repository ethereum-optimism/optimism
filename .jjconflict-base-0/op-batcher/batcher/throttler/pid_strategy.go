package throttler

import (
	"math"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
)

// ⚠️  EXPERIMENTAL FEATURE ⚠️
//
// PIDStrategy implements PID-based throttling control which is an EXPERIMENTAL feature.
// This controller should only be used by users with deep understanding of PID control theory.
// Improper configuration can lead to system instability, oscillations, or poor performance.
//
// PID (Proportional-Integral-Derivative) controllers require careful tuning of the
// Kp, Ki, and Kd parameters based on system characteristics and desired response.
// Use with extreme caution in production environments.
//
// See: https://en.wikipedia.org/wiki/PID_controller for control theory background.
//
// PIDStrategy implements PID-based throttling
type PIDStrategy struct {
	config    config.PIDConfig
	threshold uint64

	currentIntensity float64

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

func NewPIDStrategy(threshold uint64, config config.PIDConfig) *PIDStrategy {
	return &PIDStrategy{
		config:           config,
		threshold:        threshold,
		currentIntensity: 0.0,
	}
}

func (p *PIDStrategy) SetMetrics(metrics interface {
	RecordThrottleControllerState(error, integral, derivative float64)
	RecordThrottleResponseTime(time.Duration)
}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.metrics = metrics
}

func (p *PIDStrategy) Update(currentPendingBytes uint64) float64 {
	startTime := time.Now()
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	var dt time.Duration

	if !p.initialized {
		p.lastUpdateTime = now
		p.initialized = true
		p.lastError = 0
		p.integral = 0
		dt = time.Duration(0) // First update, no time delta
	} else {
		// Check if enough time has passed since last update (skip on subsequent updates)
		dt = now.Sub(p.lastUpdateTime)
		if dt < p.config.SampleTime {
			intensity := p.calculateCurrentIntensity()
			p.currentIntensity = intensity
			return intensity
		}
	}

	p.lastUpdateTime = now

	// Only apply PID control if we're above the base threshold
	var intensity float64 = 0.0
	if currentPendingBytes > p.threshold {
		// Calculate error (positive when above target)
		// Note: Error is always non-negative since we only
		// throttle when currentPendingBytes > threshold. Similarly, integral accumulates
		// only positive errors, so it remains non-negative throughout operation.
		pendingBytesError := float64(int64(currentPendingBytes) - int64(p.threshold))

		// Normalize error by threshold to get a reasonable scale
		normalizedError := pendingBytesError / float64(p.threshold)

		proportional := p.config.Kp * normalizedError

		// Update integral term with windup protection (only if dt > 0)
		if dt > 0 {
			p.integral += normalizedError * dt.Seconds()
			// Clamp integral to prevent windup (only positive clamping needed since error is always positive)
			if p.integral > p.config.IntegralMax {
				p.integral = p.config.IntegralMax
			}
		}
		integralTerm := p.config.Ki * p.integral

		// Calculate derivative term (only if dt > 0)
		var derivative float64
		if dt > 0 {
			derivative = (normalizedError - p.lastError) / dt.Seconds()
		}
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

	p.currentIntensity = intensity
	return intensity
}

func (p *PIDStrategy) calculateCurrentIntensity() float64 {
	// Return current intensity based on stored state
	return p.currentIntensity
}

func (p *PIDStrategy) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastError = 0
	p.integral = 0
	p.initialized = false
	p.currentIntensity = 0.0
}

func (p *PIDStrategy) GetType() config.ThrottleControllerType {
	return config.PIDControllerType
}

func (p *PIDStrategy) Load() (config.ThrottleControllerType, float64) {
	p.mu.Lock()
	intensity := p.currentIntensity
	p.mu.Unlock()
	return p.GetType(), intensity
}
