package sysgo

import "strconv"

const sysgoMetricsEnabledEnvVar = "SYSGO_METRICS_ENABLED"

func areMetricsEnabled() bool {
	enabledStr := getEnvVarOrDefault(sysgoMetricsEnabledEnvVar, "false")
	enabled, err := strconv.ParseBool(enabledStr)
	// NB: default to false on error parsing enabled setting
	return err == nil && enabled
}
