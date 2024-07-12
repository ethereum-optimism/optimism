package event

import "time"

type Metrics interface {
	RecordEmittedEvent(eventName string, emitter string)
	RecordProcessedEvent(eventName string, deriver string, duration time.Duration)
	RecordEventsRateLimited()
}

type NoopMetrics struct {
}

func (n NoopMetrics) RecordEmittedEvent(eventName string, emitter string) {}

func (n NoopMetrics) RecordProcessedEvent(eventName string, deriver string, duration time.Duration) {}

func (n NoopMetrics) RecordEventsRateLimited() {}

var _ Metrics = NoopMetrics{}
