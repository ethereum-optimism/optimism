package metrics

import (
	"strconv"

	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

const Namespace = "op_conductor"

type HealthCheck string

const (
	HealthCheckSyncStatusRPC HealthCheck = "sync_status_rpc"
	HealthCheckUnsafeLag     HealthCheck = "unsafe_lag"
	HealthCheckSafeLag       HealthCheck = "safe_lag"
	HealthCheckPeerCount     HealthCheck = "peer_count"
)

type HealthCheckStatus string

const (
	HealthCheckStatusHealthy    HealthCheckStatus = "healthy"
	HealthCheckStatusWarning    HealthCheckStatus = "warning"
	HealthCheckStatusRecovering HealthCheckStatus = "recovering"
	HealthCheckStatusUnhealthy  HealthCheckStatus = "unhealthy"
	HealthCheckStatusDisabled   HealthCheckStatus = "disabled"
)

type HealthCheckWindowState string

const (
	HealthCheckWindowStateSuccess      HealthCheckWindowState = "success"
	HealthCheckWindowStateFailed       HealthCheckWindowState = "failed"
	HealthCheckWindowStateInconclusive HealthCheckWindowState = "inconclusive"
)

type HealthCheckFailureReason string

const (
	HealthCheckFailureReasonRPCError              HealthCheckFailureReason = "rpc_error"
	HealthCheckFailureReasonLagExceeded           HealthCheckFailureReason = "lag_exceeded"
	HealthCheckFailureReasonPeerCountBelowMin     HealthCheckFailureReason = "peer_count_below_min"
	HealthCheckFailureReasonRecoveryWindowStalled HealthCheckFailureReason = "recovery_window_stalled"
)

type HealthCheckRecoveryEvent string

const (
	HealthCheckRecoveryEntered     HealthCheckRecoveryEvent = "entered"
	HealthCheckRecoveryWindowReset HealthCheckRecoveryEvent = "window_reset"
	HealthCheckRecoveryRecovered   HealthCheckRecoveryEvent = "recovered"
	HealthCheckRecoveryFailed      HealthCheckRecoveryEvent = "failed"
)

type Metricer interface {
	RecordInfo(version string)
	RecordUp()
	RecordStateChange(leader bool, healthy bool, active bool)
	RecordLeaderTransfer(success bool)
	RecordStartSequencer(success bool)
	RecordStopSequencer(success bool)
	RecordHealthCheck(success bool, err error)
	RecordLoopExecutionTime(duration float64)
	RecordRollupBoostConnectionAttempts(success bool, source string)
	RecordWebSocketClientCount(count int)
	RecordHealthCheckConfig(interval, unsafeInterval, safeInterval, minPeerCount, interopReorgLeniencyWindowSize uint64, safeEnabled, interopReorgLeniency bool)
	RecordHealthCheckHeads(unsafeNumber, unsafeTimestamp, safeNumber, safeTimestamp, unsafeLag, safeLag uint64)
	RecordHealthCheckPeerCount(peerCount, minPeerCount uint64)
	RecordHealthCheckWindow(check HealthCheck, state HealthCheckWindowState, successes, failures, windowSize uint64)
	RecordHealthCheckStatus(check HealthCheck, status HealthCheckStatus)
	RecordHealthCheckFailure(check HealthCheck, reason HealthCheckFailureReason)
	RecordUnsafeHeadRecovery(active bool, currentLag, initialLag, windowStartLag, wallElapsed, unsafeElapsed, polls, pollsInWindow uint64)
	RecordUnsafeHeadRecoveryEvent(event HealthCheckRecoveryEvent)
	opmetrics.RPCMetricer
}

// Metrics implementation must implement RegistryMetricer to allow the metrics server to work.
var _ opmetrics.RegistryMetricer = (*Metrics)(nil)

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	opmetrics.RPCMetrics

	info prometheus.GaugeVec
	up   prometheus.Gauge

	healthChecks                  *prometheus.CounterVec
	leaderTransfers               *prometheus.CounterVec
	sequencerStarts               *prometheus.CounterVec
	sequencerStops                *prometheus.CounterVec
	stateChanges                  *prometheus.CounterVec
	rollupBoostConnectionAttempts *prometheus.CounterVec

	loopExecutionTime prometheus.Histogram
	webSocketClients  prometheus.Gauge

	healthCheckIntervalSeconds                  prometheus.Gauge
	healthCheckUnsafeIntervalSeconds            prometheus.Gauge
	healthCheckSafeIntervalSeconds              prometheus.Gauge
	healthCheckSafeEnabled                      prometheus.Gauge
	healthCheckInteropReorgLeniencyEnabled      prometheus.Gauge
	healthCheckInteropReorgLeniencyWindowSize   prometheus.Gauge
	healthCheckUnsafeLagSeconds                 prometheus.Gauge
	healthCheckSafeLagSeconds                   prometheus.Gauge
	healthCheckUnsafeHeadNumber                 prometheus.Gauge
	healthCheckUnsafeHeadTimestamp              prometheus.Gauge
	healthCheckSafeHeadNumber                   prometheus.Gauge
	healthCheckSafeHeadTimestamp                prometheus.Gauge
	healthCheckCLPeerCount                      prometheus.Gauge
	healthCheckMinPeerCount                     prometheus.Gauge
	healthCheckUnsafeHeadRecoveryActive         prometheus.Gauge
	healthCheckUnsafeHeadRecoveryCurrentLag     prometheus.Gauge
	healthCheckUnsafeHeadRecoveryInitialLag     prometheus.Gauge
	healthCheckUnsafeHeadRecoveryWindowStartLag prometheus.Gauge
	healthCheckUnsafeHeadRecoveryWallElapsed    prometheus.Gauge
	healthCheckUnsafeHeadRecoveryUnsafeElapsed  prometheus.Gauge
	healthCheckUnsafeHeadRecoveryPolls          prometheus.Gauge
	healthCheckUnsafeHeadRecoveryPollsInWindow  prometheus.Gauge
	healthCheckWindowState                      *prometheus.GaugeVec
	healthCheckWindowObservations               *prometheus.GaugeVec
	healthCheckWindowFilled                     *prometheus.GaugeVec
	healthCheckWindowSize                       *prometheus.GaugeVec
	healthCheckStatus                           *prometheus.GaugeVec
	healthCheckFailures                         *prometheus.CounterVec
	healthCheckRecoveryEvents                   *prometheus.CounterVec
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

var _ Metricer = (*Metrics)(nil)

func NewMetrics() *Metrics {
	registry := opmetrics.NewRegistry()
	factory := opmetrics.With(registry)

	return &Metrics{
		ns:       Namespace,
		registry: registry,
		factory:  factory,

		RPCMetrics: opmetrics.MakeRPCMetrics(Namespace, factory),

		info: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "info",
			Help:      "Pseudo-metric tracking version and config info",
		}, []string{
			"version",
		}),
		up: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "up",
			Help:      "1 if the op-conductor has finished starting up",
		}),
		healthChecks: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "healthchecks_count",
			Help:      "Number of healthchecks",
		}, []string{"success", "error"}),
		leaderTransfers: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "leader_transfers_count",
			Help:      "Number of leader transfers",
		}, []string{"success"}),
		sequencerStarts: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "sequencer_starts_count",
			Help:      "Number of sequencer starts",
		}, []string{"success"}),
		sequencerStops: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "sequencer_stops_count",
			Help:      "Number of sequencer stops",
		}, []string{"success"}),
		stateChanges: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "state_changes_count",
			Help:      "Number of state changes",
		}, []string{
			"leader",
			"healthy",
			"active",
		}),
		loopExecutionTime: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: Namespace,
			Name:      "loop_execution_time",
			Help:      "Time (in seconds) to execute conductor loop iteration",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}),
		rollupBoostConnectionAttempts: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "rollup_boost_connection_attempts_count",
			Help:      "Number of rollup boost connection attempts",
		}, []string{"success", "source"}),
		webSocketClients: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "websocket_clients_connected",
			Help:      "Number of WebSocket clients currently connected to the hub",
		}),
		healthCheckIntervalSeconds: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_interval_seconds",
			Help:      "Configured interval between conductor health checks, in seconds",
		}),
		healthCheckUnsafeIntervalSeconds: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_unsafe_interval_seconds",
			Help:      "Configured maximum unsafe-head lag, in seconds",
		}),
		healthCheckSafeIntervalSeconds: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_safe_interval_seconds",
			Help:      "Configured maximum safe-head lag, in seconds",
		}),
		healthCheckSafeEnabled: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_safe_enabled",
			Help:      "1 if conductor safe-head health checks are enabled",
		}),
		healthCheckInteropReorgLeniencyEnabled: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_interop_reorg_leniency_enabled",
			Help:      "1 if conductor interop reorg health-check leniency is enabled",
		}),
		healthCheckInteropReorgLeniencyWindowSize: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_interop_reorg_leniency_window_size",
			Help:      "Configured number of observations in the conductor interop reorg health-check leniency window",
		}),
		healthCheckUnsafeLagSeconds: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_unsafe_lag_seconds",
			Help:      "Current unsafe-head lag, in seconds",
		}),
		healthCheckSafeLagSeconds: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_safe_lag_seconds",
			Help:      "Current safe-head lag, in seconds",
		}),
		healthCheckUnsafeHeadNumber: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_unsafe_head_number",
			Help:      "Current unsafe L2 head block number observed by the conductor health monitor",
		}),
		healthCheckUnsafeHeadTimestamp: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_unsafe_head_timestamp",
			Help:      "Current unsafe L2 head timestamp observed by the conductor health monitor",
		}),
		healthCheckSafeHeadNumber: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_safe_head_number",
			Help:      "Current safe L2 head block number observed by the conductor health monitor",
		}),
		healthCheckSafeHeadTimestamp: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_safe_head_timestamp",
			Help:      "Current safe L2 head timestamp observed by the conductor health monitor",
		}),
		healthCheckCLPeerCount: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_cl_peer_count",
			Help:      "Current consensus-layer peer count observed by the conductor health monitor",
		}),
		healthCheckMinPeerCount: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_min_peer_count",
			Help:      "Configured minimum consensus-layer peer count for the conductor health monitor",
		}),
		healthCheckUnsafeHeadRecoveryActive: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_unsafe_head_recovery_active",
			Help:      "1 if unsafe-head recovery leniency is currently active",
		}),
		healthCheckUnsafeHeadRecoveryCurrentLag: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_unsafe_head_recovery_current_lag_seconds",
			Help:      "Current unsafe-head lag while recovery leniency is active, in seconds",
		}),
		healthCheckUnsafeHeadRecoveryInitialLag: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_unsafe_head_recovery_initial_lag_seconds",
			Help:      "Initial unsafe-head lag when the current recovery episode began, in seconds",
		}),
		healthCheckUnsafeHeadRecoveryWindowStartLag: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_unsafe_head_recovery_window_start_lag_seconds",
			Help:      "Unsafe-head lag at the start of the current recovery window, in seconds",
		}),
		healthCheckUnsafeHeadRecoveryWallElapsed: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_unsafe_head_recovery_wall_elapsed_seconds",
			Help:      "Wall-clock elapsed time in the current unsafe-head recovery window, in seconds",
		}),
		healthCheckUnsafeHeadRecoveryUnsafeElapsed: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_unsafe_head_recovery_unsafe_elapsed_seconds",
			Help:      "Unsafe-head timestamp elapsed time in the current recovery window, in seconds",
		}),
		healthCheckUnsafeHeadRecoveryPolls: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_unsafe_head_recovery_polls",
			Help:      "Number of health-check polls in the current unsafe-head recovery episode",
		}),
		healthCheckUnsafeHeadRecoveryPollsInWindow: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_unsafe_head_recovery_polls_in_window",
			Help:      "Number of health-check polls in the current unsafe-head recovery window",
		}),
		healthCheckWindowState: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_window_state",
			Help:      "One-hot state of a conductor health-check rolling window",
		}, []string{"check", "state"}),
		healthCheckWindowObservations: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_window_observations",
			Help:      "Current conductor health-check rolling-window observations by result",
		}, []string{"check", "result"}),
		healthCheckWindowFilled: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_window_filled",
			Help:      "1 if a conductor health-check rolling window has reached its configured size",
		}, []string{"check"}),
		healthCheckWindowSize: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_window_size",
			Help:      "Configured size of a conductor health-check rolling window",
		}, []string{"check"}),
		healthCheckStatus: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "healthcheck_check_status",
			Help:      "One-hot conductor health-check status by bounded check and status labels",
		}, []string{"check", "status"}),
		healthCheckFailures: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "healthcheck_check_failures_count",
			Help:      "Number of conductor health-check failures by bounded check and reason labels",
		}, []string{"check", "reason"}),
		healthCheckRecoveryEvents: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "healthcheck_interop_reorg_recovery_events_count",
			Help:      "Number of unsafe-head recovery events from conductor interop reorg health-check leniency",
		}, []string{"event"}),
	}
}

