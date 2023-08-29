package etl

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	MetricsNamespace string = "etl"

	_ Metricer = &metricer{}
)

type Metrics interface {
	newMetricer(etl string) Metricer
}

type Metricer interface {
	RecordInterval() (done func(err error))

	// Batch Extraction
	RecordBatchFailure()
	RecordBatchLatestHeight(height *big.Int)
	RecordBatchHeaders(size int)
	RecordBatchLog(contractAddress common.Address)

	// Indexed Batches
	RecordIndexedLatestHeight(height *big.Int)
	RecordIndexedHeaders(size int)
	RecordIndexedLogs(size int)
}

type etlMetrics struct {
	intervalTick     *prometheus.CounterVec
	intervalDuration *prometheus.HistogramVec

	batchFailures     *prometheus.CounterVec
	batchLatestHeight *prometheus.GaugeVec
	batchHeaders      *prometheus.CounterVec
	batchLogs         *prometheus.CounterVec

	indexedLatestHeight *prometheus.GaugeVec
	indexedHeaders      *prometheus.CounterVec
	indexedLogs         *prometheus.CounterVec
}

type metricerFactory struct {
	metrics *etlMetrics
}

type metricer struct {
	etl     string
	metrics *etlMetrics
}

func NewMetrics(registry *prometheus.Registry) Metrics {
	return &metricerFactory{metrics: newMetrics(registry)}
}

func (factory *metricerFactory) newMetricer(etl string) Metricer {
	return &metricer{etl, factory.metrics}
}

func newMetrics(registry *prometheus.Registry) *etlMetrics {
	factory := metrics.With(registry)
	return &etlMetrics{
		intervalTick: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "intervals_total",
			Help:      "number of times the etl has run its extraction loop",
		}, []string{
			"etl",
		}),
		intervalDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: MetricsNamespace,
			Name:      "interval_seconds",
			Help:      "duration elapsed for during the processing loop",
		}, []string{
			"etl",
		}),
		batchFailures: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "failures_total",
			Help:      "number of times the etl encountered a failure to extract a batch",
		}, []string{
			"etl",
		}),
		batchLatestHeight: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "height",
			Help:      "the latest block height observed by an etl interval",
		}, []string{
			"etl",
		}),
		batchHeaders: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "headers_total",
			Help:      "number of headers observed by the etl",
		}, []string{
			"etl",
		}),
		batchLogs: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "logs_total",
			Help:      "number of logs observed by the etl",
		}, []string{
			"etl",
			"contract",
		}),
		indexedLatestHeight: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "indexed_height",
			Help:      "the latest block height indexed into the database",
		}, []string{
			"etl",
		}),
		indexedHeaders: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "indexed_headers_total",
			Help:      "number of headers indexed by the etl",
		}, []string{
			"etl",
		}),
		indexedLogs: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "indexed_logs_total",
			Help:      "number of logs indexed by the etl",
		}, []string{
			"etl",
		}),
	}
}

func (m *metricer) RecordInterval() func(error) {
	m.metrics.intervalTick.WithLabelValues(m.etl).Inc()
	timer := prometheus.NewTimer(m.metrics.intervalDuration.WithLabelValues(m.etl))
	return func(err error) {
		if err != nil {
			m.RecordBatchFailure()
		}

		timer.ObserveDuration()
	}
}

func (m *metricer) RecordBatchFailure() {
	m.metrics.batchFailures.WithLabelValues(m.etl).Inc()
}

func (m *metricer) RecordBatchLatestHeight(height *big.Int) {
	m.metrics.batchLatestHeight.WithLabelValues(m.etl).Set(float64(height.Uint64()))
}

func (m *metricer) RecordBatchHeaders(size int) {
	m.metrics.batchHeaders.WithLabelValues(m.etl).Add(float64(size))
}

func (m *metricer) RecordBatchLog(contractAddress common.Address) {
	m.metrics.batchLogs.WithLabelValues(m.etl, contractAddress.String()).Inc()
}

func (m *metricer) RecordIndexedLatestHeight(height *big.Int) {
	m.metrics.indexedLatestHeight.WithLabelValues(m.etl).Set(float64(height.Uint64()))
}

func (m *metricer) RecordIndexedHeaders(size int) {
	m.metrics.indexedHeaders.WithLabelValues(m.etl).Add(float64(size))
}

func (m *metricer) RecordIndexedLogs(size int) {
	m.metrics.indexedLogs.WithLabelValues(m.etl).Add(float64(size))
}
