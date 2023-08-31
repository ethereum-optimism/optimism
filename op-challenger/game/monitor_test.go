package game

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestMonitorMinGameTimestamp(t *testing.T) {
	t.Parallel()

	t.Run("zero game window returns zero", func(t *testing.T) {
		monitor, _, _ := setupMonitorTest(t, []common.Address{})
		monitor.gameWindow = time.Duration(0)
		require.Equal(t, monitor.minGameTimestamp(), uint64(0))
	})

	t.Run("non-zero game window with zero clock", func(t *testing.T) {
		monitor, _, _ := setupMonitorTest(t, []common.Address{})
		monitor.gameWindow = time.Minute
		monitor.clock = clock.NewDeterministicClock(time.Unix(0, 0))
		require.Equal(t, monitor.minGameTimestamp(), uint64(0))
	})

	t.Run("minimum computed correctly", func(t *testing.T) {
		monitor, _, _ := setupMonitorTest(t, []common.Address{})
		monitor.gameWindow = time.Minute
		frozen := time.Unix(int64(time.Hour.Seconds()), 0)
		monitor.clock = clock.NewDeterministicClock(frozen)
		expected := uint64(frozen.Add(-time.Minute).Unix())
		require.Equal(t, monitor.minGameTimestamp(), expected)
	})
}

func TestMonitorExitsWhenContextDone(t *testing.T) {
	monitor, _, _ := setupMonitorTest(t, []common.Address{{}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := monitor.MonitorGames(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestMonitorCreateAndProgressGameAgents(t *testing.T) {
	monitor, source, sched := setupMonitorTest(t, []common.Address{})

	addr1 := common.Address{0xaa}
	addr2 := common.Address{0xbb}
	source.games = []FaultDisputeGame{
		{
			Proxy:     addr1,
			Timestamp: 9999,
		},
		{
			Proxy:     addr2,
			Timestamp: 9999,
		},
	}

	require.NoError(t, monitor.progressGames(context.Background(), uint64(1)))

	require.Len(t, sched.scheduled, 1)
	require.Equal(t, []common.Address{addr1, addr2}, sched.scheduled[0])
}

func TestMonitorOnlyScheduleSpecifiedGame(t *testing.T) {
	addr1 := common.Address{0xaa}
	addr2 := common.Address{0xbb}
	monitor, source, sched := setupMonitorTest(t, []common.Address{addr2})

	source.games = []FaultDisputeGame{
		{
			Proxy:     addr1,
			Timestamp: 9999,
		},
		{
			Proxy:     addr2,
			Timestamp: 9999,
		},
	}

	require.NoError(t, monitor.progressGames(context.Background(), uint64(1)))

	require.Len(t, sched.scheduled, 1)
	require.Equal(t, []common.Address{addr2}, sched.scheduled[0])
}

func setupMonitorTest(t *testing.T, allowedGames []common.Address) (*gameMonitor, *stubGameSource, *stubScheduler) {
	logger := testlog.Logger(t, log.LvlDebug)
	source := &stubGameSource{}
	i := uint64(1)
	fetchBlockNum := func(ctx context.Context) (uint64, error) {
		i++
		return i, nil
	}
	sched := &stubScheduler{}
	monitor := newGameMonitor(logger, clock.SystemClock, source, sched, time.Duration(0), fetchBlockNum, allowedGames)
	return monitor, source, sched
}

type stubGameSource struct {
	games []FaultDisputeGame
}

func (s *stubGameSource) FetchAllGamesAtBlock(ctx context.Context, earliest uint64, blockNumber *big.Int) ([]FaultDisputeGame, error) {
	return s.games, nil
}

type stubScheduler struct {
	scheduled [][]common.Address
}

func (s *stubScheduler) Schedule(games []common.Address) error {
	s.scheduled = append(s.scheduled, games)
	return nil
}
