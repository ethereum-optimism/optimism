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
	RecordL1Interval() (done func(err error))
	RecordL1LatestHeight(height *big.Int)
	RecordL1LatestFinalizedHeight(height *big.Int)

	RecordL1TransactionDeposits(size int, mintedETH float64)
	RecordL1ProvenWithdrawals(size int)
	RecordL1FinalizedWithdrawals(size int)

	RecordL1CrossDomainSentMessages(size int)
	RecordL1CrossDomainRelayedMessages(size int)

	RecordL1SkippedOVM1ProvenWithdrawals(size int)
	RecordL1SkippedOVM1FinalizedWithdrawals(size int)
	RecordL1SkippedOVM1CrossDomainRelayedMessages(size int)

	RecordL1InitiatedBridgeTransfers(token common.Address, size int)
	RecordL1FinalizedBridgeTransfers(token common.Address, size int)
}

type L2Metricer interface {
	RecordL2Interval() (done func(err error))
	RecordL2LatestHeight(height *big.Int)
	RecordL2LatestFinalizedHeight(height *big.Int)

	RecordL2TransactionWithdrawals(size int, withdrawnETH float64)

	RecordL2CrossDomainSentMessages(size int)
	RecordL2CrossDomainRelayedMessages(size int)

	RecordL2InitiatedBridgeTransfers(token common.Address, size int)
	RecordL2FinalizedBridgeTransfers(token common.Address, size int)
}

type Metricer interface {
	L1Metricer
	L2Metricer
}

type bridgeMetrics struct {
	latestHeight *prometheus.GaugeVec

	intervalTick     *prometheus.CounterVec
	intervalDuration *prometheus.HistogramVec
	intervalFailures *prometheus.CounterVec

	txDeposits           prometheus.Counter
	txWithdrawals        prometheus.Counter
	provenWithdrawals    prometheus.Counter
	finalizedWithdrawals prometheus.Counter

	txMintedETH    prometheus.Counter
	txWithdrawnETH prometheus.Counter

	sentMessages    *prometheus.CounterVec
	relayedMessages *prometheus.CounterVec

	skippedOVM1Withdrawals     *prometheus.CounterVec
	skippedOVM1RelayedMessages prometheus.Counter

	initiatedBridgeTransfers *prometheus.CounterVec
	finalizedBridgeTransfers *prometheus.CounterVec
}

func NewMetrics(registry *prometheus.Registry) Metricer {
	factory := metrics.With(registry)
	return &bridgeMetrics{
		intervalTick: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "intervals_total",
			Help:      "number of times processing loop has run",
		}, []string{
			"chain",
		}),
		intervalDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: MetricsNamespace,
			Name:      "interval_seconds",
			Help:      "duration elapsed in the processing loop",
		}, []string{
			"chain",
		}),
		intervalFailures: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "interval_failures_total",
			Help:      "number of failures encountered",
		}, []string{
			"chain",
		}),
		latestHeight: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "height",
			Help:      "the latest processed l1 block height",
		}, []string{
			"chain",
			"kind",
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
			Help:      "number of initiated transaction withdrawals from l2",
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
		skippedOVM1Withdrawals: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "skipped_ovm1_withdrawals",
			Help:      "number of skipped ovm 1.0 withdrawals on l1 (proven|finalized)",
		}, []string{
			"stage",
		}),
		skippedOVM1RelayedMessages: factory.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "skipped_ovm1_relayed_messages",
			Help:      "number of skipped ovm 1.0 relayed messages on l1",
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

// L1Metricer

func (m *bridgeMetrics) RecordL1Interval() func(error) {
	m.intervalTick.WithLabelValues("l1").Inc()
	timer := prometheus.NewTimer(m.intervalDuration.WithLabelValues("l1"))
	return func(err error) {
		timer.ObserveDuration()
		if err != nil {
			m.intervalFailures.WithLabelValues("l1").Inc()
		}
	}
}

func (m *bridgeMetrics) RecordL1LatestHeight(height *big.Int) {
	m.latestHeight.WithLabelValues("l1", "initiated").Set(float64(height.Uint64()))
}

func (m *bridgeMetrics) RecordL1LatestFinalizedHeight(height *big.Int) {
	m.latestHeight.WithLabelValues("l1", "finalized").Set(float64(height.Uint64()))
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

func (m *bridgeMetrics) RecordL1SkippedOVM1ProvenWithdrawals(size int) {
	m.skippedOVM1Withdrawals.WithLabelValues("proven").Add(float64(size))
}

func (m *bridgeMetrics) RecordL1SkippedOVM1FinalizedWithdrawals(size int) {
	m.skippedOVM1Withdrawals.WithLabelValues("finalized").Add(float64(size))
}

func (m *bridgeMetrics) RecordL1SkippedOVM1CrossDomainRelayedMessages(size int) {
	m.skippedOVM1RelayedMessages.Add(float64(size))
}

func (m *bridgeMetrics) RecordL1InitiatedBridgeTransfers(tokenAddr common.Address, size int) {
	m.initiatedBridgeTransfers.WithLabelValues("l1", tokenAddr.String()).Add(float64(size))
}

func (m *bridgeMetrics) RecordL1FinalizedBridgeTransfers(tokenAddr common.Address, size int) {
	m.finalizedBridgeTransfers.WithLabelValues("l1", tokenAddr.String()).Add(float64(size))
}

// L2Metricer

func (m *bridgeMetrics) RecordL2Interval() func(error) {
	m.intervalTick.WithLabelValues("l2").Inc()
	timer := prometheus.NewTimer(m.intervalDuration.WithLabelValues("l2"))
	return func(err error) {
		timer.ObserveDuration()
		if err != nil {
			m.intervalFailures.WithLabelValues("l2").Inc()
		}
	}
}

func (m *bridgeMetrics) RecordL2LatestHeight(height *big.Int) {
	m.latestHeight.WithLabelValues("l2", "initiated").Set(float64(height.Uint64()))
}

func (m *bridgeMetrics) RecordL2LatestFinalizedHeight(height *big.Int) {
	m.latestHeight.WithLabelValues("l2", "finalized").Set(float64(height.Uint64()))
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
