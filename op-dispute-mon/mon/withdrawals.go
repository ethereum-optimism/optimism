package mon

import (
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
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
	seenWETH     map[common.Address]bool
}

func NewWithdrawalMonitor(logger log.Logger, clock RClock, metrics WithdrawalMetrics, honestActors types.HonestActors) *WithdrawalMonitor {
	return &WithdrawalMonitor{
		logger:       logger,
		clock:        clock,
		metrics:      metrics,
		honestActors: honestActors,
		seenWETH:     make(map[common.Address]bool),
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
	currentWETH := make(map[common.Address]bool, len(matching))
	for contract, count := range matching {
		currentWETH[contract] = true
		w.metrics.RecordWithdrawalRequests(contract, true, count)
		w.metrics.RecordWithdrawalRequests(contract, false, divergent[contract])
	}
	for contract := range w.seenWETH {
		if !currentWETH[contract] {
			w.metrics.RecordWithdrawalRequests(contract, true, 0)
			w.metrics.RecordWithdrawalRequests(contract, false, 0)
		}
	}
	w.seenWETH = currentWETH
	w.metrics.RecordHonestWithdrawableAmounts(honestWithdrawableAmounts)
}

func (w *WithdrawalMonitor) validateGameWithdrawals(game types.BondedGame, now time.Time, honestWithdrawableAmounts map[common.Address]*big.Int) (int, int) {
	matching := 0
	divergent := 0
	data := game.BondData()
	for recipient := range data.Recipients {
		withdrawal := data.WithdrawalRequests[recipient]
		if withdrawal == nil {
			withdrawal = &contracts.WithdrawalRequest{Amount: new(big.Int), Timestamp: new(big.Int)}
		}
		credit := data.Credits[recipient]
		if credit == nil {
			credit = new(big.Int)
		}
		expected := data.ExpectedCredits[recipient]
		if expected == nil {
			expected = new(big.Int)
		}
		hasObligation := expected.Sign() > 0 || credit.Sign() > 0 || withdrawal.Amount.Sign() > 0
		if !hasObligation {
			continue
		}
		switch data.BondDistributionMode {
		case challengerTypes.LegacyDistributionMode:
			if bigs.Equal(withdrawal.Amount, credit) {
				matching++
			} else {
				divergent++
				w.logger.Error("Withdrawal request amount does not match credit", "game", game.Common().Proxy, "recipient", recipient, "credit", credit, "withdrawal", withdrawal.Amount)
			}
		case challengerTypes.UndecidedDistributionMode:
			// DelayedWETH should not have any withdrawal request yet because the bond distribution mode is undecided
			if !bigs.IsZero(withdrawal.Amount) {
				divergent++
				w.logger.Error("Withdrawal request created before bond distribution mode set", "game", game.Common().Proxy, "recipient", recipient, "withdrawal", withdrawal.Amount)
			}
		case challengerTypes.NormalDistributionMode, challengerTypes.RefundDistributionMode:
			if validDecidedWithdrawal(game, credit, expected, withdrawal, data.WETHDelay, now) {
				matching++
			} else {
				divergent++
				w.logger.Error("Withdrawal request amount does not match expected credit", "game", game.Common().Proxy, "recipient", recipient, "expected", expected, "credit", credit, "withdrawal", withdrawal.Amount)
			}
		default:
			// Treat unknown distribution mode as divergent - better to alert than to ignore.
			divergent++
			w.logger.Error("Unsupported distribution mode", "game", game.Common().Proxy, "recipient", recipient, "mode", data.BondDistributionMode)
		}

		if w.honestActors.Contains(recipient) {
			if data.BondDistributionMode != challengerTypes.UndecidedDistributionMode && bigs.IsZero(withdrawal.Amount) && !bigs.IsZero(credit) {
				w.logger.Warn("Found uninitiated withdrawal", "recipient", recipient, "game", game.Common().Proxy, "amount", credit)
				// Treat credits as withdrawable because the first step of withdrawing can be performed
				total := honestWithdrawableAmounts[recipient]
				total = new(big.Int).Add(total, credit)
				honestWithdrawableAmounts[recipient] = total
			}
			if bigs.IsPositive(withdrawal.Amount) && !time.Unix(withdrawal.Timestamp.Int64(), 0).Add(data.WETHDelay).After(now) {
				// Credits are fully withdrawable
				total := honestWithdrawableAmounts[recipient]
				total = new(big.Int).Add(total, withdrawal.Amount)
				honestWithdrawableAmounts[recipient] = total
				w.logger.Warn("Found unclaimed credit", "recipient", recipient, "game", game.Common().Proxy, "amount", withdrawal.Amount)
			}
		}
	}
	return matching, divergent
}

func validDecidedWithdrawal(
	game types.BondedGame,
	credit *big.Int,
	expected *big.Int,
	withdrawal *contracts.WithdrawalRequest,
	delay time.Duration,
	now time.Time,
) bool {
	creditPresent := bigs.IsZero(withdrawal.Amount) && bigs.IsZero(withdrawal.Timestamp) && bigs.Equal(credit, expected)
	withdrawalCompleted := bigs.IsPositive(withdrawal.Timestamp) && bigs.IsZero(credit) && bigs.IsZero(withdrawal.Amount) &&
		!time.Unix(withdrawal.Timestamp.Int64(), 0).Add(delay).After(now)
	if creditPresent || withdrawalCompleted {
		return true
	}
	if !bigs.IsPositive(withdrawal.Timestamp) || !bigs.Equal(withdrawal.Amount, expected) {
		return false
	}
	switch game.(type) {
	case *types.FaultGameData:
		return bigs.Equal(credit, expected)
	case *types.ZKGameData:
		return bigs.IsZero(credit)
	default:
		return false
	}
}
