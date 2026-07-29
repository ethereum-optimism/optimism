package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type NodeRecorder interface {
	RecordUp()
	RecordInfo(version string)
}

type noopNodeRecorder struct{}

var NoopNodeRecorder = new(noopNodeRecorder)

func (n *noopNodeRecorder) RecordUp() {}

func (n *noopNodeRecorder) RecordInfo(string) {}

type PromNodeRecorder struct {
	Up   prometheus.Gauge
	Info *prometheus.GaugeVec
}

func NewPromNodeRecorder(r *prometheus.Registry, ns string) NodeRecorder {
	return &PromNodeRecorder{
		Up: promauto.With(r).NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "up",
			Help:      "1 if the node has finished starting up",
		}),
		Info: promauto.With(r).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "info",
			Help:      "Pseudo-metric tracking version and config info",
		}, []string{
			"version",
		}),
	}
}

func (p *PromNodeRecorder) RecordUp() {
	p.Up.Set(1)
}

func (p *PromNodeRecorder) RecordInfo(version string) {
	p.Info.WithLabelValues(version).Set(1)
}
