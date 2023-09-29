package metrics

import (
	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

type NoopMetricsImpl struct {
	txmetrics.NoopTxMetrics
}

var NoopMetrics Metricer = new(NoopMetricsImpl)

func (*NoopMetricsImpl) RecordInfo(version string) {}
func (*NoopMetricsImpl) RecordUp()                 {}

func (*NoopMetricsImpl) RecordGameMove() {}
func (*NoopMetricsImpl) RecordGameStep() {}

func (*NoopMetricsImpl) RecordCannonExecutionTime(t float64) {}

func (*NoopMetricsImpl) RecordGamesStatus(inProgress, defenderWon, challengerWon int) {}

func (*NoopMetricsImpl) RecordGameUpdateScheduled() {}
func (*NoopMetricsImpl) RecordGameUpdateCompleted() {}

func (*NoopMetricsImpl) IncActiveExecutors() {}
func (*NoopMetricsImpl) DecActiveExecutors() {}
func (*NoopMetricsImpl) IncIdleExecutors()   {}
func (*NoopMetricsImpl) DecIdleExecutors()   {}
