package metrics

import (
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type ServiceMetricer interface {
	RecordInfo(version string)
	RecordUp()
}

type ServiceMetrics struct {
	info *prometheus.GaugeVec
	up   prometheus.Gauge
}

var _ ServiceMetricer = (*ServiceMetrics)(nil)

func NewServiceMetrics(ns string, factory metrics.Factory) *ServiceMetrics {
	return &ServiceMetrics{
		info: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "info",
			Help:      "Pseudo-metric tracking version and config info",
		}, []string{
			"version",
		}),
		up: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "up",
			Help:      "1 if the op-node has finished starting up",
		}),
	}
}

// RecordInfo sets a pseudo-metric that contains versioning and config info for the op-node-v2.
func (m *ServiceMetrics) RecordInfo(version string) {
	m.info.WithLabelValues(version).Set(1)
}

// RecordUp sets the up metric to 1.
func (m *ServiceMetrics) RecordUp() {
	m.up.Set(1)
}

type NoopServiceMetrics struct{}

var _ ServiceMetricer = NoopServiceMetrics{}

func (NoopServiceMetrics) RecordInfo(version string) {}

func (NoopServiceMetrics) RecordUp() {}