func (m *Metrics) Start(host string, port int) (*httputil.HTTPServer, error) {
	return opmetrics.StartServer(m.registry, host, port)
}

// RecordInfo sets a pseudo-metric that contains versioning and
// config info for the op-proposer.
func (m *Metrics) RecordInfo(version string) {
	m.info.WithLabelValues(version).Set(1)
}

// RecordUp sets the up metric to 1.
func (m *Metrics) RecordUp() {
	m.up.Set(1)
}

// RecordHealthCheck increments the healthChecks counter.
func (m *Metrics) RecordHealthCheck(success bool, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	m.healthChecks.WithLabelValues(strconv.FormatBool(success), errStr).Inc()
}

// RecordLeaderTransfer increments the leaderTransfers counter.
func (m *Metrics) RecordLeaderTransfer(success bool) {
	m.leaderTransfers.WithLabelValues(strconv.FormatBool(success)).Inc()
}

// RecordStateChange increments the stateChanges counter.
func (m *Metrics) RecordStateChange(leader bool, healthy bool, active bool) {
	m.stateChanges.WithLabelValues(strconv.FormatBool(leader), strconv.FormatBool(healthy), strconv.FormatBool(active)).Inc()
}

// RecordStartSequencer increments the sequencerStarts counter.
func (m *Metrics) RecordStartSequencer(success bool) {
	m.sequencerStarts.WithLabelValues(strconv.FormatBool(success)).Inc()
}

