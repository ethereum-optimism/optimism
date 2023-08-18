package fault

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type gamePlayer interface {
	ProgressGame(ctx context.Context) bool
}

type playerCreator func(address common.Address) (gamePlayer, error)
type blockNumberFetcher func(ctx context.Context) (uint64, error)

// gameSource loads information about the games available to play
type gameSource interface {
	FetchAllGamesAtBlock(ctx context.Context, blockNumber *big.Int) ([]FaultDisputeGame, error)
}

type gameMonitor struct {
	logger           log.Logger
	source           gameSource
	createPlayer     playerCreator
	fetchBlockNumber blockNumberFetcher
	allowedGame      common.Address
	players          map[common.Address]gamePlayer
}

func newGameMonitor(logger log.Logger, fetchBlockNumber blockNumberFetcher, allowedGame common.Address, source gameSource, createGame playerCreator) *gameMonitor {
	return &gameMonitor{
		logger:           logger,
		source:           source,
		createPlayer:     createGame,
		fetchBlockNumber: fetchBlockNumber,
		allowedGame:      allowedGame,
		players:          make(map[common.Address]gamePlayer),
	}
}

func (m *gameMonitor) progressGames(ctx context.Context) error {
	blockNum, err := m.fetchBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to load current block number: %w", err)
	}
	games, err := m.source.FetchAllGamesAtBlock(ctx, new(big.Int).SetUint64(blockNum))
	if err != nil {
		return fmt.Errorf("failed to load games: %w", err)
	}
	for _, game := range games {
		if m.allowedGame != (common.Address{}) && m.allowedGame != game.Proxy {
			m.logger.Debug("Skipping game not on allow list", "game", game.Proxy)
			continue
		}
		player, err := m.fetchOrCreateGamePlayer(game)
		if err != nil {
			m.logger.Error("Error while progressing game", "game", game.Proxy, "err", err)
			continue
		}
		player.ProgressGame(ctx)
	}
	return nil
}

func (m *gameMonitor) fetchOrCreateGamePlayer(gameData FaultDisputeGame) (gamePlayer, error) {
	if player, ok := m.players[gameData.Proxy]; ok {
		return player, nil
	}
	player, err := m.createPlayer(gameData.Proxy)
	if err != nil {
		return nil, fmt.Errorf("failed to create game player %v: %w", gameData.Proxy, err)
	}
	m.players[gameData.Proxy] = player
	return player, nil
}

func (m *gameMonitor) MonitorGames(ctx context.Context) error {
	m.logger.Info("Monitoring fault dispute games")

	for {
		err := m.progressGames(ctx)
		if err != nil {
			m.logger.Error("Failed to progress games", "err", err)
		}
		select {
		case <-time.After(300 * time.Millisecond):
		// Continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
