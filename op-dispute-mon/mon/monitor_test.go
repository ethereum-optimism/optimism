package mon

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var mockErr = errors.New("mock error")

func TestMonitorMonitorGames(t *testing.T) {
	t.Run("failed fetch head block", func(t *testing.T) {
		monitor, _, _, _, _, _ := setupMonitorTest(t)
		boom := errors.New("boom")
		monitor.fetchHeadBlock = func(context.Context) (eth.L1BlockRef, error) {
			return eth.L1BlockRef{}, boom
		}
		require.ErrorIs(t, monitor.monitorGames(), boom)
	})

	t.Run("routes sealed variants", func(t *testing.T) {
		monitor, extractor, forecast, commonMonitor, faultMonitor, bondMonitor := setupMonitorTest(t)
		extractor.games = []monTypes.EnrichedGame{
			faultGame(gameTypes.GameStatusInProgress, true),
			&monTypes.ZKGameData{CommonGameData: commonGame(
				gameTypes.ZKDisputeGameType, gameTypes.GameStatusInProgress, true,
			)},
			&monTypes.SuperPermissionedGameData{CommonGameData: commonGame(
				gameTypes.SuperPermissionedGameType, gameTypes.GameStatusDefenderWon, true,
			)},
		}

		require.NoError(t, monitor.monitorGames())
		require.Equal(t, 1, forecast.Calls())
		require.Equal(t, 3, commonMonitor.gameCount)
		require.Equal(t, 1, faultMonitor.gameCount)
		require.Equal(t, 2, bondMonitor.gameCount)
	})

	t.Run("empty cycle still calls all consumers", func(t *testing.T) {
		monitor, _, forecast, commonMonitor, faultMonitor, bondMonitor := setupMonitorTest(t)
		require.NoError(t, monitor.monitorGames())
		require.Equal(t, 1, forecast.Calls())
		require.Equal(t, 1, commonMonitor.calls)
		require.Equal(t, 1, faultMonitor.calls)
		require.Equal(t, 1, bondMonitor.calls)
	})
}

func TestPartitionGamesOnlyReturnsFaultVariantsToFaultMonitors(t *testing.T) {
	fault := faultGame(gameTypes.GameStatusInProgress, true)
	zk := &monTypes.ZKGameData{CommonGameData: commonGame(
		gameTypes.ZKDisputeGameType, gameTypes.GameStatusInProgress, true,
	)}
	superPermissioned := &monTypes.SuperPermissionedGameData{CommonGameData: commonGame(
		gameTypes.SuperPermissionedGameType, gameTypes.GameStatusDefenderWon, true,
	)}

	commonGames, faultGames, bondedGames := partitionGames([]monTypes.EnrichedGame{fault, zk, superPermissioned})
	require.Len(t, commonGames, 3)
	require.Equal(t, []*monTypes.FaultGameData{fault}, faultGames)
	require.Equal(t, []monTypes.BondedGame{fault, zk}, bondedGames)
}

func TestMonitorStartAndStop(t *testing.T) {
	t.Run("monitors until extraction fails", func(t *testing.T) {
		monitor, extractor, forecast, _, _, _ := setupMonitorTest(t)
		extractor.games = []monTypes.EnrichedGame{faultGame(gameTypes.GameStatusInProgress, true)}
		extractor.maxSuccess = 2

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return forecast.Calls() >= 2
		}, time.Second, 20*time.Millisecond)
		require.NoError(t, monitor.StopMonitoring(stopContext(t)))
		require.Equal(t, 2, forecast.Calls())
	})

	t.Run("failed extraction is not forecast", func(t *testing.T) {
		monitor, extractor, forecast, _, _, _ := setupMonitorTest(t)
		extractor.fetchErr = errors.New("boom")

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return extractor.Calls() > 0
		}, time.Second, 20*time.Millisecond)
		require.NoError(t, monitor.StopMonitoring(stopContext(t)))
		require.Zero(t, forecast.Calls())
	})

	t.Run("stops before start", func(t *testing.T) {
		monitor, _, _, _, _, _ := setupMonitorTest(t)
		require.NoError(t, monitor.StopMonitoring(stopContext(t)))
		require.NoError(t, monitor.StopMonitoring(stopContext(t)))
	})

	t.Run("waits for in-flight monitor", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			monitor, _, _, _, _, _ := setupMonitorTest(t)
			monitor.clock.(*clock.AdvancingClock).Stop()
			cl := clock.NewDeterministicClock(time.Unix(0, 0))
			monitor.clock = cl
			entered := make(chan struct{}, 1)
			release := make(chan struct{})
			monitor.fetchHeadBlock = func(context.Context) (eth.L1BlockRef, error) {
				entered <- struct{}{}
				<-release
				return eth.L1BlockRef{}, nil
			}

			monitor.StartMonitoring()
			synctest.Wait()
			cl.AdvanceTime(monitor.monitorInterval)
			synctest.Wait()
			require.NotEmpty(t, entered)

			stopped := make(chan error, 1)
			go func() {
				stopped <- monitor.StopMonitoring(stopContext(t))
			}()
			synctest.Wait()
			select {
			case <-stopped:
				t.Fatal("stop returned before in-flight monitoring completed")
			default:
			}
			close(release)
			synctest.Wait()
			require.NoError(t, <-stopped)
		})
	})

	t.Run("honors stop context", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			monitor, _, _, _, _, _ := setupMonitorTest(t)
			monitor.clock.(*clock.AdvancingClock).Stop()
			cl := clock.NewDeterministicClock(time.Unix(0, 0))
			monitor.clock = cl
			entered := make(chan struct{}, 1)
			release := make(chan struct{})
			monitor.fetchHeadBlock = func(context.Context) (eth.L1BlockRef, error) {
				entered <- struct{}{}
				<-release
				return eth.L1BlockRef{}, nil
			}

			monitor.StartMonitoring()
			synctest.Wait()
			cl.AdvanceTime(monitor.monitorInterval)
			synctest.Wait()
			require.NotEmpty(t, entered)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			require.ErrorIs(t, monitor.StopMonitoring(ctx), context.Canceled)

			close(release)
			synctest.Wait()
			require.NoError(t, monitor.StopMonitoring(stopContext(t)))
		})
	})
}

