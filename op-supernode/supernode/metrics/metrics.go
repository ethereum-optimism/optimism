// Package metrics provides a set of metrics for the op-supernode
package metrics

import (
	"net"
	"strconv"

	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ophttp "github.com/ethereum-optimism/optimism/op-service/httputil"

)

const Namespace = "op_node"

type Metricer interface {
	RecordInfo(version string)
	RecordUp()
}

// Metrics tracks all the metrics for the op-node.
type Metrics struct {
	Info *prometheus.GaugeVec
	Up   prometheus.Gauge

	registry *prometheus.Registry
	factory  metrics.Factory
}

var _ Metricer = (*Metrics)(nil)

// NewMetrics creates a new [Metrics] instance with the given process name.
func NewMetrics(procName string, labels prometheus.Labels) *Metrics {
	if procName == "" {
		procName = "default"
	}
	ns := Namespace + "_" + procName

	registry := prometheus.NewRegistry()

	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registry.MustRegister(collectors.NewGoCollector())
	if labels != nil {
		// We use a type assertion here because the Metrics type
		// is tightly bound to the prometheus.Registry type.
		registry = prometheus.WrapRegistererWith(labels, registry).(*prometheus.Registry)
	}
	factory := metrics.With(registry)
	return &Metrics{
		Info: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "info",
			Help:      "Pseudo-metric tracking version and config info",
		}, []string{
			"version",
		}),
		Up: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "up",
			Help:      "1 if the op node has finished starting up",
		}),

		registry: registry,
		factory:  factory,
	}
}

// RecordInfo sets a pseudo-metric that contains versioning and
// config info for the opnode.
func (m *Metrics) RecordInfo(version string) {
	m.Info.WithLabelValues(version).Set(1)
}

// RecordUp sets the up metric to 1.
func (m *Metrics) RecordUp() {
	m.Up.Set(1)
}

// StartServer starts the metrics server on the given hostname and port.
func (m *Metrics) StartServer(hostname string, port int) (*ophttp.HTTPServer, error) {
	addr := net.JoinHostPort(hostname, strconv.Itoa(port))
	h := promhttp.InstrumentMetricHandler(
		m.registry, promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}),
	)
	return ophttp.StartHTTPServer(addr, h)
}
