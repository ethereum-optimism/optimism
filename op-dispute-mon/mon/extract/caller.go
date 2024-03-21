package extract

import (
	"context"
	"fmt"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

const metricsLabel = "game_caller_creator"

type CallerMetrics interface {
	caching.Metrics
	contractMetrics.ContractMetricer
}

type GameCaller interface {
	GetWithdrawals(context.Context, rpcblock.Block, common.Address, ...common.Address) ([]*contracts.WithdrawalRequest, error)
	GetGameMetadata(context.Context, rpcblock.Block) (common.Hash, uint64, common.Hash, gameTypes.GameStatus, uint64, error)
	GetAllClaims(context.Context, rpcblock.Block) ([]faultTypes.Claim, error)
	BondCaller
	BalanceCaller
}

type WethCaller interface {
	GetWithdrawals(context.Context, rpcblock.Block, common.Address, ...common.Address) ([]*contracts.WithdrawalRequest, error)
}

type CallerCreator struct {
	m      CallerMetrics
	caller *batching.MultiCaller
	games  *caching.LRUCache[common.Address, *contracts.FaultDisputeGameContract]
	weths  *caching.LRUCache[common.Address, *contracts.DelayedWethContract]
}

func NewCallerCreator(m CallerMetrics, caller *batching.MultiCaller) *CallerCreator {
	return &CallerCreator{
		m:      m,
		caller: caller,
		games:  caching.NewLRUCache[common.Address, *contracts.FaultDisputeGameContract](m, metricsLabel, 100),
		weths:  caching.NewLRUCache[common.Address, *contracts.DelayedWethContract](m, metricsLabel, 100),
	}
}

func (g *GameCallerCreator) CreateContract(game gameTypes.GameMetadata) (GameCaller, error) {
	if fdg, ok := g.cache.Get(game.Proxy); ok {
		return fdg, nil
	}
	switch game.GameType {
	case faultTypes.CannonGameType, faultTypes.AlphabetGameType:
		fdg, err := contracts.NewFaultDisputeGameContract(g.m, game.Proxy, g.caller)
		if err != nil {
			return nil, fmt.Errorf("failed to create FaultDisputeGameContract: %w", err)
		}
		g.games.Add(game.Proxy, fdg)
		return fdg, nil
	default:
		return nil, fmt.Errorf("unsupported game type: %d", game.GameType)
	}
}
