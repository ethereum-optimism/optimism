package throttler

import (
	"sync"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
)

// StepStrategy implements binary on/off throttling (existing behavior)
type StepStrategy struct {
	threshold uint64

	mu               sync.RWMutex
	currentIntensity float64
}

func NewStepStrategy(threshold uint64) *StepStrategy {
	return &StepStrategy{
		threshold:        threshold,
		currentIntensity: 0.0,
	}
}

func (s *StepStrategy) Update(currentPendingBytes uint64) float64 {
	var intensity float64 = 0.0

	if currentPendingBytes > s.threshold {
		intensity = 1.0
	}

	s.mu.Lock()
	s.currentIntensity = intensity
	s.mu.Unlock()

	return intensity
}

func (s *StepStrategy) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentIntensity = 0.0
}

func (s *StepStrategy) GetType() config.ThrottleControllerType {
	return config.StepControllerType
}

func (s *StepStrategy) Load() (config.ThrottleControllerType, float64) {
	s.mu.RLock()
	intensity := s.currentIntensity
	s.mu.RUnlock()
	return s.GetType(), intensity
}
