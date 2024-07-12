package event

import "time"

type MetricsTracer struct {
	metrics Metrics
}

var _ Tracer = (*MetricsTracer)(nil)

func NewMetricsTracer(m Metrics) *MetricsTracer {
	return &MetricsTracer{metrics: m}
}

func (mt *MetricsTracer) OnDeriveStart(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time) {
}

func (mt *MetricsTracer) OnDeriveEnd(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time, duration time.Duration, effect bool) {
	if !effect { // don't count events that were just pass-through and not of any effect
		return
	}
	mt.metrics.RecordProcessedEvent(ev.Event.String(), name, duration)
}

func (mt *MetricsTracer) OnRateLimited(name string, derivContext uint64) {
	mt.metrics.RecordEventsRateLimited()
}

func (mt *MetricsTracer) OnEmit(name string, ev AnnotatedEvent, derivContext uint64, emitTime time.Time) {
	mt.metrics.RecordEmittedEvent(ev.Event.String(), name)
}
