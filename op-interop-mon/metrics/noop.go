package metrics

import (
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

type noopMetrics struct {
	opmetrics.NoopRefMetrics
	opmetrics.NoopRPCMetrics
}

var NoopMetrics Metricer = new(noopMetrics)

func (*noopMetrics) RecordInfo(version string) {}
func (*noopMetrics) RecordUp()                 {}
func (*noopMetrics) RecordMessageStatus(executingChainID string, initiatingChainID string, status string, value float64) {
}
func (*noopMetrics) RecordTerminalStatusChange(executingChainID string, initiatingChainID string, value float64) {
}
func (*noopMetrics) RecordExecutingBlockRange(chainID string, min uint64, max uint64)        {}
func (*noopMetrics) RecordInitiatingBlockRange(chainID string, min uint64, max uint64)       {}
func (*noopMetrics) RecordInitiatingReorg(executingChainID string, initiatingChainID string) {}
func (*noopMetrics) RecordFilterDivergence(executingChainID string, initiatingChainID string, monitorStatus string, filterStatus string) {
}
func (*noopMetrics) RecordFilterFailsafe(enabled bool)                               {}
func (*noopMetrics) RecordSupernodeUp(endpoint string, up bool)                      {}
func (*noopMetrics) RecordSupernodeSafeHead(chainID string, level string, bn uint64) {}
func (*noopMetrics) RecordCrossSafetyViolation(executingChainID string, initiatingChainID string) {
}
