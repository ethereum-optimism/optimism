package mon

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	mockErr = errors.New("mock error")
)

func TestMonitor_MonitorGames(t *testing.T) {
	t.Parallel()

	t.Run("FailedFetchHeadBlock", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t)
		boom := errors.New("boom")
		monitor.fetchHeadBlock = func(ctx context.Context) (eth.L1BlockRef, error) {
			return eth.L1BlockRef{}, boom
		}
		err := monitor.monitorGames()
		require.ErrorIs(t, err, boom)
	})

	t.Run("MonitorsWithNoGames", func(t *testing.T) {
		monitor, factory, forecast, monitors := setupMonitorTest(t)
		factory.games = []*monTypes.EnrichedGameData{}
		err := monitor.monitorGames()
		require.NoError(t, err)
		require.Equal(t, 1, forecast.Calls())
		for _, m := range monitors {
			require.Equal(t, 1, m.calls)
		}
	})

	t.Run("MonitorsMultipleGames", func(t *testing.T) {
		monitor, factory, forecast, monitors := setupMonitorTest(t)
		factory.games = []*monTypes.EnrichedGameData{{}, {}, {}}
		err := monitor.monitorGames()
		require.NoError(t, err)
		require.Equal(t, 1, forecast.Calls())
		for _, m := range monitors {
			require.Equal(t, 1, m.calls)
		}
	})
}

func TestMonitor_StartMonitoring(t *testing.T) {
	t.Run("MonitorsGames", func(t *testing.T) {
		addr1 := common.Address{0xaa}
		addr2 := common.Address{0xbb}
		monitor, factory, forecaster, _ := setupMonitorTest(t)
		factory.games = []*monTypes.EnrichedGameData{newEnrichedGameData(addr1, 9999), newEnrichedGameData(addr2, 9999)}
		factory.maxSuccess = len(factory.games) // Only allow two successful fetches

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return forecaster.Calls() >= 2
		}, time.Second, 50*time.Millisecond)
		monitor.StopMonitoring()
		require.Equal(t, len(factory.games), forecaster.Calls()) // Each game's status is recorded twice
	})

	t.Run("FailsToFetchGames", func(t *testing.T) {
		monitor, factory, forecaster, _ := setupMonitorTest(t)
		factory.fetchErr = errors.New("boom")

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return factory.calls > 0
		}, time.Second, 50*time.Millisecond)
		monitor.StopMonitoring()
		require.Equal(t, 0, forecaster.Calls())
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

func setupMonitorTest(t *testing.T) (*gameMonitor, *mockExtractor, *mockForecast, []*mockMonitor) {
	logger := testlog.Logger(t, log.LvlDebug)
	fetchHeadBlock := func(ctx context.Context) (eth.L1BlockRef, error) {
		return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
	}
	monitorInterval := 100 * time.Millisecond
	cl := clock.NewAdvancingClock(10 * time.Millisecond)
	cl.Start()
	extractor := &mockExtractor{}
	forecast := &mockForecast{}
	monitor1 := &mockMonitor{}
	monitor2 := &mockMonitor{}
	monitor3 := &mockMonitor{}
	monitor := newGameMonitor(context.Background(), logger, cl, metrics.NoopMetrics, monitorInterval, 10*time.Second, fetchHeadBlock,
		extractor.Extract, forecast.Forecast, monitor1.Check, monitor2.Check, monitor3.Check)
	return monitor, extractor, forecast, []*mockMonitor{monitor1, monitor2, monitor3}
}

type mockMonitor struct {
	calls int
}

func (m *mockMonitor) Check(games []*monTypes.EnrichedGameData) {
	m.calls++
}

type mockForecast struct {
	calls atomic.Int64
}

func (m *mockForecast) Calls() int {
	return int(m.calls.Load())
}

func (m *mockForecast) Forecast(_ []*monTypes.EnrichedGameData, _, _ int) {
	m.calls.Add(1)
}

type mockExtractor struct {
	fetchErr     error
	calls        int
	maxSuccess   int
	games        []*monTypes.EnrichedGameData
	ignoredCount int
	failedCount  int
}

