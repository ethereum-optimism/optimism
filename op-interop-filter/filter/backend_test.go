package filter

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
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

	err := backend.CheckAccessList(context.Background(), nil, types.LocalUnsafe, makeExecDescriptor(0, 150, 0))
	require.NoError(t, err)
}
