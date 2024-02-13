package extract

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type CreateGameCaller func(game gameTypes.GameMetadata) (GameCaller, error)
type FactoryGameFetcher func(ctx context.Context, blockHash common.Hash, earliestTimestamp uint64) ([]gameTypes.GameMetadata, error)
type OutputFetcher func(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)

type Extractor struct {
	logger         log.Logger
	createContract CreateGameCaller
	fetchGames     FactoryGameFetcher
	fetchOutput    OutputFetcher
}

func NewExtractor(logger log.Logger, creator CreateGameCaller, fetchGames FactoryGameFetcher, outputs OutputFetcher) *Extractor {
	return &Extractor{
		logger:         logger,
		createContract: creator,
		fetchGames:     fetchGames,
		fetchOutput:    outputs,
	}
}

func (e *Extractor) Extract(ctx context.Context, blockHash common.Hash, minTimestamp uint64) ([]monTypes.EnrichedGameData, error) {
	// Fetch games from the factory
	games, err := e.fetchGames(ctx, blockHash, minTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to load games: %w", err)
	}

	gamesWithMetadata := e.enrichGameMetadata(ctx, games)
	return e.enrichGames(ctx, gamesWithMetadata)
}

func (e *Extractor) enrichGames(ctx context.Context, games []monTypes.EnrichedGameData) ([]monTypes.EnrichedGameData, error) {
	// For each game, query for the expected root claim
	for _, game := range games {
		output, err := e.fetchOutput(ctx, game.L2BlockNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch output for block %d: %w", game.L2BlockNumber, err)
		}
		expectedRoot := common.Hash(output.OutputRoot)
		game.ExpectedRoot = expectedRoot
	}
	return games, nil
}

func (e *Extractor) enrichGameMetadata(ctx context.Context, games []gameTypes.GameMetadata) []monTypes.EnrichedGameData {
	var enrichedGames []monTypes.EnrichedGameData
	for _, game := range games {
		caller, err := e.createContract(game)
		if err != nil {
			e.logger.Error("failed to create game caller", "err", err)
			continue
		}
		l2BlockNum, rootClaim, status, err := caller.GetGameMetadata(ctx)
		if err != nil {
			e.logger.Error("failed to fetch game metadata", "err", err)
			continue
		}
		enrichedGames = append(enrichedGames, monTypes.EnrichedGameData{
			GameMetadata:  game,
			L2BlockNumber: l2BlockNum,
			RootClaim:     rootClaim,
			Status:        status,
		})
	}
	return enrichedGames
}
