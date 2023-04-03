package metrics

import (
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/prometheus/client_golang/prometheus"
)

type TxMetricer interface {
	RecordL1GasFee(receipt *types.Receipt)
}

type TxMetrics struct {
	TxL1GasFee prometheus.Gauge
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
	}
}

func (t *TxMetrics) RecordL1GasFee(receipt *types.Receipt) {
	t.TxL1GasFee.Set(float64(receipt.EffectiveGasPrice.Uint64() * receipt.GasUsed / params.GWei))
}