func (m *mockExtractor) Extract(
	_ context.Context,
	_ common.Hash,
	_ uint64,
) ([]*monTypes.EnrichedGameData, int, int, error) {
	m.calls++
	if m.fetchErr != nil {
		return nil, 0, 0, m.fetchErr
	}
	if m.calls > m.maxSuccess && m.maxSuccess != 0 {
		return nil, 0, 0, mockErr
	}
	return m.games, m.ignoredCount, m.failedCount, nil
}

func TestMonitor_NodeEndpointErrorsMonitorIntegration(t *testing.T) {
	t.Parallel()

	t.Run("NodeEndpointErrorsMonitorCalledWithGamesData", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlDebug)
		fetchHeadBlock := func(ctx context.Context) (eth.L1BlockRef, error) {
			return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
		}
		monitorInterval := 100 * time.Millisecond
		cl := clock.NewAdvancingClock(10 * time.Millisecond)
		cl.Start()

		// Create games with endpoint errors
		games := []*monTypes.EnrichedGameData{
			{
				GameMetadata: types.GameMetadata{Proxy: common.Address{0x11}},
				NodeEndpointErrors: map[string]bool{
					"endpoint_1": true,
					"endpoint_2": true,
				},
			},
			{
				GameMetadata: types.GameMetadata{Proxy: common.Address{0x22}},
				NodeEndpointErrors: map[string]bool{
					"endpoint_2": true, // Overlapping with first game
					"endpoint_3": true,
				},
			},
		}

		extractor := &mockExtractor{games: games}
		forecast := &mockForecast{}
		nodeEndpointErrorsMetrics := &stubNodeEndpointErrorsMetrics{}
		nodeEndpointErrorsMonitor := NewNodeEndpointErrorsMonitor(logger, nodeEndpointErrorsMetrics)

		monitor := newGameMonitor(context.Background(), logger, cl, metrics.NoopMetrics, monitorInterval, 10*time.Second, fetchHeadBlock,
			extractor.Extract, forecast.Forecast, nodeEndpointErrorsMonitor.CheckNodeEndpointErrors)

		err := monitor.monitorGames()
		require.NoError(t, err)

		// Verify that NodeEndpointErrorsMonitor was called and recorded the correct count
		// Should count unique endpoints: endpoint_1, endpoint_2, endpoint_3 = 3 total
		require.Equal(t, 3, nodeEndpointErrorsMetrics.recordedCount)
	})
}

func TestMonitor_NodeEndpointErrorCountMonitorIntegration(t *testing.T) {
	t.Parallel()

	t.Run("NodeEndpointErrorCountMonitorCalledWithGamesData", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlDebug)
		fetchHeadBlock := func(ctx context.Context) (eth.L1BlockRef, error) {
			return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
		}
		monitorInterval := 100 * time.Millisecond
		cl := clock.NewAdvancingClock(10 * time.Millisecond)
		cl.Start()

		// Create games with endpoint error counts
		games := []*monTypes.EnrichedGameData{
			{
				GameMetadata:           types.GameMetadata{Proxy: common.Address{0x11}},
				NodeEndpointErrorCount: 5, // First game has 5 errors
			},
			{
				GameMetadata:           types.GameMetadata{Proxy: common.Address{0x22}},
				NodeEndpointErrorCount: 3, // Second game has 3 errors
			},
			{
				GameMetadata:           types.GameMetadata{Proxy: common.Address{0x33}},
				NodeEndpointErrorCount: 0, // Third game has no errors
			},
		}

		extractor := &mockExtractor{games: games}
		forecast := &mockForecast{}
		nodeEndpointErrorCountMetrics := &mockNodeEndpointErrorCountMetrics{}
		nodeEndpointErrorCountMonitor := NewNodeEndpointErrorCountMonitor(logger, nodeEndpointErrorCountMetrics)

		monitor := newGameMonitor(context.Background(), logger, cl, metrics.NoopMetrics, monitorInterval, 10*time.Second, fetchHeadBlock,
			extractor.Extract, forecast.Forecast, nodeEndpointErrorCountMonitor.CheckNodeEndpointErrorCount)

		err := monitor.monitorGames()
		require.NoError(t, err)

		// Verify that NodeEndpointErrorCountMonitor was called and recorded the correct total
		// Should sum all error counts: 5 + 3 + 0 = 8 total errors
		require.Equal(t, 8, nodeEndpointErrorCountMetrics.recordedCount)
	})
}

