package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type EventMetrics struct {
	Total    prometheus.Counter
	LastTime prometheus.Gauge
}

func (e *EventMetrics) RecordEvent() {
	e.Total.Inc()
	e.LastTime.Set(float64(time.Now().Unix()))
}

func NewEventMetrics(registry prometheus.Registerer, ns string, name string, displayName string) *EventMetrics {
	return &EventMetrics{
		Total: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      fmt.Sprintf("%s_total", name),
			Help:      fmt.Sprintf("Count of %s events", displayName),
		}),
		LastTime: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      fmt.Sprintf("last_%s_unix", name),
			Help:      fmt.Sprintf("Timestamp of last %s event", displayName),
		}),
	}
}
