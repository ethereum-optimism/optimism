package scheduler

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestSchedulerProcessesGames(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	ctx := context.Background()
	createPlayer := func(g types.GameMetadata, dir string) (GamePlayer, error) {
		return &test.StubGamePlayer{}, nil
	}
	removeExceptCalls := make(chan []common.Address)
	disk := &trackingDiskManager{removeExceptCalls: removeExceptCalls}
	s := NewScheduler(logger, metrics.NoopMetrics, disk, 2, createPlayer)
	s.Start(ctx)

	gameAddr1 := common.Address{0xaa}
	gameAddr2 := common.Address{0xbb}
	gameAddr3 := common.Address{0xcc}
	games := asGames(gameAddr1, gameAddr2, gameAddr3)

	require.NoError(t, s.Schedule(games))

	// All jobs should be executed and completed, the last step being to clean up disk resources
	for i := 0; i < len(games); i++ {
		kept := <-removeExceptCalls
		require.Len(t, kept, len(games), "should keep all games")
		for _, game := range games {
			require.Containsf(t, kept, game.Proxy, "should keep game %v", game.Proxy)
		}
	}
	require.NoError(t, s.Close())
}

func TestReturnBusyWhenScheduleQueueFull(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	createPlayer := func(game types.GameMetadata, dir string) (GamePlayer, error) {
		return &test.StubGamePlayer{}, nil
	}
	removeExceptCalls := make(chan []common.Address)
	disk := &trackingDiskManager{removeExceptCalls: removeExceptCalls}
	s := NewScheduler(logger, metrics.NoopMetrics, disk, 2, createPlayer)

	// Scheduler not started - first call fills the queue
	require.NoError(t, s.Schedule(asGames(common.Address{0xaa})))

	// Second call should return busy
	err := s.Schedule(asGames(common.Address{0xaa}))
	require.ErrorIs(t, err, ErrBusy)
}

type trackingDiskManager struct {
	removeExceptCalls chan []common.Address
}

func (t *trackingDiskManager) DirForGame(addr common.Address) string {
	return addr.Hex()
}

func (t *trackingDiskManager) RemoveAllExcept(addrs []common.Address) error {
	t.removeExceptCalls <- addrs
	return nil
}
