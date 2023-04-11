package metrics

import (
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/prometheus/client_golang/prometheus"
)

type TxMetricer interface {
	RecordGasBumpCount(int)
	RecordTxConfirmationLatency(int64)
	RecordNonce(uint64)
	TxConfirmed(*types.Receipt)
	TxPublished(string)
	RPCError()
}

type TxMetrics struct {
	TxL1GasFee         prometheus.Gauge
	txFees             prometheus.Counter
	TxGasBump          prometheus.Gauge
	txFeeHistogram     prometheus.Histogram
	LatencyConfirmedTx prometheus.Gauge
	currentNonce       prometheus.Gauge
	txPublishError     *prometheus.CounterVec
	publishEvent       metrics.Event
	confirmEvent       metrics.EventVec
	rpcError           prometheus.Counter
}

func receiptStatusString(receipt *types.Receipt) string {
	switch receipt.Status {
	case types.ReceiptStatusSuccessful:
		return "success"
	case types.ReceiptStatusFailed:
		return "failed"
	default:
		return "unknown_status"
	}
}

var _ TxMetricer = (*TxMetrics)(nil)

func MakeTxMetrics(ns string, factory metrics.Factory) TxMetrics {
	return TxMetrics{
		TxL1GasFee: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "tx_fee_gwei",
			Help:      "L1 gas fee for transactions in GWEI",
			Subsystem: "txmgr",
		}),
		txFees: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "tx_fee_gwei_total",
			Help:      "Sum of fees spent for all transactions in GWEI",
			Subsystem: "txmgr",
		}),
		txFeeHistogram: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "tx_fee_histogram_gwei",
			Help:      "Tx Fee in GWEI",
			Subsystem: "txmgr",
			Buckets:   []float64{0.5, 1, 2, 5, 10, 20, 40, 60, 80, 100, 200, 400, 800, 1600},
		}),
		TxGasBump: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "tx_gas_bump",
			Help:      "Number of times a transaction gas needed to be bumped before it got included",
			Subsystem: "txmgr",
		}),
		LatencyConfirmedTx: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "tx_confirmed_latency_ms",
			Help:      "Latency of a confirmed transaction in milliseconds",
			Subsystem: "txmgr",
		}),
		currentNonce: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "current_nonce",
			Help:      "Current nonce of the from address",
			Subsystem: "txmgr",
		}),
		txPublishError: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "tx_publish_error_count",
			Help:      "Count of publish errors. Labells are sanitized error strings",
			Subsystem: "txmgr",
		}, []string{"error"}),
		confirmEvent: metrics.NewEventVec(factory, ns, "confirm", "tx confirm", []string{"status"}),
		publishEvent: metrics.NewEvent(factory, ns, "publish", "tx publish"),
		rpcError: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "rpc_error_count",
			Help:      "Temporrary: Count of RPC errors (like timeouts) that have occurrred",
			Subsystem: "txmgr",
		}),
	}
}

func (t *TxMetrics) RecordNonce(nonce uint64) {
	t.currentNonce.Set(float64(nonce))
}

// TxConfirmed records lots of information about the confirmed transaction
func (t *TxMetrics) TxConfirmed(receipt *types.Receipt) {
	fee := float64(receipt.EffectiveGasPrice.Uint64() * receipt.GasUsed / params.GWei)
	t.confirmEvent.Record(receiptStatusString(receipt))
	t.TxL1GasFee.Set(fee)
	t.txFees.Add(fee)
	t.txFeeHistogram.Observe(fee)

}

func (t *TxMetrics) RecordGasBumpCount(times int) {
	t.TxGasBump.Set(float64(times))
}

func (t *TxMetrics) RecordTxConfirmationLatency(latency int64) {
	t.LatencyConfirmedTx.Set(float64(latency))
}

func (t *TxMetrics) TxPublished(errString string) {
	if errString != "" {
		t.txPublishError.WithLabelValues(errString).Inc()
	} else {
		t.publishEvent.Record()
	}
}

func (t *TxMetrics) RPCError() {
	t.rpcError.Inc()
}
