package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

const Namespace = "op_interop_filter"

type Metricer interface {
	RecordInfo(version string)
	RecordUp()
	RecordFailsafeEnabled(enabled bool)
	RecordChainHead(chainID uint64, blockNum uint64)
	RecordCheckAccessList(success bool)
}

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	info             *prometheus.GaugeVec
	up               prometheus.Gauge
	failsafeEnabled  prometheus.Gauge
	chainHead        *prometheus.GaugeVec
	checkAccessTotal *prometheus.CounterVec
}

var _ Metricer = (*Metrics)(nil)
var _ opmetrics.RegistryMetricer = (*Metrics)(nil)

func NewMetrics(procName string) *Metrics {
	if procName == "" {
		procName = "default"
	}
	ns := Namespace + "_" + procName

	registry := opmetrics.NewRegistry()
	factory := opmetrics.With(registry)

	return &Metrics{
		ns:       ns,
		registry: registry,
		factory:  factory,

		info: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "info",
			Help:      "Service info",
		}, []string{"version"}),

		up: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "up",
			Help:      "1 if service is up",
		}),

		failsafeEnabled: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "failsafe_enabled",
			Help:      "1 if failsafe is enabled",
		}),

		chainHead: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "chain_head",
			Help:      "Latest ingested block number",
		}, []string{"chain_id"}),

		checkAccessTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "check_access_list_total",
			Help:      "Total checkAccessList requests",
		}, []string{"success"}),
	}
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

func (m *Metrics) Document() []opmetrics.DocumentedMetric {
	return m.factory.Document()
}

func (m *Metrics) RecordInfo(version string) {
	m.info.WithLabelValues(version).Set(1)
}

func (m *Metrics) RecordUp() {
	m.up.Set(1)
}

func (m *Metrics) RecordFailsafeEnabled(enabled bool) {
	if enabled {
		m.failsafeEnabled.Set(1)
	} else {
		m.failsafeEnabled.Set(0)
	}
}

func (m *Metrics) RecordChainHead(chainID uint64, blockNum uint64) {
	m.chainHead.WithLabelValues(strconv.FormatUint(chainID, 10)).Set(float64(blockNum))
}

func (m *Metrics) RecordCheckAccessList(success bool) {
	label := "false"
	if success {
		label = "true"
	}
	m.checkAccessTotal.WithLabelValues(label).Inc()
}

// NoopMetrics is a no-op implementation for testing
var NoopMetrics Metricer = &noopMetrics{}

type noopMetrics struct{}

func (n *noopMetrics) RecordInfo(version string)                       {}
func (n *noopMetrics) RecordUp()                                       {}
func (n *noopMetrics) RecordFailsafeEnabled(enabled bool)              {}
func (n *noopMetrics) RecordChainHead(chainID uint64, blockNum uint64) {}
func (n *noopMetrics) RecordCheckAccessList(success bool)              {}
