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

type GameCallerMetrics interface {
	caching.Metrics
	contractMetrics.ContractMetricer
}

type GameCaller interface {
	GetWithdrawals(context.Context, rpcblock.Block, common.Address, ...common.Address) ([]*contracts.WithdrawalRequest, error)
	GetGameMetadata(context.Context, rpcblock.Block) (common.Hash, uint64, common.Hash, gameTypes.GameStatus, uint64, error)
	GetAllClaims(context.Context, rpcblock.Block) ([]faultTypes.Claim, error)
	BondCaller
	BalanceCaller
	ClaimCaller
}

type GameCallerCreator struct {
	m      GameCallerMetrics
	cache  *caching.LRUCache[common.Address, *contracts.FaultDisputeGameContract]
	caller *batching.MultiCaller
}

func NewGameCallerCreator(m GameCallerMetrics, caller *batching.MultiCaller) *GameCallerCreator {
	return &GameCallerCreator{
		m:      m,
		caller: caller,
		cache:  caching.NewLRUCache[common.Address, *contracts.FaultDisputeGameContract](m, metricsLabel, 100),
	}
}

func (g *GameCallerCreator) CreateContract(game gameTypes.GameMetadata) (GameCaller, error) {
	if fdg, ok := g.cache.Get(game.Proxy); ok {
		return fdg, nil
	}
	switch game.GameType {
	case faultTypes.CannonGameType, faultTypes.AsteriscGameType, faultTypes.AlphabetGameType:
		fdg := contracts.NewFaultDisputeGameContract(g.m, game.Proxy, g.caller)
		g.cache.Add(game.Proxy, fdg)
		return fdg, nil
	default:
		return nil, fmt.Errorf("unsupported game type: %d", game.GameType)
	}
}
