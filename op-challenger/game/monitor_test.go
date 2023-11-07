package game

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

func TestMonitorMinGameTimestamp(t *testing.T) {
	t.Parallel()

	t.Run("zero game window returns zero", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t, []common.Address{})
		monitor.gameWindow = time.Duration(0)
		require.Equal(t, monitor.minGameTimestamp(), uint64(0))
	})

	t.Run("non-zero game window with zero clock", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t, []common.Address{})
		monitor.gameWindow = time.Minute
		monitor.clock = clock.NewDeterministicClock(time.Unix(0, 0))
		require.Equal(t, monitor.minGameTimestamp(), uint64(0))
	})

	t.Run("minimum computed correctly", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t, []common.Address{})
		monitor.gameWindow = time.Minute
		frozen := time.Unix(int64(time.Hour.Seconds()), 0)
		monitor.clock = clock.NewDeterministicClock(frozen)
		expected := uint64(frozen.Add(-time.Minute).Unix())
		require.Equal(t, monitor.minGameTimestamp(), expected)
	})
}

// TestMonitorGames tests that the monitor can handle a new head event
// and resubscribe to new heads if the subscription errors.
func TestMonitorGames(t *testing.T) {
	t.Run("Schedules games", func(t *testing.T) {
		addr1 := common.Address{0xaa}
		addr2 := common.Address{0xbb}
		monitor, source, sched, mockHeadSource := setupMonitorTest(t, []common.Address{})
		source.games = []types.GameMetadata{newFDG(addr1, 9999), newFDG(addr2, 9999)}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			headerNotSent := true
			for {
				if len(sched.scheduled) >= 1 {
					break
				}
				if mockHeadSource.sub == nil {
					continue
				}
				if headerNotSent {
					select {
					case mockHeadSource.sub.headers <- &ethtypes.Header{
						Number: big.NewInt(1),
					}:
						headerNotSent = false
					case <-ctx.Done():
						break
					default:
					}
				}
				// Just to avoid a tight loop
				time.Sleep(100 * time.Millisecond)
			}
			mockHeadSource.err = fmt.Errorf("eth subscribe test error")
			cancel()
		}()

		err := monitor.MonitorGames(ctx)
		require.NoError(t, err)
		require.Len(t, sched.scheduled, 1)
		require.Equal(t, []common.Address{addr1, addr2}, sched.scheduled[0])
	})

	t.Run("Resubscribes on error", func(t *testing.T) {
		addr1 := common.Address{0xaa}
		addr2 := common.Address{0xbb}
		monitor, source, sched, mockHeadSource := setupMonitorTest(t, []common.Address{})
		source.games = []types.GameMetadata{newFDG(addr1, 9999), newFDG(addr2, 9999)}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			// Wait for the subscription to be created
			waitErr := wait.For(context.Background(), 5*time.Second, func() (bool, error) {
				return mockHeadSource.sub != nil, nil
			})
			require.NoError(t, waitErr)
			mockHeadSource.sub.errChan <- fmt.Errorf("test error")
			for {
				if len(sched.scheduled) >= 1 {
					break
				}
				if mockHeadSource.sub == nil {
					continue
				}
				select {
				case mockHeadSource.sub.headers <- &ethtypes.Header{
					Number: big.NewInt(1),
				}:
				case <-ctx.Done():
					break
				default:
				}
				// Just to avoid a tight loop
				time.Sleep(100 * time.Millisecond)
			}
			require.NoError(t, waitErr)
			mockHeadSource.err = fmt.Errorf("eth subscribe test error")
			cancel()
		}()

		err := monitor.MonitorGames(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, sched.scheduled) // We might get more than one update scheduled.
		require.Equal(t, []common.Address{addr1, addr2}, sched.scheduled[0])
	})
}

func TestMonitorCreateAndProgressGameAgents(t *testing.T) {
	monitor, source, sched, _ := setupMonitorTest(t, []common.Address{})

	addr1 := common.Address{0xaa}
	addr2 := common.Address{0xbb}
	source.games = []types.GameMetadata{newFDG(addr1, 9999), newFDG(addr2, 9999)}

	require.NoError(t, monitor.progressGames(context.Background(), uint64(1)))

	require.Len(t, sched.scheduled, 1)
	require.Equal(t, []common.Address{addr1, addr2}, sched.scheduled[0])
}

func TestMonitorOnlyScheduleSpecifiedGame(t *testing.T) {
	addr1 := common.Address{0xaa}
	addr2 := common.Address{0xbb}
	monitor, source, sched, _ := setupMonitorTest(t, []common.Address{addr2})
	source.games = []types.GameMetadata{newFDG(addr1, 9999), newFDG(addr2, 9999)}

	require.NoError(t, monitor.progressGames(context.Background(), uint64(1)))

	require.Len(t, sched.scheduled, 1)
	require.Equal(t, []common.Address{addr2}, sched.scheduled[0])
}

func newFDG(proxy common.Address, timestamp uint64) types.GameMetadata {
	return types.GameMetadata{
		Proxy:     proxy,
		Timestamp: timestamp,
	}
}

func setupMonitorTest(
	t *testing.T,
	allowedGames []common.Address,
) (*gameMonitor, *stubGameSource, *stubScheduler, *mockNewHeadSource) {
	logger := testlog.Logger(t, log.LvlDebug)
	source := &stubGameSource{}
	i := uint64(1)
	fetchBlockNum := func(ctx context.Context) (uint64, error) {
		i++
		return i, nil
	}
	sched := &stubScheduler{}
	mockHeadSource := &mockNewHeadSource{}
	monitor := newGameMonitor(
		logger,
		clock.SystemClock,
		source,
		sched,
		time.Duration(0),
		fetchBlockNum,
		allowedGames,
		mockHeadSource,
	)
	return monitor, source, sched, mockHeadSource
}

type mockNewHeadSource struct {
	sub *mockSubscription
	err error
}

func (m *mockNewHeadSource) EthSubscribe(
	ctx context.Context,
	ch any,
	args ...any,
) (ethereum.Subscription, error) {
	errChan := make(chan error)
	m.sub = &mockSubscription{errChan, (ch).(chan<- *ethtypes.Header)}
	if m.err != nil {
		return nil, m.err
	}
	return m.sub, nil
}

type mockSubscription struct {
	errChan chan error
	headers chan<- *ethtypes.Header
}

func (m *mockSubscription) Unsubscribe() {}

func (m *mockSubscription) Err() <-chan error {
	return m.errChan
}

type stubGameSource struct {
	games []types.GameMetadata
}

func (s *stubGameSource) FetchAllGamesAtBlock(
	ctx context.Context,
	earliest uint64,
	blockNumber uint64,
) ([]types.GameMetadata, error) {
	return s.games, nil
}

type stubScheduler struct {
	scheduled [][]common.Address
}

func (s *stubScheduler) Schedule(games []types.GameMetadata) error {
	var addrs []common.Address
	for _, game := range games {
		addrs = append(addrs, game.Proxy)
	}
	s.scheduled = append(s.scheduled, addrs)
	return nil
}
