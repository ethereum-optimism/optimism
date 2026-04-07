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
func makeAccess(chainID, timestamp, blockNum uint64, logIdx uint32, checksum types.MessageChecksum) types.Access {
	return types.Access{
		ChainID:     eth.ChainIDFromUInt64(chainID),
		Timestamp:   timestamp,
		BlockNumber: blockNum,
		LogIndex:    logIdx,
		Checksum:    checksum,
	}
}

// makeExecDescriptor creates a test executing descriptor
func makeExecDescriptor(chainID, timestamp, timeout uint64) types.ExecutingDescriptor {
	return types.ExecutingDescriptor{
		ChainID:   eth.ChainIDFromUInt64(chainID),
		Timestamp: timestamp,
		Timeout:   timeout,
	}
}

// =============================================================================
// Backend Failsafe Tests
// =============================================================================

func TestBackend_Failsafe_ManualEnabled(t *testing.T) {
	backend, _ := newTestBackendWithMockChain(testChainA)

	// Initially not enabled
	require.False(t, backend.FailsafeEnabled())

	// Enable manually
	backend.SetFailsafeEnabled(true)
	require.True(t, backend.FailsafeEnabled())

	// Disable
	backend.SetFailsafeEnabled(false)
	require.False(t, backend.FailsafeEnabled())
}

func TestBackend_Failsafe_ChainError(t *testing.T) {
	backend, mock := newTestBackendWithMockChain(testChainA)

	// Initially not enabled
	require.False(t, backend.FailsafeEnabled())

	// Chain error enables failsafe
	mock.SetError(ErrorReorg, "reorg detected")
	require.True(t, backend.FailsafeEnabled())

	// Clearing error disables failsafe
	mock.ClearError()
	require.False(t, backend.FailsafeEnabled())
}

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
		ExecutingMessage: &types.ExecutingMessage{
			ChainID:   eth.ChainIDFromUInt64(testChainA),
			BlockNum:  999, // Non-existent
			LogIdx:    0,
			Timestamp: 50,
			Checksum:  types.MessageChecksum{0xFF},
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

// =============================================================================
// Backend Ready State Tests
// =============================================================================

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

// =============================================================================
// Backend CheckAccessList Tests
// =============================================================================

func TestBackend_CheckAccessList(t *testing.T) {
	// Failsafe enabled returns error
	backend, _ := newTestBackendWithMockChain(testChainA)
	backend.SetFailsafeEnabled(true)
	err := backend.CheckAccessList(context.Background(), nil, types.LocalUnsafe, makeExecDescriptor(testChainA, 100, 0))
	require.ErrorIs(t, err, types.ErrFailsafeEnabled)

	// Not ready returns error
	backend = newTestBackend() // No chains = not ready
	err = backend.CheckAccessList(context.Background(), nil, types.LocalUnsafe, makeExecDescriptor(testChainA, 100, 0))
	require.ErrorIs(t, err, types.ErrUninitialized)

	// Unsupported safety level returns error
	backend, mock := newTestBackendWithMockChain(testChainA)
	mock.SetReady(true)
	err = backend.CheckAccessList(context.Background(), nil, types.Finalized, makeExecDescriptor(testChainA, 100, 0))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported safety level")

	// Unknown executing chain returns error
	mock.SetLatestTimestamp(200)
	unknownChainID := uint64(999)
	err = backend.CheckAccessList(context.Background(), nil, types.LocalUnsafe, makeExecDescriptor(unknownChainID, 150, 0))
	require.ErrorIs(t, err, types.ErrUnknownChain)

	// LocalUnsafe with empty access list passes
	err = backend.CheckAccessList(context.Background(), nil, types.LocalUnsafe, makeExecDescriptor(testChainA, 150, 0))
	require.NoError(t, err)
}
