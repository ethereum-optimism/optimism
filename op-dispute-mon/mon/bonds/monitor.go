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
	RecordHonestActorBonds(address common.Address, data *metrics.HonestActorBondData)
}

type Bonds struct {
	logger       log.Logger
	clock        RClock
	metrics      BondMetrics
	honestActors types.HonestActors
}

func NewBonds(logger log.Logger, metrics BondMetrics, clock RClock, honestActors types.HonestActors) *Bonds {
	return &Bonds{
		logger:       logger,
		clock:        clock,
		metrics:      metrics,
		honestActors: honestActors,
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
	b.checkHonestActorBonds(games)
}

func (b *Bonds) checkHonestActorBonds(games []types.BondedGame) {
	honest := make(map[common.Address]*metrics.HonestActorBondData, len(b.honestActors))
	for actor := range b.honestActors {
		honest[actor] = &metrics.HonestActorBondData{
			Pending: new(big.Int),
			Lost:    new(big.Int),
			Won:     new(big.Int),
		}
	}
	for _, game := range games {
		for _, bond := range game.BondData().Bonds {
			if !bond.Resolved {
				if b.honestActors.Contains(bond.Depositor) {
					honest[bond.Depositor].Pending.Add(honest[bond.Depositor].Pending, bond.Amount)
				}
				continue
			}
			if !bond.Forfeited {
				continue
			}
			if b.honestActors.Contains(bond.Depositor) {
				honest[bond.Depositor].Lost.Add(honest[bond.Depositor].Lost, bond.Amount)
			}
			if b.honestActors.Contains(bond.Recipient) {
				honest[bond.Recipient].Won.Add(honest[bond.Recipient].Won, bond.Amount)
			}
		}
	}
	for actor, data := range honest {
		b.metrics.RecordHonestActorBonds(actor, data)
	}
}

func (b *Bonds) checkCredits(games []types.BondedGame) {
	creditMetrics := make(map[metrics.CreditExpectation]int)

	for _, game := range games {
		data := game.BondData()
		now := b.clock.Now()
		_, isZK := game.(*types.ZKGameData)

		allRecipients := make(map[common.Address]bool)
		for address := range data.ExpectedCredits {
			allRecipients[address] = true
		}
		for address := range data.Credits {
			allRecipients[address] = true
		}
		if isZK {
			for address := range data.WithdrawalRequests {
				allRecipients[address] = true
			}
		}

		for recipient := range allRecipients {
			actual := data.Credits[recipient]
			if actual == nil {
				actual = big.NewInt(0)
			}
			maxDurationReached := !data.CreditWithdrawableAt.After(now)
			completedPayout := false
			if isZK {
				request := data.WithdrawalRequests[recipient]
				if request != nil && request.Amount != nil && request.Amount.Cmp(actual) > 0 {
					actual = request.Amount
				}
				maxDurationReached = request != nil && request.Timestamp != nil && request.Timestamp.Sign() > 0 &&
					!now.Before(time.Unix(request.Timestamp.Int64(), 0).Add(data.WETHDelay))
				completedPayout = request != nil && request.Amount != nil && request.Amount.Sign() == 0 && actual.Sign() == 0 && maxDurationReached
			}
			expected := data.ExpectedCredits[recipient]
			if completedPayout && expected != nil && expected.Sign() > 0 {
				actual = expected
			}
			if isZK && expected == nil && actual.Sign() == 0 {
				continue
			}
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
