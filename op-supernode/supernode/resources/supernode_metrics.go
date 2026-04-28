package resources

import "github.com/prometheus/client_golang/prometheus"

// SupernodeMetrics holds supernode-level metrics that outlive individual
// virtual node restarts. Created once in supernode.New() and shared with
// chain containers and activities. Callers that receive nil default to
// NewSupernodeMetrics(), which creates functional counters not attached
// to any scraped registry (safe for tests).
type SupernodeMetrics struct {
	VNRestarts                *prometheus.CounterVec
	InteropTimestampsVerified prometheus.Counter
	InteropInvalidations      *prometheus.CounterVec
	InteropVerifiedTimestamp  prometheus.Gauge
	InteropRoundDecisions     *prometheus.CounterVec
	InteropRewinds            prometheus.Counter

	registry *prometheus.Registry
}

// NewSupernodeMetrics creates a new SupernodeMetrics backed by a dedicated registry.
func NewSupernodeMetrics() *SupernodeMetrics {
	reg := prometheus.NewRegistry()
	m := &SupernodeMetrics{
		VNRestarts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "supernode",
			Name:      "virtual_node_restarts_total",
			Help:      "Total number of virtual node restarts.",
		}, []string{"chain_id"}),
		InteropTimestampsVerified: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "supernode",
			Name:      "interop_timestamps_verified_total",
			Help:      "Total number of timestamps successfully verified by interop.",
		}),
		InteropInvalidations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "supernode",
			Name:      "interop_invalidations_total",
			Help:      "Total number of successful block invalidations triggered by interop.",
		}, []string{"chain_id"}),
		InteropVerifiedTimestamp: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "supernode",
			Name:      "interop_verified_timestamp",
			Help:      "Latest L2 timestamp successfully verified by interop.",
		}),
		InteropRoundDecisions: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "supernode",
			Name:      "interop_round_decisions_total",
			Help:      "Total number of interop round decisions by type.",
		}, []string{"decision"}),
		InteropRewinds: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "supernode",
			Name:      "interop_rewinds_total",
			Help:      "Total number of interop rewinds due to L1 consistency failures.",
		}),
		registry: reg,
	}
	reg.MustRegister(
		m.VNRestarts,
		m.InteropTimestampsVerified,
		m.InteropInvalidations,
		m.InteropVerifiedTimestamp,
		m.InteropRoundDecisions,
		m.InteropRewinds,
	)
	return m
}

// Registry returns the prometheus gatherer for these metrics.
func (m *SupernodeMetrics) Registry() prometheus.Gatherer {
	return m.registry
}
