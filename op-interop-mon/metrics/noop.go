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
func (*noopMetrics) RecordInitiatingMessageStats(chainID string, blockHeight uint64, status string, value float64) {
}
func (*noopMetrics) RecordExecutingMessageStats(chainID string, blockHeight uint64, blockHash string, status string, value float64) {
}
func (*noopMetrics) RecordTerminalStatusChange(executingChainID string, initiatingChainID string, value float64) {
}
