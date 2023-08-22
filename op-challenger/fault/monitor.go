package fault

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type gamePlayer interface {
	ProgressGame(ctx context.Context) bool
	Cleanup() error
}

type playerCreator func(address common.Address) (gamePlayer, error)
type blockNumberFetcher func(ctx context.Context) (uint64, error)

// gameSource loads information about the games available to play
type gameSource interface {
	FetchAllGamesAtBlock(ctx context.Context, earliest uint64, blockNumber *big.Int) ([]FaultDisputeGame, error)
}

type gameMonitor struct {
	logger           log.Logger
	clock            clock.Clock
	source           gameSource
	gameWindow       time.Duration
	createPlayer     playerCreator
	fetchBlockNumber blockNumberFetcher
	allowedGames     []common.Address
	players          map[common.Address]gamePlayer
}

func newGameMonitor(logger log.Logger, gameWindow time.Duration, cl clock.Clock, fetchBlockNumber blockNumberFetcher, allowedGames []common.Address, source gameSource, createGame playerCreator) *gameMonitor {
	return &gameMonitor{
		logger:           logger,
		clock:            cl,
		source:           source,
		gameWindow:       gameWindow,
		createPlayer:     createGame,
		fetchBlockNumber: fetchBlockNumber,
		allowedGames:     allowedGames,
		players:          make(map[common.Address]gamePlayer),
	}
}

func (m *gameMonitor) allowedGame(game common.Address) bool {
	if len(m.allowedGames) == 0 {
		return true
	}
	for _, allowed := range m.allowedGames {
		if allowed == game {
			return true
		}
	}
	return false
}

func (m *gameMonitor) minGameTimestamp() uint64 {
	if m.gameWindow.Seconds() == 0 {
		return 0
	}
	// time: "To compute t-d for a duration d, use t.Add(-d)."
	// https://pkg.go.dev/time#Time.Sub
	if m.clock.Now().Unix() > int64(m.gameWindow.Seconds()) {
		return uint64(m.clock.Now().Add(-m.gameWindow).Unix())
	}
	return 0
}

func (m *gameMonitor) progressGames(ctx context.Context) error {
	blockNum, err := m.fetchBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to load current block number: %w", err)
	}
	games, err := m.source.FetchAllGamesAtBlock(ctx, m.minGameTimestamp(), new(big.Int).SetUint64(blockNum))
	if err != nil {
		return fmt.Errorf("failed to load games: %w", err)
	}
	requiredGames := make(map[common.Address]bool)
	for _, game := range games {
		if !m.allowedGame(game.Proxy) {
			m.logger.Debug("Skipping game not on allow list", "game", game.Proxy)
			continue
		}
		requiredGames[game.Proxy] = true
		player, err := m.fetchOrCreateGamePlayer(game)
		if err != nil {
			m.logger.Error("Error while progressing game", "game", game.Proxy, "err", err)
			continue
		}
		done := player.ProgressGame(ctx)
		if done {
			// Remove resources on disk as soon as the game is complete to save disk space.
			// We keep the player in memory to avoid recreating it on every update but will no longer
			// need the resources on disk because there are no further actions required on the game.
			if err := player.Cleanup(); err != nil {
				m.logger.Error("Unable to cleanup player data", "err", err)
			}
		}
	}
	// Remove the player for any game that's no longer being returned from the list of active games
	for addr := range m.players {
		if _, ok := requiredGames[addr]; ok {
			// Game still required
			continue
		}
		delete(m.players, addr)
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
		if err := m.clock.SleepCtx(ctx, 300*time.Millisecond); err != nil {
			return err
		}
	}
}
