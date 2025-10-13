package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetrics(t *testing.T) {
	tests := []struct {
		name     string
		procName string
		expected string
	}{
		{
			name:     "default process name",
			procName: "",
			expected: "op_conductor_default",
		},
		{
			name:     "custom process name",
			procName: "rbuilder-1",
			expected: "op_conductor_rbuilder-1",
		},
		{
			name:     "explicit default",
			procName: "default",
			expected: "op_conductor_default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := NewMetrics(tt.procName)
			require.NotNil(t, metrics)
			assert.Equal(t, tt.expected, metrics.ns)
		})
	}
}

func TestRecordHealthCheck(t *testing.T) {
	metrics := NewMetrics("test")

	// Test successful health check
	metrics.RecordHealthCheck(true, nil)

	// Test failed health check
	err := assert.AnError
	metrics.RecordHealthCheck(false, err)

	// Verify metrics were recorded
	// Note: In a real test, you would collect and verify the metric values
	// This is a basic smoke test to ensure no panics occur
}

func TestRecordHealthCheckWithReplicaID(t *testing.T) {
	metrics := NewMetrics("test")

	tests := []struct {
		name      string
		success   bool
		err       error
		replicaID string
	}{
		{
			name:      "successful health check with replica ID",
			success:   true,
			err:       nil,
			replicaID: "rbuilder-1",
		},
		{
			name:      "failed health check with replica ID",
			success:   false,
			err:       assert.AnError,
			replicaID: "rbuilder-2",
		},
		{
			name:      "empty replica ID defaults to default",
			success:   true,
			err:       nil,
			replicaID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.RecordHealthCheckWithReplicaID(tt.success, tt.err, tt.replicaID)
			// Note: In a real test, you would collect and verify the metric values
			// This is a basic smoke test to ensure no panics occur
		})
	}
}

func TestMetricsRegistry(t *testing.T) {
	metrics := NewMetrics("test")
	registry := metrics.Registry()
	require.NotNil(t, registry)

	// Verify registry contains expected metrics
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Note: The metric might not be found if no health checks have been recorded yet
	// This is expected behavior for a new metrics instance
	t.Logf("Found %d metric families", len(metricFamilies))
	for _, mf := range metricFamilies {
		t.Logf("Metric family: %s", mf.GetName())
	}

	// Verify that the registry is working properly
	assert.NotEmpty(t, metricFamilies, "Registry should contain some metrics")
}

func TestNoopMetrics(t *testing.T) {
	noop := &NoopMetricsImpl{}

	// Test that noop methods don't panic
	noop.RecordHealthCheck(true, nil)
	noop.RecordHealthCheckWithReplicaID(false, assert.AnError, "test-replica")
	noop.RecordInfo("test-version")
	noop.RecordUp()
	noop.RecordStateChange(true, true, true)
	noop.RecordLeaderTransfer(true)
	noop.RecordStartSequencer(true)
	noop.RecordStopSequencer(true)
	noop.RecordLoopExecutionTime(1.0)
	noop.RecordRollupBoostConnectionAttempts(true, "test-source")
	noop.RecordWebSocketClientCount(5)
}
