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
