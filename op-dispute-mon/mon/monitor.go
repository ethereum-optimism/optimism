package mon

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type blockNumberFetcher func(ctx context.Context) (uint64, error)
type blockHashFetcher func(ctx context.Context, number *big.Int) (common.Hash, error)

// gameSource loads information about the games available to play
type gameSource interface {
	GetGamesAtOrAfter(ctx context.Context, blockHash common.Hash, earliestTimestamp uint64) ([]types.GameMetadata, error)
}

type StatusFetcher interface {
	GetStatus(context.Context, common.Address) (types.GameStatus, error)
}

type RWClock interface {
	SetTime(uint64)
	Now() time.Time
}

type MonitorMetricer interface {
	RecordGamesStatus(inProgress, defenderWon, challengerWon int)
}

type gameMonitor struct {
	logger           log.Logger
	metrics          MonitorMetricer
	clock            RWClock
	monitorInterval  time.Duration
	done             chan struct{}
	source           gameSource
	status           StatusFetcher
	gameWindow       time.Duration
	fetchBlockNumber blockNumberFetcher
	fetchBlockHash   blockHashFetcher
}

func newGameMonitor(
	logger log.Logger,
	metrics MonitorMetricer,
	cl RWClock,
	monitorInterval time.Duration,
	source gameSource,
	status StatusFetcher,
	gameWindow time.Duration,
	fetchBlockNumber blockNumberFetcher,
	fetchBlockHash blockHashFetcher,
) *gameMonitor {
	return &gameMonitor{
		logger:           logger,
		metrics:          metrics,
		clock:            cl,
		done:             make(chan struct{}),
		monitorInterval:  monitorInterval,
		source:           source,
		status:           status,
		gameWindow:       gameWindow,
		fetchBlockNumber: fetchBlockNumber,
		fetchBlockHash:   fetchBlockHash,
	}
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

func (m *gameMonitor) monitorGames(ctx context.Context) error {
	blockNumber, err := m.fetchBlockNumber(context.Background())
	if err != nil {
		return fmt.Errorf("Failed to fetch block number: %w", err)
	}
	m.logger.Debug("Fetched block number", "blockNumber", blockNumber)
	blockHash, err := m.fetchBlockHash(context.Background(), new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return fmt.Errorf("Failed to fetch block hash: %w", err)
	}
	games, err := m.source.GetGamesAtOrAfter(ctx, blockHash, m.minGameTimestamp())
	if err != nil {
		return fmt.Errorf("failed to load games: %w", err)
	}
	for _, game := range games {
		if err := m.recordGameStatus(ctx, game); err != nil {
			m.logger.Error("Failed to record game status", "err", err)
		}
	}
	return nil
}

func (m *gameMonitor) recordGameStatus(ctx context.Context, game types.GameMetadata) error {
	status, err := m.status.GetStatus(ctx, game.Proxy)
	if err != nil {
		return fmt.Errorf("failed to get game status: %w", err)
	}
	switch status {
	case types.GameStatusInProgress:
		m.metrics.RecordGamesStatus(1, 0, 0)
	case types.GameStatusDefenderWon:
		m.metrics.RecordGamesStatus(0, 1, 0)
	case types.GameStatusChallengerWon:
		m.metrics.RecordGamesStatus(0, 0, 1)
	}
	return nil
}

func (m *gameMonitor) loop() {
	ticker := time.NewTicker(m.monitorInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := m.monitorGames(context.Background()); err != nil {
				m.logger.Error("Failed to monitor games", "err", err)
			}
		case <-m.done:
			m.logger.Info("Stopping game monitor")
			return
		}
	}
}

func (m *gameMonitor) StartMonitoring() {
	if m.done == nil {
		m.done = make(chan struct{})
	}
	go m.loop()
}

func (m *gameMonitor) StopMonitoring() {
	m.logger.Info("Stopping game monitor")
	close(m.done)
}
