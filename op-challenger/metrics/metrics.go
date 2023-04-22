package metrics

import (
	"context"
	"math/big"

	common "github.com/ethereum/go-ethereum/common"
	types "github.com/ethereum/go-ethereum/core/types"
	ethclient "github.com/ethereum/go-ethereum/ethclient"
	log "github.com/ethereum/go-ethereum/log"
	params "github.com/ethereum/go-ethereum/params"
	prometheus "github.com/prometheus/client_golang/prometheus"

	eth "github.com/ethereum-optimism/optimism/op-node/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

const Namespace = "op_challenger"

type Metricer interface {
	RecordInfo(version string)
	RecordUp()

	// Records all L1 and L2 block events
	opmetrics.RefMetricer

	RecordValidOutput(l2ref eth.L2BlockRef)
	RecordInvalidOutput(l2ref eth.L2BlockRef)
	RecordChallengeSent(l2BlockNumber *big.Int, outputRoot common.Hash)

	RecordDisputeGameCreated(l2BlockNumber *big.Int, outputRoot common.Hash, contract common.Address)

	RecordL1GasFee(receipt *types.Receipt)
}

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	opmetrics.RefMetrics

	TxL1GasFee prometheus.Gauge

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

		TxL1GasFee: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "tx_fee_gwei",
			Help:      "L1 gas fee for transactions in GWEI",
			Subsystem: "txmgr",
		}),

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
			Help:      "1 if the op-challenger has finished starting up",
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
// config info for the op-challenger.
func (m *Metrics) RecordInfo(version string) {
	m.Info.WithLabelValues(version).Set(1)
}

// RecordUp sets the up metric to 1.
func (m *Metrics) RecordUp() {
	prometheus.MustRegister()
	m.Up.Set(1)
}

const (
	ValidOutput      = "valid_output"
	InvalidOutput    = "invalid_output"
	OutputChallenged = "output_challenged"
)

// RecordValidOutput should be called when a valid output is found
func (m *Metrics) RecordValidOutput(l2ref eth.L2BlockRef) {
	m.RecordL2Ref(ValidOutput, l2ref)
}

// RecordInvalidOutput should be called when an invalid output is found
func (m *Metrics) RecordInvalidOutput(l2ref eth.L2BlockRef) {
	m.RecordL2Ref(InvalidOutput, l2ref)
}

func (m *Metrics) RecordChallengeSent(l2BlockNumber *big.Int, outputRoot common.Hash) {
	m.RecordL2Ref(OutputChallenged, eth.L2BlockRef{
		Number: l2BlockNumber.Uint64(),
		Hash:   outputRoot,
	})
}

func (m *Metrics) RecordDisputeGameCreated(l2BlockNumber *big.Int, outputRoot common.Hash, contract common.Address) {
	// TODO: record dg created here
}

// RecordL1GasFee records the L1 gas fee for a transaction
func (m *Metrics) RecordL1GasFee(receipt *types.Receipt) {
	m.TxL1GasFee.Set(float64(receipt.EffectiveGasPrice.Uint64() * receipt.GasUsed / params.GWei))
}
