package metrics

import (
	"io"
	"time"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

type NoopMetricsImpl struct {
	txmetrics.NoopTxMetrics
	contractMetrics.NoopMetrics
}

func (i *NoopMetricsImpl) StartBalanceMetrics(l log.Logger, client *ethclient.Client, account common.Address) io.Closer {
	return nil
}

var NoopMetrics Metricer = new(NoopMetricsImpl)

func (*NoopMetricsImpl) RecordInfo(version string) {}
func (*NoopMetricsImpl) RecordUp()                 {}

func (*NoopMetricsImpl) RecordGameMove()        {}
func (*NoopMetricsImpl) RecordGameStep()        {}
func (*NoopMetricsImpl) RecordGameL2Challenge() {}

func (*NoopMetricsImpl) RecordActedL1Block(_ uint64) {}

func (*NoopMetricsImpl) RecordPreimageChallenged()      {}
func (*NoopMetricsImpl) RecordPreimageChallengeFailed() {}
func (*NoopMetricsImpl) RecordLargePreimageCount(_ int) {}

func (*NoopMetricsImpl) RecordBondClaimFailed()   {}
func (*NoopMetricsImpl) RecordBondClaimed(uint64) {}

func (*NoopMetricsImpl) RecordVmExecutionTime(_ string, _ time.Duration) {}
func (*NoopMetricsImpl) RecordVmMemoryUsed(_ string, _ uint64)           {}
func (*NoopMetricsImpl) RecordClaimResolutionTime(t float64)             {}
func (*NoopMetricsImpl) RecordGameActTime(t float64)                     {}

func (*NoopMetricsImpl) RecordGamesStatus(inProgress, defenderWon, challengerWon int) {}

func (*NoopMetricsImpl) RecordGameUpdateScheduled() {}
func (*NoopMetricsImpl) RecordGameUpdateCompleted() {}

func (*NoopMetricsImpl) IncActiveExecutors() {}
func (*NoopMetricsImpl) DecActiveExecutors() {}
func (*NoopMetricsImpl) IncIdleExecutors()   {}
func (*NoopMetricsImpl) DecIdleExecutors()   {}

func (*NoopMetricsImpl) CacheAdd(_ string, _ int, _ bool) {}
func (*NoopMetricsImpl) CacheGet(_ string, _ bool)        {}

func (m *NoopMetricsImpl) VmMetrics(vmType string) *VmMetrics {
	return NewVmMetrics(m, vmType)
}
