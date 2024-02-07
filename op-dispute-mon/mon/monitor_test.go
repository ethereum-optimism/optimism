package mon

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
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

	t.Run("zero game window returns zero", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t)
		monitor.gameWindow = time.Duration(0)
		require.Equal(t, monitor.minGameTimestamp(), uint64(0))
	})

	t.Run("non-zero game window with zero clock", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t)
		monitor.gameWindow = time.Minute
		monitor.clock = clock.NewDeterministicClock(time.Unix(0, 0))
		require.Equal(t, uint64(0), monitor.minGameTimestamp())
	})

	t.Run("minimum computed correctly", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t)
		monitor.gameWindow = time.Minute
		frozen := time.Unix(int64(time.Hour.Seconds()), 0)
		monitor.clock = clock.NewDeterministicClock(frozen)
		expected := uint64(frozen.Add(-time.Minute).Unix())
		require.Equal(t, monitor.minGameTimestamp(), expected)
	})
}

func TestMonitor_RecordGamesStatus(t *testing.T) {
	tests := []struct {
		name    string
		games   []types.GameMetadata
		status  func(loader *mockMetadataLoader)
		creator func(creator *mockMetadataCreator)
		metrics func(m *stubMonitorMetricer)
	}{
		{
			name:  "NoGames",
			games: []types.GameMetadata{},
			metrics: func(m *stubMonitorMetricer) {
				require.Equal(t, 0, m.inProgress)
				require.Equal(t, 0, m.defenderWon)
				require.Equal(t, 0, m.challengerWon)
			},
		},
		{
			name:  "InProgress",
			games: []types.GameMetadata{{}},
			metrics: func(m *stubMonitorMetricer) {
				require.Equal(t, 1, m.inProgress)
				require.Equal(t, 0, m.defenderWon)
				require.Equal(t, 0, m.challengerWon)
			},
		},
		{
			name:  "DefenderWon",
			games: []types.GameMetadata{{}},
			status: func(loader *mockMetadataLoader) {
				loader.status = types.GameStatusDefenderWon
			},
			metrics: func(m *stubMonitorMetricer) {
				require.Equal(t, 0, m.inProgress)
				require.Equal(t, 1, m.defenderWon)
				require.Equal(t, 0, m.challengerWon)
			},
		},
		{
			name:  "ChallengerWon",
			games: []types.GameMetadata{{}},
			status: func(loader *mockMetadataLoader) {
				loader.status = types.GameStatusChallengerWon
			},
			metrics: func(m *stubMonitorMetricer) {
				require.Equal(t, 0, m.inProgress)
				require.Equal(t, 0, m.defenderWon)
				require.Equal(t, 1, m.challengerWon)
			},
		},
		{
			name:  "MetadataLoaderError",
			games: []types.GameMetadata{{}},
			status: func(loader *mockMetadataLoader) {
				loader.err = mockErr
			},
			metrics: func(m *stubMonitorMetricer) {
				require.Equal(t, 0, m.inProgress)
				require.Equal(t, 0, m.defenderWon)
				require.Equal(t, 0, m.challengerWon)
			},
		},
		{
			name:    "MetadataCreatorError",
			games:   []types.GameMetadata{{}},
			creator: func(creator *mockMetadataCreator) { creator.err = mockErr },
			metrics: func(m *stubMonitorMetricer) {
				require.Equal(t, 0, m.inProgress)
				require.Equal(t, 0, m.defenderWon)
				require.Equal(t, 0, m.challengerWon)
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			monitor, _, metrics, creator := setupMonitorTest(t)
			if test.status != nil {
				test.status(creator.loader)
			}
			if test.creator != nil {
				test.creator(creator)
			}
			err := monitor.recordGamesStatus(context.Background(), test.games)
			require.NoError(t, err) // All errors are handled gracefully
			test.metrics(metrics)
		})
	}
}

func TestMonitor_MonitorGames(t *testing.T) {
	t.Parallel()

	t.Run("FailedFetchBlocknumber", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t)
		boom := errors.New("boom")
		monitor.fetchBlockNumber = func(ctx context.Context) (uint64, error) {
			return 0, boom
		}
		err := monitor.monitorGames()
		require.ErrorIs(t, err, boom)
	})

	t.Run("FailedFetchBlockHash", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t)
		boom := errors.New("boom")
		monitor.fetchBlockHash = func(ctx context.Context, number *big.Int) (common.Hash, error) {
			return common.Hash{}, boom
		}
		err := monitor.monitorGames()
		require.ErrorIs(t, err, boom)
	})

	t.Run("NoGames", func(t *testing.T) {
		monitor, source, _, creator := setupMonitorTest(t)
		source.games = []types.GameMetadata{}
		err := monitor.monitorGames()
		require.NoError(t, err)
		require.Equal(t, 0, creator.calls)
	})

	t.Run("CreatorErrorsHandled", func(t *testing.T) {
		monitor, source, _, creator := setupMonitorTest(t)
		source.games = []types.GameMetadata{{}}
		creator.err = errors.New("boom")
		err := monitor.monitorGames()
		require.NoError(t, err)
		require.Equal(t, 1, creator.calls)
	})

	t.Run("Success", func(t *testing.T) {
		monitor, source, metrics, _ := setupMonitorTest(t)
		source.games = []types.GameMetadata{{}, {}, {}}
		err := monitor.monitorGames()
		require.NoError(t, err)
		require.Equal(t, len(source.games), metrics.inProgress)
	})
}

