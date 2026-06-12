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
	// InteropHandoffGapSeconds is firstVerifiableTimestamp - activationTimestamp:
	// the width of the startup-handoff window reported "verified" without ever
	// being verified (covered only by the pre-activation / startup-handoff trust
	// assumption). Should be ~0 on a clean cold start at activation; a large
	// value means a chain's first SafeDB entry is far past activation (e.g. a
	// reseeded node), widening the trusted-but-unverified window.
	InteropHandoffGapSeconds prometheus.Gauge
	// LogsDBEntries is the current number of sealed blocks retained in each
	// chain's logsDB (latest - first + 1), after pruning.
	LogsDBEntries *prometheus.GaugeVec
	// LogsDBPruned counts sealed blocks dropped from each chain's logsDB by the
	// retention-window pruner.
	LogsDBPruned *prometheus.CounterVec
	// LogsDBPruneHorizon is the timestamp below which logsDB entries are pruned
	// (verifier frontier minus the retention window). 0 until pruning engages.
	LogsDBPruneHorizon prometheus.Gauge

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
		InteropHandoffGapSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "supernode",
			Name:      "interop_handoff_gap_seconds",
			Help:      "firstVerifiableTimestamp - activationTimestamp: width of the startup-handoff window reported verified without being verified. ~0 on a clean cold start; large means a chain's first SafeDB entry is far past activation.",
		}),
		LogsDBEntries: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "supernode",
			Name:      "logsdb_entries",
			Help:      "Number of sealed blocks retained in each chain's logsDB (after retention-window pruning).",
		}, []string{"chain_id"}),
		LogsDBPruned: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "supernode",
			Name:      "logsdb_pruned_total",
			Help:      "Total sealed blocks pruned from each chain's logsDB below the retention horizon.",
		}, []string{"chain_id"}),
		LogsDBPruneHorizon: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "supernode",
			Name:      "logsdb_prune_horizon_timestamp",
			Help:      "Timestamp below which logsDB entries are pruned (verifier frontier minus retention window).",
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
		m.InteropHandoffGapSeconds,
		m.LogsDBEntries,
		m.LogsDBPruned,
		m.LogsDBPruneHorizon,
	)
	return m
}

// Registry returns the prometheus gatherer for these metrics.
func (m *SupernodeMetrics) Registry() prometheus.Gatherer {
	return m.registry
}
