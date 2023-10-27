package etl

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	MetricsNamespace string = "op_indexer_etl"
)

type Metricer interface {
	RecordInterval() (done func(err error))
	RecordLatestHeight(height *big.Int)

	// Indexed Batches
	RecordIndexedLatestHeight(height *big.Int)
	RecordIndexedHeaders(size int)
	RecordIndexedLog(contractAddress common.Address)
}

type etlMetrics struct {
	intervalTick     prometheus.Counter
	intervalDuration prometheus.Histogram
	intervalFailures prometheus.Counter
	latestHeight     prometheus.Gauge

	indexedLatestHeight prometheus.Gauge
	indexedHeaders      prometheus.Counter
	indexedLogs         *prometheus.CounterVec
}

func NewMetrics(registry *prometheus.Registry, subsystem string) Metricer {
	factory := metrics.With(registry)
	return &etlMetrics{
		intervalTick: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "intervals_total",
			Help:      "number of times the etl has run its extraction loop",
		}),
		intervalDuration: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "interval_seconds",
			Help:      "duration elapsed for during the processing loop",
		}),
		intervalFailures: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "interval_failures_total",
			Help:      "number of times the etl encountered a failure during the processing loop",
		}),
		latestHeight: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "latest_height",
			Help:      "the latest height reported by the connected client",
		}),
		indexedLatestHeight: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "indexed_height",
			Help:      "the latest block height indexed into the database",
		}),
		indexedHeaders: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "indexed_headers_total",
			Help:      "number of headers indexed by the etl",
		}),
		indexedLogs: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "indexed_logs_total",
			Help:      "number of logs indexed by the etl",
		}, []string{
			"contract",
		}),
	}
}

func (m *etlMetrics) RecordInterval() func(error) {
	m.intervalTick.Inc()
	timer := prometheus.NewTimer(m.intervalDuration)
	return func(err error) {
		if err != nil {
			m.intervalFailures.Inc()
		}
		timer.ObserveDuration()
	}
}

func (m *etlMetrics) RecordLatestHeight(height *big.Int) {
	m.latestHeight.Set(float64(height.Uint64()))
}

func (m *etlMetrics) RecordIndexedLatestHeight(height *big.Int) {
	m.indexedLatestHeight.Set(float64(height.Uint64()))
}

func (m *etlMetrics) RecordIndexedHeaders(size int) {
	m.indexedHeaders.Add(float64(size))
}

func (m *etlMetrics) RecordIndexedLog(addr common.Address) {
	m.indexedLogs.WithLabelValues(addr.String()).Inc()
}
