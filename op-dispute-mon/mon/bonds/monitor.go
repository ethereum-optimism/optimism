package bonds

import (
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type RClock interface {
	Now() time.Time
}

type BondMetrics interface {
	RecordCredit(expectation metrics.CreditExpectation, count int)
	RecordBondCollateral(addr common.Address, required *big.Int, available *big.Int)
}

type Bonds struct {
	logger  log.Logger
	clock   RClock
	metrics BondMetrics
}

func NewBonds(logger log.Logger, metrics BondMetrics, clock RClock) *Bonds {
	return &Bonds{
		logger:  logger,
		clock:   clock,
		metrics: metrics,
	}
}

func (b *Bonds) CheckBonds(games []types.BondedGame) {
	data := CalculateRequiredCollateral(games)
	for addr, collateral := range data {
		if collateral.Required.Cmp(collateral.Actual) > 0 {
			b.logger.Error("Insufficient collateral", "delayedWETH", addr, "required", collateral.Required, "actual", collateral.Actual)
		}
		b.metrics.RecordBondCollateral(addr, collateral.Required, collateral.Actual)
	}

	b.checkCredits(games)
}

func (b *Bonds) checkCredits(games []types.BondedGame) {
	creditMetrics := make(map[metrics.CreditExpectation]int)

	for _, game := range games {
		data := game.BondData()
		maxDurationReached := !data.CreditWithdrawableAt.After(b.clock.Now())

		allRecipients := make(map[common.Address]bool)
		for address := range data.ExpectedCredits {
			allRecipients[address] = true
		}
		for address := range data.Credits {
			allRecipients[address] = true
		}

		for recipient := range allRecipients {
			actual := data.Credits[recipient]
			if actual == nil {
				actual = big.NewInt(0)
			}
			expected := data.ExpectedCredits[recipient]
			if expected == nil {
				expected = big.NewInt(0)
			}
			comparison := actual.Cmp(expected)
			if maxDurationReached {
				if comparison > 0 {
					creditMetrics[metrics.CreditAboveWithdrawable] += 1
					b.logger.Warn("Credit above expected amount", "recipient", recipient, "expected", expected, "actual", actual, "game", game.Common().Proxy, "withdrawable", "withdrawable")
				} else if comparison == 0 {
					creditMetrics[metrics.CreditEqualWithdrawable] += 1
				} else {
					creditMetrics[metrics.CreditBelowWithdrawable] += 1
				}
			} else {
				if comparison > 0 {
					creditMetrics[metrics.CreditAboveNonWithdrawable] += 1
					b.logger.Warn("Credit above expected amount", "recipient", recipient, "expected", expected, "actual", actual, "game", game.Common().Proxy, "withdrawable", "non_withdrawable")
				} else if comparison == 0 {
					creditMetrics[metrics.CreditEqualNonWithdrawable] += 1
				} else {
					creditMetrics[metrics.CreditBelowNonWithdrawable] += 1
					b.logger.Error("Credit withdrawn early", "recipient", recipient, "expected", expected, "actual", actual, "game", game.Common().Proxy, "withdrawable", "non_withdrawable")
				}
			}
		}
	}

	b.metrics.RecordCredit(metrics.CreditBelowWithdrawable, creditMetrics[metrics.CreditBelowWithdrawable])
	b.metrics.RecordCredit(metrics.CreditEqualWithdrawable, creditMetrics[metrics.CreditEqualWithdrawable])
	b.metrics.RecordCredit(metrics.CreditAboveWithdrawable, creditMetrics[metrics.CreditAboveWithdrawable])

	b.metrics.RecordCredit(metrics.CreditBelowNonWithdrawable, creditMetrics[metrics.CreditBelowNonWithdrawable])
	b.metrics.RecordCredit(metrics.CreditEqualNonWithdrawable, creditMetrics[metrics.CreditEqualNonWithdrawable])
	b.metrics.RecordCredit(metrics.CreditAboveNonWithdrawable, creditMetrics[metrics.CreditAboveNonWithdrawable])
}
