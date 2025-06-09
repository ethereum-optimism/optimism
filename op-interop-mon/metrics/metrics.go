package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

const Namespace = "op_interop_mon"

var _ opmetrics.RegistryMetricer = (*Metrics)(nil)

type Metricer interface {
	RecordInfo(version string)
	RecordUp()
	RecordInitiatingMessageStats(chainID string, blockHeight uint64, status string, value float64)
	RecordExecutingMessageStats(chainID string, blockHeight uint64, blockHash string, status string, value float64)
	RecordTerminalStatusChange(executingChainID string, initiatingChainID string, value float64)

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
	executingMessages     prometheus.GaugeVec
	initiatingMessages    prometheus.GaugeVec
	terminalStatusChanges prometheus.GaugeVec
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
		executingMessages: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "executing_messages",
			Help:      "Number of messages being executed",
		}, []string{
			"chain_id",
			"block_height",
			"block_hash",
			"status",
		}),
		initiatingMessages: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "initiating_messages",
			Help:      "Number of messages being referenced by executing messages",
		}, []string{
			"chain_id",
			"block_height",
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

// RecordExecutingMessageStats records metrics for messages being executed
func (m *Metrics) RecordExecutingMessageStats(
	chainID string,
	blockHeight uint64,
	blockHash string,
	status string,
	value float64,
) {
	m.executingMessages.WithLabelValues(
		chainID,
		fmt.Sprintf("%d", blockHeight),
		blockHash,
		status,
	).Set(value)
}

// RecordInitiatingMessageStats records metrics for messages being initiated
func (m *Metrics) RecordInitiatingMessageStats(
	chainID string,
	blockHeight uint64,
	status string,
	value float64,
) {
	m.initiatingMessages.WithLabelValues(
		chainID,
		fmt.Sprintf("%d", blockHeight),
		status,
	).Set(value)
}

// RecordTerminalStatusChange records a terminal status change with detailed logging
func (m *Metrics) RecordTerminalStatusChange(
	executingChainID string,
	initiatingChainID string,
	value float64,
) {
	m.terminalStatusChanges.WithLabelValues(
		executingChainID,
		initiatingChainID,
	).Set(value)
}
