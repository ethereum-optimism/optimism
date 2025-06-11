package metrics

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/prometheus/client_golang/prometheus"
)

type TxMetricer interface {
	RecordGasBumpCount(int)
	RecordTxConfirmationLatency(int64)
	RecordNonce(uint64)
	RecordPendingTx(pending int64)
	TxConfirmed(*types.Receipt)
	TxPublished(string)
	RecordBaseFee(*big.Int)
	RecordBlobBaseFee(*big.Int)
	RecordTipCap(*big.Int)
	RPCError()
}

type TxMetrics struct {
	txL1GasFee         prometheus.Gauge
	txFeesTotal        prometheus.Counter
	txGasBump          prometheus.Gauge
	txFeeHistogram     prometheus.Histogram
	txType             prometheus.Gauge
	latencyConfirmedTx prometheus.Gauge
	currentNonce       prometheus.Gauge
	pendingTxs         prometheus.Gauge
	txPublishError     *prometheus.CounterVec
	publishEvent       *metrics.Event
	confirmEvent       metrics.EventVec
	baseFee            prometheus.Gauge
	blobBaseFee        prometheus.Gauge
	tipCap             prometheus.Gauge
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
		txL1GasFee: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "tx_fee_gwei",
			Help:      "L1 gas fee for transactions in GWEI",
			Subsystem: "txmgr",
		}),
		txFeesTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "tx_fee_gwei_total",
			Help:      "Sum of fees spent for all transactions in GWEI",
			Subsystem: "txmgr",
		}),
		txGasBump: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "tx_gas_bump",
			Help:      "Number of times a transaction gas needed to be bumped before it got included",
			Subsystem: "txmgr",
		}),
		txFeeHistogram: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "tx_fee_histogram_gwei",
			Help:      "Tx Fee in GWEI",
			Subsystem: "txmgr",
			Buckets:   []float64{0.5, 1, 2, 5, 10, 20, 40, 60, 80, 100, 200, 400, 800, 1600},
		}),
		txType: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "tx_type",
			Help:      "Transaction type (receipt field uint8)",
			Subsystem: "txmgr",
		}),
		latencyConfirmedTx: factory.NewGauge(prometheus.GaugeOpts{
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
		pendingTxs: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "pending_txs",
			Help:      "Number of transactions pending receipts",
			Subsystem: "txmgr",
		}),
		txPublishError: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "tx_publish_error_count",
			Help:      "Count of publish errors. Labels are sanitized error strings",
			Subsystem: "txmgr",
		}, []string{"error"}),
		confirmEvent: metrics.NewEventVec(factory, ns, "txmgr", "confirm", "tx confirm", []string{"status"}),
		publishEvent: metrics.NewEvent(factory, ns, "txmgr", "publish", "tx publish"),
		baseFee: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "basefee_wei",
			Help:      "Latest L1 base fee (in Wei)",
			Subsystem: "txmgr",
		}),
		blobBaseFee: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "blob_basefee_wei",
			Help:      "Latest Blob base fee (in Wei)",
			Subsystem: "txmgr",
		}),
		tipCap: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "tipcap_wei",
			Help:      "Latest L1 suggested tip cap (in Wei)",
			Subsystem: "txmgr",
		}),
		rpcError: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "rpc_error_count",
			Help:      "Temporary: Count of RPC errors (like timeouts) that have occurred",
			Subsystem: "txmgr",
		}),
	}
}

func (t *TxMetrics) RecordNonce(nonce uint64) {
	t.currentNonce.Set(float64(nonce))
}

func (t *TxMetrics) RecordPendingTx(pending int64) {
	t.pendingTxs.Set(float64(pending))
}

// TxConfirmed records lots of information about the confirmed transaction
func (t *TxMetrics) TxConfirmed(receipt *types.Receipt) {
	fee := float64(receipt.EffectiveGasPrice.Uint64() * receipt.GasUsed / params.GWei)
	t.confirmEvent.Record(receiptStatusString(receipt))
	t.txL1GasFee.Set(fee)
	t.txFeesTotal.Add(fee)
	t.txFeeHistogram.Observe(fee)
	t.txType.Set(float64(receipt.Type))
}

func (t *TxMetrics) RecordGasBumpCount(times int) {
	t.txGasBump.Set(float64(times))
}

func (t *TxMetrics) RecordTxConfirmationLatency(latency int64) {
	t.latencyConfirmedTx.Set(float64(latency))
}

func (t *TxMetrics) TxPublished(errString string) {
	if errString != "" {
		t.txPublishError.WithLabelValues(errString).Inc()
	} else {
		t.publishEvent.Record()
	}
}

func (t *TxMetrics) RecordBaseFee(baseFee *big.Int) {
	bff, _ := baseFee.Float64()
	t.baseFee.Set(bff)
}

func (t *TxMetrics) RecordBlobBaseFee(blobBaseFee *big.Int) {
	bff, _ := blobBaseFee.Float64()
	t.blobBaseFee.Set(bff)
}

func (t *TxMetrics) RecordTipCap(tipcap *big.Int) {
	tcf, _ := tipcap.Float64()
	t.tipCap.Set(tcf)
}

func (t *TxMetrics) RPCError() {
	t.rpcError.Inc()
}
