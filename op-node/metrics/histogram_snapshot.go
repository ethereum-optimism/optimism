package metrics

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

type ReplaceableHistogramVec struct {
	current        *atomic.Value
	opts           prometheus.HistogramOpts
	variableLabels []string
}

func NewReplaceableHistogramVec(registry *prometheus.Registry, opts prometheus.HistogramOpts, variableLabels []string) *ReplaceableHistogramVec {
	metric := &ReplaceableHistogramVec{
		current:        &atomic.Value{},
		opts:           opts,
		variableLabels: variableLabels,
	}
	metric.current.Store(prometheus.NewHistogramVec(opts, variableLabels))
	registry.MustRegister(metric)
	return metric
}

func (c *ReplaceableHistogramVec) Replace(updater func(h *prometheus.HistogramVec)) {
	h := prometheus.NewHistogramVec(c.opts, c.variableLabels)
	updater(h)
	c.current.Store(h)
}

func (c *ReplaceableHistogramVec) Describe(ch chan<- *prometheus.Desc) {
	collector, ok := c.current.Load().(prometheus.Collector)
	if ok {
		collector.Describe(ch)
	}
}

func (c *ReplaceableHistogramVec) Collect(ch chan<- prometheus.Metric) {
	collector, ok := c.current.Load().(prometheus.Collector)
	if ok {
		collector.Collect(ch)
	}
}
