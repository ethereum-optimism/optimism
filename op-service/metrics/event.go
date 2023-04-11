package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

type Event struct {
	Total    prometheus.Counter
	LastTime prometheus.Gauge
}

func (e *Event) Record() {
	e.Total.Inc()
	e.LastTime.SetToCurrentTime()
}

func NewEvent(factory Factory, ns string, name string, displayName string) Event {
	return Event{
		Total: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      fmt.Sprintf("%s_total", name),
			Help:      fmt.Sprintf("Count of %s events", displayName),
		}),
		LastTime: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      fmt.Sprintf("last_%s_unix", name),
			Help:      fmt.Sprintf("Timestamp of last %s event", displayName),
		}),
	}
}

type EventVec struct {
	Total    prometheus.CounterVec
	LastTime prometheus.GaugeVec
}

func (e *EventVec) Record(lvs ...string) {
	e.Total.WithLabelValues(lvs...).Inc()
	e.LastTime.WithLabelValues(lvs...).SetToCurrentTime()
}

func NewEventVec(factory Factory, ns string, name string, displayName string, labelNames []string) EventVec {
	return EventVec{
		Total: *factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      fmt.Sprintf("%s_total", name),
			Help:      fmt.Sprintf("Count of %s events", displayName),
		}, labelNames),
		LastTime: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      fmt.Sprintf("last_%s_unix", name),
			Help:      fmt.Sprintf("Timestamp of last %s event", displayName),
		}, labelNames),
	}
}
