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

type (
	CreateGameCaller   func(ctx context.Context, game gameTypes.GameMetadata) (GameCaller, error)
	FactoryGameFetcher func(ctx context.Context, blockHash common.Hash, earliestTimestamp uint64) ([]gameTypes.GameMetadata, error)
)

type Enricher interface {
	Enrich(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.EnrichedGameData) error
}

type Extractor struct {
	logger         log.Logger
	createContract CreateGameCaller
	fetchGames     FactoryGameFetcher
	enrichers      []Enricher
	ignoredGames   map[common.Address]bool
}

func NewExtractor(logger log.Logger, creator CreateGameCaller, fetchGames FactoryGameFetcher, ignoredGames []common.Address, enrichers ...Enricher) *Extractor {
	ignored := make(map[common.Address]bool)
	for _, game := range ignoredGames {
		ignored[game] = true
	}
	return &Extractor{
		logger:         logger,
		createContract: creator,
		fetchGames:     fetchGames,
		enrichers:      enrichers,
		ignoredGames:   ignored,
	}
}

func (e *Extractor) Extract(ctx context.Context, blockHash common.Hash, minTimestamp uint64) ([]*monTypes.EnrichedGameData, int, int, error) {
	games, err := e.fetchGames(ctx, blockHash, minTimestamp)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to load games: %w", err)
	}
	enriched, ignored, failed := e.enrichGames(ctx, blockHash, games)
	return enriched, ignored, failed, nil
}

func (e *Extractor) enrichGames(ctx context.Context, blockHash common.Hash, games []gameTypes.GameMetadata) ([]*monTypes.EnrichedGameData, int, int) {
	var enrichedGames []*monTypes.EnrichedGameData
	ignored := 0
	failed := 0
	for _, game := range games {
		if e.ignoredGames[game.Proxy] {
			ignored++
			e.logger.Warn("Ignoring game", "game", game.Proxy)
			continue
		}
		caller, err := e.createContract(ctx, game)
		if err != nil {
			e.logger.Error("Failed to create game caller", "err", err)
			failed++
			continue
		}
		l1Head, l2BlockNum, rootClaim, status, duration, err := caller.GetGameMetadata(ctx, rpcblock.ByHash(blockHash))
		if err != nil {
			e.logger.Error("Failed to fetch game metadata", "err", err)
			continue
		}
		claims, err := caller.GetAllClaims(ctx, rpcblock.ByHash(blockHash))
		if err != nil {
			e.logger.Error("Failed to fetch game claims", "err", err)
			continue
		}
		enrichedClaims := make([]monTypes.EnrichedClaim, len(claims))
		for i, claim := range claims {
			enrichedClaims[i] = monTypes.EnrichedClaim{Claim: claim}
		}
		enrichedGame := &monTypes.EnrichedGameData{
			GameMetadata:     game,
			L1Head:           l1Head,
			L2BlockNumber:    l2BlockNum,
			RootClaim:        rootClaim,
			Status:           status,
			MaxClockDuration: duration,
			Claims:           enrichedClaims,
		}
		if err := e.applyEnrichers(ctx, blockHash, caller, enrichedGame); err != nil {
			e.logger.Error("Failed to enrich game", "err", err)
			continue
		}
		enrichedGames = append(enrichedGames, enrichedGame)
	}
	return enrichedGames, ignored, failed
}

func (e *Extractor) applyEnrichers(ctx context.Context, blockHash common.Hash, caller GameCaller, game *monTypes.EnrichedGameData) error {
	for _, enricher := range e.enrichers {
		if err := enricher.Enrich(ctx, rpcblock.ByHash(blockHash), caller, game); err != nil {
			return err
		}
	}
	return nil
}
