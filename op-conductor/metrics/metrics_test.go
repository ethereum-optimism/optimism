package metrics

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestHealthCheckDebugGauges(t *testing.T) {
	metricer := NewMetrics()

	metricer.RecordHealthCheckConfig(1, 10, 60, 2, 7, true, true)
	metricer.RecordHealthCheckHeads(100, 1234, 90, 1200, 5, 39)
	metricer.RecordHealthCheckPeerCount(3, 2)
	metricer.RecordUnsafeHeadRecovery(true, 12, 20, 15, 4, 8, 3, 2)

	require.Equal(t, float64(1), metricValue(t, metricer, "op_conductor_healthcheck_interval_seconds", nil))
	require.Equal(t, float64(10), metricValue(t, metricer, "op_conductor_healthcheck_unsafe_interval_seconds", nil))
	require.Equal(t, float64(60), metricValue(t, metricer, "op_conductor_healthcheck_safe_interval_seconds", nil))
	require.Equal(t, float64(1), metricValue(t, metricer, "op_conductor_healthcheck_safe_enabled", nil))
	require.Equal(t, float64(1), metricValue(t, metricer, "op_conductor_healthcheck_interop_reorg_leniency_enabled", nil))
	require.Equal(t, float64(7), metricValue(t, metricer, "op_conductor_healthcheck_interop_reorg_leniency_window_size", nil))
	require.Equal(t, float64(100), metricValue(t, metricer, "op_conductor_healthcheck_unsafe_head_number", nil))
	require.Equal(t, float64(1234), metricValue(t, metricer, "op_conductor_healthcheck_unsafe_head_timestamp", nil))
	require.Equal(t, float64(5), metricValue(t, metricer, "op_conductor_healthcheck_unsafe_lag_seconds", nil))
	require.Equal(t, float64(90), metricValue(t, metricer, "op_conductor_healthcheck_safe_head_number", nil))
	require.Equal(t, float64(1200), metricValue(t, metricer, "op_conductor_healthcheck_safe_head_timestamp", nil))
	require.Equal(t, float64(39), metricValue(t, metricer, "op_conductor_healthcheck_safe_lag_seconds", nil))
	require.Equal(t, float64(3), metricValue(t, metricer, "op_conductor_healthcheck_cl_peer_count", nil))
	require.Equal(t, float64(2), metricValue(t, metricer, "op_conductor_healthcheck_min_peer_count", nil))
	require.Equal(t, float64(1), metricValue(t, metricer, "op_conductor_healthcheck_unsafe_head_recovery_active", nil))
	require.Equal(t, float64(12), metricValue(t, metricer, "op_conductor_healthcheck_unsafe_head_recovery_current_lag_seconds", nil))
	require.Equal(t, float64(20), metricValue(t, metricer, "op_conductor_healthcheck_unsafe_head_recovery_initial_lag_seconds", nil))
	require.Equal(t, float64(15), metricValue(t, metricer, "op_conductor_healthcheck_unsafe_head_recovery_window_start_lag_seconds", nil))
	require.Equal(t, float64(4), metricValue(t, metricer, "op_conductor_healthcheck_unsafe_head_recovery_wall_elapsed_seconds", nil))
	require.Equal(t, float64(8), metricValue(t, metricer, "op_conductor_healthcheck_unsafe_head_recovery_unsafe_elapsed_seconds", nil))
	require.Equal(t, float64(3), metricValue(t, metricer, "op_conductor_healthcheck_unsafe_head_recovery_polls", nil))
	require.Equal(t, float64(2), metricValue(t, metricer, "op_conductor_healthcheck_unsafe_head_recovery_polls_in_window", nil))
}

func TestHealthCheckBoundedLabelMetrics(t *testing.T) {
	metricer := NewMetrics()

	metricer.RecordHealthCheckStatus(HealthCheckUnsafeLag, HealthCheckStatusRecovering)
	metricer.RecordHealthCheckWindow(HealthCheckSyncStatusRPC, HealthCheckWindowStateInconclusive, 2, 3, 5)
	metricer.RecordHealthCheckFailure(HealthCheckPeerCount, HealthCheckFailureReasonPeerCountBelowMin)
	metricer.RecordUnsafeHeadRecoveryEvent(HealthCheckRecoveryEntered)

	require.Equal(t, float64(1), metricValue(t, metricer, "op_conductor_healthcheck_check_status", map[string]string{
		"check":  string(HealthCheckUnsafeLag),
		"status": string(HealthCheckStatusRecovering),
	}))
	require.Equal(t, float64(0), metricValue(t, metricer, "op_conductor_healthcheck_check_status", map[string]string{
		"check":  string(HealthCheckUnsafeLag),
		"status": string(HealthCheckStatusUnhealthy),
	}))
	require.Equal(t, float64(1), metricValue(t, metricer, "op_conductor_healthcheck_window_state", map[string]string{
		"check": string(HealthCheckSyncStatusRPC),
		"state": string(HealthCheckWindowStateInconclusive),
	}))
	require.Equal(t, float64(0), metricValue(t, metricer, "op_conductor_healthcheck_window_state", map[string]string{
		"check": string(HealthCheckSyncStatusRPC),
		"state": string(HealthCheckWindowStateFailed),
	}))
	require.Equal(t, float64(2), metricValue(t, metricer, "op_conductor_healthcheck_window_observations", map[string]string{
		"check":  string(HealthCheckSyncStatusRPC),
		"result": "success",
	}))
	require.Equal(t, float64(3), metricValue(t, metricer, "op_conductor_healthcheck_window_observations", map[string]string{
		"check":  string(HealthCheckSyncStatusRPC),
		"result": "failure",
	}))
	require.Equal(t, float64(1), metricValue(t, metricer, "op_conductor_healthcheck_window_filled", map[string]string{
		"check": string(HealthCheckSyncStatusRPC),
	}))
	require.Equal(t, float64(5), metricValue(t, metricer, "op_conductor_healthcheck_window_size", map[string]string{
		"check": string(HealthCheckSyncStatusRPC),
	}))
	require.Equal(t, float64(1), metricValue(t, metricer, "op_conductor_healthcheck_check_failures_count", map[string]string{
		"check":  string(HealthCheckPeerCount),
		"reason": string(HealthCheckFailureReasonPeerCountBelowMin),
	}))
	require.Equal(t, float64(1), metricValue(t, metricer, "op_conductor_healthcheck_interop_reorg_recovery_events_count", map[string]string{
		"event": string(HealthCheckRecoveryEntered),
	}))
}

func metricValue(t *testing.T, metricer *Metrics, name string, labels map[string]string) float64 {
	t.Helper()
	metricFamilies, err := metricer.Registry().Gather()
	require.NoError(t, err)
	for _, family := range metricFamilies {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			if labelsMatch(metric, labels) {
				return sampleValue(t, metric)
			}
		}
	}
	t.Fatalf("metric %s with labels %v not found", name, labels)
	return 0
}

func labelsMatch(metric *dto.Metric, labels map[string]string) bool {
	actual := make(map[string]string, len(metric.GetLabel()))
	for _, label := range metric.GetLabel() {
		actual[label.GetName()] = label.GetValue()
	}
	for key, want := range labels {
		if actual[key] != want {
			return false
		}
	}
	return len(actual) == len(labels)
}

func sampleValue(t *testing.T, metric *dto.Metric) float64 {
	t.Helper()
	if metric.GetGauge() != nil {
		return metric.GetGauge().GetValue()
	}
	if metric.GetCounter() != nil {
		return metric.GetCounter().GetValue()
	}
	t.Fatalf("metric has no gauge or counter sample")
	return 0
}
