package metrics

import (
	"github.com/ethereum/go-ethereum/core/types"
)

type TestMetrics struct {
	noopMetrics
	PendingBlocksBytesCurrent float64
	ChannelQueueLength        int
	pendingDABytes            float64
}

var _ Metricer = new(TestMetrics)

func (m *TestMetrics) RecordL2BlockInPendingQueue(block *types.Block) {
	daSize, rawSize := estimateBatchSize(block)
	m.PendingBlocksBytesCurrent += float64(rawSize)
	m.pendingDABytes += float64(daSize)
}
func (m *TestMetrics) RecordL2BlockInChannel(block *types.Block) {
	daSize, rawSize := estimateBatchSize(block)
	m.PendingBlocksBytesCurrent -= float64(rawSize)
	m.pendingDABytes -= float64(daSize)
}
func (m *TestMetrics) RecordChannelQueueLength(l int) {
	m.ChannelQueueLength = l
}
func (m *TestMetrics) PendingDABytes() float64 {
	return m.pendingDABytes
}
func (m *TestMetrics) ClearAllStateMetrics() {
	m.PendingBlocksBytesCurrent = 0
	m.ChannelQueueLength = 0
	m.pendingDABytes = 0
}
