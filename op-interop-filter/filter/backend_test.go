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

func newTestBackendWithMockChain(chainID uint64) (*Backend, *MockChainIngester) {
	mock := NewMockChainIngester()
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

func newTestCrossValidator(chains map[eth.ChainID]ChainIngester, expiryWindow, startTs uint64) *LockstepCrossValidator {
	return NewLockstepCrossValidator(
		context.Background(),
		testlog.Logger(&testing.T{}, log.LevelCrit),
		metrics.NoopMetrics,
		expiryWindow,
		time.Hour, // Long interval - won't tick in tests
		chains,
		startTs,
	)
}

// mockCrossValidator is a minimal mock for backend tests that don't need validation
type mockCrossValidator struct {
	validateErr error
}

func (m *mockCrossValidator) Start() error { return nil }
func (m *mockCrossValidator) Stop() error  { return nil }
func (m *mockCrossValidator) ValidateAccessEntry(access types.Access, minSafety types.SafetyLevel, execDescriptor types.ExecutingDescriptor) error {
	return m.validateErr
}
func (m *mockCrossValidator) CrossValidatedTimestamp() (uint64, bool) { return 0, false }

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
// ValidateMessageTiming Tests
// =============================================================================

func TestValidateMessageTiming(t *testing.T) {
	tests := []struct {
		name       string
		init       uint64
		inclusion  uint64
		expiry     uint64
		wantErr    bool
		errContain string
	}{
		{
			name:      "happy path - init before inclusion",
			init:      100,
			inclusion: 101,
			expiry:    100,
			wantErr:   false,
		},
		{
			name:       "init equals inclusion",
			init:       100,
			inclusion:  100,
			expiry:     100,
			wantErr:    true,
			errContain: "not before",
		},
		{
			name:      "exact expiry boundary - passes",
			init:      100,
			inclusion: 200, // init + expiry = 200
			expiry:    100,
			wantErr:   false,
		},
		{
			name:       "just past expiry - fails",
			init:       100,
			inclusion:  201, // init + expiry = 200 < 201
			expiry:     100,
			wantErr:    true,
			errContain: "expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessageTiming(tt.init, tt.inclusion, tt.expiry)
			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, types.ErrConflict)
				if tt.errContain != "" {
					require.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
			}
		})
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

// =============================================================================
// Backend Ready State Tests
// =============================================================================

func TestBackend_Ready_NoChains(t *testing.T) {
	backend := newTestBackend()
	require.False(t, backend.Ready())
}

func TestBackend_Ready_WithChains(t *testing.T) {
	backend, mock := newTestBackendWithMockChain(testChainA)

	mock.SetReady(true)
	require.True(t, backend.Ready())

	mock.SetReady(false)
	require.False(t, backend.Ready())
}

// =============================================================================
// Backend CheckAccessList Tests
// =============================================================================

func TestBackend_CheckAccessList_FailsafeEnabled(t *testing.T) {
	backend, _ := newTestBackendWithMockChain(testChainA)
	backend.SetFailsafeEnabled(true)

	err := backend.CheckAccessList(context.Background(), nil, types.LocalUnsafe, makeExecDescriptor(testChainA, 100, 0))
	require.ErrorIs(t, err, types.ErrFailsafeEnabled)
}

func TestBackend_CheckAccessList_NotReady(t *testing.T) {
	backend := newTestBackend() // No chains = not ready

	err := backend.CheckAccessList(context.Background(), nil, types.LocalUnsafe, makeExecDescriptor(testChainA, 100, 0))
	require.ErrorIs(t, err, types.ErrUninitialized)
}

// =============================================================================
// Safety Level Tests
// =============================================================================

func TestBackend_CheckAccessList_LocalUnsafe(t *testing.T) {
	backend, mock := newTestBackendWithMockChain(testChainA)
	mock.SetReady(true)
	mock.SetLatestTimestamp(200)

	// Empty access list with LocalUnsafe should pass
	err := backend.CheckAccessList(context.Background(), nil, types.LocalUnsafe, makeExecDescriptor(testChainA, 150, 0))
	require.NoError(t, err)
}

func TestBackend_CheckAccessList_UnsupportedSafetyLevel(t *testing.T) {
	backend, mock := newTestBackendWithMockChain(testChainA)
	mock.SetReady(true)

	err := backend.CheckAccessList(context.Background(), nil, types.Finalized, makeExecDescriptor(testChainA, 100, 0))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported safety level")
}

// =============================================================================
// CrossValidator Timeout Expiry Tests
// =============================================================================

func TestCrossValidator_TimeoutZero(t *testing.T) {
	mock := NewMockChainIngester()
	checksum := types.MessageChecksum{0x01}
	mock.AddLog(100, 10, 0, checksum, types.BlockSeal{})
	mock.SetLatestTimestamp(200)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	access := makeAccess(testChainA, 100, 10, 0, checksum)
	exec := makeExecDescriptor(testChainA, 150, 0) // timeout = 0, skip check

	err := cv.ValidateAccessEntry(access, types.LocalUnsafe, exec)
	require.NoError(t, err)
}

func TestCrossValidator_TimeoutExceedsExpiry(t *testing.T) {
	mock := NewMockChainIngester()
	checksum := types.MessageChecksum{0x01}
	mock.AddLog(100, 10, 0, checksum, types.BlockSeal{})
	mock.SetLatestTimestamp(200)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	access := makeAccess(testChainA, 100, 10, 0, checksum)
	// init=100, expiry=100, so expiresAt=200
	// exec=150, timeout=51, so maxExecTs=201
	// 200 < 201, so should fail
	exec := makeExecDescriptor(testChainA, 150, 51)

	err := cv.ValidateAccessEntry(access, types.LocalUnsafe, exec)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrConflict)
	require.Contains(t, err.Error(), "expire before timeout")
}

