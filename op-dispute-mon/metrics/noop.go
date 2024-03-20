package metrics

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type NoopMetricsImpl struct{}

var NoopMetrics Metricer = new(NoopMetricsImpl)

func (*NoopMetricsImpl) RecordInfo(_ string) {}
func (*NoopMetricsImpl) RecordUp()           {}

func (*NoopMetricsImpl) CacheAdd(_ string, _ int, _ bool) {}
func (*NoopMetricsImpl) CacheGet(_ string, _ bool)        {}

func (*NoopMetricsImpl) RecordCredit(_ CreditExpectation, _ int) {}

func (*NoopMetricsImpl) RecordClaimResolutionDelayMax(_ float64) {}

func (*NoopMetricsImpl) RecordOutputFetchTime(_ float64) {}

func (*NoopMetricsImpl) RecordGameAgreement(_ GameAgreementStatus, _ int) {}

func (i *NoopMetricsImpl) RecordBondCollateral(_ common.Address, _ *big.Int, _ *big.Int) {}
