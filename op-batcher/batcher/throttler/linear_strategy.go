package throttler

import (
	"sync"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/ethereum/go-ethereum/log"
)

// LinearStrategy implements linear throttling for a smoother and more eager response than the step strategy
type LinearStrategy struct {
	threshold    uint64
	maxThreshold uint64

	mu               sync.RWMutex
	currentIntensity float64
}

func NewLinearStrategy(threshold uint64, multiplier float64, log log.Logger) *LinearStrategy {
	maxThreshold := uint64(float64(threshold) * multiplier)
	// Ensure maxThreshold is always greater than threshold to prevent division by zero
	if maxThreshold <= threshold {
		maxThreshold = threshold + 1
		log.Warn("maxThreshold is less than or equal to threshold, setting maxThreshold to threshold + 1", "threshold", threshold, "multiplier", multiplier, "maxThreshold", maxThreshold)
	}
	return &LinearStrategy{
		threshold:        threshold,
		maxThreshold:     maxThreshold,
		currentIntensity: 0.0,
	}
}

func (q *LinearStrategy) Update(currentPendingBytes uint64) float64 {
	var intensity float64 = 0.0

	if currentPendingBytes > q.threshold {
		// Linear scaling from threshold to maxThreshold
		if currentPendingBytes >= q.maxThreshold {
			intensity = 1.0
		} else {
			// Linear interpolation (x curve for more aggressive throttling)
			intensity = float64(currentPendingBytes-q.threshold) / float64(q.maxThreshold-q.threshold)
		}
	}

	q.mu.Lock()
	q.currentIntensity = intensity
	q.mu.Unlock()

	return intensity
}

func (q *LinearStrategy) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.currentIntensity = 0.0
}

func (q *LinearStrategy) GetType() config.ThrottleControllerType {
	return config.LinearControllerType
}

func (q *LinearStrategy) Load() (config.ThrottleControllerType, float64) {
	q.mu.RLock()
	intensity := q.currentIntensity
	q.mu.RUnlock()
	return q.GetType(), intensity
}
