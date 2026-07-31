package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

const Namespace = "op_interop_mon"

var _ opmetrics.RegistryMetricer = (*Metrics)(nil)

type Metricer interface {
	RecordInfo(version string)
	RecordUp()
	RecordMessageStatus(executingChainID string, initiatingChainID string, status string, count float64)
	RecordTerminalStatusChange(executingChainID string, initiatingChainID string, count float64)
	RecordExecutingBlockRange(chainID string, min uint64, max uint64)
	RecordInitiatingBlockRange(chainID string, min uint64, max uint64)
	RecordInitiatingReorg(executingChainID string, initiatingChainID string)
	RecordFilterDivergence(executingChainID string, initiatingChainID string, monitorStatus string, filterStatus string)
	RecordFilterFailsafe(enabled bool)
	RecordSupernodeUp(endpoint string, up bool)
	RecordSupernodeSafeHead(chainID string, level string, blockNumber uint64)
	RecordCrossSafetyViolation(executingChainID string, initiatingChainID string)

	opmetrics.RefMetricer
	opmetrics.RPCMetricer
}

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	opmetrics.RefMetrics
	opmetrics.RPCMetrics

	info prometheus.GaugeVec
	up   prometheus.Gauge

	// Message metrics
	messageStatus         prometheus.GaugeVec
	terminalStatusChanges prometheus.GaugeVec
	executingBlockRange   prometheus.GaugeVec
	initiatingBlockRange  prometheus.GaugeVec
	initiatingReorgs      prometheus.CounterVec
	filterDivergence      prometheus.CounterVec
	filterFailsafe        prometheus.Gauge

	// Supernode observability
	supernodeUp           prometheus.GaugeVec
	supernodeSafeHead     prometheus.GaugeVec
	crossSafetyViolations prometheus.CounterVec
}

var _ Metricer = (*Metrics)(nil)

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

		RefMetrics: opmetrics.MakeRefMetrics(ns, factory),
		RPCMetrics: opmetrics.MakeRPCMetrics(ns, factory),

		info: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "info",
			Help:      "Information about the monitor",
		}, []string{
			"version",
		}),
		up: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "up",
			Help:      "1 if the op-interop-mon has finished starting up",
		}),
		messageStatus: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "message_status",
			Help:      "Number of messages by executing chain, initiating chain, and status",
		}, []string{
			"executing_chain_id",
			"initiating_chain_id",
			"status",
		}),
		terminalStatusChanges: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "terminal_status_changes",
			Help:      "Number of terminal status changes",
		}, []string{
			"executing_chain_id",
			"initiating_chain_id",
		}),
		executingBlockRange: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "executing_block_range",
			Help:      "Range of blocks containing Executing Messages currently tracked by the monitor",
		}, []string{
			"chain_id",
			"range_type",
		}),
		initiatingBlockRange: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "initiating_block_range",
			Help:      "Range of blocks being referenced by Executing Messages currently tracked by the monitor",
		}, []string{
			"chain_id",
			"range_type",
		}),
		initiatingReorgs: *factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "initiating_reorgs_total",
			Help:      "Count of jobs whose initiating block was observed at more than one hash (initiating-chain reorg)",
		}, []string{
			"executing_chain_id",
			"initiating_chain_id",
		}),
		filterDivergence: *factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "filter_divergence_total",
			Help:      "Count of executing messages where the interop-filter verdict disagreed with the monitor verdict",
		}, []string{
			"executing_chain_id",
			"initiating_chain_id",
			"monitor_status",
			"filter_status",
		}),
		filterFailsafe: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "interop_filter_failsafe",
			Help:      "1 if the observed interop-filter reports failsafe enabled, 0 otherwise",
		}),
		supernodeUp: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "supernode_up",
			Help:      "1 if the observed supernode answered its heartbeat probe, 0 otherwise",
		}, []string{
			"endpoint",
		}),
		supernodeSafeHead: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "supernode_safe_head",
			Help:      "Latest per-chain L2 head block number reported by the supernode, by safety level",
		}, []string{
			"chain_id",
			"level",
		}),
		crossSafetyViolations: *factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "cross_safety_violations_total",
			Help:      "Count of bad executing messages observed at/below the supernode cross-safe head",
		}, []string{
			"executing_chain_id",
			"initiating_chain_id",
		}),
	}
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

