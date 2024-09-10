package mon

import (
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
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

func (w *WithdrawalMonitor) CheckWithdrawals(games []*types.EnrichedGameData) {
	now := w.clock.Now() // Use a consistent time for all checks
	matching := make(map[common.Address]int)
	divergent := make(map[common.Address]int)
	honestWithdrawableAmounts := make(map[common.Address]*big.Int)
	for address := range w.honestActors {
		honestWithdrawableAmounts[address] = big.NewInt(0)
	}
	for _, game := range games {
		matches, diverges := w.validateGameWithdrawals(game, now, honestWithdrawableAmounts)
		matching[game.WETHContract] += matches
		divergent[game.WETHContract] += diverges
	}
	for contract, count := range matching {
		w.metrics.RecordWithdrawalRequests(contract, true, count)
	}
	for contract, count := range divergent {
		w.metrics.RecordWithdrawalRequests(contract, false, count)
	}
	w.metrics.RecordHonestWithdrawableAmounts(honestWithdrawableAmounts)
}

func (w *WithdrawalMonitor) validateGameWithdrawals(game *types.EnrichedGameData, now time.Time, honestWithdrawableAmounts map[common.Address]*big.Int) (int, int) {
	matching := 0
	divergent := 0
	for recipient, withdrawalAmount := range game.WithdrawalRequests {
		if withdrawalAmount.Amount != nil && withdrawalAmount.Amount.Cmp(game.Credits[recipient]) == 0 {
			matching++
		} else {
			divergent++
			w.logger.Error("Withdrawal request amount does not match credit", "game", game.Proxy, "recipient", recipient, "credit", game.Credits[recipient], "withdrawal", game.WithdrawalRequests[recipient].Amount)
		}

		if withdrawalAmount.Amount.Cmp(big.NewInt(0)) > 0 && w.honestActors.Contains(recipient) {
			if time.Unix(withdrawalAmount.Timestamp.Int64(), 0).Add(game.WETHDelay).Before(now) {
				// Credits are withdrawable
				total := honestWithdrawableAmounts[recipient]
				total = new(big.Int).Add(total, withdrawalAmount.Amount)
				honestWithdrawableAmounts[recipient] = total
				w.logger.Warn("Found unclaimed credit", "recipient", recipient, "game", game.Proxy, "amount", withdrawalAmount.Amount)
			}
		}
	}
	return matching, divergent
}
