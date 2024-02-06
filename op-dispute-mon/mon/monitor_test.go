package mon

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
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
		monitor.clock = clock.NewSimpleClock()
		monitor.clock.SetTime(0)
		require.Equal(t, uint64(0), monitor.minGameTimestamp())
	})

	t.Run("minimum computed correctly", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t)
		monitor.gameWindow = time.Minute
		monitor.clock = clock.NewSimpleClock()
		frozen := uint64(time.Hour.Seconds())
		monitor.clock.SetTime(frozen)
		expected := uint64(time.Unix(int64(frozen), 0).Add(-time.Minute).Unix())
		require.Equal(t, monitor.minGameTimestamp(), expected)
	})
}

func TestMonitor_RecordGameStatus(t *testing.T) {
	t.Parallel()

	t.Run("in_progress", func(t *testing.T) {
		monitor, _, metrics, status := setupMonitorTest(t)
		status.status = types.GameStatusInProgress
		err := monitor.recordGameStatus(context.Background(), types.GameMetadata{})
		require.NoError(t, err)
		require.Equal(t, 1, metrics.inProgress)
		require.Equal(t, 0, metrics.defenderWon)
		require.Equal(t, 0, metrics.challengerWon)
	})

	t.Run("defender_won", func(t *testing.T) {
		monitor, _, metrics, status := setupMonitorTest(t)
		status.status = types.GameStatusDefenderWon
		err := monitor.recordGameStatus(context.Background(), types.GameMetadata{})
		require.NoError(t, err)
		require.Equal(t, 0, metrics.inProgress)
		require.Equal(t, 1, metrics.defenderWon)
		require.Equal(t, 0, metrics.challengerWon)
	})

	t.Run("challenger_won", func(t *testing.T) {
		monitor, _, metrics, status := setupMonitorTest(t)
		status.status = types.GameStatusChallengerWon
		err := monitor.recordGameStatus(context.Background(), types.GameMetadata{})
		require.NoError(t, err)
		require.Equal(t, 0, metrics.inProgress)
		require.Equal(t, 0, metrics.defenderWon)
		require.Equal(t, 1, metrics.challengerWon)
	})

	t.Run("status_error", func(t *testing.T) {
		monitor, _, metrics, status := setupMonitorTest(t)
		status.err = errors.New("boom")
		err := monitor.recordGameStatus(context.Background(), types.GameMetadata{})
		require.ErrorIs(t, err, status.err)
		require.Equal(t, 0, metrics.inProgress)
		require.Equal(t, 0, metrics.defenderWon)
		require.Equal(t, 0, metrics.challengerWon)
	})
}

func TestMonitor_MonitorGames(t *testing.T) {
	t.Parallel()

	t.Run("Fails to fetch block number", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t)
		boom := errors.New("boom")
		monitor.fetchBlockNumber = func(ctx context.Context) (uint64, error) {
			return 0, boom
		}
		err := monitor.monitorGames(context.Background())
		require.ErrorIs(t, err, boom)
	})

	t.Run("Fails to fetch block hash", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t)
		boom := errors.New("boom")
		monitor.fetchBlockHash = func(ctx context.Context, number *big.Int) (common.Hash, error) {
			return common.Hash{}, boom
		}
		err := monitor.monitorGames(context.Background())
		require.ErrorIs(t, err, boom)
	})

	t.Run("Fails to fetch games", func(t *testing.T) {
		monitor, source, _, status := setupMonitorTest(t)
		source.games = []types.GameMetadata{}
		err := monitor.monitorGames(context.Background())
		require.NoError(t, err)
		require.Equal(t, 0, status.calls) // No status loader should be created
	})

	t.Run("Record status loader errors gracefully", func(t *testing.T) {
		monitor, source, _, status := setupMonitorTest(t)
		source.games = []types.GameMetadata{{}}
		status.err = errors.New("boom")
		err := monitor.monitorGames(context.Background())
		require.NoError(t, err)
		require.Equal(t, 1, status.calls)
	})

	t.Run("Record status success", func(t *testing.T) {
		monitor, source, metrics, _ := setupMonitorTest(t)
		source.games = []types.GameMetadata{{}, {}, {}}
		err := monitor.monitorGames(context.Background())
		require.NoError(t, err)
		require.Equal(t, len(source.games), metrics.inProgress)
	})
}

