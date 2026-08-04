package mon

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
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
		factory.games = []monTypes.EnrichedGame{}
		err := monitor.monitorGames()
		require.NoError(t, err)
		require.Equal(t, 1, forecast.Calls())
		for _, m := range monitors {
			require.Equal(t, 1, m.calls)
		}
	})

	t.Run("MonitorsMultipleGames", func(t *testing.T) {
		monitor, factory, forecast, monitors := setupMonitorTest(t)
		factory.games = []monTypes.EnrichedGame{newEnrichedGameData(common.Address{1}, 1), newEnrichedGameData(common.Address{2}, 2), newEnrichedGameData(common.Address{3}, 3)}
		err := monitor.monitorGames()
		require.NoError(t, err)
		require.Equal(t, 1, forecast.Calls())
		for _, m := range monitors {
			require.Equal(t, 1, m.calls)
		}
	})

	t.Run("RunsTypedMonitorLanesInOrder", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t)
		var calls []string
		monitor.bondMonitors = []BondMonitor{func([]monTypes.BondedGame) {
			calls = append(calls, "bond")
		}}
		monitor.faultMonitors = []FaultMonitor{func([]*monTypes.FaultGameData) {
			calls = append(calls, "fault")
		}}
		monitor.commonMonitors = []CommonMonitor{func([]*monTypes.CommonGameData) {
			calls = append(calls, "common")
		}}

		require.NoError(t, monitor.monitorGames())
		require.Equal(t, []string{"bond", "fault", "common"}, calls)
	})
}

func TestMonitorRoutesGamesToResolution(t *testing.T) {
	// Mutation killed: omitting a supported variant from the common lane or
	// routing SuperPermissioned through the Fault lane survives monitor unit tests.
	monitor, extractor, _, _ := setupMonitorTest(t)
	terminal := newEnrichedGameData(common.Address{0xaa}, 1)
	terminal.Status = types.GameStatusDefenderWon
	super := &monTypes.SuperPermissionedGameData{CommonGameData: monTypes.CommonGameData{Status: types.GameStatusDefenderWon}}
	extractor.games = []monTypes.EnrichedGame{terminal, super}
	var commonReceived []*monTypes.CommonGameData
	monitor.commonMonitors = []CommonMonitor{func(games []*monTypes.CommonGameData) {
		commonReceived = games
	}}
	var faultReceived []*monTypes.FaultGameData
	monitor.faultMonitors = []FaultMonitor{func(games []*monTypes.FaultGameData) {
		faultReceived = games
	}}
	var bondReceived []monTypes.BondedGame
	monitor.bondMonitors = []BondMonitor{func(games []monTypes.BondedGame) {
		bondReceived = games
	}}

	require.NoError(t, monitor.monitorGames())
	require.Equal(t, []*monTypes.CommonGameData{terminal.Common(), super.Common()}, commonReceived)
	require.Equal(t, []*monTypes.FaultGameData{terminal}, faultReceived)
	require.Equal(t, []monTypes.BondedGame{terminal}, bondReceived)
}

func TestPartitionGamesRejectsUnknownGameType(t *testing.T) {
	require.PanicsWithValue(t, "unsupported enriched game type <nil>", func() {
		partitionGames([]monTypes.EnrichedGame{nil})
	})
}

func TestMonitorChecksAnchorOnceOnEmptyLane(t *testing.T) {
	monitor, _, _, _ := setupMonitorTest(t)
	calls := 0
	monitor.checkAnchorState = func(_ context.Context, _ common.Hash, games []*monTypes.CommonGameData) {
		calls++
		require.Empty(t, games)
	}

	require.NoError(t, monitor.monitorGames())
	require.Equal(t, 1, calls)
}

