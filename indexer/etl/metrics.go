package etl

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	MetricsNamespace string = "etl"
)

type Metricer interface {
	RecordInterval() (done func(err error))

	// Batch Extraction
	RecordBatchLatestHeight(height *big.Int)
	RecordBatchHeaders(size int)
	RecordBatchLog(contractAddress common.Address)

	// Indexed Batches
	RecordIndexedLatestHeight(height *big.Int)
	RecordIndexedHeaders(size int)
	RecordIndexedLogs(size int)
}

type etlMetrics struct {
	intervalTick     prometheus.Counter
	intervalDuration prometheus.Histogram

	batchFailures     prometheus.Counter
	batchLatestHeight prometheus.Gauge
	batchHeaders      prometheus.Counter
	batchLogs         *prometheus.CounterVec

	indexedLatestHeight prometheus.Gauge
	indexedHeaders      prometheus.Counter
	indexedLogs         prometheus.Counter
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
		batchFailures: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "failures_total",
			Help:      "number of times the etl encountered a failure to extract a batch",
		}),
		batchLatestHeight: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "height",
			Help:      "the latest block height observed by an etl interval",
		}),
		batchHeaders: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "headers_total",
			Help:      "number of headers observed by the etl",
		}),
		batchLogs: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "logs_total",
			Help:      "number of logs observed by the etl",
		}, []string{
			"contract",
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
		indexedLogs: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "indexed_logs_total",
			Help:      "number of logs indexed by the etl",
		}),
	}
}

func (m *etlMetrics) RecordInterval() func(error) {
	m.intervalTick.Inc()
	timer := prometheus.NewTimer(m.intervalDuration)
	return func(err error) {
		if err != nil {
			m.batchFailures.Inc()
		}

		timer.ObserveDuration()
	}
}

func (m *etlMetrics) RecordBatchLatestHeight(height *big.Int) {
	m.batchLatestHeight.Set(float64(height.Uint64()))
}

func (m *etlMetrics) RecordBatchHeaders(size int) {
	m.batchHeaders.Add(float64(size))
}

func (m *etlMetrics) RecordBatchLog(contractAddress common.Address) {
	m.batchLogs.WithLabelValues(contractAddress.String()).Inc()
}

func (m *etlMetrics) RecordIndexedLatestHeight(height *big.Int) {
	m.indexedLatestHeight.Set(float64(height.Uint64()))
}

func (m *etlMetrics) RecordIndexedHeaders(size int) {
	m.indexedHeaders.Add(float64(size))
}

func (m *etlMetrics) RecordIndexedLogs(size int) {
	m.indexedLogs.Add(float64(size))
}
