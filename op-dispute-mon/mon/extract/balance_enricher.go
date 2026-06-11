package extract

import (
	"context"
	"fmt"
	"math/big"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
)

var _ Enricher = (*BalanceEnricher)(nil)

type BalanceCaller interface {
	GetBalanceAndDelay(context.Context, rpcblock.Block) (*big.Int, time.Duration, common.Address, error)
}

type BalanceEnricher struct{}

func NewBalanceEnricher() *BalanceEnricher {
	return &BalanceEnricher{}
}

func (b *BalanceEnricher) Enrich(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.EnrichedGameData) error {
	if gameTypes.GameType(game.GameType) == gameTypes.SuperPermissionedGameType {
		return nil
	}
	balance, delay, holdingAddr, err := caller.GetBalanceAndDelay(ctx, block)
	if err != nil {
		return fmt.Errorf("failed to fetch balance: %w", err)
	}
	game.ETHCollateral = balance
	game.WETHContract = holdingAddr
	game.WETHDelay = delay
	return nil
}
