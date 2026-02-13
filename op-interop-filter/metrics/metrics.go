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
	RecordBackfillProgress(chainID uint64, progress float64)
	RecordReorgDetected(chainID uint64)
	RecordLogsAdded(chainID uint64, count int64)
	RecordBlocksSealed(chainID uint64, count int64)
	RecordCrossUnsafeValidatedTimestamp(timestamp uint64)
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

	// Chain-specific metrics
	backfillProgress              *prometheus.GaugeVec
	reorgDetectedTotal            *prometheus.CounterVec
	logsAddedTotal                *prometheus.CounterVec
	blocksSealedTotal             *prometheus.CounterVec
	crossUnsafeValidatedTimestamp prometheus.Gauge
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

		backfillProgress: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "backfill_progress",
			Help:      "Backfill progress (0-1) per chain",
		}, []string{"chain_id"}),

		reorgDetectedTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "reorg_detected_total",
			Help:      "Total reorgs detected per chain",
		}, []string{"chain_id"}),

		logsAddedTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "logs_added_total",
			Help:      "Total logs added to DB per chain",
		}, []string{"chain_id"}),

		blocksSealedTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "blocks_sealed_total",
			Help:      "Total blocks sealed per chain",
		}, []string{"chain_id"}),

		crossUnsafeValidatedTimestamp: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "cross_unsafe_validated_timestamp",
			Help:      "Latest cross-unsafe validated timestamp",
		}),
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

func (m *Metrics) RecordBackfillProgress(chainID uint64, progress float64) {
	m.backfillProgress.WithLabelValues(strconv.FormatUint(chainID, 10)).Set(progress)
}

func (m *Metrics) RecordReorgDetected(chainID uint64) {
	m.reorgDetectedTotal.WithLabelValues(strconv.FormatUint(chainID, 10)).Inc()
}

func (m *Metrics) RecordLogsAdded(chainID uint64, count int64) {
	m.logsAddedTotal.WithLabelValues(strconv.FormatUint(chainID, 10)).Add(float64(count))
}

func (m *Metrics) RecordBlocksSealed(chainID uint64, count int64) {
	m.blocksSealedTotal.WithLabelValues(strconv.FormatUint(chainID, 10)).Add(float64(count))
}

func (m *Metrics) RecordCrossUnsafeValidatedTimestamp(timestamp uint64) {
	m.crossUnsafeValidatedTimestamp.Set(float64(timestamp))
}

// NoopMetrics is a no-op implementation for testing
var NoopMetrics Metricer = &noopMetrics{}

type noopMetrics struct{}

func (n *noopMetrics) RecordInfo(version string)                               {}
func (n *noopMetrics) RecordUp()                                               {}
func (n *noopMetrics) RecordFailsafeEnabled(enabled bool)                      {}
func (n *noopMetrics) RecordChainHead(chainID uint64, blockNum uint64)         {}
func (n *noopMetrics) RecordCheckAccessList(success bool)                      {}
func (n *noopMetrics) RecordBackfillProgress(chainID uint64, progress float64) {}
func (n *noopMetrics) RecordReorgDetected(chainID uint64)                      {}
func (n *noopMetrics) RecordLogsAdded(chainID uint64, count int64)             {}
func (n *noopMetrics) RecordBlocksSealed(chainID uint64, count int64)          {}
func (n *noopMetrics) RecordCrossUnsafeValidatedTimestamp(timestamp uint64)    {}
