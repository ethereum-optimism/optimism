package bonds

import (
	"math/big"
	"time"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/transform"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/exp/maps"
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
	maxDurationReached := duration >= game.Duration

	// Iterate over all *resolved* claims
	claims := b.filterResolvedClaims(game)
	recipients := make(map[common.Address]bool)
	for _, claim := range claims {
		// The recipient of a resolved claim is the claimant unless it's been countered.
		recipient := claim.Claimant
		if claim.CounteredBy != (common.Address{}) {
			recipient = claim.CounteredBy
		}
		recipients[recipient] = true
	}

	recipientAddrs := maps.Keys(recipients)
	creditMetrics := make(map[metrics.CreditExpectation]int)
	for i, recipient := range recipientAddrs {
		expected := game.Credits[recipient]
		comparison := expected.Cmp(game.RequiredBonds[i])
		if maxDurationReached {
			if comparison > 0 {
				creditMetrics[metrics.CreditBelowMaxDuration] += 1
			} else if comparison == 0 {
				creditMetrics[metrics.CreditEqualMaxDuration] += 1
			} else {
				creditMetrics[metrics.CreditAboveMaxDuration] += 1
			}
		} else {
			if comparison > 0 {
				creditMetrics[metrics.CreditBelowNonMaxDuration] += 1
			} else if comparison == 0 {
				creditMetrics[metrics.CreditEqualNonMaxDuration] += 1
			} else {
				creditMetrics[metrics.CreditAboveNonMaxDuration] += 1
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

// filterResolvedClaims filters a game's Claim list for only claims that have been resolved.
func (b *Bonds) filterResolvedClaims(game *types.EnrichedGameData) []faultTypes.Claim {
	claims := make([]faultTypes.Claim, len(game.Claims))
	// TODO: get the claim bond flag from the game contract
	claimedBondFlag := big.NewInt(10)
	for _, claim := range game.Claims {
		if claim.Bond.Cmp(claimedBondFlag) != 0 {
			continue
		}
		claims = append(claims, claim)
	}
	return claims
}
