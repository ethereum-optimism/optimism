package engine

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var Namespace = "op_node"

type Metricer interface {
	RecordBlockFail()
	RecordBlockStats(hash common.Hash, num uint64, time uint64, txs uint64, gas uint64, baseFee float64)
}

type Metrics struct {
	BlockFails prometheus.Counter

	BlockHash    prometheus.Gauge
	BlockNum     prometheus.Gauge
	BlockTime    prometheus.Gauge
	BlockTxs     prometheus.Gauge
	BlockGas     prometheus.Gauge
	BlockBaseFee prometheus.Gauge
}

func NewMetrics(procName string, registry *prometheus.Registry) *Metrics {
	if procName == "" {
		procName = "default"
	}
	ns := Namespace + "_" + procName
	return &Metrics{
		BlockFails: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "engine",
			Name:      "block_fails",
			Help:      "Total block building attempts that fail",
		}),
		BlockHash: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "block_hash",
			Help:      "current head block hash",
		}),
		BlockNum: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "block_num",
			Help:      "current head block number",
		}),
		BlockTime: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "block_time",
			Help:      "current head block time",
		}),
		BlockTxs: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "block_txs",
			Help:      "current head block txs",
		}),
		BlockGas: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "block_gas",
			Help:      "current head block gas",
		}),
		BlockBaseFee: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "block_base_fee",
			Help:      "current head block basefee",
		}),
	}
}

func (r *Metrics) RecordBlockFail() {
	r.BlockFails.Inc()
}

func (r *Metrics) RecordBlockStats(hash common.Hash, num uint64, time uint64, txs uint64, gas uint64, baseFee float64) {
	r.BlockHash.Set(float64(binary.LittleEndian.Uint64(hash[:]))) // for pretty block-color changing charts
	r.BlockNum.Set(float64(num))
	r.BlockTime.Set(float64(time))
	r.BlockTxs.Set(float64(txs))
	r.BlockGas.Set(float64(gas))
	r.BlockGas.Set(float64(baseFee))
}

var _ Metricer = (*Metrics)(nil)
