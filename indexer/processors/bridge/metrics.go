package bridge

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	MetricsNamespace string = "op_indexer_bridge"
)

type L1Metricer interface {
	RecordLatestIndexedL1Height(height *big.Int)

	RecordL1TransactionDeposits(size int, mintedETH float64)
	RecordL1ProvenWithdrawals(size int)
	RecordL1FinalizedWithdrawals(size int)

	RecordL1CrossDomainSentMessages(size int)
	RecordL1CrossDomainRelayedMessages(size int)

	RecordL1InitiatedBridgeTransfers(token common.Address, size int)
	RecordL1FinalizedBridgeTransfers(token common.Address, size int)
}

type L2Metricer interface {
	RecordLatestIndexedL2Height(height *big.Int)

	RecordL2TransactionWithdrawals(size int, withdrawnETH float64)

	RecordL2CrossDomainSentMessages(size int)
	RecordL2CrossDomainRelayedMessages(size int)

	RecordL2InitiatedBridgeTransfers(token common.Address, size int)
	RecordL2FinalizedBridgeTransfers(token common.Address, size int)
}

type Metricer interface {
	L1Metricer
	L2Metricer

	RecordInterval() (done func(err error))
}

type bridgeMetrics struct {
	intervalTick     prometheus.Counter
	intervalDuration prometheus.Histogram
	intervalFailures prometheus.Counter

	latestL1Height prometheus.Gauge
	latestL2Height prometheus.Gauge

	txDeposits           prometheus.Counter
	txMintedETH          prometheus.Counter
	txWithdrawals        prometheus.Counter
	txWithdrawnETH       prometheus.Counter
	provenWithdrawals    prometheus.Counter
	finalizedWithdrawals prometheus.Counter

	sentMessages    *prometheus.CounterVec
	relayedMessages *prometheus.CounterVec

	initiatedBridgeTransfers *prometheus.CounterVec
	finalizedBridgeTransfers *prometheus.CounterVec
}

func NewMetrics(registry *prometheus.Registry) Metricer {
	factory := metrics.With(registry)
	return &bridgeMetrics{
		intervalTick: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "intervals_total",
			Help:      "number of times processing loop has run",
		}),
		intervalDuration: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: MetricsNamespace,
			Name:      "interval_seconds",
			Help:      "duration elapsed in the processing loop",
		}),
		intervalFailures: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "interval_failures_total",
			Help:      "number of failures encountered",
		}),
		latestL1Height: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Subsystem: "l1",
			Name:      "height",
			Help:      "the latest processed l1 block height",
		}),
		latestL2Height: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Subsystem: "l2",
			Name:      "height",
			Help:      "the latest processed l2 block height",
		}),
		txDeposits: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "tx_deposits",
			Help:      "number of processed transactions deposited from l1",
		}),
		txMintedETH: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "tx_minted_eth",
			Help:      "amount of eth bridged from l1",
		}),
		txWithdrawals: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "tx_withdrawals",
			Help:      "number of processed transactions withdrawn from l2",
		}),
		txWithdrawnETH: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "tx_withdrawn_eth",
			Help:      "amount of eth withdrawn from l2",
		}),
		provenWithdrawals: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "proven_withdrawals",
			Help:      "number of proven tx withdrawals on l1",
		}),
		finalizedWithdrawals: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "finalized_withdrawals",
			Help:      "number of finalized tx withdrawals on l1",
		}),
		sentMessages: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "sent_messages",
			Help:      "number of bridged messages between l1 and l2",
		}, []string{
			"chain",
		}),
		relayedMessages: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "relayed_messages",
			Help:      "number of relayed messages between l1 and l2",
		}, []string{
			"chain",
		}),
		initiatedBridgeTransfers: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "initiated_token_transfers",
			Help:      "number of bridged tokens between l1 and l2",
		}, []string{
			"chain",
			"token_address",
		}),
		finalizedBridgeTransfers: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "finalized_token_transfers",
			Help:      "number of finalized token transfers between l1 and l2",
		}, []string{
			"chain",
			"token_address",
		}),
	}
}

func (m *bridgeMetrics) RecordInterval() func(error) {
	m.intervalTick.Inc()
	timer := prometheus.NewTimer(m.intervalDuration)
	return func(err error) {
		timer.ObserveDuration()
		if err != nil {
			m.intervalFailures.Inc()
		}
	}
}

// L1Metricer

func (m *bridgeMetrics) RecordLatestIndexedL1Height(height *big.Int) {
	m.latestL1Height.Set(float64(height.Uint64()))
}

func (m *bridgeMetrics) RecordL1TransactionDeposits(size int, mintedETH float64) {
	m.txDeposits.Add(float64(size))
	m.txMintedETH.Add(mintedETH)
}

func (m *bridgeMetrics) RecordL1ProvenWithdrawals(size int) {
	m.provenWithdrawals.Add(float64(size))
}

func (m *bridgeMetrics) RecordL1FinalizedWithdrawals(size int) {
	m.finalizedWithdrawals.Add(float64(size))
}

func (m *bridgeMetrics) RecordL1CrossDomainSentMessages(size int) {
	m.sentMessages.WithLabelValues("l1").Add(float64(size))
}

func (m *bridgeMetrics) RecordL1CrossDomainRelayedMessages(size int) {
	m.relayedMessages.WithLabelValues("l1").Add(float64(size))
}

func (m *bridgeMetrics) RecordL1InitiatedBridgeTransfers(tokenAddr common.Address, size int) {
	m.initiatedBridgeTransfers.WithLabelValues("l1", tokenAddr.String()).Add(float64(size))
}

func (m *bridgeMetrics) RecordL1FinalizedBridgeTransfers(tokenAddr common.Address, size int) {
	m.finalizedBridgeTransfers.WithLabelValues("l1", tokenAddr.String()).Add(float64(size))
}

// L2Metricer

func (m *bridgeMetrics) RecordLatestIndexedL2Height(height *big.Int) {
	m.latestL2Height.Set(float64(height.Uint64()))
}

func (m *bridgeMetrics) RecordL2TransactionWithdrawals(size int, withdrawnETH float64) {
	m.txWithdrawals.Add(float64(size))
	m.txWithdrawnETH.Add(withdrawnETH)
}

func (m *bridgeMetrics) RecordL2CrossDomainSentMessages(size int) {
	m.sentMessages.WithLabelValues("l2").Add(float64(size))
}

func (m *bridgeMetrics) RecordL2CrossDomainRelayedMessages(size int) {
	m.relayedMessages.WithLabelValues("l2").Add(float64(size))
}

func (m *bridgeMetrics) RecordL2InitiatedBridgeTransfers(tokenAddr common.Address, size int) {
	m.initiatedBridgeTransfers.WithLabelValues("l2", tokenAddr.String()).Add(float64(size))
}

func (m *bridgeMetrics) RecordL2FinalizedBridgeTransfers(tokenAddr common.Address, size int) {
	m.finalizedBridgeTransfers.WithLabelValues("l2", tokenAddr.String()).Add(float64(size))
}