// RecordStopSequencer increments the sequencerStops counter.
func (m *Metrics) RecordStopSequencer(success bool) {
	m.sequencerStops.WithLabelValues(strconv.FormatBool(success)).Inc()
}

// RecordLoopExecutionTime records the time it took to execute the conductor loop.
func (m *Metrics) RecordLoopExecutionTime(duration float64) {
	m.loopExecutionTime.Observe(duration)
}

// RecordRollupBoostConnectionAttempts increments the rollupBoostConnectionAttempts counter.
func (m *Metrics) RecordRollupBoostConnectionAttempts(success bool, source string) {
	m.rollupBoostConnectionAttempts.WithLabelValues(strconv.FormatBool(success), source).Inc()
}

// RecordWebSocketClientCount sets the current number of WebSocket clients connected.
func (m *Metrics) RecordWebSocketClientCount(count int) {
	m.webSocketClients.Set(float64(count))
}

func (m *Metrics) RecordHealthCheckConfig(interval, unsafeInterval, safeInterval, minPeerCount, interopReorgLeniencyWindowSize uint64, safeEnabled, interopReorgLeniency bool) {
	m.healthCheckIntervalSeconds.Set(float64(interval))
	m.healthCheckUnsafeIntervalSeconds.Set(float64(unsafeInterval))
	m.healthCheckSafeIntervalSeconds.Set(float64(safeInterval))
	m.healthCheckSafeEnabled.Set(boolToFloat(safeEnabled))
	m.healthCheckInteropReorgLeniencyEnabled.Set(boolToFloat(interopReorgLeniency))
	m.healthCheckInteropReorgLeniencyWindowSize.Set(float64(interopReorgLeniencyWindowSize))
	m.healthCheckMinPeerCount.Set(float64(minPeerCount))
}

