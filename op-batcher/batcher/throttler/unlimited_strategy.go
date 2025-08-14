package throttler

import "github.com/ethereum-optimism/optimism/op-batcher/config"

type UnlimitedStrategy struct {
}

func NewUnlimitedStrategy() *UnlimitedStrategy {
	return &UnlimitedStrategy{}
}

func (s *UnlimitedStrategy) GetType() config.ThrottleControllerType {
	return config.UnlimitedControllerType
}

func (s *UnlimitedStrategy) Update(currentPendingBytes uint64) float64 {
	return 0.0
}

func (s *UnlimitedStrategy) Reset() {
}

func (s *UnlimitedStrategy) Load() (config.ThrottleControllerType, float64) {
	return s.GetType(), 0.0
}
