package metrics

import (
	"math/big"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type ProposerMetricsRegistry struct {
	BlockNumberGauge prometheus.Gauge
}

func InitProposerMetricsRegistry(r *prometheus.Registry, ns string) *ProposerMetricsRegistry {
	blockNumberGauge := promauto.With(r).NewGauge(prometheus.GaugeOpts{
		Namespace: ns,
		Name:      "latest_block_number",
		Help:      "Latest L2 proposed block number",
	})

	return &ProposerMetricsRegistry{BlockNumberGauge: blockNumberGauge}
}
func EmitBlockNumber(gauge prometheus.Gauge, blockNumber *big.Int) {
	gauge.Set(float64(blockNumber.Uint64()))
}