func TestMonitorGames(t *testing.T) {
	t.Run("Monitors games", func(t *testing.T) {
		addr1 := common.Address{0xaa}
		addr2 := common.Address{0xbb}
		monitor, source, metrics, _ := setupMonitorTest(t)
		source.games = []types.GameMetadata{newFDG(addr1, 9999), newFDG(addr2, 9999)}
		source.maxSuccess = 2 // Only allow two successful fetches

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return metrics.inProgress == 2
		}, time.Second, 100*time.Millisecond)
		monitor.StopMonitoring()
		require.Equal(t, source.maxSuccess*len(source.games), metrics.inProgress) // Each game's status is recorded twice
	})

	t.Run("Fails to monitor games", func(t *testing.T) {
		monitor, source, metrics, _ := setupMonitorTest(t)
		source.fetchErr = errors.New("boom")

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return source.calls > 0
		}, time.Second, 100*time.Millisecond)
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

func setupMonitorTest(t *testing.T) (*gameMonitor, *stubGameSource, *stubMonitorMetricer, *mockStatusLoader) {
	logger := testlog.Logger(t, log.LvlDebug)
	source := &stubGameSource{}
	i := uint64(1)
	fetchBlockNum := func(ctx context.Context) (uint64, error) {
		i++
		return i, nil
	}
	fetchBlockHash := func(ctx context.Context, number *big.Int) (common.Hash, error) {
		return common.Hash{}, nil
	}
	metrics := &stubMonitorMetricer{}
	monitorInterval := time.Duration(100 * time.Millisecond)
	status := &mockStatusLoader{}
	rollupClient := &stubRollupClient{}
	monitor := newGameMonitor(
		logger,
		metrics,
		clock.NewSimpleClock(),
		monitorInterval,
		source,
		status,
		time.Duration(10*time.Second),
		rollupClient,
		fetchBlockNum,
		fetchBlockHash,
	)
	return monitor, source, metrics, status
}

type stubRollupClient struct {
	blockNum uint64
	err      error
}

func (s *stubRollupClient) OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	s.blockNum = blockNum
	return &eth.OutputResponse{OutputRoot: eth.Bytes32(common.HexToHash("0x10"))}, s.err
}

type mockStatusLoader struct {
	calls  int
	status types.GameStatus
	err    error
}

func (m *mockStatusLoader) GetStatus(ctx context.Context, _ common.Address) (types.GameStatus, error) {
	m.calls++
	if m.err != nil {
		return 0, m.err
	}
	return m.status, nil
}

func (m *mockStatusLoader) GetRootClaim(ctx context.Context, _ common.Address) (common.Hash, error) {
	return common.Hash{}, nil
}

func (m *mockStatusLoader) GetL2BlockNumber(ctx context.Context, _ common.Address) (uint64, error) {
	return 0, nil
}

type stubMonitorMetricer struct {
	inProgress    int
	defenderWon   int
	challengerWon int
}

func (s *stubMonitorMetricer) RecordGamesStatus(inProgress, defenderWon, challengerWon int) {
	s.inProgress += inProgress
	s.defenderWon += defenderWon
	s.challengerWon += challengerWon
}

func (s *stubMonitorMetricer) RecordGameAgreement(status string, count int) {
	panic("implement me")
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
	if s.fetchErr != nil || (s.calls > s.maxSuccess && s.maxSuccess != 0) {
		return nil, s.fetchErr
	}
	return s.games, nil
}
