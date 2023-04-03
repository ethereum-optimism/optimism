package metrics

import (
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/prometheus/client_golang/prometheus"
)

type TxMetricer interface {
	RecordL1GasFee(receipt *types.Receipt)
	RecordGasBumpCount(times int)
	RecordTxConfirmationLatency(latency int64)
}

type TxMetrics struct {
	TxL1GasFee         prometheus.Gauge
	TxGasBump          prometheus.Gauge
	LatencyConfirmedTx prometheus.Gauge
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
	}
}

func (t *TxMetrics) RecordL1GasFee(receipt *types.Receipt) {
	t.TxL1GasFee.Set(float64(receipt.EffectiveGasPrice.Uint64() * receipt.GasUsed / params.GWei))
}

func (t *TxMetrics) RecordGasBumpCount(times int) {
	t.TxGasBump.Set(float64(times))
}

func (t *TxMetrics) RecordTxConfirmationLatency(latency int64) {
	t.LatencyConfirmedTx.Set(float64(latency))
}
