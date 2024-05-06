package mon

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	failedGameClaimCountLog = "Failed to count game claims"
)

func TestClaimCounter_Count(t *testing.T) {
	t.Parallel()

	t.Run("OutputValidatorErrors", func(t *testing.T) {
		claims, cl, metrics, outputs, logs := setupClaimCounterTest(t)
		outputs.err = errors.New("boom")
		games := makeClaimTestGames(uint64(cl.Now().Unix()))
		claims.Count(context.Background(), games)

		require.Equal(t, 2, outputs.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedGameClaimCountLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		err := l.AttrValue("err")
		expectedErr := fmt.Errorf("%w: %w", ErrRootAgreement, outputs.err)
		require.Equal(t, expectedErr, err)

		require.Zero(t, metrics.invalidClaims)
		require.Zero(t, metrics.validClaims)
		require.Zero(t, metrics.honestActorValidClaims)
	})

	t.Run("OutputMismatch", func(t *testing.T) {
		claims, cl, metrics, outputs, logs := setupClaimCounterTest(t)
		games := makeClaimTestGames(uint64(cl.Now().Unix()))
		claims.Count(context.Background(), games)

		require.Equal(t, 2, outputs.calls)
		require.Equal(t, 6, metrics.invalidClaims)
		require.Equal(t, 4, metrics.validClaims)
		require.Equal(t, 2, metrics.honestActorValidClaims)

		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedGameClaimCountLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
	})
}

func makeClaimTestGames(duration uint64) []*types.EnrichedGameData {
	games := makeMultipleTestGames(duration)
	for _, game := range games {
		for i := range game.Claims {
			game.Claims[i].Resolved = false
		}
	}
	return games
}

func setupClaimCounterTest(t *testing.T) (*ClaimCounter, *clock.AdvancingClock, *mockClaimCounterMetrics, *stubOutputValidator, *testlog.CapturingHandler) {
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	cl := clock.NewAdvancingClock(10 * time.Millisecond)
	cl.Start()
	outputs := &stubOutputValidator{}
	metrics := &mockClaimCounterMetrics{}
	return NewClaimCounter(logger, cl, []common.Address{common.Address{0x01}}, outputs, metrics), cl, metrics, outputs, capturedLogs
}

type mockClaimCounterMetrics struct {
	invalidClaims          int
	validClaims            int
	honestActorValidClaims int
}

func (m *mockClaimCounterMetrics) RecordInvalidClaims(count int) {
	m.invalidClaims += count
}

func (m *mockClaimCounterMetrics) RecordValidClaims(count int) {
	m.validClaims += count
}

func (m *mockClaimCounterMetrics) RecordHonestActorValidClaimCount(count int) {
	m.honestActorValidClaims += count
}
