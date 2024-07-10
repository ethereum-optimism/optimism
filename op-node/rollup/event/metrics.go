package event

type Metrics interface {
	RecordEmittedEvent(name string)
	RecordProcessedEvent(name string)
	RecordEventsRateLimited()
}

type NoopMetrics struct {
}

func (n NoopMetrics) RecordEmittedEvent(name string) {}

func (n NoopMetrics) RecordProcessedEvent(name string) {}

func (n NoopMetrics) RecordEventsRateLimited() {}

var _ Metrics = NoopMetrics{}