func TestServiceStopStopsMonitoring(t *testing.T) {
	monitor, _, _, _, _, _ := setupMonitorTest(t)
	service := &Service{logger: monitor.logger, monitor: monitor}
	require.NoError(t, service.Start(context.Background()))
	require.NoError(t, service.Stop(stopContext(t)))
	select {
	case <-monitor.done:
	default:
		t.Fatal("service stop did not stop the game monitor")
	}
}

func stopContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	t.Cleanup(cancel)
	return ctx
}

func setupMonitorTest(t *testing.T) (*gameMonitor, *mockExtractor, *mockForecast, *mockCommonMonitor, *mockFaultMonitor, *mockBondMonitor) {
	logger := testlog.Logger(t, log.LvlDebug)
	cl := clock.NewAdvancingClock()
	cl.Start()
	t.Cleanup(cl.Stop)
	extractor := &mockExtractor{}
	forecast := &mockForecast{}
	commonMonitor := &mockCommonMonitor{}
	faultMonitor := &mockFaultMonitor{}
	bondMonitor := &mockBondMonitor{}
	monitor := newGameMonitor(
		context.Background(),
		logger,
		cl,
		metrics.NoopMetrics,
		100*time.Millisecond,
		10*time.Second,
		func(context.Context) (eth.L1BlockRef, error) {
			return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
		},
		extractor.Extract,
		forecast.Forecast,
		func(context.Context, common.Hash, []*monTypes.CommonGameData) {},
		[]CommonMonitor{commonMonitor.Check},
		[]FaultMonitor{faultMonitor.Check},
		[]BondMonitor{bondMonitor.Check},
	)
	return monitor, extractor, forecast, commonMonitor, faultMonitor, bondMonitor
}

type mockCommonMonitor struct {
	calls     int
	gameCount int
}

func (m *mockCommonMonitor) Check(games []*monTypes.CommonGameData) {
	m.calls++
	m.gameCount = len(games)
}

type mockFaultMonitor struct {
	calls     int
	gameCount int
}

type mockBondMonitor struct {
	calls     int
	gameCount int
}

func (m *mockBondMonitor) Check(games []monTypes.BondedGame) {
	m.calls++
	m.gameCount = len(games)
}

func (m *mockFaultMonitor) Check(games []*monTypes.FaultGameData) {
	m.calls++
	m.gameCount = len(games)
}

type mockForecast struct {
	calls atomic.Int64
}

func (m *mockForecast) Calls() int {
	return int(m.calls.Load())
}

func (m *mockForecast) Forecast([]monTypes.EnrichedGame, int, int) {
	m.calls.Add(1)
}

type mockExtractor struct {
	fetchErr     error
	calls        atomic.Int64
	maxSuccess   int
	games        []monTypes.EnrichedGame
	ignoredCount int
	failedCount  int
}

func (m *mockExtractor) Extract(context.Context, common.Hash, uint64) ([]monTypes.EnrichedGame, int, int, error) {
	calls := int(m.calls.Add(1))
	if m.fetchErr != nil {
		return nil, 0, 0, m.fetchErr
	}
	if calls > m.maxSuccess && m.maxSuccess != 0 {
		return nil, 0, 0, mockErr
	}
	return m.games, m.ignoredCount, m.failedCount, nil
}

func (m *mockExtractor) Calls() int {
	return int(m.calls.Load())
}
