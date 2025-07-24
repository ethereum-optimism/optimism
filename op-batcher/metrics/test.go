package metrics

import (
	"github.com/ethereum/go-ethereum/core/types"
)

type TestMetrics struct {
	noopMetrics
	UnsafeBlocksBytesCurrent float64
	ChannelQueueLength       int
	unsafeDABytes            float64
}

var _ Metricer = new(TestMetrics)

func (m *TestMetrics) RecordL2BlockInUnsafeQueue(block *types.Block) {
	daSize, rawSize := estimateBatchSize(block)
	m.UnsafeBlocksBytesCurrent += float64(rawSize)
	m.unsafeDABytes += float64(daSize)
}
func (m *TestMetrics) RecordL2BlockDequeued(block *types.Block) {
	daSize, rawSize := estimateBatchSize(block)
	m.UnsafeBlocksBytesCurrent -= float64(rawSize)
	m.unsafeDABytes -= float64(daSize)
}
func (m *TestMetrics) RecordChannelQueueLength(l int) {
	m.ChannelQueueLength = l
}
func (m *TestMetrics) ClearAllStateMetrics() {
	m.UnsafeBlocksBytesCurrent = 0
	m.ChannelQueueLength = 0
	m.unsafeDABytes = 0
}
