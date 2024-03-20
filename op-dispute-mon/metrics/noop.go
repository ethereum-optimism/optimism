package metrics

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type NoopMetricsImpl struct{}

var NoopMetrics Metricer = new(NoopMetricsImpl)

func (*NoopMetricsImpl) RecordInfo(version string) {}
func (*NoopMetricsImpl) RecordUp()                 {}

func (*NoopMetricsImpl) CacheAdd(_ string, _ int, _ bool) {}
func (*NoopMetricsImpl) CacheGet(_ string, _ bool)        {}

func (*NoopMetricsImpl) RecordClaimResolutionDelayMax(delay float64) {}

func (*NoopMetricsImpl) RecordOutputFetchTime(timestamp float64) {}

func (*NoopMetricsImpl) RecordGameAgreement(status GameAgreementStatus, count int) {}

func (i *NoopMetricsImpl) RecordBondCollateral(_ common.Address, _ *big.Int, _ *big.Int) {}