func TestMonitor_StartMonitoring(t *testing.T) {
	t.Run("Monitors games", func(t *testing.T) {
		addr1 := common.Address{0xaa}
		addr2 := common.Address{0xbb}
		monitor, source, metrics, _ := setupMonitorTest(t)
		source.games = []types.GameMetadata{newFDG(addr1, 9999), newFDG(addr2, 9999)}
		source.maxSuccess = len(source.games) // Only allow two successful fetches

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return metrics.inProgress == 2
		}, time.Second, 50*time.Millisecond)
		monitor.StopMonitoring()
		require.Equal(t, len(source.games), metrics.inProgress) // Each game's status is recorded twice
	})

	t.Run("Fails to monitor games", func(t *testing.T) {
		monitor, source, metrics, _ := setupMonitorTest(t)
		source.fetchErr = errors.New("boom")

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return source.calls > 0
		}, time.Second, 50*time.Millisecond)
		monitor.StopMonitoring()
		require.Equal(t, 0, metrics.inProgress)
		require.Equal(t, 0, metrics.defenderWon)
		require.Equal(t, 0, metrics.challengerWon)
	})
}

func newFDG(proxy common.Address, timestamp uint64) types.GameMetadata {
	return types.GameMetadata{
		Proxy:     proxy,
		Timestamp: timestamp,
	}
}

func setupMonitorTest(t *testing.T) (*gameMonitor, *stubGameSource, *stubMonitorMetricer, *mockMetadataCreator) {
	logger := testlog.Logger(t, log.LvlDebug)
	source := &stubGameSource{}
	fetchBlockNum := func(ctx context.Context) (uint64, error) {
		return 1, nil
	}
	fetchBlockHash := func(ctx context.Context, number *big.Int) (common.Hash, error) {
		return common.Hash{}, nil
	}
	metrics := &stubMonitorMetricer{}
	monitorInterval := time.Duration(100 * time.Millisecond)
	loader := &mockMetadataLoader{}
	creator := &mockMetadataCreator{loader: loader}
	cl := clock.NewAdvancingClock(10 * time.Millisecond)
	cl.Start()
	monitor := newGameMonitor(
		context.Background(),
		logger,
		metrics,
		cl,
		monitorInterval,
		source,
		creator,
		time.Duration(10*time.Second),
		fetchBlockNum,
		fetchBlockHash,
	)
	return monitor, source, metrics, creator
}

type mockMetadataCreator struct {
	calls  int
	err    error
	loader *mockMetadataLoader
}

func (m *mockMetadataCreator) CreateContract(game types.GameMetadata) (MetadataLoader, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.loader, nil
}

type mockMetadataLoader struct {
	calls  int
	status types.GameStatus
	err    error
}

func (m *mockMetadataLoader) GetGameMetadata(ctx context.Context) (uint64, common.Hash, types.GameStatus, error) {
	m.calls++
	if m.err != nil {
		return 0, common.Hash{}, m.status, m.err
	}
	return 0, common.Hash{}, m.status, nil
}

type stubMonitorMetricer struct {
	inProgress    int
	defenderWon   int
	challengerWon int
}

func (s *stubMonitorMetricer) RecordGamesStatus(inProgress, defenderWon, challengerWon int) {
	s.inProgress = inProgress
	s.defenderWon = defenderWon
	s.challengerWon = challengerWon
}

type stubGameSource struct {
	fetchErr   error
	calls      int
	maxSuccess int
	games      []types.GameMetadata
}

func (s *stubGameSource) GetGamesAtOrAfter(
	_ context.Context,
	_ common.Hash,
	_ uint64,
) ([]types.GameMetadata, error) {
	s.calls++
	if s.fetchErr != nil {
		return nil, s.fetchErr
	}
	if s.calls > s.maxSuccess && s.maxSuccess != 0 {
		return nil, mockErr
	}
	return s.games, nil
}
