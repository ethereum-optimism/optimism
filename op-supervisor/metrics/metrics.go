package metrics

import (
	"math/big"

	"github.com/prometheus/client_golang/prometheus"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

const Namespace = "op_supervisor"

type Metricer interface {
	RecordInfo(version string)
	RecordUp()

	opmetrics.RPCMetricer

	CacheAdd(chainID *big.Int, label string, cacheSize int, evicted bool)
	CacheGet(chainID *big.Int, label string, hit bool)

	Document() []opmetrics.DocumentedMetric
}

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	opmetrics.RPCMetrics

	SizeVec *prometheus.GaugeVec
	GetVec  *prometheus.CounterVec
	AddVec  *prometheus.CounterVec

	info prometheus.GaugeVec
	up   prometheus.Gauge
}

var _ Metricer = (*Metrics)(nil)

// implements the Registry getter, for metrics HTTP server to hook into
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

		RPCMetrics: opmetrics.MakeRPCMetrics(ns, factory),

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
			Help:      "1 if the op-supervisor has finished starting up",
		}),

		SizeVec: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "source_rpc_cache_size",
			Help:      "source rpc cache cache size",
		}, []string{
			"chain",
			"type",
		}),
		GetVec: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "source_rpc_cache_get",
			Help:      "source rpc cache lookups, hitting or not",
		}, []string{
			"chain",
			"type",
			"hit",
		}),
		AddVec: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "source_rpc_cache_add",
			Help:      "source rpc cache additions, evicting previous values or not",
		}, []string{
			"chain",
			"type",
			"evicted",
		}),
	}
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

func (m *Metrics) Document() []opmetrics.DocumentedMetric {
	return m.factory.Document()
}

// RecordInfo sets a pseudo-metric that contains versioning and config info for the op-supervisor.
func (m *Metrics) RecordInfo(version string) {
	m.info.WithLabelValues(version).Set(1)
}

// RecordUp sets the up metric to 1.
func (m *Metrics) RecordUp() {
	prometheus.MustRegister()
	m.up.Set(1)
}

func (m *Metrics) CacheAdd(chainID *big.Int, label string, cacheSize int, evicted bool) {
	chain := chainID.String()
	m.SizeVec.WithLabelValues(chain, label).Set(float64(cacheSize))
	if evicted {
		m.AddVec.WithLabelValues(chain, label, "true").Inc()
	} else {
		m.AddVec.WithLabelValues(chain, label, "false").Inc()
	}
}

func (m *Metrics) CacheGet(chainID *big.Int, label string, hit bool) {
	chain := chainID.String()
	if hit {
		m.GetVec.WithLabelValues(chain, label, "true").Inc()
	} else {
		m.GetVec.WithLabelValues(chain, label, "false").Inc()
	}
}
