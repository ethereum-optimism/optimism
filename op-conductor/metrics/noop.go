package metrics

import opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"

type NoopMetricsImpl struct {
	opmetrics.NoopRPCMetrics
}

var NoopMetrics Metricer = new(NoopMetricsImpl)

func (*NoopMetricsImpl) RecordInfo(version string)                                       {}
func (*NoopMetricsImpl) RecordUp()                                                       {}
func (*NoopMetricsImpl) RecordStateChange(leader bool, healthy bool, active bool)        {}
func (*NoopMetricsImpl) RecordLeaderTransfer(success bool)                               {}
func (*NoopMetricsImpl) RecordStartSequencer(success bool)                               {}
func (*NoopMetricsImpl) RecordStopSequencer(success bool)                                {}
func (*NoopMetricsImpl) RecordHealthCheck(success bool, err error)                       {}
func (*NoopMetricsImpl) RecordLoopExecutionTime(duration float64)                        {}
func (*NoopMetricsImpl) RecordRollupBoostConnectionAttempts(success bool, source string) {}
func (*NoopMetricsImpl) RecordWebSocketClientCount(count int)                            {}
func (*NoopMetricsImpl) RecordHealthCheckConfig(interval, unsafeInterval, safeInterval, minPeerCount, interopReorgLeniencyWindowSize uint64, safeEnabled, interopReorgLeniency bool) {
}
func (*NoopMetricsImpl) RecordHealthCheckHeads(unsafeNumber, unsafeTimestamp, safeNumber, safeTimestamp, unsafeLag, safeLag uint64) {
}
func (*NoopMetricsImpl) RecordHealthCheckPeerCount(peerCount, minPeerCount uint64) {}
func (*NoopMetricsImpl) RecordHealthCheckWindow(check HealthCheck, state HealthCheckWindowState, successes, failures, windowSize uint64) {
}
func (*NoopMetricsImpl) RecordHealthCheckStatus(check HealthCheck, status HealthCheckStatus) {}
func (*NoopMetricsImpl) RecordHealthCheckFailure(check HealthCheck, reason HealthCheckFailureReason) {
}
func (*NoopMetricsImpl) RecordUnsafeHeadRecovery(active bool, currentLag, initialLag, windowStartLag, wallElapsed, unsafeElapsed, polls, pollsInWindow uint64) {
}
func (*NoopMetricsImpl) RecordUnsafeHeadRecoveryEvent(event HealthCheckRecoveryEvent) {}
