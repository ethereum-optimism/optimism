package metrics

import (
	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

type noopMetrics struct {
	txmetrics.NoopTxMetrics
}

var NoopMetrics Metricer = new(noopMetrics)

func (*noopMetrics) RecordInfo(version string) {}
func (*noopMetrics) RecordUp()                 {}
