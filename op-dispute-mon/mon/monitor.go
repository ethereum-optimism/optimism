package mon

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type blockNumberFetcher func(ctx context.Context) (uint64, error)
type blockHashFetcher func(ctx context.Context, number *big.Int) (common.Hash, error)

// gameSource loads information about the games available to play
type gameSource interface {
	GetGamesAtOrAfter(ctx context.Context, blockHash common.Hash, earliestTimestamp uint64) ([]types.GameMetadata, error)
}

type MonitorMetricer interface {
	RecordGamesStatus(inProgress, defenderWon, challengerWon int)
}

type MetadataCreator interface {
	CreateContract(game types.GameMetadata) (MetadataLoader, error)
}

type gameMonitor struct {
	logger  log.Logger
	metrics MonitorMetricer

	ctx    context.Context
	cancel context.CancelFunc

	clock            clock.Clock
	monitorInterval  time.Duration
	done             chan struct{}
	source           gameSource
	metadata         MetadataCreator
	gameWindow       time.Duration
	fetchBlockNumber blockNumberFetcher
	fetchBlockHash   blockHashFetcher
}

func newGameMonitor(
	ctx context.Context,
	logger log.Logger,
	metrics MonitorMetricer,
	cl clock.Clock,
	monitorInterval time.Duration,
	source gameSource,
	metadata MetadataCreator,
	gameWindow time.Duration,
	fetchBlockNumber blockNumberFetcher,
	fetchBlockHash blockHashFetcher,
) *gameMonitor {
	return &gameMonitor{
		logger:           logger,
		metrics:          metrics,
		ctx:              ctx,
		clock:            cl,
		done:             make(chan struct{}),
		monitorInterval:  monitorInterval,
		source:           source,
		metadata:         metadata,
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

func (m *gameMonitor) monitorGames() error {
	blockNumber, err := m.fetchBlockNumber(m.ctx)
	if err != nil {
		return fmt.Errorf("Failed to fetch block number: %w", err)
	}
	m.logger.Debug("Fetched block number", "blockNumber", blockNumber)
	blockHash, err := m.fetchBlockHash(context.Background(), new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return fmt.Errorf("Failed to fetch block hash: %w", err)
	}
	games, err := m.source.GetGamesAtOrAfter(m.ctx, blockHash, m.minGameTimestamp())
	if err != nil {
		return fmt.Errorf("failed to load games: %w", err)
	}
	return m.recordGamesStatus(m.ctx, games)
}

func (m *gameMonitor) recordGamesStatus(ctx context.Context, games []types.GameMetadata) error {
	inProgress, defenderWon, challengerWon := 0, 0, 0
	for _, game := range games {
		loader, err := m.metadata.CreateContract(game)
		if err != nil {
			m.logger.Error("Failed to create contract", "err", err)
			continue
		}
		_, _, status, err := loader.GetGameMetadata(ctx)
		if err != nil {
			m.logger.Error("Failed to get game metadata", "err", err)
			continue
		}
		switch status {
		case types.GameStatusInProgress:
			inProgress++
		case types.GameStatusDefenderWon:
			defenderWon++
		case types.GameStatusChallengerWon:
			challengerWon++
		}
	}
	m.metrics.RecordGamesStatus(inProgress, defenderWon, challengerWon)
	return nil
}

func (m *gameMonitor) loop() {
	ticker := m.clock.NewTicker(m.monitorInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.Ch():
			if err := m.monitorGames(); err != nil {
				m.logger.Error("Failed to monitor games", "err", err)
			}
		case <-m.done:
			m.logger.Info("Stopping game monitor")
			return
		}
	}
}

func (m *gameMonitor) StartMonitoring() {
	// Setup the cancellation only if it's not already set.
	// This prevents overwriting the context and cancel function
	// if, for example, this function is called multiple times.
	if m.cancel == nil {
		ctx, cancel := context.WithCancel(m.ctx)
		m.ctx = ctx
		m.cancel = cancel
	}
	m.logger.Info("Starting game monitor")
	go m.loop()
}

func (m *gameMonitor) StopMonitoring() {
	m.logger.Info("Stopping game monitor")
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	close(m.done)
}
