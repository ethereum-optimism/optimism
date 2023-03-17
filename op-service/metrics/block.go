package metrics

import (
	"math/big"

	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/prometheus/client_golang/prometheus"
)

func EmitBlockNumber(r *prometheus.Registry, ns string, blockNumber *big.Int) {
	promauto.With(r).NewGauge(prometheus.GaugeOpts{
		Namespace: ns,
		Name:      "latest_block_number",
		Help:      "Latest L2 proposed block number",
	}).Set(float64(blockNumber.Uint64()))
}
