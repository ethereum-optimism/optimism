package throttler

import (
	"sync"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/ethereum/go-ethereum/log"
)

// LinearStrategy implements linear throttling for a smoother and more eager response than the step strategy
type LinearStrategy struct {
	lowerThreshold uint64
	upperThreshold uint64

	mu               sync.RWMutex
	currentIntensity float64
}

func NewLinearStrategy(lowerThreshold uint64, upperThreshold uint64, log log.Logger) *LinearStrategy {
	if upperThreshold <= lowerThreshold {
		panic("maxThreshold must be greater than threshold")
	}

	return &LinearStrategy{
		lowerThreshold:   lowerThreshold,
		upperThreshold:   upperThreshold,
		currentIntensity: 0.0,
	}
}

func (q *LinearStrategy) Update(currentPendingBytes uint64) float64 {
	var intensity float64 = 0.0

	if currentPendingBytes > q.lowerThreshold {
		// Linear scaling from threshold to maxThreshold
		if currentPendingBytes >= q.upperThreshold {
			intensity = 1.0
		} else {
			// Linear interpolation (x curve for more aggressive throttling)
			intensity = float64(currentPendingBytes-q.lowerThreshold) / float64(q.upperThreshold-q.lowerThreshold)
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
