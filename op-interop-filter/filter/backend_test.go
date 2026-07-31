package filter

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// Test constants
const (
	testExpiryWindow = uint64(100)
	testChainA       = uint64(900)
)

// =============================================================================
// Test Helpers
// =============================================================================

func newTestBackend() *Backend {
	return NewBackend(context.Background(), BackendParams{
		Logger:         testlog.Logger(&testing.T{}, log.LevelCrit),
		Metrics:        metrics.NoopMetrics,
		Chains:         make(map[eth.ChainID]ChainIngester),
		CrossValidator: &mockCrossValidator{},
	})
}

func newTestBackendWithMockChain(chainID uint64) (*Backend, *mockChainIngester) {
	mock := newMockChainIngester()
	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(chainID): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)
	return NewBackend(context.Background(), BackendParams{
		Logger:         testlog.Logger(&testing.T{}, log.LevelCrit),
		Metrics:        metrics.NoopMetrics,
		Chains:         chains,
		CrossValidator: cv,
	}), mock
}

func newTestCrossValidator(chains map[eth.ChainID]ChainIngester, expiryWindow uint64, startTimestamp uint64) *LockstepCrossValidator {
	return NewLockstepCrossValidator(
		context.Background(),
		testlog.Logger(&testing.T{}, log.LevelCrit),
		metrics.NoopMetrics,
		expiryWindow,
		startTimestamp,
		time.Hour, // Long interval - won't tick in tests
		chains,
	)
}

// makeAccess creates a test access entry
func makeAccess(chainID, timestamp, blockNum uint64, logIdx uint32, checksum messages.MessageChecksum) messages.Access {
	return messages.Access{
		ChainID:     eth.ChainIDFromUInt64(chainID),
		Timestamp:   timestamp,
		BlockNumber: blockNum,
		LogIndex:    logIdx,
		Checksum:    checksum,
	}
}

// makeExecDescriptor creates a test executing descriptor
func makeExecDescriptor(chainID, timestamp, timeout uint64) messages.ExecutingDescriptor {
	return messages.ExecutingDescriptor{
		ChainID:   eth.ChainIDFromUInt64(chainID),
		Timestamp: timestamp,
		Timeout:   timeout,
	}
}

// =============================================================================
// Backend Failsafe Tests
// =============================================================================

func TestBackend_Failsafe_CrossValidatorError(t *testing.T) {
	mock := newMockChainIngester()
	mock.SetReady(true)
	mock.SetLatestTimestamp(100) // Lazy init will set crossValidatedTs=100

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	backend := NewBackend(context.Background(), BackendParams{
		Logger:         testlog.Logger(t, log.LevelCrit),
		Metrics:        metrics.NoopMetrics,
		Chains:         chains,
		CrossValidator: cv,
	})

	// Initially not enabled
	require.False(t, backend.FailsafeEnabled())

	// Trigger lazy init (sets crossValidatedTs=100)
	cv.advanceValidation()

	// Add an invalid exec message at timestamp 101 (which we'll validate next)
	mock.AddExecMsg(IncludedMessage{
		ExecutingMessage: &messages.ExecutingMessage{
			ChainID:   eth.ChainIDFromUInt64(testChainA),
			BlockNum:  999, // Non-existent
			LogIdx:    0,
			Timestamp: 50,
			Checksum:  messages.MessageChecksum{0xFF},
		},
		InclusionBlockNum:  10,
		InclusionTimestamp: 101, // Will be validated when advancing from 100 to 101
	})
	mock.SetLatestTimestamp(101)

	// This should try to validate timestamp 101 and fail
	cv.advanceValidation()

	// Cross-validator error should enable failsafe
	require.NotNil(t, cv.Error())
	require.True(t, backend.FailsafeEnabled())
}

func TestBackend_Failsafe_AllClear(t *testing.T) {
	backend, mock := newTestBackendWithMockChain(testChainA)

	// Set everything to a good state
	mock.SetReady(true)
	mock.SetLatestTimestamp(200)

	// Failsafe should be off
	require.False(t, backend.FailsafeEnabled())

	// Even after some operations, failsafe stays off
	mock.SetLatestTimestamp(300)
	require.False(t, backend.FailsafeEnabled())
}

func TestBackend_ReorgRecovery_NoErrorIsNotResolvable(t *testing.T) {
	mock := newMockChainIngester()
	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	backend := NewBackend(context.Background(), BackendParams{Logger: testlog.Logger(t, log.LevelCrit), Metrics: metrics.NoopMetrics, Chains: chains, CrossValidator: &mockCrossValidator{}})

	_, _, err := backend.recoverChainReorg(context.Background(), eth.ChainIDFromUInt64(testChainA), mock)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no ingester error")
	require.Equal(t, 0, mock.rewindToFinalizedCount)
}

func TestBackend_Ready(t *testing.T) {
	// No chains = not ready
	backend := newTestBackend()
	require.False(t, backend.Ready(), "should not be ready with no chains")

	// With chains
	backend, mock := newTestBackendWithMockChain(testChainA)
	mock.SetReady(true)
	require.True(t, backend.Ready(), "should be ready when chains are ready")

	mock.SetReady(false)
	require.False(t, backend.Ready(), "should not be ready when chains are not ready")
}