// mockNodeEndpointErrorCountMetrics for integration test
type mockNodeEndpointErrorCountMetrics struct {
	recordedCount int
}

func (m *mockNodeEndpointErrorCountMetrics) RecordNodeEndpointErrorCount(count int) {
	m.recordedCount = count
}

func TestMonitor_MixedAvailabilityMonitorIntegration(t *testing.T) {
	t.Parallel()

	t.Run("MixedAvailabilityMonitorCalledWithGamesData", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlDebug)
		fetchHeadBlock := func(ctx context.Context) (eth.L1BlockRef, error) {
			return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
		}
		monitorInterval := 100 * time.Millisecond
		cl := clock.NewAdvancingClock(10 * time.Millisecond)
		cl.Start()

		// Create games with mixed availability scenarios
		games := []*monTypes.EnrichedGameData{
			{
				GameMetadata:              types.GameMetadata{Proxy: common.Address{0x11}},
				NodeEndpointTotalCount:    3,
				NodeEndpointNotFoundCount: 1, // Mixed availability: some found, some not found
				NodeEndpointErrorCount:    0,
			},
			{
				GameMetadata:              types.GameMetadata{Proxy: common.Address{0x22}},
				NodeEndpointTotalCount:    2,
				NodeEndpointNotFoundCount: 2, // All endpoints not found - not mixed availability
				NodeEndpointErrorCount:    0,
			},
			{
				GameMetadata:              types.GameMetadata{Proxy: common.Address{0x33}},
				NodeEndpointTotalCount:    4,
				NodeEndpointNotFoundCount: 2, // Mixed availability: some found, some not found
				NodeEndpointErrorCount:    0,
			},
			{
				GameMetadata:              types.GameMetadata{Proxy: common.Address{0x44}},
				NodeEndpointTotalCount:    3,
				NodeEndpointNotFoundCount: 0, // All endpoints found - not mixed availability
				NodeEndpointErrorCount:    0,
			},
		}

		extractor := &mockExtractor{games: games}
		forecast := &mockForecast{}
		mixedAvailabilityMetrics := &mockMixedAvailabilityMetrics{}
		mixedAvailabilityMonitor := NewMixedAvailability(logger, mixedAvailabilityMetrics)

		monitor := newGameMonitor(context.Background(), logger, cl, metrics.NoopMetrics, monitorInterval, 10*time.Second, fetchHeadBlock,
			extractor.Extract, forecast.Forecast, mixedAvailabilityMonitor.CheckMixedAvailability)

		err := monitor.monitorGames()
		require.NoError(t, err)

		// Verify that MixedAvailabilityMonitor was called and recorded the correct count
		// Should count games with mixed availability: game 0x11 and 0x33 = 2 total
		require.Equal(t, 2, mixedAvailabilityMetrics.recordedCount)
	})
}

// mockMixedAvailabilityMetrics for integration test
type mockMixedAvailabilityMetrics struct {
	recordedCount int
}

func (m *mockMixedAvailabilityMetrics) RecordMixedAvailabilityGames(count int) {
	m.recordedCount = count
}

