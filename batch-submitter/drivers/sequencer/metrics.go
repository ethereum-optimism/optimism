package sequencer

import (
	"github.com/ethereum-optimism/optimism/bss-core/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics extends the BSS core metrics with additional metrics tracked by the
// sequencer driver.
type Metrics struct {
	*metrics.Base

	// BatchPruneCount tracks the number of times a batch of sequencer
	// transactions is pruned in order to meet the desired size requirements.
	BatchPruneCount prometheus.Gauge
}

// NewMetrics initializes a new, extended metrics object.
func NewMetrics(subsystem string) *Metrics {
	base := metrics.NewBase("batch_submitter", subsystem)
	return &Metrics{
		Base: base,
		BatchPruneCount: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "batch_prune_count",
			Help:      "Number of times a batch is pruned",
			Subsystem: base.SubsystemName(),
		}),
	}
}