// =============================================================================
// CrossValidator CrossUnsafe Timestamp Tests
// =============================================================================

func TestCrossValidator_CrossUnsafe_AtBoundary(t *testing.T) {
	mock := NewMockChainIngester()
	checksum := types.MessageChecksum{0x01}
	mock.AddLog(100, 10, 0, checksum, types.BlockSeal{})
	mock.SetLatestTimestamp(200)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	// Start timestamp = 100 (cross-validated up to timestamp 100)
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	// Access at init=100, which equals crossValidatedTs=100
	access := makeAccess(testChainA, 100, 10, 0, checksum)
	exec := makeExecDescriptor(testChainA, 150, 0)

	err := cv.ValidateAccessEntry(access, types.CrossUnsafe, exec)
	require.NoError(t, err)
}

func TestCrossValidator_CrossUnsafe_BeyondBoundary(t *testing.T) {
	mock := NewMockChainIngester()
	checksum := types.MessageChecksum{0x01}
	mock.AddLog(101, 10, 0, checksum, types.BlockSeal{}) // Note: timestamp 101
	mock.SetLatestTimestamp(200)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	// Start timestamp = 100 (cross-validated up to timestamp 100)
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	// Access at init=101, which is beyond crossValidatedTs=100
	access := makeAccess(testChainA, 101, 10, 0, checksum)
	exec := makeExecDescriptor(testChainA, 150, 0)

	err := cv.ValidateAccessEntry(access, types.CrossUnsafe, exec)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrOutOfScope)
}

// =============================================================================
// CrossValidator Unknown Chain Tests
// =============================================================================

func TestCrossValidator_KnownChain(t *testing.T) {
	mock := NewMockChainIngester()
	checksum := types.MessageChecksum{0x01}
	mock.AddLog(100, 10, 0, checksum, types.BlockSeal{})
	mock.SetLatestTimestamp(200)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	access := makeAccess(testChainA, 100, 10, 0, checksum)
	exec := makeExecDescriptor(testChainA, 150, 0)

	err := cv.ValidateAccessEntry(access, types.LocalUnsafe, exec)
	require.NoError(t, err)
}

