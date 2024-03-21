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

func (w *WithdrawalsEnricher) Enrich(ctx context.Context, block rpcblock.Block, _ GameCaller, caller WethCaller, game *monTypes.EnrichedGameData) error {
	recipients := make(map[common.Address]bool)
	for _, claim := range game.Claims {
		if claim.CounteredBy != (common.Address{}) {
			recipients[claim.CounteredBy] = true
		} else {
			recipients[claim.Claimant] = true
		}
	}
	recipientAddrs := maps.Keys(recipients)
	withdrawals, err := caller.GetWithdrawals(ctx, block, game.Proxy, recipientAddrs...)
	if err != nil {
		return fmt.Errorf("failed to fetch withdrawals: %w", err)
	}
	if len(withdrawals) != len(recipientAddrs) {
		return fmt.Errorf("%w, requested %v values but got %v", ErrIncorrectWithdrawalsCount, len(recipientAddrs), len(withdrawals))
	}
	if game.WithdrawalRequests == nil {
		game.WithdrawalRequests = make(map[common.Address]*contracts.WithdrawalRequest)
	}
	for i, recipient := range recipientAddrs {
		game.WithdrawalRequests[recipient] = withdrawals[i]
	}
	return nil
}
