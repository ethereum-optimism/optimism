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
		b.metrics.RecordBondCollateral(addr, collateral.Required, collateral.Actual)
	}

	for _, game := range games {
		b.checkCredits(game)
	}
}

func (b *Bonds) checkCredits(game *types.EnrichedGameData) {
	// Check if the max duration has been reached for this game
	duration := uint64(b.clock.Now().Unix()) - game.Timestamp
	maxDurationReached := duration >= game.MaxClockDuration*2

	// Iterate over claims and filter out resolved ones
	recipients := make(map[int]common.Address)
	for i, claim := range game.Claims {
		// Skip unresolved claims since these bonds will not appear in the credits.
		if !claim.Resolved {
			continue
		}
		// The recipient of a resolved claim is the claimant unless it's been countered.
		recipient := claim.Claimant
		if claim.CounteredBy != (common.Address{}) {
			recipient = claim.CounteredBy
		}
		recipients[i] = recipient
	}

	creditMetrics := make(map[metrics.CreditExpectation]int)
	for i, recipient := range recipients {
		expected := game.Credits[recipient]
		comparison := expected.Cmp(game.RequiredBonds[i])
		if maxDurationReached {
			if comparison > 0 {
				creditMetrics[metrics.CreditBelowMaxDuration] += 1
			} else if comparison == 0 {
				creditMetrics[metrics.CreditEqualMaxDuration] += 1
			} else {
				creditMetrics[metrics.CreditAboveMaxDuration] += 1
				b.logger.Warn("credit above expected amount", "recipient", recipient, "expected", expected, "gameAddr", game.Proxy, "duration", "max_duration")
			}
		} else {
			if comparison > 0 {
				creditMetrics[metrics.CreditBelowNonMaxDuration] += 1
			} else if comparison == 0 {
				creditMetrics[metrics.CreditEqualNonMaxDuration] += 1
			} else {
				creditMetrics[metrics.CreditAboveNonMaxDuration] += 1
				b.logger.Warn("credit above expected amount", "recipient", recipient, "expected", expected, "gameAddr", game.Proxy, "duration", "non_max_duration")
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