func (m *Metrics) RecordInfo(version string) {
	m.info.WithLabelValues(version).Set(1)
}

func (m *Metrics) RecordUp() {
	m.up.Set(1)
}

func (m *Metrics) Document() []opmetrics.DocumentedMetric {
	return m.factory.Document()
}

// RecordMessageStatus records metrics for messages by their executing chain, initiating chain, and status
func (m *Metrics) RecordMessageStatus(
	executingChainID string,
	initiatingChainID string,
	status string,
	count float64,
) {
	m.messageStatus.WithLabelValues(
		executingChainID,
		initiatingChainID,
		status,
	).Set(count)
}

// RecordTerminalStatusChange records a terminal status change with detailed logging
func (m *Metrics) RecordTerminalStatusChange(
	executingChainID string,
	initiatingChainID string,
	count float64,
) {
	m.terminalStatusChanges.WithLabelValues(
		executingChainID,
		initiatingChainID,
	).Set(count)
}

// RecordExecutingBlockRange records the min/max executing block numbers seen for a chain
func (m *Metrics) RecordExecutingBlockRange(chainID string, min uint64, max uint64) {
	m.executingBlockRange.WithLabelValues(chainID, "min").Set(float64(min))
	m.executingBlockRange.WithLabelValues(chainID, "max").Set(float64(max))
}

// RecordInitiatingBlockRange records the min/max initiating block numbers seen for a chain
func (m *Metrics) RecordInitiatingBlockRange(chainID string, min uint64, max uint64) {
	m.initiatingBlockRange.WithLabelValues(chainID, "min").Set(float64(min))
	m.initiatingBlockRange.WithLabelValues(chainID, "max").Set(float64(max))
}

// RecordInitiatingReorg increments when a job's initiating block is seen at multiple hashes.
func (m *Metrics) RecordInitiatingReorg(executingChainID string, initiatingChainID string) {
	m.initiatingReorgs.WithLabelValues(executingChainID, initiatingChainID).Inc()
}

// RecordFilterDivergence increments when the interop-filter verdict disagrees with the monitor's.
func (m *Metrics) RecordFilterDivergence(executingChainID string, initiatingChainID string, monitorStatus string, filterStatus string) {
	m.filterDivergence.WithLabelValues(executingChainID, initiatingChainID, monitorStatus, filterStatus).Inc()
}

// RecordFilterFailsafe records the observed interop-filter failsafe state.
func (m *Metrics) RecordFilterFailsafe(enabled bool) {
	if enabled {
		m.filterFailsafe.Set(1)
	} else {
		m.filterFailsafe.Set(0)
	}
}

// RecordSupernodeUp records whether the observed supernode answered its heartbeat probe.
func (m *Metrics) RecordSupernodeUp(endpoint string, up bool) {
	v := 0.0
	if up {
		v = 1.0
	}
	m.supernodeUp.WithLabelValues(endpoint).Set(v)
}

// RecordSupernodeSafeHead records the supernode's per-chain L2 head block number at a safety level.
func (m *Metrics) RecordSupernodeSafeHead(chainID string, level string, blockNumber uint64) {
	m.supernodeSafeHead.WithLabelValues(chainID, level).Set(float64(blockNumber))
}

// RecordCrossSafetyViolation increments when a bad executing message is observed at/below the supernode cross-safe head.
func (m *Metrics) RecordCrossSafetyViolation(executingChainID string, initiatingChainID string) {
	m.crossSafetyViolations.WithLabelValues(executingChainID, initiatingChainID).Inc()
}
