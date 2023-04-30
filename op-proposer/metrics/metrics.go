package metrics

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-node/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

const Namespace = "op_proposer"

type Metricer interface {
	RecordInfo(version string)
	RecordUp()

	// Records all L1 and L2 block events
	opmetrics.RefMetricer

	// Record Tx metrics
	txmetrics.TxMetricer

	RecordL2BlocksProposed(l2ref eth.L2BlockRef)
	RecordValidOutputAlreadyProposed(block *big.Int, output common.Hash)
	RecordInvalidOutputAlreadyProposed(block *big.Int, output common.Hash)
}

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	opmetrics.RefMetrics
	txmetrics.TxMetrics

	Info prometheus.GaugeVec
	Up   prometheus.Gauge
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
		TxMetrics:  txmetrics.MakeTxMetrics(ns, factory),

		Info: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "info",
			Help:      "Pseudo-metric tracking version and config info",
		}, []string{
			"version",
		}),
		Up: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "up",
			Help:      "1 if the op-proposer has finished starting up",
		}),
	}
}

func (m *Metrics) Serve(ctx context.Context, host string, port int) error {
	return opmetrics.ListenAndServe(ctx, m.registry, host, port)
}

func (m *Metrics) StartBalanceMetrics(ctx context.Context,
	l log.Logger, client *ethclient.Client, account common.Address) {
	opmetrics.LaunchBalanceMetrics(ctx, l, m.registry, m.ns, client, account)
}

// RecordInfo sets a pseudo-metric that contains versioning and
// config info for the op-proposer.
func (m *Metrics) RecordInfo(version string) {
	m.Info.WithLabelValues(version).Set(1)
}

// RecordUp sets the up metric to 1.
func (m *Metrics) RecordUp() {
	prometheus.MustRegister()
	m.Up.Set(1)
}

const (
	BlockProposed = "proposed"
	InvalidOutput = "invalid_output"
	ValidOutput   = "valid_output"
)

// RecordL2BlocksProposed should be called when new L2 block is proposed
func (m *Metrics) RecordL2BlocksProposed(l2ref eth.L2BlockRef) {
	m.RecordL2Ref(BlockProposed, l2ref)
}

func (m *Metrics) Document() []opmetrics.DocumentedMetric {
	return m.factory.Document()
}

// RecordValidOutputAlreadyProposed should be called when the proposer
// sees an valid output root is already proposed for the given block.
func (m *Metrics) RecordValidOutputAlreadyProposed(block *big.Int, output common.Hash) {
	m.RecordL2Ref(ValidOutput, eth.L2BlockRef{
		Number: block.Uint64(),
		Hash:   output,
	})
}

// RecordInvalidOutputAlreadyProposed should be called when the proposer
// sees an invalid output root is already proposed for the given block.
func (m *Metrics) RecordInvalidOutputAlreadyProposed(block *big.Int, output common.Hash) {
	m.RecordL2Ref(InvalidOutput, eth.L2BlockRef{
		Number: block.Uint64(),
		Hash:   output,
	})
}
