package mon

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type blockNumberFetcher func(ctx context.Context) (uint64, error)
type blockHashFetcher func(ctx context.Context, number *big.Int) (common.Hash, error)

type factorySource interface {
	GetGamesAtOrAfter(ctx context.Context, blockHash common.Hash, earliestTimestamp uint64) ([]types.GameMetadata, error)
}

type gameSource interface {
	GetRootClaim(context.Context, common.Address) (common.Hash, error)
	GetStatus(context.Context, common.Address) (types.GameStatus, error)
	GetL2BlockNumber(context.Context, common.Address) (uint64, error)
}

type RWClock interface {
	SetTime(uint64)
	Now() time.Time
}

type OutputRollupClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

type MonitorMetricer interface {
	RecordGameAgreement(status string, count int)
	RecordGamesStatus(inProgress, defenderWon, challengerWon int)
}

type gameMonitor struct {
	logger           log.Logger
	metrics          MonitorMetricer
	clock            RWClock
	monitorInterval  time.Duration
	done             chan struct{}
	factory          factorySource
	game             gameSource
	gameWindow       time.Duration
	outputClient     OutputRollupClient
	fetchBlockNumber blockNumberFetcher
	fetchBlockHash   blockHashFetcher
}

func newGameMonitor(
	logger log.Logger,
	metrics MonitorMetricer,
	cl RWClock,
	monitorInterval time.Duration,
	factory factorySource,
	game gameSource,
	gameWindow time.Duration,
	outputClient OutputRollupClient,
	fetchBlockNumber blockNumberFetcher,
	fetchBlockHash blockHashFetcher,
) *gameMonitor {
	return &gameMonitor{
		logger:           logger,
		metrics:          metrics,
		clock:            cl,
		done:             make(chan struct{}),
		monitorInterval:  monitorInterval,
		factory:          factory,
		game:             game,
		gameWindow:       gameWindow,
		outputClient:     outputClient,
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

func (m *gameMonitor) checkRootAgreement(ctx context.Context, game types.GameMetadata) (bool, error) {
	root, err := m.game.GetRootClaim(ctx, game.Proxy)
	if err != nil {
		return false, fmt.Errorf("failed to get game root claim: %w", err)
	}
	l2BlockNum, err := m.game.GetL2BlockNumber(ctx, game.Proxy)
	if err != nil {
		return false, fmt.Errorf("failed to get game L2 block number: %w", err)
	}
	output, err := m.outputClient.OutputAtBlock(ctx, l2BlockNum)
	if err != nil {
		return false, fmt.Errorf("failed to get output at block: %w", err)
	}
	return root == common.Hash(output.OutputRoot), nil
}

func (m *gameMonitor) monitorGames(ctx context.Context) error {
	blockNum, err := m.fetchBlockNumber(context.Background())
	if err != nil {
		return fmt.Errorf("Failed to fetch block number: %w", err)
	}
	m.logger.Debug("Fetched block number", "blockNumber", blockNum)
	blockHash, err := m.fetchBlockHash(context.Background(), new(big.Int).SetUint64(blockNum))
	if err != nil {
		return fmt.Errorf("Failed to fetch block hash: %w", err)
	}
	games, err := m.factory.GetGamesAtOrAfter(ctx, blockHash, m.minGameTimestamp())
	if err != nil {
		return fmt.Errorf("failed to load games: %w", err)
	}
	for _, game := range games {
		if err := m.recordGameStatus(ctx, game); err != nil {
			m.logger.Error("Failed to record game status", "err", err)
		}
		agree, err := m.checkRootAgreement(ctx, game)
		if err != nil {
			m.logger.Error("Failed to check root agreement", "err", err)
		}
		m.logger.Debug("Checked root agreement", "game", game.Proxy, "agree", agree)
	}
	return nil
}

func (m *gameMonitor) recordGameStatus(ctx context.Context, game types.GameMetadata) error {
	status, err := m.game.GetStatus(ctx, game.Proxy)
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
