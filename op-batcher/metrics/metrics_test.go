package metrics

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/stretchr/testify/require"
)

func TestThrottleMetrics(t *testing.T) {

	metrics := NewMetrics("test")

	// Initial metrics
	metrics.RecordThrottleControllerType(config.StepControllerType)
	metrics.RecordThrottleIntensity(37.4, config.StepControllerType)

	//  Metrics after a change from (say) an RPC call
	metrics.RecordThrottleControllerType(config.QuadraticControllerType)
	metrics.RecordThrottleIntensity(37.4, config.QuadraticControllerType)

	// Get the metrics checker to verify recorded values
	c := opmetrics.NewMetricChecker(t, metrics.Registry())

	prefix := "op_batcher_test_"

	// Verify throttle controller type metrics
	// QuadraticControllerType should be set to 1
	record := c.FindByName(prefix + "throttle_controller_type").FindByLabels(map[string]string{
		"type": string(config.QuadraticControllerType),
	})
	require.Equal(t, 1.0, record.Gauge.GetValue())

	// StepControllerType should be set to 0 (since we set QuadraticControllerType)
	record = c.FindByName(prefix + "throttle_controller_type").FindByLabels(map[string]string{
		"type": string(config.StepControllerType),
	})
	require.Equal(t, 0.0, record.Gauge.GetValue())

	// Verify throttle intensity metrics
	// StepControllerType should have intensity 37.4
	record = c.FindByName(prefix + "throttle_intensity").FindByLabels(map[string]string{
		"type": string(config.StepControllerType),
	})
	require.Equal(t, 0.0, record.Gauge.GetValue())

	// QuadraticControllerType should have intensity 0 (since we set intensity for StepControllerType)
	record = c.FindByName(prefix + "throttle_intensity").FindByLabels(map[string]string{
		"type": string(config.QuadraticControllerType),
	})
	require.Equal(t, 37.4, record.Gauge.GetValue())
}
