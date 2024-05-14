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
		maxDurationReached := duration >= game.MaxClockDuration*2

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
					creditMetrics[metrics.CreditAboveMaxDuration] += 1
					b.logger.Warn("Credit above expected amount", "recipient", recipient, "expected", expected, "actual", actual, "gameAddr", game.Proxy, "duration", "reached")
				} else if comparison == 0 {
					creditMetrics[metrics.CreditEqualMaxDuration] += 1
				} else {
					creditMetrics[metrics.CreditBelowMaxDuration] += 1
				}
			} else {
				if comparison > 0 {
					creditMetrics[metrics.CreditAboveNonMaxDuration] += 1
					b.logger.Warn("Credit above expected amount", "recipient", recipient, "expected", expected, "actual", actual, "gameAddr", game.Proxy, "duration", "unreached")
				} else if comparison == 0 {
					creditMetrics[metrics.CreditEqualNonMaxDuration] += 1
				} else {
					creditMetrics[metrics.CreditBelowNonMaxDuration] += 1
					b.logger.Warn("Credit withdrawn early", "recipient", recipient, "expected", expected, "actual", actual, "gameAddr", game.Proxy, "duration", "unreached")
				}
			}
		}
	}

	b.metrics.RecordCredit(metrics.CreditBelowMaxDuration, creditMetrics[metrics.CreditBelowMaxDuration])
	b.metrics.RecordCredit(metrics.CreditEqualMaxDuration, creditMetrics[metrics.CreditEqualMaxDuration])
	b.metrics.RecordCredit(metrics.CreditAboveMaxDuration, creditMetrics[metrics.CreditAboveMaxDuration])

	b.metrics.RecordCredit(metrics.CreditBelowNonMaxDuration, creditMetrics[metrics.CreditBelowNonMaxDuration])
	b.metrics.RecordCredit(metrics.CreditEqualNonMaxDuration, creditMetrics[metrics.CreditEqualNonMaxDuration])
	b.metrics.RecordCredit(metrics.CreditAboveNonMaxDuration, creditMetrics[metrics.CreditAboveNonMaxDuration])
}
