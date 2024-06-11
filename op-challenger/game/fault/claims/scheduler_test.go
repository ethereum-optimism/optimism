package claims

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

var mockClaimError = errors.New("mock claim error")

func TestBondClaimScheduler_Schedule(t *testing.T) {
	tests := []struct {
		name                string
		claimErr            error
		games               []types.GameMetadata
		expectedMetricCalls int
		expectedClaimCalls  int
	}{
		{
			name:                "SingleGame_Succeeds",
			games:               []types.GameMetadata{{}},
			expectedMetricCalls: 0,
			expectedClaimCalls:  1,
		},
		{
			name:                "SingleGame_Fails",
			claimErr:            mockClaimError,
			games:               []types.GameMetadata{{}},
			expectedMetricCalls: 1,
			expectedClaimCalls:  1,
		},
		{
			name:                "MultipleGames_Succeed",
			games:               []types.GameMetadata{{}, {}, {}},
			expectedMetricCalls: 0,
			expectedClaimCalls:  1,
		},
		{
			name:                "MultipleGames_Fails",
			claimErr:            mockClaimError,
			games:               []types.GameMetadata{{}, {}, {}},
			expectedMetricCalls: 1,
			expectedClaimCalls:  1,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			scheduler, metrics, claimer := setupTestBondClaimScheduler(t)
			claimer.claimErr = test.claimErr
			scheduler.Start(ctx)
			defer scheduler.Close()

			err := scheduler.Schedule(1, test.games)
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				return int(claimer.claimCalls.Load()) == test.expectedClaimCalls
			}, 10*time.Second, 10*time.Millisecond)
			require.Eventually(t, func() bool {
				return int(metrics.failedCalls.Load()) == test.expectedMetricCalls
			}, 10*time.Second, 10*time.Millisecond)
		})
	}
}

func setupTestBondClaimScheduler(t *testing.T) (*BondClaimScheduler, *stubMetrics, *stubClaimer) {
	logger := testlog.Logger(t, log.LvlInfo)
	metrics := &stubMetrics{}
	claimer := &stubClaimer{}
	scheduler := NewBondClaimScheduler(logger, metrics, claimer)
	return scheduler, metrics, claimer
}

type stubMetrics struct {
	failedCalls atomic.Int64
}

func (s *stubMetrics) RecordBondClaimFailed() {
	s.failedCalls.Add(1)
}

type stubClaimer struct {
	claimCalls atomic.Int64
	claimErr   error
}

func (s *stubClaimer) ClaimBonds(ctx context.Context, games []types.GameMetadata) error {
	s.claimCalls.Add(1)
	return s.claimErr
}
