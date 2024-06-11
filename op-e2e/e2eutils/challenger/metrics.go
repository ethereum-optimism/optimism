package challenger

import (
	"sync/atomic"

	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
)

type CapturingMetrics struct {
	metrics.NoopMetricsImpl

	HighestActedL1Block atomic.Uint64
}

func NewCapturingMetrics() *CapturingMetrics {
	return &CapturingMetrics{}
}

var _ metrics.Metricer = (*CapturingMetrics)(nil)

func (c *CapturingMetrics) RecordActedL1Block(block uint64) {
	c.HighestActedL1Block.Store(block)
}