func TestMonitor_MixedSafetyMonitorIntegration(t *testing.T) {
	t.Parallel()

	t.Run("MixedSafetyMonitorCalledWithGamesData", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlDebug)
		fetchHeadBlock := func(ctx context.Context) (eth.L1BlockRef, error) {
			return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
		}
		monitorInterval := 100 * time.Millisecond
		cl := clock.NewAdvancingClock(10 * time.Millisecond)
		cl.Start()

		// Create games with mixed safety scenarios
		games := []*monTypes.EnrichedGameData{
			{
				GameMetadata:            types.GameMetadata{Proxy: common.Address{0x11}},
				NodeEndpointSafeCount:   2, // Mixed safety: some safe, some unsafe
				NodeEndpointUnsafeCount: 1,
			},
			{
				GameMetadata:            types.GameMetadata{Proxy: common.Address{0x22}},
				NodeEndpointSafeCount:   3, // All endpoints safe - not mixed safety
				NodeEndpointUnsafeCount: 0,
			},
			{
				GameMetadata:            types.GameMetadata{Proxy: common.Address{0x33}},
				NodeEndpointSafeCount:   1, // Mixed safety: some safe, some unsafe
				NodeEndpointUnsafeCount: 4,
			},
			{
				GameMetadata:            types.GameMetadata{Proxy: common.Address{0x44}},
				NodeEndpointSafeCount:   0, // All endpoints unsafe - not mixed safety
				NodeEndpointUnsafeCount: 2,
			},
			{
				GameMetadata:            types.GameMetadata{Proxy: common.Address{0x55}},
				NodeEndpointSafeCount:   0, // No safety checks performed - not mixed safety
				NodeEndpointUnsafeCount: 0,
			},
		}

		extractor := &mockExtractor{games: games}
		forecast := &mockForecast{}
		mixedSafetyMetrics := &mockMixedSafetyMetrics{}
		mixedSafetyMonitor := NewMixedSafetyMonitor(logger, mixedSafetyMetrics)

		monitor := newGameMonitor(context.Background(), logger, cl, metrics.NoopMetrics, monitorInterval, 10*time.Second, fetchHeadBlock,
			extractor.Extract, forecast.Forecast, mixedSafetyMonitor.CheckMixedSafety)

		err := monitor.monitorGames()
		require.NoError(t, err)

		// Verify that MixedSafetyMonitor was called and recorded the correct count
		// Should count games with mixed safety: game 0x11 and 0x33 = 2 total
		require.Equal(t, 2, mixedSafetyMetrics.recordedCount)
	})

	t.Run("OnlyGamesWithMixedSafetyAreCounted", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlDebug)
		fetchHeadBlock := func(ctx context.Context) (eth.L1BlockRef, error) {
			return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
		}
		monitorInterval := 100 * time.Millisecond
		cl := clock.NewAdvancingClock(10 * time.Millisecond)
		cl.Start()

		// Create games without mixed safety
		games := []*monTypes.EnrichedGameData{
			{
				GameMetadata:            types.GameMetadata{Proxy: common.Address{0x11}},
				NodeEndpointSafeCount:   5, // All safe
				NodeEndpointUnsafeCount: 0,
			},
			{
				GameMetadata:            types.GameMetadata{Proxy: common.Address{0x22}},
				NodeEndpointSafeCount:   0, // All unsafe
				NodeEndpointUnsafeCount: 3,
			},
			{
				GameMetadata:            types.GameMetadata{Proxy: common.Address{0x33}},
				NodeEndpointSafeCount:   0, // No checks performed
				NodeEndpointUnsafeCount: 0,
			},
		}

		extractor := &mockExtractor{games: games}
		forecast := &mockForecast{}
		mixedSafetyMetrics := &mockMixedSafetyMetrics{}
		mixedSafetyMonitor := NewMixedSafetyMonitor(logger, mixedSafetyMetrics)

		monitor := newGameMonitor(context.Background(), logger, cl, metrics.NoopMetrics, monitorInterval, 10*time.Second, fetchHeadBlock,
			extractor.Extract, forecast.Forecast, mixedSafetyMonitor.CheckMixedSafety)

		err := monitor.monitorGames()
		require.NoError(t, err)

		// Verify that no games were counted as having mixed safety
		require.Equal(t, 0, mixedSafetyMetrics.recordedCount)
	})

	t.Run("EdgeCaseMinimalMixedSafety", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlDebug)
		fetchHeadBlock := func(ctx context.Context) (eth.L1BlockRef, error) {
			return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
		}
		monitorInterval := 100 * time.Millisecond
		cl := clock.NewAdvancingClock(10 * time.Millisecond)
		cl.Start()

		// Create a game with minimal mixed safety (1 safe, 1 unsafe)
		games := []*monTypes.EnrichedGameData{
			{
				GameMetadata:            types.GameMetadata{Proxy: common.Address{0x11}},
				NodeEndpointSafeCount:   1, // Minimal mixed safety
				NodeEndpointUnsafeCount: 1,
			},
		}

		extractor := &mockExtractor{games: games}
		forecast := &mockForecast{}
		mixedSafetyMetrics := &mockMixedSafetyMetrics{}
		mixedSafetyMonitor := NewMixedSafetyMonitor(logger, mixedSafetyMetrics)

		monitor := newGameMonitor(context.Background(), logger, cl, metrics.NoopMetrics, monitorInterval, 10*time.Second, fetchHeadBlock,
			extractor.Extract, forecast.Forecast, mixedSafetyMonitor.CheckMixedSafety)

		err := monitor.monitorGames()
		require.NoError(t, err)

		// Verify that the minimal mixed safety case is counted
		require.Equal(t, 1, mixedSafetyMetrics.recordedCount)
	})
}

