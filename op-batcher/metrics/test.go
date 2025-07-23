package metrics

type TestMetrics struct {
	noopMetrics
	PendingBlocksBytesCurrent float64
	ChannelQueueLength        int
}

var _ Metricer = new(TestMetrics)

func (m *TestMetrics) RecordChannelQueueLength(l int) {
	m.ChannelQueueLength = l
}
func (m *TestMetrics) ClearAllStateMetrics() {
	m.PendingBlocksBytesCurrent = 0
	m.ChannelQueueLength = 0
}
