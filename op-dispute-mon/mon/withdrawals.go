package mon

import (
	"math/big"
	"time"

	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type WithdrawalMetrics interface {
	RecordWithdrawalRequests(delayedWeth common.Address, matches bool, count int)
	RecordHonestWithdrawableAmounts(map[common.Address]*big.Int)
}

type WithdrawalMonitor struct {
	logger       log.Logger
	clock        RClock
	metrics      WithdrawalMetrics
	honestActors types.HonestActors
}

func NewWithdrawalMonitor(logger log.Logger, clock RClock, metrics WithdrawalMetrics, honestActors types.HonestActors) *WithdrawalMonitor {
	return &WithdrawalMonitor{
		logger:       logger,
		clock:        clock,
		metrics:      metrics,
		honestActors: honestActors,
	}
}

func (w *WithdrawalMonitor) CheckWithdrawals(games []types.BondedGame) {
	now := w.clock.Now() // Use a consistent time for all checks
	matching := make(map[common.Address]int)
	divergent := make(map[common.Address]int)
	honestWithdrawableAmounts := make(map[common.Address]*big.Int)
	for address := range w.honestActors {
		honestWithdrawableAmounts[address] = big.NewInt(0)
	}
	for _, game := range games {
		matches, diverges := w.validateGameWithdrawals(game, now, honestWithdrawableAmounts)
		matching[game.BondData().WETHContract] += matches
		divergent[game.BondData().WETHContract] += diverges
	}
	for contract, count := range matching {
		w.metrics.RecordWithdrawalRequests(contract, true, count)
	}
	for contract, count := range divergent {
		w.metrics.RecordWithdrawalRequests(contract, false, count)
	}
	w.metrics.RecordHonestWithdrawableAmounts(honestWithdrawableAmounts)
}

func (w *WithdrawalMonitor) validateGameWithdrawals(game types.BondedGame, now time.Time, honestWithdrawableAmounts map[common.Address]*big.Int) (int, int) {
	if _, ok := game.(*types.ZKGameData); ok {
		return w.validateZKGameWithdrawals(game, now, honestWithdrawableAmounts)
	}
	matching := 0
	divergent := 0
	data := game.BondData()
	for recipient, withdrawalAmount := range data.WithdrawalRequests {
		switch data.BondDistributionMode {
		case challengerTypes.LegacyDistributionMode:
			if bigs.Equal(withdrawalAmount.Amount, data.Credits[recipient]) {
				matching++
			} else {
				divergent++
				w.logger.Error("Withdrawal request amount does not match credit", "game", game.Common().Proxy, "recipient", recipient, "credit", data.Credits[recipient], "withdrawal", data.WithdrawalRequests[recipient].Amount)
			}
		case challengerTypes.UndecidedDistributionMode:
			// DelayedWETH should not have any withdrawal request yet because the bond distribution mode is undecided
			if !bigs.IsZero(withdrawalAmount.Amount) {
				divergent++
				w.logger.Error("Withdrawal request created before bond distribution mode set", "game", game.Common().Proxy, "recipient", recipient, "withdrawal", data.WithdrawalRequests[recipient].Amount)
			}
		case challengerTypes.NormalDistributionMode, challengerTypes.RefundDistributionMode:
			// The withdrawal request is only created on the first claim to claimCredit, so it may not have been set.
			// If it has been set, it should match the game credit amount.
			if bigs.IsZero(withdrawalAmount.Amount) || bigs.Equal(withdrawalAmount.Amount, data.Credits[recipient]) {
				matching++
			} else {
				divergent++
				w.logger.Error("Withdrawal request amount does not match credit", "game", game.Common().Proxy, "recipient", recipient, "credit", data.Credits[recipient], "withdrawal", data.WithdrawalRequests[recipient].Amount)
			}
		default:
			// Treat unknown distribution mode as divergent - better to alert than to ignore.
			divergent++
			w.logger.Error("Unsupported distribution mode", "game", game.Common().Proxy, "recipient", recipient, "mode", data.BondDistributionMode)
		}

		if w.honestActors.Contains(recipient) {
			if data.BondDistributionMode != challengerTypes.UndecidedDistributionMode && bigs.IsZero(withdrawalAmount.Amount) && !bigs.IsZero(data.Credits[recipient]) {
				w.logger.Warn("Found uninitiated withdrawal", "recipient", recipient, "game", game.Common().Proxy, "amount", data.Credits[recipient])
				// Treat credits as withdrawable because the first step of withdrawing can be performed
				total := honestWithdrawableAmounts[recipient]
				total = new(big.Int).Add(total, data.Credits[recipient])
				honestWithdrawableAmounts[recipient] = total
			}
			if bigs.IsPositive(withdrawalAmount.Amount) && time.Unix(withdrawalAmount.Timestamp.Int64(), 0).Add(data.WETHDelay).Before(now) {
				// Credits are fully withdrawable
				total := honestWithdrawableAmounts[recipient]
				total = new(big.Int).Add(total, withdrawalAmount.Amount)
				honestWithdrawableAmounts[recipient] = total
				w.logger.Warn("Found unclaimed credit", "recipient", recipient, "game", game.Common().Proxy, "amount", withdrawalAmount.Amount)
			}
		}
	}
	return matching, divergent
}

func (w *WithdrawalMonitor) validateZKGameWithdrawals(game types.BondedGame, now time.Time, honestWithdrawableAmounts map[common.Address]*big.Int) (int, int) {
	matching := 0
	divergent := 0
	data := game.BondData()
	for recipient, request := range data.WithdrawalRequests {
		credit := data.Credits[recipient]
		if credit == nil {
			credit = new(big.Int)
		}
		expected := data.ExpectedCredits[recipient]
		if expected == nil {
			expected = new(big.Int)
		}
		if expected.Sign() == 0 && credit.Sign() == 0 && request.Amount.Sign() == 0 && request.Timestamp.Sign() == 0 {
			continue
		}
		mature := request.Timestamp.Sign() > 0 && !now.Before(time.Unix(request.Timestamp.Int64(), 0).Add(data.WETHDelay))
		switch data.BondDistributionMode {
		case challengerTypes.UndecidedDistributionMode:
			if request.Amount.Sign() > 0 || request.Timestamp.Sign() > 0 {
				divergent++
				w.logger.Error("Withdrawal request created before bond distribution mode set", "game", game.Common().Proxy, "recipient", recipient, "withdrawal", request.Amount)
			}
		case challengerTypes.NormalDistributionMode, challengerTypes.RefundDistributionMode:
			observed := credit
			if request.Amount.Cmp(observed) > 0 {
				observed = request.Amount
			}
			uninitiated := request.Amount.Sign() == 0 && request.Timestamp.Sign() == 0 && credit.Cmp(expected) == 0
			completed := request.Amount.Sign() == 0 && credit.Sign() == 0 && mature
			pending := request.Amount.Sign() > 0 && request.Timestamp.Sign() > 0 && credit.Sign() == 0 && observed.Cmp(expected) == 0
			if uninitiated || completed || pending {
				matching++
			} else {
				divergent++
				w.logger.Error("ZK withdrawal state does not match expected credit", "game", game.Common().Proxy, "recipient", recipient, "expected", expected, "credit", credit, "withdrawal", request.Amount)
			}
		default:
			divergent++
			w.logger.Error("Unsupported distribution mode", "game", game.Common().Proxy, "recipient", recipient, "mode", data.BondDistributionMode)
		}

		if w.honestActors.Contains(recipient) && request.Amount.Sign() > 0 && mature {
			total := honestWithdrawableAmounts[recipient]
			honestWithdrawableAmounts[recipient] = new(big.Int).Add(total, request.Amount)
			w.logger.Warn("Found unclaimed credit", "recipient", recipient, "game", game.Common().Proxy, "amount", request.Amount)
		}
	}
	return matching, divergent
}
