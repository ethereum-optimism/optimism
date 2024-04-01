package mon

import (
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type WithdrawalMetrics interface {
	RecordWithdrawalRequests(delayedWeth common.Address, matches bool, count int)
}

type WithdrawalMonitor struct {
	logger  log.Logger
	metrics WithdrawalMetrics
}

func NewWithdrawalMonitor(logger log.Logger, metrics WithdrawalMetrics) *WithdrawalMonitor {
	return &WithdrawalMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

func (w *WithdrawalMonitor) CheckWithdrawals(games []*types.EnrichedGameData) {
	matching := make(map[common.Address]int)
	divergent := make(map[common.Address]int)
	for _, game := range games {
		matches, diverges := w.validateGameWithdrawals(game)
		matching[game.WETHContract] += matches
		divergent[game.WETHContract] += diverges
	}
	for contract, count := range matching {
		w.metrics.RecordWithdrawalRequests(contract, true, count)
	}
	for contract, count := range divergent {
		w.metrics.RecordWithdrawalRequests(contract, false, count)
	}
}

func (w *WithdrawalMonitor) validateGameWithdrawals(game *types.EnrichedGameData) (int, int) {
	matching := 0
	divergent := 0
	for recipient, withdrawalAmount := range game.WithdrawalRequests {
		if withdrawalAmount.Amount != nil && withdrawalAmount.Amount.Cmp(game.Credits[recipient]) == 0 {
			matching++
		} else {
			divergent++
			w.logger.Error("Withdrawal request amount does not match credit", "game", game.Proxy, "recipient", recipient, "credit", game.Credits[recipient], "withdrawal", game.WithdrawalRequests[recipient].Amount)
		}
	}
	return matching, divergent
}