func (m *Metrics) RecordHealthCheckHeads(unsafeNumber, unsafeTimestamp, safeNumber, safeTimestamp, unsafeLag, safeLag uint64) {
	m.healthCheckUnsafeHeadNumber.Set(float64(unsafeNumber))
	m.healthCheckUnsafeHeadTimestamp.Set(float64(unsafeTimestamp))
	m.healthCheckUnsafeLagSeconds.Set(float64(unsafeLag))
	m.healthCheckSafeHeadNumber.Set(float64(safeNumber))
	m.healthCheckSafeHeadTimestamp.Set(float64(safeTimestamp))
	m.healthCheckSafeLagSeconds.Set(float64(safeLag))
}

func (m *Metrics) RecordHealthCheckPeerCount(peerCount, minPeerCount uint64) {
	m.healthCheckCLPeerCount.Set(float64(peerCount))
	m.healthCheckMinPeerCount.Set(float64(minPeerCount))
}

func (m *Metrics) RecordHealthCheckWindow(check HealthCheck, state HealthCheckWindowState, successes, failures, windowSize uint64) {
	for _, windowState := range []HealthCheckWindowState{
		HealthCheckWindowStateSuccess,
		HealthCheckWindowStateFailed,
		HealthCheckWindowStateInconclusive,
	} {
		m.healthCheckWindowState.WithLabelValues(string(check), string(windowState)).Set(boolToFloat(windowState == state))
	}

	m.healthCheckWindowObservations.WithLabelValues(string(check), "success").Set(float64(successes))
	m.healthCheckWindowObservations.WithLabelValues(string(check), "failure").Set(float64(failures))
	m.healthCheckWindowSize.WithLabelValues(string(check)).Set(float64(windowSize))
	m.healthCheckWindowFilled.WithLabelValues(string(check)).Set(boolToFloat(successes+failures >= windowSize))
}

