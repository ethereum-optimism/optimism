package event

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

type Metrics interface {
	RecordEmittedEvent(eventName string, emitter string)
	RecordProcessedEvent(eventName string, deriver string, duration time.Duration)
	RecordEventsRateLimited()
	RecordDequeuedEvents(eventName string, deriver string)
	EnqueuedEventIncrement(eventName string)
	EnqueuedEventDecrement(eventName string)
	SetTotalEnqueuedEvents(current uint64)
}

type NoopMetrics struct {
}

func (n NoopMetrics) RecordEmittedEvent(eventName string, emitter string) {}

func (n NoopMetrics) RecordProcessedEvent(eventName string, deriver string, duration time.Duration) {}

func (n NoopMetrics) RecordEventsRateLimited() {}

func (n NoopMetrics) RecordDequeuedEvents(eventName string, deriver string) {}

func (n NoopMetrics) SetTotalEnqueuedEvents(current uint64) {}

func (n NoopMetrics) EnqueuedEventIncrement(eventName string) {}

func (n NoopMetrics) EnqueuedEventDecrement(eventName string) {}

var _ Metrics = NoopMetrics{}

type EventMetricsTracker struct {
	EmittedEvents       *prometheus.CounterVec
	EnqueuedEvents      *prometheus.GaugeVec
	TotalEnqueuedEvents prometheus.Gauge
	ProcessedEvents     *prometheus.CounterVec

	// We don't use a histogram for observing time durations,
	// as each vec entry (event-type, deriver type) is synchronous with other occurrences of the same entry key,
	// so we can get a reasonably good understanding of execution by looking at the rate.
	// Bucketing to detect outliers would be nice, but also increases the overhead by a lot,
	// where we already track many event-type/deriver combinations.
	EventsProcessTime *prometheus.CounterVec

	EventsRateLimited *metrics.Event

	DequeuedEvents *prometheus.CounterVec
}

func NewMetricsTracker(ns string, factory metrics.Factory) *EventMetricsTracker {
	return &EventMetricsTracker{
		EmittedEvents: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: ns,
				Subsystem: "events",
				Name:      "emitted",
				Help:      "number of emitted events",
			}, []string{"event_type", "emitter"}),

		TotalEnqueuedEvents: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: "events",
			Name:      "total_enqueued",
			Help:      "Gauge representing the current total number of enqueued events",
		}),

		EnqueuedEvents: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: ns,
				Subsystem: "events",
				Name:      "enqueued",
				Help:      "Gauge representing the number of enqueued events per type",
			}, []string{"event_type"}),

		ProcessedEvents: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: ns,
				Subsystem: "events",
				Name:      "processed",
				Help:      "number of processed events",
			}, []string{"event_type", "deriver"}),

		EventsProcessTime: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: ns,
				Subsystem: "events",
				Name:      "process_time",
				Help:      "total duration in seconds of processed events",
			}, []string{"event_type", "deriver"}),

		EventsRateLimited: metrics.NewEvent(factory, ns, "events", "rate_limited", "events rate limiter hits"),

		DequeuedEvents: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: ns,
				Subsystem: "events",
				Name:      "dequeued",
				Help:      "number of dequeued events",
			}, []string{"event_type", "deriver"}),
	}
}

func (m *EventMetricsTracker) RecordEmittedEvent(eventName string, emitter string) {
	m.EmittedEvents.WithLabelValues(eventName, emitter).Inc()
}

func (m *EventMetricsTracker) EnqueuedEventIncrement(eventName string) {
	m.EnqueuedEvents.WithLabelValues(eventName).Inc()
}

func (m *EventMetricsTracker) EnqueuedEventDecrement(eventName string) {
	m.EnqueuedEvents.WithLabelValues(eventName).Dec()
}

func (m *EventMetricsTracker) SetTotalEnqueuedEvents(current uint64) {
	m.TotalEnqueuedEvents.Set(float64(current))
}

func (m *EventMetricsTracker) RecordProcessedEvent(eventName string, deriver string, duration time.Duration) {
	m.ProcessedEvents.WithLabelValues(eventName, deriver).Inc()
	// We take the absolute value; if the clock was not monotonically increased between start and top,
	// there still was a duration gap. And the Counter metrics-type would panic if the duration is negative.
	m.EventsProcessTime.WithLabelValues(eventName, deriver).Add(float64(duration.Abs()) / float64(time.Second))
}

func (m *EventMetricsTracker) RecordEventsRateLimited() {
	m.EventsRateLimited.Record()
}

func (m *EventMetricsTracker) RecordDequeuedEvents(eventName string, deriver string) {
	m.DequeuedEvents.WithLabelValues(eventName, deriver).Inc()
}
