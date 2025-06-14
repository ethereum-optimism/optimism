package metrics

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type L1Metricer interface {
	RecordL1ReorgDepth(d uint64)
	RecordL1RequestTime(method string, duration time.Duration)
}

type L1Metrics struct {
	// Historically from op-node
	L1RequestDurationSeconds *prometheus.HistogramVec
	L1ReorgDepth             prometheus.Histogram
}

var _ L1Metricer = (*L1Metrics)(nil)

func NewL1Metrics(ns string, factory metrics.Factory) *L1Metrics {
	return &L1Metrics{
		L1ReorgDepth: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "l1_reorg_depth",
			Buckets:   []float64{0.5, 1.5, 2.5, 3.5, 4.5, 5.5, 6.5, 7.5, 8.5, 9.5, 10.5, 20.5, 50.5, 100.5},
			Help:      "Histogram of L1 Reorg Depths",
		}),

		L1RequestDurationSeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "l1_request_seconds",
			Buckets: []float64{
				.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			Help: "Histogram of L1 request time",
		}, []string{"request"}),
	}
}

func (m *L1Metrics) RecordL1ReorgDepth(d uint64) {
	m.L1ReorgDepth.Observe(float64(d))
}

// RecordL1RequestTime tracks the amount of time the derivation pipeline spent waiting for L1 data requests.
func (m *L1Metrics) RecordL1RequestTime(method string, duration time.Duration) {
	m.L1RequestDurationSeconds.WithLabelValues(method).Observe(float64(duration) / float64(time.Second))
}

type NoopL1Metrics struct{}

var _ L1Metricer = NoopL1Metrics{}

func (NoopL1Metrics) RecordL1ReorgDepth(d uint64) {}

func (NoopL1Metrics) RecordL1RequestTime(method string, duration time.Duration) {}
