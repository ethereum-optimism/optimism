package sync

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestScheduler(t *testing.T) {
	t.Run("ImmediateShutdown", func(t *testing.T) {
		runner := func(ctx context.Context, item int) {}
		s := NewSchedulerFromBufferSize(runner, 1)
		s.Start(context.Background())
		err := s.Close()
		require.NoError(t, err)
	})

	t.Run("Drain", func(t *testing.T) {
		runnerCalls := uint64(0)
		runner := func(ctx context.Context, item int) {
			atomic.AddUint64(&runnerCalls, 1)
			for {
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(10 * time.Millisecond)
				}
			}
		}
		s := NewSchedulerFromBufferSize(runner, 10)
		s.Start(context.Background())
		for i := 0; i < 10; i++ {
			err := s.Schedule(1)
			require.NoError(t, err)
		}
		require.Eventually(t, func() bool {
			return atomic.LoadUint64(&runnerCalls) > 0
		}, 10*time.Second, 10*time.Millisecond)
		require.Equal(t, 9, len(s.receiver))
		s.Drain()
		require.Equal(t, 0, len(s.receiver))
		s.Close()
		require.Equal(t, 0, len(s.receiver))
	})

	t.Run("ScheduleMessage", func(t *testing.T) {
		runnerCalls := 0
		runner := func(ctx context.Context, item int) {
			runnerCalls++
		}
		s := NewSchedulerFromBufferSize(runner, 1)
		s.Start(context.Background())
		err := s.Schedule(1)
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			return runnerCalls > 0
		}, 10*time.Second, 10*time.Millisecond)
	})

	t.Run("ScheduleMessageBufferFull", func(t *testing.T) {
		runnerCalls := 0
		runner := func(ctx context.Context, item int) {
			runnerCalls++
		}
		s := NewSchedulerFromBufferSize(runner, 1)
		s.Start(context.Background())
		err := s.Schedule(1)
		require.NoError(t, err)
		err = s.Schedule(2)
		require.Error(t, err)
		require.Eventually(t, func() bool {
			return runnerCalls > 0
		}, 10*time.Second, 10*time.Millisecond)
	})
}
