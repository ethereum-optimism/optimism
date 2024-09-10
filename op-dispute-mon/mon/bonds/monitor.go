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

func (b *Bonds) CheckBonds(games []*types.EnrichedGameData) {
	data := CalculateRequiredCollateral(games)
	for addr, collateral := range data {
		if collateral.Required.Cmp(collateral.Actual) > 0 {
			b.logger.Error("Insufficient collateral", "delayedWETH", addr, "required", collateral.Required, "actual", collateral.Actual)
		}
		b.metrics.RecordBondCollateral(addr, collateral.Required, collateral.Actual)
	}

	b.checkCredits(games)
}

func (b *Bonds) checkCredits(games []*types.EnrichedGameData) {
	creditMetrics := make(map[metrics.CreditExpectation]int)

	for _, game := range games {
		// Check if the max duration has been reached for this game
		duration := uint64(b.clock.Now().Unix()) - game.Timestamp
		maxDurationReached := duration >= game.MaxClockDuration+uint64(game.WETHDelay.Seconds())

		// Iterate over claims, filter out resolved ones and sum up expected credits per recipient
		expectedCredits := make(map[common.Address]*big.Int)
		for _, claim := range game.Claims {
			// Skip unresolved claims since these bonds will not appear in the credits.
			if !claim.Resolved {
				continue
			}
			// The recipient of a resolved claim is the claimant unless it's been countered.
			recipient := claim.Claimant
			if claim.IsRoot() && game.BlockNumberChallenged {
				// The bond for the root claim is paid to the block number challenger if present
				recipient = game.BlockNumberChallenger
			} else if claim.CounteredBy != (common.Address{}) {
				recipient = claim.CounteredBy
			}
			current := expectedCredits[recipient]
			if current == nil {
				current = big.NewInt(0)
			}
			expectedCredits[recipient] = new(big.Int).Add(current, claim.Bond)
		}

		allRecipients := make(map[common.Address]bool)
		for address := range expectedCredits {
			allRecipients[address] = true
		}
		for address := range game.Credits {
			allRecipients[address] = true
		}

		for recipient := range allRecipients {
			actual := game.Credits[recipient]
			if actual == nil {
				actual = big.NewInt(0)
			}
			expected := expectedCredits[recipient]
			if expected == nil {
				expected = big.NewInt(0)
			}
			comparison := actual.Cmp(expected)
			if maxDurationReached {
				if comparison > 0 {
					creditMetrics[metrics.CreditAboveWithdrawable] += 1
					b.logger.Warn("Credit above expected amount", "recipient", recipient, "expected", expected, "actual", actual, "game", game.Proxy, "withdrawable", "withdrawable")
				} else if comparison == 0 {
					creditMetrics[metrics.CreditEqualWithdrawable] += 1
				} else {
					creditMetrics[metrics.CreditBelowWithdrawable] += 1
				}
			} else {
				if comparison > 0 {
					creditMetrics[metrics.CreditAboveNonWithdrawable] += 1
					b.logger.Warn("Credit above expected amount", "recipient", recipient, "expected", expected, "actual", actual, "game", game.Proxy, "withdrawable", "non_withdrawable")
				} else if comparison == 0 {
					creditMetrics[metrics.CreditEqualNonWithdrawable] += 1
				} else {
					creditMetrics[metrics.CreditBelowNonWithdrawable] += 1
					b.logger.Error("Credit withdrawn early", "recipient", recipient, "expected", expected, "actual", actual, "game", game.Proxy, "withdrawable", "non_withdrawable")
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
