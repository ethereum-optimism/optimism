package metrics

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/prometheus/client_golang/prometheus"
)

type TxMetricer interface {
	RecordNonce(nonce uint64)
	TxConfirmed(*types.Receipt)
	TxPublished(err error)
	RPCError()
}

type TxMetrics struct {
	TxL1GasFee      prometheus.Gauge
	currentNonce    prometheus.Gauge
	txConfirmed     *prometheus.CounterVec
	txPublished     prometheus.Counter
	txPublishError  *prometheus.CounterVec
	lastPublishTime prometheus.Gauge
	lastConfirmTime prometheus.Gauge
	rpcError        prometheus.Counter
}

func receiptStatusString(receipt *types.Receipt) string {
	switch receipt.Status {
	case types.ReceiptStatusSuccessful:
		return "success"
	case types.ReceiptStatusFailed:
		return "failed"
	default:
		return "unkown_status"
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
		currentNonce: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "current_nonce",
			Help:      "",
			Subsystem: "txmgr",
		}),
		txConfirmed: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "tx_confirmed_count",
			Help:      "",
			Subsystem: "txmgr",
		}, []string{"status"}),
		txPublished: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "tx_published_count",
			Help:      "",
			Subsystem: "txmgr",
		}),
		txPublishError: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "tx_publish_error_count",
			Help:      "",
			Subsystem: "txmgr",
		}, []string{"error"}),
		lastPublishTime: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "last_publish_time_unix_secs",
			Help:      "",
			Subsystem: "txmgr",
		}),
		lastConfirmTime: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "last_confirm_time_unix_secs",
			Help:      "",
			Subsystem: "txmgr",
		}),
		rpcError: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "rpc_error_count",
			Help:      "",
			Subsystem: "txmgr",
		}),
	}
}

func (t *TxMetrics) RecordNonce(nonce uint64) {
	t.currentNonce.Set(float64(nonce))
}

// TxConfirmed records lots of information about the confirmed transaction
func (t *TxMetrics) TxConfirmed(receipt *types.Receipt) {
	t.lastConfirmTime.Set(float64(time.Now().Unix()))
	t.txConfirmed.WithLabelValues(receiptStatusString(receipt)).Inc()
	t.TxL1GasFee.Set(float64(receipt.EffectiveGasPrice.Uint64() * receipt.GasUsed / params.GWei))
}

func (t *TxMetrics) TxPublished(err error) {
	if err != nil {
		t.txPublishError.WithLabelValues(err.Error()).Inc()
	} else {
		t.txPublishError.WithLabelValues("nil").Inc() // TODO: DO we want this?
		t.txPublished.Inc()
		t.lastPublishTime.Set(float64(time.Now().Unix()))
	}
}

func (t *TxMetrics) RPCError() {
	t.rpcError.Inc()
}
