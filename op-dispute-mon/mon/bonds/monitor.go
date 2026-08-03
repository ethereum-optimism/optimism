package bonds

import (
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
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
	seenWETH     map[common.Address]bool
}

func NewBonds(logger log.Logger, metrics BondMetrics, clock RClock, honestActors types.HonestActors) *Bonds {
	return &Bonds{
		logger:       logger,
		clock:        clock,
		metrics:      metrics,
		honestActors: honestActors,
		seenWETH:     make(map[common.Address]bool),
	}
}

func (b *Bonds) CheckBonds(games []types.BondedGame) {
	data := CalculateRequiredCollateral(games)
	currentWETH := make(map[common.Address]bool, len(data))
	for addr, collateral := range data {
		currentWETH[addr] = true
		if collateral.BalancesDiffer {
			b.logger.Error("Games returned different balances for shared DelayedWETH", "delayedWETH", addr, "conservativeBalance", collateral.Actual)
		}
		if collateral.Required.Cmp(collateral.Actual) > 0 {
			b.logger.Error("Insufficient collateral", "delayedWETH", addr, "required", collateral.Required, "actual", collateral.Actual)
		}
		b.metrics.RecordBondCollateral(addr, collateral.Required, collateral.Actual)
	}
	for addr := range b.seenWETH {
		if !currentWETH[addr] {
			b.metrics.RecordBondCollateral(addr, new(big.Int), new(big.Int))
		}
	}
	b.seenWETH = currentWETH

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
			if bond.Recipient == bond.Depositor {
				continue
			}
			if b.honestActors.Contains(bond.Depositor) {
				honest[bond.Depositor].Lost.Add(honest[bond.Depositor].Lost, bond.Amount)
				b.logger.Error("Bond resolved against honest actor", "game", game.Common().Proxy, "honestActor", bond.Depositor, "recipient", bond.Recipient, "bondAmount", bond.Amount)
			}
			if !bond.Burned && b.honestActors.Contains(bond.Recipient) {
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

	now := b.clock.Now()
	for _, game := range games {
		data := game.BondData()
		for _, recipient := range data.RecipientAddresses() {
			expected := data.ExpectedCredits[recipient]
			if expected == nil && effectiveCredit(data, recipient).Sign() == 0 {
				continue
			}
			if expected == nil {
				expected = big.NewInt(0)
			}
			actual := observedCredit(data, recipient, expected, now)
			withdrawable := creditCanBeWithdrawn(game, recipient, now)
			comparison := actual.Cmp(expected)
			if withdrawable {
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

func observedCredit(data *types.BondGameData, recipient common.Address, expected *big.Int, now time.Time) *big.Int {
	actual := effectiveCredit(data, recipient)
	request := data.WithdrawalRequests[recipient]
	if actual.Sign() == 0 && withdrawalMature(request, data.WETHDelay, now) {
		// DelayedWETH retains the unlock timestamp after the full amount has been withdrawn.
		// A zero credit and zero request amount with a mature timestamp therefore means the payout
		// completed. DelayedWETH does not retain the paid amount, so a partial payout cannot be
		// distinguished from a full payout after maturity and is represented as the expected amount.
		return expected
	}
	return actual
}

func creditCanBeWithdrawn(game types.BondedGame, recipient common.Address, now time.Time) bool {
	data := game.BondData()
	request := data.WithdrawalRequests[recipient]
	if request != nil && request.Timestamp != nil && request.Timestamp.Sign() > 0 {
		return withdrawalMature(request, data.WETHDelay, now)
	}
	return game.RequiresCreditForWithdrawal() &&
		!data.CreditWithdrawableAt.IsZero() && !data.CreditWithdrawableAt.After(now)
}

func withdrawalMature(request *contracts.WithdrawalRequest, delay time.Duration, now time.Time) bool {
	return request != nil && request.Timestamp != nil && request.Timestamp.Sign() > 0 &&
		!time.Unix(request.Timestamp.Int64(), 0).Add(delay).After(now)
}
