package extract

import (
	"context"
	"fmt"
	"math/big"

	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
)

var _ Enricher = (*BalanceEnricher)(nil)

type BalanceCaller interface {
	GetBalance(context.Context, rpcblock.Block) (*big.Int, common.Address, error)
}

type BalanceEnricher struct{}

func NewBalanceEnricher() *BalanceEnricher {
	return &BalanceEnricher{}
}

func (b *BalanceEnricher) Enrich(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.EnrichedGameData) error {
	balance, holdingAddr, err := caller.GetBalance(ctx, block)
	if err != nil {
		return fmt.Errorf("failed to fetch balance: %w", err)
	}
	game.ETHCollateral = balance
	game.WETHContract = holdingAddr
	return nil
}
