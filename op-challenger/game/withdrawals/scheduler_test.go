package withdrawals

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestScheduler_Schedule(t *testing.T) {
	tests := []struct {
		name                string
		deleteErr           error
		expectedMetricCalls int
	}{
		{name: "Succeeds"},
		{name: "Fails", deleteErr: errors.New("mock delete error"), expectedMetricCalls: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metrics := &stubSchedulerMetrics{}
			deleter := &stubDeleter{err: test.deleteErr}
			scheduler := NewScheduler(testlog.Logger(t, log.LevelInfo), metrics, deleter)
			scheduler.Start(context.Background())
			defer scheduler.Close()

			require.NoError(t, scheduler.Schedule(1, []types.GameMetadata{{}, {}}))
			require.Eventually(t, func() bool {
				return deleter.calls.Load() == 1 && int(metrics.failedCalls.Load()) == test.expectedMetricCalls
			}, 10*time.Second, 10*time.Millisecond)
		})
	}
}

func TestScheduler_DropsOverlappingRuns(t *testing.T) {
	metrics := &stubSchedulerMetrics{}
	deleter := &stubDeleter{}
	scheduler := NewScheduler(testlog.Logger(t, log.LevelInfo), metrics, deleter)

	// Not started, so nothing consumes the channel and only the first message is accepted.
	require.NoError(t, scheduler.Schedule(1, nil))
	require.NoError(t, scheduler.Schedule(2, nil))
	require.Len(t, scheduler.ch, 1)
	require.Equal(t, uint64(1), (<-scheduler.ch).blockNumber)
}

type stubSchedulerMetrics struct {
	failedCalls atomic.Int64
}

func (s *stubSchedulerMetrics) RecordWithdrawalDeletionFailed() {
	s.failedCalls.Add(1)
}

type stubDeleter struct {
	calls atomic.Int64
	err   error
}

func (s *stubDeleter) DeleteInvalidatedWithdrawals(_ context.Context, _ uint64, _ []types.GameMetadata) error {
	s.calls.Add(1)
	return s.err
}
