package sysgo

import "strconv"

const SysgoMetricsEnabledEnvVar = "SYSGO_METRICS_ENABLED"

func AreMetricsEnabled() bool {
	enabledStr := GetEnvVarOrDefault(SysgoMetricsEnabledEnvVar, "false")
	enabled, err := strconv.ParseBool(enabledStr)
	// NB: default to false on error parsing enabled setting
	return err == nil && enabled
}
