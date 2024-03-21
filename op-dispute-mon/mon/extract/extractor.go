package extract

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
)

type CreateWethCaller func(proxy common.Address) (WethCaller, error)
type CreateGameCaller func(game gameTypes.GameMetadata) (GameCaller, error)
type FactoryGameFetcher func(ctx context.Context, blockHash common.Hash, earliestTimestamp uint64) ([]gameTypes.GameMetadata, error)

type Enricher interface {
	Enrich(context.Context, rpcblock.Block, GameCaller, WethCaller, *monTypes.EnrichedGameData) error
}

type Extractor struct {
	logger           log.Logger
	createGameCaller CreateGameCaller
	createWethCaller CreateWethCaller
	fetchGames       FactoryGameFetcher
	enrichers        []Enricher
}

func NewExtractor(logger log.Logger, gameCreator CreateGameCaller, wethCreator CreateWethCaller, fetchGames FactoryGameFetcher, enrichers ...Enricher) *Extractor {
	return &Extractor{
		logger:           logger,
		createGameCaller: gameCreator,
		createWethCaller: wethCreator,
		fetchGames:       fetchGames,
		enrichers:        enrichers,
	}
}

func (e *Extractor) Extract(ctx context.Context, blockHash common.Hash, minTimestamp uint64) ([]*monTypes.EnrichedGameData, error) {
	games, err := e.fetchGames(ctx, blockHash, minTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to load games: %w", err)
	}
	return e.enrichGames(ctx, blockHash, games), nil
}

func (e *Extractor) enrichGames(ctx context.Context, blockHash common.Hash, games []gameTypes.GameMetadata) []*monTypes.EnrichedGameData {
	var enrichedGames []*monTypes.EnrichedGameData
	for _, game := range games {
		gameCaller, err := e.createGameCaller(game)
		if err != nil {
			e.logger.Error("Failed to create game caller", "err", err)
			continue
		}
		wethCaller, err := e.createWethCaller(game.Proxy)
		if err != nil {
			e.logger.Error("Failed to create weth caller", "err", err)
			continue
		}
		l1Head, l2BlockNum, rootClaim, status, duration, err := gameCaller.GetGameMetadata(ctx, rpcblock.ByHash(blockHash))
		if err != nil {
			e.logger.Error("Failed to fetch game metadata", "err", err)
			continue
		}
		claims, err := gameCaller.GetAllClaims(ctx, rpcblock.ByHash(blockHash))
		if err != nil {
			e.logger.Error("Failed to fetch game claims", "err", err)
			continue
		}
		enrichedClaims := make([]monTypes.EnrichedClaim, len(claims))
		for i, claim := range claims {
			enrichedClaims[i] = monTypes.EnrichedClaim{Claim: claim}
		}
		enrichedGame := &monTypes.EnrichedGameData{
			GameMetadata:  game,
			L1Head:        l1Head,
			L2BlockNumber: l2BlockNum,
			RootClaim:     rootClaim,
			Status:        status,
			Duration:      duration,
			Claims:        enrichedClaims,
		}
		if err := e.applyEnrichers(ctx, blockHash, gameCaller, wethCaller, enrichedGame); err != nil {
			e.logger.Error("Failed to enrich game", "err", err)
			continue
		}
		enrichedGames = append(enrichedGames, enrichedGame)
	}
	return enrichedGames
}

func (e *Extractor) applyEnrichers(ctx context.Context, blockHash common.Hash, gameCaller GameCaller, wethCaller WethCaller, game *monTypes.EnrichedGameData) error {
	for _, enricher := range e.enrichers {
		if err := enricher.Enrich(ctx, rpcblock.ByHash(blockHash), gameCaller, wethCaller, game); err != nil {
			return err
		}
	}
	return nil
}