// mockMixedSafetyMetrics for integration test
type mockMixedSafetyMetrics struct {
	recordedCount int
}

func (m *mockMixedSafetyMetrics) RecordMixedSafetyGames(count int) {
	m.recordedCount = count
}

func TestMonitor_DifferentOutputRootMonitorIntegration(t *testing.T) {
	t.Parallel()

	t.Run("DifferentOutputRootMonitorCalledWithGamesData", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlDebug)
		fetchHeadBlock := func(ctx context.Context) (eth.L1BlockRef, error) {
			return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
		}
		monitorInterval := 100 * time.Millisecond
		cl := clock.NewAdvancingClock(10 * time.Millisecond)
		cl.Start()

		// Create games with different output root scenarios
		games := []*monTypes.EnrichedGameData{
			{
				GameMetadata:               types.GameMetadata{Proxy: common.Address{0x11}},
				NodeEndpointDifferentRoots: true, // Has different output roots
			},
			{
				GameMetadata:               types.GameMetadata{Proxy: common.Address{0x22}},
				NodeEndpointDifferentRoots: false, // No disagreement
			},
			{
				GameMetadata:               types.GameMetadata{Proxy: common.Address{0x33}},
				NodeEndpointDifferentRoots: true, // Has different output roots
			},
			{
				GameMetadata:               types.GameMetadata{Proxy: common.Address{0x44}},
				NodeEndpointDifferentRoots: false, // No disagreement
			},
		}

		extractor := &mockExtractor{games: games}
		forecast := &mockForecast{}
		differentOutputRootMetrics := &mockDifferentOutputRootMetrics{}
		differentOutputRootMonitor := NewDifferentRootMonitor(logger, differentOutputRootMetrics)

		monitor := newGameMonitor(context.Background(), logger, cl, metrics.NoopMetrics, monitorInterval, 10*time.Second, fetchHeadBlock,
			extractor.Extract, forecast.Forecast, differentOutputRootMonitor.CheckDifferentRoots)

		err := monitor.monitorGames()
		require.NoError(t, err)

		// Verify that DifferentOutputRootMonitor was called and recorded the correct count
		// Should count games with different output roots: game 0x11 and 0x33 = 2 total
		require.Equal(t, 2, differentOutputRootMetrics.recordedCount)
	})

	t.Run("OnlyGamesWithDifferentOutputRootsAreCounted", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlDebug)
		fetchHeadBlock := func(ctx context.Context) (eth.L1BlockRef, error) {
			return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
		}
		monitorInterval := 100 * time.Millisecond
		cl := clock.NewAdvancingClock(10 * time.Millisecond)
		cl.Start()

		// Create games without different output roots
		games := []*monTypes.EnrichedGameData{
			{
				GameMetadata:               types.GameMetadata{Proxy: common.Address{0x11}},
				NodeEndpointDifferentRoots: false, // No disagreement
			},
			{
				GameMetadata:               types.GameMetadata{Proxy: common.Address{0x22}},
				NodeEndpointDifferentRoots: false, // No disagreement
			},
			{
				GameMetadata:               types.GameMetadata{Proxy: common.Address{0x33}},
				NodeEndpointDifferentRoots: false, // No disagreement
			},
		}

		extractor := &mockExtractor{games: games}
		forecast := &mockForecast{}
		differentOutputRootMetrics := &mockDifferentOutputRootMetrics{}
		differentOutputRootMonitor := NewDifferentRootMonitor(logger, differentOutputRootMetrics)

		monitor := newGameMonitor(context.Background(), logger, cl, metrics.NoopMetrics, monitorInterval, 10*time.Second, fetchHeadBlock,
			extractor.Extract, forecast.Forecast, differentOutputRootMonitor.CheckDifferentRoots)

		err := monitor.monitorGames()
		require.NoError(t, err)

		// Verify that no games were counted as having different output roots
		require.Equal(t, 0, differentOutputRootMetrics.recordedCount)
	})

	t.Run("AllGamesHaveDifferentOutputRoots", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlDebug)
		fetchHeadBlock := func(ctx context.Context) (eth.L1BlockRef, error) {
			return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
		}
		monitorInterval := 100 * time.Millisecond
		cl := clock.NewAdvancingClock(10 * time.Millisecond)
		cl.Start()

		// Create games where all have different output roots
		games := []*monTypes.EnrichedGameData{
			{
				GameMetadata:               types.GameMetadata{Proxy: common.Address{0x11}},
				NodeEndpointDifferentRoots: true,
			},
			{
				GameMetadata:               types.GameMetadata{Proxy: common.Address{0x22}},
				NodeEndpointDifferentRoots: true,
			},
			{
				GameMetadata:               types.GameMetadata{Proxy: common.Address{0x33}},
				NodeEndpointDifferentRoots: true,
			},
		}

		extractor := &mockExtractor{games: games}
		forecast := &mockForecast{}
		differentOutputRootMetrics := &mockDifferentOutputRootMetrics{}
		differentOutputRootMonitor := NewDifferentRootMonitor(logger, differentOutputRootMetrics)

		monitor := newGameMonitor(context.Background(), logger, cl, metrics.NoopMetrics, monitorInterval, 10*time.Second, fetchHeadBlock,
			extractor.Extract, forecast.Forecast, differentOutputRootMonitor.CheckDifferentRoots)

		err := monitor.monitorGames()
		require.NoError(t, err)

		// Verify that all games were counted
		require.Equal(t, 3, differentOutputRootMetrics.recordedCount)
	})

	t.Run("EmptyGamesListReturnsZero", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlDebug)
		fetchHeadBlock := func(ctx context.Context) (eth.L1BlockRef, error) {
			return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
		}
		monitorInterval := 100 * time.Millisecond
		cl := clock.NewAdvancingClock(10 * time.Millisecond)
		cl.Start()

		// Create empty games list
		games := []*monTypes.EnrichedGameData{}

		extractor := &mockExtractor{games: games}
		forecast := &mockForecast{}
		differentOutputRootMetrics := &mockDifferentOutputRootMetrics{}
		differentOutputRootMonitor := NewDifferentRootMonitor(logger, differentOutputRootMetrics)

		monitor := newGameMonitor(context.Background(), logger, cl, metrics.NoopMetrics, monitorInterval, 10*time.Second, fetchHeadBlock,
			extractor.Extract, forecast.Forecast, differentOutputRootMonitor.CheckDifferentRoots)

		err := monitor.monitorGames()
		require.NoError(t, err)

		// Verify that count is zero
		require.Equal(t, 0, differentOutputRootMetrics.recordedCount)
	})
}

// mockDifferentOutputRootMetrics for integration test
type mockDifferentOutputRootMetrics struct {
	recordedCount int
}

func (m *mockDifferentOutputRootMetrics) RecordDifferentRootGames(count int) {
	m.recordedCount = count
}