func TestMonitor_StartMonitoring(t *testing.T) {
	t.Run("MonitorsGames", func(t *testing.T) {
		addr1 := common.Address{0xaa}
		addr2 := common.Address{0xbb}
		monitor, factory, forecaster, _ := setupMonitorTest(t)
		factory.games = []monTypes.EnrichedGame{newEnrichedGameData(addr1, 9999), newEnrichedGameData(addr2, 9999)}
		factory.maxSuccess = len(factory.games) // Only allow two successful fetches

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return forecaster.Calls() >= 2
		}, time.Second, 50*time.Millisecond)
		require.NoError(t, monitor.StopMonitoring(stopContext(t)))
		require.Equal(t, len(factory.games), forecaster.Calls()) // Each game's status is recorded twice
	})

	t.Run("FailsToFetchGames", func(t *testing.T) {
		monitor, factory, forecaster, _ := setupMonitorTest(t)
		factory.fetchErr = errors.New("boom")

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return factory.Calls() > 0
		}, time.Second, 50*time.Millisecond)
		require.NoError(t, monitor.StopMonitoring(stopContext(t)))
		require.Equal(t, 0, forecaster.Calls())
	})

	t.Run("WaitsForInFlightMonitor", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			monitor, _, _, _ := setupMonitorTest(t)
			monitor.clock.(*clock.AdvancingClock).Stop()
			cl := clock.NewDeterministicClock(time.Unix(0, 0))
			monitor.clock = cl
			entered := make(chan struct{}, 1)
			release := make(chan struct{})
			var fetchReturned atomic.Bool
			monitor.fetchHeadBlock = func(context.Context) (eth.L1BlockRef, error) {
				select {
				case entered <- struct{}{}:
				default:
				}
				<-release
				fetchReturned.Store(true)
				return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
			}

			monitor.StartMonitoring()
			synctest.Wait()
			cl.AdvanceTime(monitor.monitorInterval)
			synctest.Wait()
			select {
			case <-entered:
			default:
				t.Fatal("monitor did not start fetching the head block")
			}

			stopReturned := make(chan error, 1)
			go func() {
				stopReturned <- monitor.StopMonitoring(stopContext(t))
			}()
			synctest.Wait()

			select {
			case <-monitor.done:
			default:
				t.Fatal("monitor stop was not signaled")
			}
			select {
			case <-stopReturned:
				t.Fatal("monitor stop returned while a monitor operation was still in flight")
			default:
			}

			close(release)
			synctest.Wait()
			require.True(t, fetchReturned.Load(), "in-flight monitor operation did not complete")
			require.NoError(t, <-stopReturned, "monitor stop did not complete successfully")
		})
	})

	t.Run("StopsBeforeStart", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			monitor, _, _, _ := setupMonitorTest(t)
			monitor.clock.(*clock.AdvancingClock).Stop()
			stopReturned := make(chan error, 1)
			go func() {
				stopReturned <- monitor.StopMonitoring(stopContext(t))
			}()
			synctest.Wait()
			select {
			case err := <-stopReturned:
				require.NoError(t, err)
			default:
				t.Fatal("monitor stop blocked before monitoring started")
			}
			require.NoError(t, monitor.StopMonitoring(stopContext(t)), "second stop should be idempotent")
		})
	})

	t.Run("HonorsStopContext", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			monitor, _, _, _ := setupMonitorTest(t)
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
			select {
			case <-entered:
			default:
				t.Fatal("monitor did not start fetching the head block")
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			require.ErrorIs(t, monitor.StopMonitoring(ctx), context.Canceled)

			close(release)
			synctest.Wait()
			require.NoError(t, monitor.StopMonitoring(stopContext(t)))
		})
	})
}

func stopContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	t.Cleanup(cancel)
	return ctx
}

func newEnrichedGameData(proxy common.Address, timestamp uint64) *monTypes.FaultGameData {
	return &monTypes.FaultGameData{
		CommonGameData: monTypes.CommonGameData{
			GameMetadata: types.GameMetadata{
				Proxy:     proxy,
				Timestamp: timestamp,
			},
			Status: types.GameStatusInProgress,
		},
	}
}

func setupMonitorTest(t *testing.T) (*gameMonitor, *mockExtractor, *mockForecast, []*mockMonitor) {
	logger := testlog.Logger(t, log.LvlDebug)
	fetchHeadBlock := func(ctx context.Context) (eth.L1BlockRef, error) {
		return eth.L1BlockRef{Number: 1, Hash: common.Hash{0xaa}}, nil
	}
	monitorInterval := 100 * time.Millisecond
	cl := clock.NewAdvancingClock()
	cl.Start()
	extractor := &mockExtractor{}
	forecast := &mockForecast{}
	monitor1 := &mockMonitor{}
	monitor2 := &mockMonitor{}
	monitor3 := &mockMonitor{}
	monitor4 := &mockMonitor{}
	monitor := newGameMonitor(context.Background(), logger, cl, metrics.NoopMetrics, monitorInterval, 10*time.Second, fetchHeadBlock,
		extractor.Extract, forecast.Forecast, noopAnchorStateCheck,
		[]CommonMonitor{monitor1.CheckCommon, monitor2.CheckCommon},
		[]FaultMonitor{monitor3.CheckFault},
		[]BondMonitor{monitor4.CheckBond})
	return monitor, extractor, forecast, []*mockMonitor{monitor1, monitor2, monitor3, monitor4}
}

func noopAnchorStateCheck(_ context.Context, _ common.Hash, _ []*monTypes.CommonGameData) {}

type mockMonitor struct {
	calls int
}

func (m *mockMonitor) CheckCommon(_ []*monTypes.CommonGameData) {
	m.calls++
}

func (m *mockMonitor) CheckFault(_ []*monTypes.FaultGameData) {
	m.calls++
}

func (m *mockMonitor) CheckBond(_ []monTypes.BondedGame) {
	m.calls++
}

type mockForecast struct {
	calls atomic.Int64
}

func (m *mockForecast) Calls() int {
	return int(m.calls.Load())
}

func (m *mockForecast) Forecast(_ []monTypes.EnrichedGame, _, _ int) {
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

func (m *mockExtractor) Extract(
	_ context.Context,
	_ common.Hash,
	_ uint64,
) ([]monTypes.EnrichedGame, int, int, error) {
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

func TestServiceStopStopsMonitoring(t *testing.T) {
	monitor, _, _, _ := setupMonitorTest(t)
	service := &Service{logger: monitor.logger, monitor: monitor}

	require.NoError(t, service.Start(context.Background()))
	require.NoError(t, service.Stop(stopContext(t)))

	select {
	case <-monitor.done:
		// The monitor loop was stopped.
	default:
		t.Fatal("service stop did not stop the game monitor")
	}
}
