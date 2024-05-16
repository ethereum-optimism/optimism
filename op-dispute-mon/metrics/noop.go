package metrics

import (
	"math/big"
	"time"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum/go-ethereum/common"
)

type NoopMetricsImpl struct {
	contractMetrics.NoopMetrics
}

var NoopMetrics Metricer = new(NoopMetricsImpl)

func (*NoopMetricsImpl) RecordInfo(_ string) {}
func (*NoopMetricsImpl) RecordUp()           {}

func (*NoopMetricsImpl) RecordMonitorDuration(_ time.Duration) {}

func (*NoopMetricsImpl) CacheAdd(_ string, _ int, _ bool) {}
func (*NoopMetricsImpl) CacheGet(_ string, _ bool)        {}

func (*NoopMetricsImpl) RecordHonestActorClaims(_ common.Address, _ *HonestActorData) {}

func (*NoopMetricsImpl) RecordGameResolutionStatus(_ ResolutionStatus, _ int) {}

func (*NoopMetricsImpl) RecordCredit(_ CreditExpectation, _ int) {}

func (*NoopMetricsImpl) RecordClaims(_ ClaimStatus, _ int) {}

func (*NoopMetricsImpl) RecordWithdrawalRequests(_ common.Address, _ bool, _ int) {}

func (*NoopMetricsImpl) RecordOutputFetchTime(_ float64) {}

func (*NoopMetricsImpl) RecordGameAgreement(_ GameAgreementStatus, _ int) {}

func (*NoopMetricsImpl) RecordLatestInvalidProposal(_ uint64) {}

func (*NoopMetricsImpl) RecordIgnoredGames(_ int) {}

func (*NoopMetricsImpl) RecordFailedGames(_ int) {}

func (*NoopMetricsImpl) RecordBondCollateral(_ common.Address, _ *big.Int, _ *big.Int) {}

func (*NoopMetricsImpl) RecordL2Challenges(_ bool, _ int) {}
