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
	GetAnchorStateRegistry(context.Context, rpcblock.Block) (common.Address, error)
}

// MetadataCaller exposes metadata shared by fault and SuperPermissioned games.
type MetadataCaller interface {
	GetExtendedMetadata(context.Context, rpcblock.Block) (contracts.GameMetadata, error)
}

// BondGameCaller exposes reads shared by every bond-bearing game.
type BondGameCaller interface {
	GetWithdrawals(context.Context, rpcblock.Block, ...common.Address) ([]*contracts.WithdrawalRequest, error)
	BondCaller
	BalanceCaller
}

// FaultGameCaller exposes reads used only to enrich fault games.
type FaultGameCaller interface {
	GameCaller
	MetadataCaller
	GetAllClaims(context.Context, rpcblock.Block) ([]faultTypes.Claim, error)
	BondGameCaller
	ClaimCaller
}

// ZKGameCaller exposes the generic and challenger views of a ZK game.
type ZKGameCaller interface {
	GameCaller
	GetMetadata(context.Context, rpcblock.Block) (contracts.GenericGameMetadata, error)
	GetChallengerMetadata(context.Context, rpcblock.Block) (contracts.ChallengerMetadata, error)
}

// ZKBondGameCaller exposes the complete pinned ZK snapshot used by the monitor.
type ZKBondGameCaller interface {
	ZKGameCaller
	BondGameCaller
	GetBondMetadata(context.Context, rpcblock.Block) (contracts.ZKBondMetadata, error)
}

var _ FaultGameCaller = (contracts.FaultDisputeGameContract)(nil)
var _ ZKGameCaller = (*contracts.ZKDisputeGameContractLatest)(nil)
var _ ZKBondGameCaller = (*contracts.ZKDisputeGameContractLatest)(nil)
var _ BondGameCaller = (contracts.FaultDisputeGameContract)(nil)
var _ MetadataCaller = (*contracts.SuperPermissionedDisputeGameContract)(nil)

type GameCallerCreator struct {
	m      GameCallerMetrics
	cache  *caching.LRUCache[common.Address, GameCaller]
	caller *batching.MultiCaller
}

func NewGameCallerCreator(m GameCallerMetrics, caller *batching.MultiCaller) *GameCallerCreator {
	return &GameCallerCreator{
		m:      m,
		caller: caller,
		cache:  caching.NewLRUCache[common.Address, GameCaller](m, metricsLabel, 100),
	}
}

func (g *GameCallerCreator) CreateContract(ctx context.Context, game gameTypes.GameMetadata) (GameCaller, error) {
	if fdg, ok := g.cache.Get(game.Proxy); ok {
		return fdg, nil
	}
	switch gameTypes.GameType(game.GameType) {
	case gameTypes.SuperPermissionedGameType:
		fdg := contracts.NewSuperPermissionedDisputeGameContract(g.m, game.Proxy, g.caller)
		g.cache.Add(game.Proxy, fdg)
		return fdg, nil
	case gameTypes.CannonGameType,
		gameTypes.PermissionedGameType,
		gameTypes.CannonKonaGameType,
		gameTypes.AlphabetGameType,
		gameTypes.FastGameType,
		gameTypes.SuperCannonKonaGameType:
		fdg, err := contracts.NewFaultDisputeGameContract(ctx, g.m, game.Proxy, g.caller)
		if err != nil {
			return nil, fmt.Errorf("failed to create fault dispute game contract: %w", err)
		}
		g.cache.Add(game.Proxy, fdg)
		return fdg, nil
	default:
		return nil, fmt.Errorf("unsupported game type: %d", game.GameType)
	}
}
