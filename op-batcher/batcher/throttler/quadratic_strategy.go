package throttler

import (
	"sync"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/ethereum/go-ethereum/log"
)

// QuadraticStrategy implements quadratic throttling for more aggressive scaling
type QuadraticStrategy struct {
	lowerThreshold uint64
	upperThreshold uint64

	mu               sync.RWMutex
	currentIntensity float64
}

func NewQuadraticStrategy(lowerThreshold uint64, upperThreshold uint64, log log.Logger) *QuadraticStrategy {
	if upperThreshold <= lowerThreshold {
		panic("maxThreshold must be greater than threshold")
	}
	return &QuadraticStrategy{
		lowerThreshold:   lowerThreshold,
		upperThreshold:   upperThreshold,
		currentIntensity: 0.0,
	}
}

func (q *QuadraticStrategy) Update(currentPendingBytes uint64) float64 {
	var intensity float64 = 0.0

	if currentPendingBytes > q.lowerThreshold {
		// Quadratic scaling from threshold to maxThreshold
		if currentPendingBytes >= q.upperThreshold {
			intensity = 1.0
		} else {
			// Quadratic interpolation (x^2 curve for more aggressive throttling)
			linear := float64(currentPendingBytes-q.lowerThreshold) / float64(q.upperThreshold-q.lowerThreshold)
			intensity = linear * linear
		}
	}

	q.mu.Lock()
	q.currentIntensity = intensity
	q.mu.Unlock()

	return intensity
}

func (q *QuadraticStrategy) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.currentIntensity = 0.0
}

func (q *QuadraticStrategy) GetType() config.ThrottleControllerType {
	return config.QuadraticControllerType
}

func (q *QuadraticStrategy) Load() (config.ThrottleControllerType, float64) {
	q.mu.RLock()
	intensity := q.currentIntensity
	q.mu.RUnlock()
	return q.GetType(), intensity
}
