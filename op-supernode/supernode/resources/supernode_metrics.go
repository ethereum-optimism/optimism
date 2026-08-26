package resources

import "github.com/prometheus/client_golang/prometheus"

// SupernodeMetrics holds supernode-level metrics that outlive individual
// virtual node restarts. Created once in supernode.New() and shared with
// chain containers and activities. Callers that receive nil default to
// NewSupernodeMetrics(), which creates functional counters not attached
// to any scraped registry (safe for tests).
type SupernodeMetrics struct {
	Info                        *prometheus.GaugeVec
	VNRestarts                  *prometheus.CounterVec
	InteropTimestampsVerified   prometheus.Counter
	InteropInvalidations        *prometheus.CounterVec
	InteropVerifiedTimestamp    prometheus.Gauge
	InteropRoundDecisions       *prometheus.CounterVec
	InteropRewinds              prometheus.Counter
	InteropVerificationDuration prometheus.Histogram
	ChainRewindDepthBlocks      *prometheus.HistogramVec
	ChainRewindDuration         *prometheus.HistogramVec
	ChainRewindFailures         *prometheus.CounterVec
	DenyListEntries             *prometheus.CounterVec
	LogBackfillProgress         *prometheus.GaugeVec
	LogBackfillRetries          *prometheus.CounterVec
	ActivityErrors              *prometheus.CounterVec
	// InteropActivityState tracks the interop activity lifecycle:
	// 0=not_started, 1=cold_start_waiting, 2=running, 3=halted.
	InteropActivityState prometheus.Gauge

	// The silhouette gauges. All three exist because a proof-carried chain fails QUIETLY: a halted
	// shim keeps its process running and refuses every request, and a proof stream that stops
	// arriving looks exactly like a chain with nothing to say. Every one of these is exported for a
	// chain the moment it is declared, at zero, so that "no series" means "no silhouette chain"
	// rather than "nothing has gone wrong yet" — an alert on an absent series never fires.
	SilhouetteShimHalted           *prometheus.GaugeVec
	SilhouetteProvenHead           *prometheus.GaugeVec
	SilhouetteTrackerL1            *prometheus.GaugeVec
	SilhouetteInvalidationsRefused *prometheus.CounterVec

	registry *prometheus.Registry
}

// NewSupernodeMetrics creates a new SupernodeMetrics backed by a dedicated registry.
func NewSupernodeMetrics() *SupernodeMetrics {
	reg := prometheus.NewRegistry()
	m := &SupernodeMetrics{
		Info: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "supernode",
			Name:      "info",
			Help:      "Supernode build information.",
		}, []string{"version", "commit"}),
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
		ChainRewindDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "supernode",
			Name:      "chain_rewind_duration_seconds",
			Help:      "Duration in seconds of chain rewind attempts.",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		}, []string{"chain_id"}),
		ChainRewindFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "supernode",
			Name:      "chain_rewind_failures_total",
			Help:      "Total number of failed chain rewind attempts by stage.",
		}, []string{"chain_id", "stage"}),
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
		SilhouetteShimHalted: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "supernode",
			Name:      "silhouette_shim_halted",
			Help: "1 when a proof-carried chain's shim has fail-stopped (the DR-1 honesty assertion, " +
				"logged as SILHOUETTE SHIM HALTED). The process keeps running and refuses every " +
				"request, so this is the only machine-readable difference between halted and slow.",
		}, []string{"chain_id"}),
		SilhouetteProvenHead: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "supernode",
			Name:      "silhouette_proven_head",
			Help: "Highest block of a proof-carried chain this node holds a proven-or-forced fact for. " +
				"In the sequencer posture this is where the chain's PUBLIC safety labels come from, " +
				"and it is not visible in optimism_syncStatus.",
		}, []string{"chain_id"}),
		SilhouetteTrackerL1: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "supernode",
			Name:      "silhouette_tracker_l1_cursor",
			Help: "Next L1 block the sequencer posture's proven-head walk will read. Against the L1 " +
				"head it separates 'no proofs are landing' from 'this node stopped looking', which " +
				"are the same symptom and different incidents.",
		}, []string{"chain_id"}),
		SilhouetteInvalidationsRefused: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "supernode",
			Name:      "silhouette_invalidations_refused_total",
			Help: "Times the cross-safety judge found a block of a proof-carried chain INVALID and this " +
				"node refused to replace it (G7G D3: a proven chain is never replaced, only stopped). " +
				"Any non-zero value is an incident: the dependency set's cross-safe frontier is pinned " +
				"at that block and only the chain's prover can clear it, by re-proving from the last " +
				"valid point. It is deliberately separate from interop_invalidations, which counts " +
				"invalidations that were CARRIED OUT.",
		}, []string{"chain_id"}),
		registry: reg,
	}
	reg.MustRegister(
		m.Info,
		m.VNRestarts,
		m.InteropTimestampsVerified,
		m.InteropInvalidations,
		m.InteropVerifiedTimestamp,
		m.InteropRoundDecisions,
		m.InteropRewinds,
		m.InteropVerificationDuration,
		m.ChainRewindDepthBlocks,
		m.ChainRewindDuration,
		m.ChainRewindFailures,
		m.DenyListEntries,
		m.LogBackfillProgress,
		m.LogBackfillRetries,
		m.ActivityErrors,
		m.InteropActivityState,
		m.SilhouetteShimHalted,
		m.SilhouetteProvenHead,
		m.SilhouetteTrackerL1,
		m.SilhouetteInvalidationsRefused,
	)
	return m
}

// Registry returns the prometheus gatherer for these metrics.
func (m *SupernodeMetrics) Registry() prometheus.Gatherer {
	return m.registry
}
