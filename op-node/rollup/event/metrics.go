package event

type Metrics interface {
	RecordEmittedEvent(name string)
	RecordProcessedEvent(name string)
}

type NoopMetrics struct {
}

func (n NoopMetrics) RecordEmittedEvent(name string) {}

func (n NoopMetrics) RecordProcessedEvent(name string) {}

var _ Metrics = NoopMetrics{}
