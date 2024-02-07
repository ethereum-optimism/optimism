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

type factorySource interface {
	GetGamesAtOrAfter(ctx context.Context, blockHash common.Hash, earliestTimestamp uint64) ([]types.GameMetadata, error)
}

type RWClock interface {
	SetTime(uint64)
	Now() time.Time
}

type MonitorMetricer interface {
	RecordGameAgreement(status string, count int)
	RecordGamesStatus(inProgress, defenderWon, challengerWon int)
}

type Detector interface {
	Detect(ctx context.Context, games []types.GameMetadata)
}

type gameMonitor struct {
	logger  log.Logger
	metrics MonitorMetricer

	ctx    context.Context
	cancel context.CancelFunc

	clock            RWClock
	monitorInterval  time.Duration
	done             chan struct{}
	factory          factorySource
	gameWindow       time.Duration
	detector         Detector
	fetchBlockNumber blockNumberFetcher
	fetchBlockHash   blockHashFetcher
}

func newGameMonitor(
	logger log.Logger,
	metrics MonitorMetricer,
	cl RWClock,
	monitorInterval time.Duration,
	factory factorySource,
	gameWindow time.Duration,
	detector Detector,
	fetchBlockNumber blockNumberFetcher,
	fetchBlockHash blockHashFetcher,
) *gameMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &gameMonitor{
		logger:           logger,
		metrics:          metrics,
		ctx:              ctx,
		cancel:           cancel,
		clock:            cl,
		done:             make(chan struct{}),
		monitorInterval:  monitorInterval,
		factory:          factory,
		gameWindow:       gameWindow,
		detector:         detector,
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
	blockNum, err := m.fetchBlockNumber(m.ctx)
	if err != nil {
		return fmt.Errorf("Failed to fetch block number: %w", err)
	}
	blockHash, err := m.fetchBlockHash(m.ctx, new(big.Int).SetUint64(blockNum))
	if err != nil {
		return fmt.Errorf("Failed to fetch block hash: %w", err)
	}
	games, err := m.factory.GetGamesAtOrAfter(m.ctx, blockHash, m.minGameTimestamp())
	if err != nil {
		return fmt.Errorf("failed to load games: %w", err)
	}
	m.detector.Detect(m.ctx, games)
	return nil
}

func (m *gameMonitor) loop() {
	ticker := time.NewTicker(m.monitorInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
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
	if m.done == nil {
		m.done = make(chan struct{})
	}
	go m.loop()
}

func (m *gameMonitor) StopMonitoring() {
	m.logger.Info("Stopping game monitor")
	m.cancel()
	close(m.done)
}
