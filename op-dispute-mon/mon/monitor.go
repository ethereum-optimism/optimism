package mon

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// ForecastResolution records forecasts for all enriched game variants.
type ForecastResolution func(games []types.EnrichedGame, ignoredCount, failedCount int)

// AnchorStateCheck checks anchor state shared by every game kind.
type AnchorStateCheck func(ctx context.Context, blockHash common.Hash, games []*types.CommonGameData)

// CommonMonitor checks fields shared by every game kind.
type CommonMonitor func(games []*types.CommonGameData)

// FaultMonitor checks fields that exist only on fault games.
type FaultMonitor func(games []*types.FaultGameData)

// HeadBlockFetcher returns the L1 snapshot used for a monitor cycle.
type HeadBlockFetcher func(ctx context.Context) (eth.L1BlockRef, error)

// Extract creates enriched game snapshots pinned to an L1 block.
type Extract func(ctx context.Context, blockHash common.Hash, minTimestamp uint64) ([]types.EnrichedGame, int, int, error)

type MonitorMetrics interface {
	RecordMonitorDuration(dur time.Duration)
}

type gameMonitor struct {
	logger  log.Logger
	clock   clock.Clock
	metrics MonitorMetrics

	done     chan struct{}
	stopped  chan struct{}
	stopOnce sync.Once
	ctx      context.Context
	cancel   context.CancelFunc
	loopWG   sync.WaitGroup

	gameWindow      time.Duration
	monitorInterval time.Duration

	forecast         ForecastResolution
	checkAnchorState AnchorStateCheck
	commonMonitors   []CommonMonitor
	faultMonitors    []FaultMonitor
	extract          Extract
	fetchHeadBlock   HeadBlockFetcher
}

func newGameMonitor(
	ctx context.Context,
	logger log.Logger,
	cl clock.Clock,
	metrics MonitorMetrics,
	monitorInterval time.Duration,
	gameWindow time.Duration,
	fetchHeadBlock HeadBlockFetcher,
	extract Extract,
	forecast ForecastResolution,
	checkAnchorState AnchorStateCheck,
	commonMonitors []CommonMonitor,
	faultMonitors []FaultMonitor,
) *gameMonitor {
	return &gameMonitor{
		logger:           logger,
		clock:            cl,
		ctx:              ctx,
		done:             make(chan struct{}),
		stopped:          make(chan struct{}),
		metrics:          metrics,
		monitorInterval:  monitorInterval,
		gameWindow:       gameWindow,
		forecast:         forecast,
		checkAnchorState: checkAnchorState,
		commonMonitors:   commonMonitors,
		faultMonitors:    faultMonitors,
		extract:          extract,
		fetchHeadBlock:   fetchHeadBlock,
	}
}

func (m *gameMonitor) monitorGames() error {
	start := m.clock.Now()
	headBlock, err := m.fetchHeadBlock(m.ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch block number: %w", err)
	}
	m.logger.Debug("Fetched current head block", "block", headBlock)
	minGameTimestamp := clock.MinCheckedTimestamp(m.clock, m.gameWindow)
	enrichedGames, ignored, failed, err := m.extract(m.ctx, headBlock.Hash, minGameTimestamp)
	if err != nil {
		return fmt.Errorf("failed to load games: %w", err)
	}
	commonGames, faultGames := partitionGames(enrichedGames)
	m.forecast(enrichedGames, ignored, failed)
	m.checkAnchorState(m.ctx, headBlock.Hash, commonGames)
	for _, monitor := range m.commonMonitors {
		monitor(commonGames)
	}
	for _, monitor := range m.faultMonitors {
		monitor(faultGames)
	}
	timeTaken := m.clock.Since(start)
	m.metrics.RecordMonitorDuration(timeTaken)
	m.logger.Info("Completed monitoring update",
		"blockNumber", headBlock.Number,
		"blockHash", headBlock.Hash,
		"duration", timeTaken,
		"games", len(enrichedGames),
		"ignored", ignored,
		"failed", failed)
	return nil
}

func partitionGames(games []types.EnrichedGame) ([]*types.CommonGameData, []*types.FaultGameData) {
	commonGames := make([]*types.CommonGameData, 0, len(games))
	faultGames := make([]*types.FaultGameData, 0, len(games))
	for _, game := range games {
		commonGames = append(commonGames, game.Common())
		if faultGame, ok := game.(*types.FaultGameData); ok {
			faultGames = append(faultGames, faultGame)
		}
	}
	return commonGames, faultGames
}

func (m *gameMonitor) loop() {
	defer m.loopWG.Done()
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
	m.loopWG.Add(1)
	go m.loop()
}

func (m *gameMonitor) StopMonitoring(ctx context.Context) error {
	m.logger.Info("Stopping game monitor")
	m.stopOnce.Do(func() {
		if m.cancel != nil {
			m.cancel()
		}
		close(m.done)
		go func() {
			m.loopWG.Wait()
			close(m.stopped)
		}()
	})
	select {
	case <-m.stopped:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("waiting for game monitor to stop: %w", ctx.Err())
	}
}