func TestCrossValidator_UnknownChain(t *testing.T) {
	mock := NewMockChainIngester()
	mock.SetLatestTimestamp(200)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	// Access from chain 902 which is not registered
	unknownChainID := uint64(902)
	access := makeAccess(unknownChainID, 100, 10, 0, types.MessageChecksum{0x01})
	exec := makeExecDescriptor(testChainA, 150, 0)

	err := cv.ValidateAccessEntry(access, types.LocalUnsafe, exec)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrUnknownChain)
}

// =============================================================================
// CrossValidator Checksum Tests
// =============================================================================

func TestCrossValidator_ChecksumMatch(t *testing.T) {
	mock := NewMockChainIngester()
	checksum := types.MessageChecksum{0x01, 0x02, 0x03}
	mock.AddLog(100, 10, 0, checksum, types.BlockSeal{})
	mock.SetLatestTimestamp(200)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	access := makeAccess(testChainA, 100, 10, 0, checksum)
	exec := makeExecDescriptor(testChainA, 150, 0)

	err := cv.ValidateAccessEntry(access, types.LocalUnsafe, exec)
	require.NoError(t, err)
}

func TestCrossValidator_ChecksumNotFound(t *testing.T) {
	mock := NewMockChainIngester()
	mock.SetLatestTimestamp(200)
	// Don't add any logs - checksum won't be found

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	access := makeAccess(testChainA, 100, 10, 0, types.MessageChecksum{0x01})
	exec := makeExecDescriptor(testChainA, 150, 0)

	err := cv.ValidateAccessEntry(access, types.LocalUnsafe, exec)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrConflict)
}

// =============================================================================
// Validation Failure Propagation Test
// =============================================================================

func TestCrossValidator_ValidationFailureSetsErrorOnAllChains(t *testing.T) {
	// Setup two chains
	mockA := NewMockChainIngester()
	mockB := NewMockChainIngester()

	checksumA := types.MessageChecksum{0x01}

	// Add valid log on chain A
	mockA.AddLog(100, 10, 0, checksumA, types.BlockSeal{})
	mockA.SetLatestTimestamp(101)

	// Add INVALID executing message on chain B that references a non-existent log
	// This will cause validation to fail when we try to validate timestamp 101
	mockB.AddExecMsg(IncludedMessage{
		ExecutingMessage: &types.ExecutingMessage{
			ChainID:   eth.ChainIDFromUInt64(testChainA), // References chain A
			BlockNum:  999,                               // Non-existent block
			LogIdx:    0,
			Timestamp: 50,                          // Init timestamp
			Checksum:  types.MessageChecksum{0xFF}, // Non-existent checksum
		},
		InclusionBlockNum:  11,
		InclusionTimestamp: 101,
	})
	mockB.SetLatestTimestamp(101)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA):     mockA,
		eth.ChainIDFromUInt64(testChainA + 1): mockB,
	}

	cv := NewLockstepCrossValidator(
		context.Background(),
		testlog.Logger(t, log.LevelCrit),
		metrics.NoopMetrics,
		testExpiryWindow,
		time.Millisecond, // Short interval for test
		chains,
		100, // Start at timestamp 100
	)

	// Both chains should have no errors initially
	require.Nil(t, mockA.Error())
	require.Nil(t, mockB.Error())

	// Manually trigger advanceValidation
	// This will try to validate timestamp 101, which will fail because
	// the executing message on chain B references a non-existent log
	cv.advanceValidation()

	// Both chains should now have errors set
	require.NotNil(t, mockA.Error(), "chain A should have error after validation failure")
	require.NotNil(t, mockB.Error(), "chain B should have error after validation failure")
	require.Equal(t, ErrorValidationFailed, mockA.Error().Reason)
	require.Equal(t, ErrorValidationFailed, mockB.Error().Reason)

	// Cross-validated timestamp should NOT have advanced past 100
	ts, ok := cv.CrossValidatedTimestamp()
	require.True(t, ok)
	require.Equal(t, uint64(100), ts, "cross-validated timestamp should not advance after failure")
}
