package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

const Namespace = "op_sync_tester"

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	info prometheus.GaugeVec
	up   prometheus.Gauge
}

var _ Metricer = (*Metrics)(nil)

func NewMetrics(procName string) *Metrics {
	return newMetrics(procName, opmetrics.NewRegistry())
}

func newMetrics(procName string, registry *prometheus.Registry) *Metrics {
	if procName == "" {
		procName = "default"
	}
	ns := Namespace + "_" + procName

	factory := opmetrics.With(registry)
	return &Metrics{
		ns:       ns,
		registry: registry,
		factory:  factory,

		info: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "info",
			Help:      "Pseudo-metric tracking version and config info",
		}, []string{
			"version",
		}),
		up: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "up",
			Help:      "1 if op-sync-tester has finished starting up",
		}),
	}
}

// RecordInfo sets a pseudo-metric that contains versioning and config info.
func (m *Metrics) RecordInfo(version string) {
	m.info.WithLabelValues(version).Set(1)
}

// RecordUp sets the up metric to 1.
func (m *Metrics) RecordUp() {
	m.up.Set(1)
}

func (m *Metrics) Document() []opmetrics.DocumentedMetric {
	return m.factory.Document()
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}
