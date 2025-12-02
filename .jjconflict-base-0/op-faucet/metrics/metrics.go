package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	ftypes "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

const Namespace = "op_faucet"

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	// Global metrics for now. We can parametrize and add a label per faucet later.
	txmetrics.TxMetrics

	totalFundingETH *prometheus.CounterVec
	totalFundingTxs *prometheus.CounterVec

	txDuration *prometheus.HistogramVec

	info prometheus.GaugeVec
	up   prometheus.Gauge
}

var _ Metricer = (*Metrics)(nil)

func NewMetrics(procName string) *Metrics {
	return newMetrics(procName, opmetrics.NewRegistry())
}

func newMetrics(procName string, registry *prometheus.Registry) *Metrics {
	if procName == "" {
		procName = "default"
	}
	ns := Namespace + "_" + procName

	factory := opmetrics.With(registry)
	return &Metrics{
		ns:       ns,
		registry: registry,
		factory:  factory,

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
			Help:      "1 if op-faucet has finished starting up",
		}),

		TxMetrics: txmetrics.MakeTxMetrics(ns, factory),

		totalFundingETH: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "funding_eth_total",
			Help:      "Total of funding ETH",
		}, []string{"faucet", "chain", "err"}),

		totalFundingTxs: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "funding_txs_total",
			Help:      "Count of funding txs",
		}, []string{"faucet", "chain", "err"}),

		txDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "funding_duration_seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			Help:      "Duration it takes to confirm a tx",
		}, []string{"faucet", "chain"}),
	}
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

func (m *Metrics) Document() []opmetrics.DocumentedMetric {
	return m.factory.Document()
}

// RecordInfo sets a pseudo-metric that contains versioning and config info.
func (m *Metrics) RecordInfo(version string) {
	m.info.WithLabelValues(version).Set(1)
}

// RecordUp sets the up metric to 1.
func (m *Metrics) RecordUp() {
	m.up.Set(1)
}

func (m *Metrics) RecordFundAction(faucet ftypes.FaucetID, chainID eth.ChainID, amount eth.ETH) (onDone func(err error)) {
	timer := prometheus.NewTimer(m.txDuration.WithLabelValues(faucet.String(), chainID.String()))
	return func(err error) {
		timer.ObserveDuration()
		errStr := "success"
		if err != nil {
			errStr = "failed"
		}
		m.totalFundingTxs.WithLabelValues(faucet.String(), chainID.String(), errStr).Inc()
		m.totalFundingETH.WithLabelValues(faucet.String(), chainID.String(), errStr).Add(amount.WeiFloat())
	}
}
