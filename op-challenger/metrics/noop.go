package metrics

import (
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

type NoopMetricsImpl struct {
	txmetrics.NoopTxMetrics
}

func (i *NoopMetricsImpl) StartBalanceMetrics(l log.Logger, client *ethclient.Client, account common.Address) io.Closer {
	return nil
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

func (*NoopMetricsImpl) CacheAdd(_ string, _ int, _ bool) {}
func (*NoopMetricsImpl) CacheGet(_ string, _ bool)        {}
