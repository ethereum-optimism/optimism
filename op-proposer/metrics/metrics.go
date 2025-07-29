package metrics

import (
	"io"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

const Namespace = "op_proposer"

// implements the Registry getter, for metrics HTTP server to hook into.
var _ opmetrics.RegistryMetricer = (*Metrics)(nil)

// Metricer is the interface that wraps the basic methods for recording metrics.
type Metricer interface {
	RecordInfo(version string)
	RecordUp()

	// Records all L1 and L2 block events
	opmetrics.RefMetricer

	// Record Tx metrics
	txmetrics.TxMetricer

	opmetrics.RPCMetricer

	StartBalanceMetrics(l log.Logger, client *ethclient.Client, account common.Address) io.Closer

	RecordL2Proposal(sequenceNum uint64)
	RecordL2BlocksProposed(l2ref eth.L2BlockRef)
}

// Metrics implements the Metricer interface.
type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	opmetrics.RefMetrics
	txmetrics.TxMetrics
	opmetrics.RPCMetrics

	proposalSequenceNum prometheus.Gauge

	info prometheus.GaugeVec
	up   prometheus.Gauge
}

var _ Metricer = (*Metrics)(nil)

// NewMetrics constructs a new Metrics instance.
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
		TxMetrics:  txmetrics.MakeTxMetrics(ns, factory),
		RPCMetrics: opmetrics.MakeRPCMetrics(ns, factory),

		proposalSequenceNum: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "proposed_sequence_number",
			Help:      "Sequence number (block number or timestamp) of the latest proposal",
		}),
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
			Help:      "1 if the op-proposer has finished starting up",
		}),
	}
}

// Registry returns the prometheus registry.
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

// StartBalanceMetrics starts the balance metrics.
func (m *Metrics) StartBalanceMetrics(l log.Logger, client *ethclient.Client, account common.Address) io.Closer {
	return opmetrics.LaunchBalanceMetrics(l, m.registry, m.ns, client, account)
}

// RecordInfo sets a pseudo-metric that contains versioning and config info for the op-proposer.
func (m *Metrics) RecordInfo(version string) {
	m.info.WithLabelValues(version).Set(1)
}

// RecordUp sets the `up` metric to 1.
func (m *Metrics) RecordUp() {
	m.up.Set(1)
}

const (
	BlockProposed = "proposed"
)

// RecordL2BlocksProposed records when new L2 block is proposed.
func (m *Metrics) RecordL2BlocksProposed(l2ref eth.L2BlockRef) {
	m.RecordL2Ref(BlockProposed, l2ref)
}

// RecordL2Proposal records when a new L2 proposal is created.
func (m *Metrics) RecordL2Proposal(seqNum uint64) {
	m.proposalSequenceNum.Set(float64(seqNum))
}

// Document returns the documented metrics.
func (m *Metrics) Document() []opmetrics.DocumentedMetric {
	return m.factory.Document()
}