func TestBackend_CheckAccessList_SupportLegacyCheckAccessListFormat(t *testing.T) {
	backend, mock := newTestBackendWithMockChain(testChainA)
	backend.legacyCheckAccessListFormat = true
	mock.SetReady(true)
	mock.SetLatestTimestamp(200)

	err := backend.CheckAccessList(context.Background(), nil, safety.LocalUnsafe, makeExecDescriptor(0, 150, 0))
	require.NoError(t, err)
}

// =============================================================================
// Failsafe reason logging
// =============================================================================

func TestFailsafeReasonDetail(t *testing.T) {
	chain900 := eth.ChainIDFromUInt64(900)
	chain1000 := eth.ChainIDFromUInt64(1000)

	tests := []struct {
		name      string
		manual    bool
		chainErrs map[eth.ChainID]*IngesterError
		cvErr     *ValidatorError
		want      string
	}{
		{
			name: "none active",
			want: failsafeReasonNone,
		},
		{
			name:   "manual only",
			manual: true,
			want:   "manual override",
		},
		{
			name:      "chain error includes reason and message",
			chainErrs: map[eth.ChainID]*IngesterError{chain900: {Reason: ErrorReorg, Message: "parent hash mismatch at block 175901"}},
			want:      "chain[900] reorg: parent hash mismatch at block 175901",
		},
		{
			name:  "cross-validation includes message",
			cvErr: &ValidatorError{Message: "invalid executing message at ts 42"},
			want:  "cross-validation: invalid executing message at ts 42",
		},
		{
			name:      "combined sources joined with semicolons",
			manual:    true,
			chainErrs: map[eth.ChainID]*IngesterError{chain900: {Reason: ErrorConflict, Message: "db conflict"}},
			cvErr:     &ValidatorError{Message: "validation failed"},
			want:      "manual override; chain[900] conflict: db conflict; cross-validation: validation failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, failsafeReasonDetail(tt.manual, tt.chainErrs, tt.cvErr))
		})
	}

	t.Run("chains ordered numerically not lexicographically", func(t *testing.T) {
		errs := map[eth.ChainID]*IngesterError{
			chain1000: {Reason: ErrorReorg, Message: "r1000"},
			chain900:  {Reason: ErrorReorg, Message: "r900"},
		}
		got := failsafeReasonDetail(false, errs, nil)
		require.Less(t, strings.Index(got, "chain[900]"), strings.Index(got, "chain[1000]"),
			"chains must be ordered numerically, got %q", got)
	})
}

func TestBackend_FailsafeHeartbeat_LogsReasonWhileActive(t *testing.T) {
	logger, logs := testlog.CaptureLogger(t, log.LevelInfo)
	mock := newMockChainIngester()
	b := NewBackend(context.Background(), BackendParams{
		Logger:         logger,
		Metrics:        metrics.NoopMetrics,
		Chains:         map[eth.ChainID]ChainIngester{eth.ChainIDFromUInt64(testChainA): mock},
		CrossValidator: &mockCrossValidator{},
	})

	const heartbeatMsg = "Failsafe still active"

	// Not in failsafe -> heartbeat is silent.
	b.logFailsafeIfActive()
	require.Nil(t, logs.FindLog(testlog.NewMessageFilter(heartbeatMsg)),
		"heartbeat must not log when failsafe is inactive")

	// In failsafe -> heartbeat logs the reason and the underlying "why" at Warn.
	mock.SetError(ErrorReorg, "parent hash mismatch at block 175901")
	b.logFailsafeIfActive()
	rec := logs.FindLog(testlog.NewMessageFilter(heartbeatMsg), testlog.NewLevelFilter(slog.LevelWarn))
	require.NotNil(t, rec, "heartbeat must log at Warn while failsafe is active")
	require.Contains(t, rec.AttrValue("reasons"), "reorg")
	require.Contains(t, rec.AttrValue("detail"), "parent hash mismatch at block 175901")

	// Cleared -> heartbeat goes silent again.
	logs.Clear()
	mock.ClearError()
	b.logFailsafeIfActive()
	require.Nil(t, logs.FindLog(testlog.NewMessageFilter(heartbeatMsg)),
		"heartbeat must stop once failsafe clears")
}

func TestBackend_FailsafeLogInterval_Configured(t *testing.T) {
	newBackend := func(interval time.Duration) *Backend {
		return NewBackend(context.Background(), BackendParams{
			Logger:              testlog.Logger(t, log.LevelError),
			Metrics:             metrics.NoopMetrics,
			Chains:              map[eth.ChainID]ChainIngester{},
			CrossValidator:      &mockCrossValidator{},
			FailsafeLogInterval: interval,
		})
	}

	// Configured interval is honored.
	require.Equal(t, 15*time.Second, newBackend(15*time.Second).failsafeLogInterval)

	// Unset (zero) falls back to the default — guards against time.NewTicker(0) panicking.
	require.Equal(t, defaultFailsafeLogInterval, newBackend(0).failsafeLogInterval)
}
