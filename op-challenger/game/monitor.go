package game

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type blockNumberFetcher func(ctx context.Context) (uint64, error)

// gameSource loads information about the games available to play
type gameSource interface {
	FetchAllGamesAtBlock(ctx context.Context, earliest uint64, blockNumber *big.Int) ([]FaultDisputeGame, error)
}

type gameScheduler interface {
	Schedule([]common.Address) error
}

type gameMonitor struct {
	logger           log.Logger
	clock            clock.Clock
	source           gameSource
	scheduler        gameScheduler
	gameWindow       time.Duration
	fetchBlockNumber blockNumberFetcher
	allowedGames     []common.Address
}

func newGameMonitor(
	logger log.Logger,
	cl clock.Clock,
	source gameSource,
	scheduler gameScheduler,
	gameWindow time.Duration,
	fetchBlockNumber blockNumberFetcher,
	allowedGames []common.Address,
) *gameMonitor {
	return &gameMonitor{
		logger:           logger,
		clock:            cl,
		scheduler:        scheduler,
		source:           source,
		gameWindow:       gameWindow,
		fetchBlockNumber: fetchBlockNumber,
		allowedGames:     allowedGames,
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

func (m *gameMonitor) progressGames(ctx context.Context, blockNum uint64) error {
	games, err := m.source.FetchAllGamesAtBlock(ctx, m.minGameTimestamp(), new(big.Int).SetUint64(blockNum))
	if err != nil {
		return fmt.Errorf("failed to load games: %w", err)
	}
	var gamesToPlay []common.Address
	for _, game := range games {
		if !m.allowedGame(game.Proxy) {
			m.logger.Debug("Skipping game not on allow list", "game", game.Proxy)
			continue
		}
		gamesToPlay = append(gamesToPlay, game.Proxy)
	}
	if err := m.scheduler.Schedule(gamesToPlay); errors.Is(err, scheduler.ErrBusy) {
		m.logger.Info("Scheduler still busy with previous update")
	} else if err != nil {
		return fmt.Errorf("failed to schedule games: %w", err)
	}
	return nil
}

func (m *gameMonitor) MonitorGames(ctx context.Context) error {
	m.logger.Info("Monitoring fault dispute games")

	blockNum := uint64(0)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			nextBlockNum, err := m.fetchBlockNumber(ctx)
			if err != nil {
				m.logger.Error("Failed to load current block number", "err", err)
				continue
			}
			if nextBlockNum > blockNum {
				blockNum = nextBlockNum
				if err := m.progressGames(ctx, nextBlockNum); err != nil {
					m.logger.Error("Failed to progress games", "err", err)
				}
			}
			if err := m.clock.SleepCtx(ctx, time.Second); err != nil {
				return err
			}
		}
	}
}
