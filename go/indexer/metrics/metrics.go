package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	// BlockIndexingTime tracks the time for indexing a block.
	// FIXME: add this metric
	BlockIndexingTime prometheus.Gauge
}

func NewMetrics(subsystem string) *Metrics {
	return &Metrics{
		BlockIndexingTime: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "indexer_block_indexing_time",
			Help:      "Time to index a block",
			Subsystem: subsystem,
		}),
	}
}