func (m *Metrics) RecordHealthCheckStatus(check HealthCheck, status HealthCheckStatus) {
	for _, checkStatus := range []HealthCheckStatus{
		HealthCheckStatusHealthy,
		HealthCheckStatusWarning,
		HealthCheckStatusRecovering,
		HealthCheckStatusUnhealthy,
		HealthCheckStatusDisabled,
	} {
		m.healthCheckStatus.WithLabelValues(string(check), string(checkStatus)).Set(boolToFloat(checkStatus == status))
	}
}

func (m *Metrics) RecordHealthCheckFailure(check HealthCheck, reason HealthCheckFailureReason) {
	m.healthCheckFailures.WithLabelValues(string(check), string(reason)).Inc()
}

func (m *Metrics) RecordUnsafeHeadRecovery(active bool, currentLag, initialLag, windowStartLag, wallElapsed, unsafeElapsed, polls, pollsInWindow uint64) {
	m.healthCheckUnsafeHeadRecoveryActive.Set(boolToFloat(active))
	m.healthCheckUnsafeHeadRecoveryCurrentLag.Set(float64(currentLag))
	m.healthCheckUnsafeHeadRecoveryInitialLag.Set(float64(initialLag))
	m.healthCheckUnsafeHeadRecoveryWindowStartLag.Set(float64(windowStartLag))
	m.healthCheckUnsafeHeadRecoveryWallElapsed.Set(float64(wallElapsed))
	m.healthCheckUnsafeHeadRecoveryUnsafeElapsed.Set(float64(unsafeElapsed))
	m.healthCheckUnsafeHeadRecoveryPolls.Set(float64(polls))
	m.healthCheckUnsafeHeadRecoveryPollsInWindow.Set(float64(pollsInWindow))
}

func (m *Metrics) RecordUnsafeHeadRecoveryEvent(event HealthCheckRecoveryEvent) {
	m.healthCheckRecoveryEvents.WithLabelValues(string(event)).Inc()
}

func boolToFloat(v bool) float64 {
	if v {
		return 1
	}
	return 0
}
