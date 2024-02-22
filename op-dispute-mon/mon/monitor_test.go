package mon

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	mockErr = errors.New("mock error")
)

func TestMonitor_MinGameTimestamp(t *testing.T) {
	t.Parallel()

	t.Run("ZeroGameWindow", func(t *testing.T) {
		monitor, _, _, _, _ := setupMonitorTest(t)
		monitor.gameWindow = time.Duration(0)
		require.Equal(t, monitor.minGameTimestamp(), uint64(0))
	})

	t.Run("ZeroClock", func(t *testing.T) {
		monitor, _, _, _, _ := setupMonitorTest(t)
		monitor.gameWindow = time.Minute
		monitor.clock = clock.NewDeterministicClock(time.Unix(0, 0))
		require.Equal(t, uint64(0), monitor.minGameTimestamp())
	})

	t.Run("ValidArithmetic", func(t *testing.T) {
		monitor, _, _, _, _ := setupMonitorTest(t)
		monitor.gameWindow = time.Minute
		frozen := time.Unix(int64(time.Hour.Seconds()), 0)
		monitor.clock = clock.NewDeterministicClock(frozen)
		expected := uint64(frozen.Add(-time.Minute).Unix())
		require.Equal(t, monitor.minGameTimestamp(), expected)
	})
}

func TestMonitor_MonitorGames(t *testing.T) {
	t.Parallel()

	t.Run("FailedFetchBlocknumber", func(t *testing.T) {
		monitor, _, _, _, _ := setupMonitorTest(t)
		boom := errors.New("boom")
		monitor.fetchBlockNumber = func(ctx context.Context) (uint64, error) {
			return 0, boom
		}
		err := monitor.monitorGames()
		require.ErrorIs(t, err, boom)
	})

	t.Run("FailedFetchBlockHash", func(t *testing.T) {
		monitor, _, _, _, _ := setupMonitorTest(t)
		boom := errors.New("boom")
		monitor.fetchBlockHash = func(ctx context.Context, number *big.Int) (common.Hash, error) {
			return common.Hash{}, boom
		}
		err := monitor.monitorGames()
		require.ErrorIs(t, err, boom)
	})

	t.Run("DetectsWithNoGames", func(t *testing.T) {
		monitor, factory, detector, forecast, delays := setupMonitorTest(t)
		factory.games = []*monTypes.EnrichedGameData{}
		err := monitor.monitorGames()
		require.NoError(t, err)
		require.Equal(t, 1, detector.calls)
		require.Equal(t, 1, forecast.calls)
		require.Equal(t, 1, delays.calls)
	})

	t.Run("DetectsMultipleGames", func(t *testing.T) {
		monitor, factory, detector, forecast, delays := setupMonitorTest(t)
		factory.games = []*monTypes.EnrichedGameData{{}, {}, {}}
		err := monitor.monitorGames()
		require.NoError(t, err)
		require.Equal(t, 1, detector.calls)
		require.Equal(t, 1, forecast.calls)
		require.Equal(t, 1, delays.calls)
	})
}

func TestMonitor_StartMonitoring(t *testing.T) {
	t.Run("MonitorsGames", func(t *testing.T) {
		addr1 := common.Address{0xaa}
		addr2 := common.Address{0xbb}
		monitor, factory, detector, _, _ := setupMonitorTest(t)
		factory.games = []*monTypes.EnrichedGameData{newEnrichedGameData(addr1, 9999), newEnrichedGameData(addr2, 9999)}
		factory.maxSuccess = len(factory.games) // Only allow two successful fetches

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return detector.calls >= 2
		}, time.Second, 50*time.Millisecond)
		monitor.StopMonitoring()
		require.Equal(t, len(factory.games), detector.calls) // Each game's status is recorded twice
	})

	t.Run("FailsToFetchGames", func(t *testing.T) {
		monitor, factory, detector, _, _ := setupMonitorTest(t)
		factory.fetchErr = errors.New("boom")

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return factory.calls > 0
		}, time.Second, 50*time.Millisecond)
		monitor.StopMonitoring()
		require.Equal(t, 0, detector.calls)
	})
}

func newEnrichedGameData(proxy common.Address, timestamp uint64) *monTypes.EnrichedGameData {
	return &monTypes.EnrichedGameData{
		GameMetadata: types.GameMetadata{
			Proxy:     proxy,
			Timestamp: timestamp,
		},
		Status: types.GameStatusInProgress,
	}
}

func setupMonitorTest(t *testing.T) (*gameMonitor, *mockExtractor, *mockDetector, *mockForecast, *mockDelayCalculator) {
	logger := testlog.Logger(t, log.LvlDebug)
	fetchBlockNum := func(ctx context.Context) (uint64, error) {
		return 1, nil
	}
	fetchBlockHash := func(ctx context.Context, number *big.Int) (common.Hash, error) {
		return common.Hash{}, nil
	}
	monitorInterval := time.Duration(100 * time.Millisecond)
	cl := clock.NewAdvancingClock(10 * time.Millisecond)
	cl.Start()
	extractor := &mockExtractor{}
	detect := &mockDetector{}
	forecast := &mockForecast{}
	delays := &mockDelayCalculator{}
	monitor := newGameMonitor(
		context.Background(),
		logger,
		cl,
		monitorInterval,
		time.Duration(10*time.Second),
		delays.RecordClaimResolutionDelayMax,
		detect.Detect,
		forecast.Forecast,
		extractor.Extract,
		fetchBlockNum,
		fetchBlockHash,
	)
	return monitor, extractor, detect, forecast, delays
}

type mockDelayCalculator struct {
	calls int
}

func (m *mockDelayCalculator) RecordClaimResolutionDelayMax(games []*monTypes.EnrichedGameData) {
	m.calls++
}

type mockForecast struct {
	calls int
}

func (m *mockForecast) Forecast(ctx context.Context, games []*monTypes.EnrichedGameData) {
	m.calls++
}

type mockDetector struct {
	calls int
}

func (m *mockDetector) Detect(ctx context.Context, games []*monTypes.EnrichedGameData) {
	m.calls++
}

type mockExtractor struct {
	fetchErr   error
	calls      int
	maxSuccess int
	games      []*monTypes.EnrichedGameData
}

func (m *mockExtractor) Extract(
	_ context.Context,
	_ common.Hash,
	_ uint64,
) ([]*monTypes.EnrichedGameData, error) {
	m.calls++
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	if m.calls > m.maxSuccess && m.maxSuccess != 0 {
		return nil, mockErr
	}
	return m.games, nil
}
