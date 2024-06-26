package extract

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/exp/maps"
)

var ErrIncorrectWithdrawalsCount = errors.New("incorrect withdrawals count")

var _ Enricher = (*WithdrawalsEnricher)(nil)

type WithdrawalsEnricher struct{}

func NewWithdrawalsEnricher() *WithdrawalsEnricher {
	return &WithdrawalsEnricher{}
}

func (w *WithdrawalsEnricher) Enrich(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.EnrichedGameData) error {
	recipients := maps.Keys(game.Recipients)
	withdrawals, err := caller.GetWithdrawals(ctx, block, recipients...)
	if err != nil {
		return fmt.Errorf("failed to fetch withdrawals: %w", err)
	}
	if len(withdrawals) != len(recipients) {
		return fmt.Errorf("%w, requested %v values but got %v", ErrIncorrectWithdrawalsCount, len(recipients), len(withdrawals))
	}
	if game.WithdrawalRequests == nil {
		game.WithdrawalRequests = make(map[common.Address]*contracts.WithdrawalRequest)
	}
	for i, recipient := range recipients {
		game.WithdrawalRequests[recipient] = withdrawals[i]
	}
	return nil
}
