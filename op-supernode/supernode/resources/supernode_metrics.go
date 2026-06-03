package resources

import "github.com/prometheus/client_golang/prometheus"

// SupernodeMetrics holds supernode-level metrics that outlive individual
// virtual node restarts. Created once in supernode.New() and shared with
// chain containers and activities. Callers that receive nil default to
// NewSupernodeMetrics(), which creates functional counters not attached
// to any scraped registry (safe for tests).
type SupernodeMetrics struct {
	VNRestarts                  *prometheus.CounterVec
	InteropTimestampsVerified   prometheus.Counter
	InteropInvalidations        *prometheus.CounterVec
	InteropVerifiedTimestamp    prometheus.Gauge
	InteropRoundDecisions       *prometheus.CounterVec
	InteropRewinds              prometheus.Counter
	InteropVerificationDuration prometheus.Histogram
	ChainRewindDepthBlocks      *prometheus.HistogramVec
	DenyListEntries             *prometheus.CounterVec
	LogBackfillProgress         *prometheus.GaugeVec
	LogBackfillRetries          *prometheus.CounterVec
	ActivityErrors              *prometheus.CounterVec
	// InteropActivityState tracks the interop activity lifecycle:
	// 0=not_started, 1=cold_start_waiting, 2=running, 3=halted.
	InteropActivityState prometheus.Gauge

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
		InteropVerificationDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "supernode",
			Name:      "interop_verification_duration_seconds",
			Help:      "Time from timestamp available on all chains to verified result persisted.",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		}),
		ChainRewindDepthBlocks: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "supernode",
			Name:      "chain_rewind_depth_blocks",
			Help:      "Depth in blocks of chain rewinds triggered by invalidation.",
			Buckets:   []float64{1, 2, 5, 10, 50, 100, 500},
		}, []string{"chain_id"}),
		DenyListEntries: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "supernode",
			Name:      "denylist_entries_total",
			Help:      "Total number of deny list entries added per chain.",
		}, []string{"chain_id"}),
		LogBackfillProgress: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "supernode",
			Name:      "log_backfill_progress",
			Help:      "Log backfill progress per chain (0.0 to 1.0).",
		}, []string{"chain_id"}),
		LogBackfillRetries: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "supernode",
			Name:      "log_backfill_retries_total",
			Help:      "Total number of log backfill retry attempts per chain.",
		}, []string{"chain_id"}),
		ActivityErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "supernode",
			Name:      "activity_errors_total",
			Help:      "Total number of activity errors by activity name and error type.",
		}, []string{"activity", "error_type"}),
		InteropActivityState: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "supernode",
			Name:      "interop_activity_state",
			Help:      "Interop activity lifecycle state: 0=not_started, 1=cold_start_waiting, 2=running, 3=halted.",
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
		m.InteropVerificationDuration,
		m.ChainRewindDepthBlocks,
		m.DenyListEntries,
		m.LogBackfillProgress,
		m.LogBackfillRetries,
		m.ActivityErrors,
		m.InteropActivityState,
	)
	return m
}

// Registry returns the prometheus gatherer for these metrics.
func (m *SupernodeMetrics) Registry() prometheus.Gatherer {
	return m.registry
}
