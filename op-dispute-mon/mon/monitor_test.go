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

func TestMonitor_MonitorGames(t *testing.T) {
	t.Parallel()

	t.Run("Fails to fetch block number", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t)
		boom := errors.New("boom")
		monitor.fetchBlockNumber = func(ctx context.Context) (uint64, error) {
			return 0, boom
		}
		err := monitor.monitorGames()
		require.ErrorIs(t, err, boom)
	})

	t.Run("Fails to fetch block hash", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t)
		boom := errors.New("boom")
		monitor.fetchBlockHash = func(ctx context.Context, number *big.Int) (common.Hash, error) {
			return common.Hash{}, boom
		}
		err := monitor.monitorGames()
		require.ErrorIs(t, err, boom)
	})

	t.Run("Record status success", func(t *testing.T) {
		monitor, source, _, _ := setupMonitorTest(t)
		source.games = []types.GameMetadata{{}, {}, {}}
		err := monitor.monitorGames()
		require.NoError(t, err)
	})
}

func TestMonitorGames(t *testing.T) {
	t.Run("MonitorsGames", func(t *testing.T) {
		addr1 := common.Address{0xaa}
		addr2 := common.Address{0xbb}
		monitor, source, _, detector := setupMonitorTest(t)
		source.games = []types.GameMetadata{newFDG(addr1, 9999), newFDG(addr2, 9999)}
		source.maxSuccess = 2 // Only allow two successful fetches

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return source.calls > 3
		}, 10*time.Second, 50*time.Millisecond)
		require.Equal(t, 2, detector.calls)
		monitor.StopMonitoring()
	})

	t.Run("FailsToFetch", func(t *testing.T) {
		monitor, source, metrics, _ := setupMonitorTest(t)
		source.fetchErr = errors.New("boom")

		monitor.StartMonitoring()
		require.Eventually(t, func() bool {
			return source.calls > 3
		}, 10*time.Second, 50*time.Millisecond)
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

func setupMonitorTest(t *testing.T) (*gameMonitor, *stubGameSource, *stubMonitorMetricer, *stubDetector) {
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
	detector := &stubDetector{}
	monitor := newGameMonitor(
		logger,
		metrics,
		clock.NewSimpleClock(),
		monitorInterval,
		source,
		time.Duration(10*time.Second),
		detector,
		fetchBlockNum,
		fetchBlockHash,
	)
	return monitor, source, metrics, detector
}

type stubDetector struct {
	calls int
}

func (s *stubDetector) Detect(_ context.Context, _ []types.GameMetadata) {
	s.calls++
}

type stubMonitorMetricer struct {
	inProgress    int
	defenderWon   int
	challengerWon int
	gameAgreement map[string]int
}

func (s *stubMonitorMetricer) RecordGamesStatus(inProgress, defenderWon, challengerWon int) {
	s.inProgress += inProgress
	s.defenderWon += defenderWon
	s.challengerWon += challengerWon
}

func (s *stubMonitorMetricer) RecordGameAgreement(status string, count int) {
	if s.gameAgreement == nil {
		s.gameAgreement = make(map[string]int)
	}
	s.gameAgreement[status] += count
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
		return nil, errors.New("max success")
	}
	return s.games, nil
}
