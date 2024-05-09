package extract

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrIgnored = errors.New("ignored")
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
	var ignored atomic.Int32
	var failed atomic.Int32

	concurrencyLimit := 5
	var wg sync.WaitGroup
	wg.Add(concurrencyLimit)
	gameCh := make(chan gameTypes.GameMetadata)
	enrichedCh := make(chan *monTypes.EnrichedGameData, concurrencyLimit)
	for i := 0; i < concurrencyLimit; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case game, ok := <-gameCh:
					if !ok {
						e.logger.Debug("Enriching complete")
						// Channel closed
						return
					}
					e.logger.Trace("Enriching game", "game", game.Proxy)
					enrichedGame, err := e.enrichGame(ctx, blockHash, game)
					if errors.Is(err, ErrIgnored) {
						ignored.Add(1)
						e.logger.Warn("Ignoring game", "game", game.Proxy)
						continue
					} else if err != nil {
						failed.Add(1)
						e.logger.Error("Failed to fetch game data", "game", game.Proxy, "err", err)
						continue
					}
					enrichedCh <- enrichedGame
				}
			}
		}()
	}

	var resultsWg sync.WaitGroup
	resultsWg.Add(1)
	go func() {
		defer resultsWg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case enrichedGame, ok := <-enrichedCh:
				if !ok {
					e.logger.Debug("Result reading complete")
					return
				}
				e.logger.Trace("Enriched game", "game", enrichedGame.Proxy)
				enrichedGames = append(enrichedGames, enrichedGame)
			}
		}
	}()
	for _, game := range games {
		gameCh <- game
	}
	close(gameCh)

	wg.Wait()
	close(enrichedCh)
	resultsWg.Wait()
	return enrichedGames, int(ignored.Load()), int(failed.Load())
}

func (e *Extractor) enrichGame(ctx context.Context, blockHash common.Hash, game gameTypes.GameMetadata) (*monTypes.EnrichedGameData, error) {
	if e.ignoredGames[game.Proxy] {
		return nil, ErrIgnored
	}
	caller, err := e.createContract(ctx, game)
	if err != nil {
		return nil, fmt.Errorf("failed to create contracts: %w", err)
	}
	l1Head, l2BlockNum, rootClaim, status, duration, err := caller.GetGameMetadata(ctx, rpcblock.ByHash(blockHash))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game metadata: %w", err)
	}
	claims, err := caller.GetAllClaims(ctx, rpcblock.ByHash(blockHash))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game claims: %w", err)
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
		return nil, fmt.Errorf("failed to enrich game: %w", err)
	}
	return enrichedGame, nil
}

func (e *Extractor) applyEnrichers(ctx context.Context, blockHash common.Hash, caller GameCaller, game *monTypes.EnrichedGameData) error {
	for _, enricher := range e.enrichers {
		if err := enricher.Enrich(ctx, rpcblock.ByHash(blockHash), caller, game); err != nil {
			return err
		}
	}
	return nil
}
