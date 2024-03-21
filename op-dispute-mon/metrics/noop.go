package metrics

import (
	"math/big"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum/go-ethereum/common"
)

type NoopMetricsImpl struct {
	contractMetrics.NoopMetrics
}

var NoopMetrics Metricer = new(NoopMetricsImpl)

func (*NoopMetricsImpl) RecordInfo(version string) {}
func (*NoopMetricsImpl) RecordUp()                 {}

func (*NoopMetricsImpl) CacheAdd(_ string, _ int, _ bool) {}
func (*NoopMetricsImpl) CacheGet(_ string, _ bool)        {}

func (*NoopMetricsImpl) RecordClaimResolutionDelayMax(delay float64) {}

func (*NoopMetricsImpl) RecordOutputFetchTime(timestamp float64) {}

func (*NoopMetricsImpl) RecordGameAgreement(status GameAgreementStatus, count int) {}

func (i *NoopMetricsImpl) RecordBondCollateral(_ common.Address, _ *big.Int, _ *big.Int) {}
