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

// Note: Test helpers (newTestCrossValidator, makeAccess, makeExecDescriptor) and
// constants (testExpiryWindow, testChainA) are defined in backend_test.go

// =============================================================================
// Timeout Expiry Tests
// =============================================================================

func TestCrossValidator_TimeoutExceedsExpiry(t *testing.T) {
	mock := newMockChainIngester()
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
// CrossUnsafe Timestamp Tests
// =============================================================================

func TestCrossValidator_CrossUnsafe_Boundary(t *testing.T) {
	mock := newMockChainIngester()
	checksum := types.MessageChecksum{0x01}
	mock.AddLog(100, 10, 0, checksum, types.BlockSeal{})
	mock.AddLog(101, 10, 0, checksum, types.BlockSeal{})
	mock.SetLatestTimestamp(100)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	// Trigger initialization (sets crossValidatedTs=100)
	cv.advanceValidation()

	// At boundary: access at timestamp 100 == crossValidatedTs=100 should pass
	access := makeAccess(testChainA, 100, 10, 0, checksum)
	exec := makeExecDescriptor(testChainA, 150, 0)
	err := cv.ValidateAccessEntry(access, types.CrossUnsafe, exec)
	require.NoError(t, err)

	// Beyond boundary: access at timestamp 101 > crossValidatedTs=100 should fail
	access = makeAccess(testChainA, 101, 10, 0, checksum)
	err = cv.ValidateAccessEntry(access, types.CrossUnsafe, exec)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrOutOfScope)
}

// =============================================================================
// Unknown Chain Tests
// =============================================================================

func TestCrossValidator_KnownChain(t *testing.T) {
	mock := newMockChainIngester()
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
	mock := newMockChainIngester()
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

func TestCrossValidator_InitiatingMessageNotFound(t *testing.T) {
	mock := newMockChainIngester()
	mock.SetLatestTimestamp(200)
	// Don't add any logs - the initiating message won't exist

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	// Valid chain, valid timing, but log doesn't exist
	access := makeAccess(testChainA, 100, 10, 0, types.MessageChecksum{0x01})
	exec := makeExecDescriptor(testChainA, 150, 0)

	err := cv.ValidateAccessEntry(access, types.LocalUnsafe, exec)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrConflict)
}

// =============================================================================
// Validation Failure Propagation Test
// =============================================================================

func TestCrossValidator_ValidationFailureSetsError(t *testing.T) {
	// Setup two chains
	mockA := newMockChainIngester()
	mockB := newMockChainIngester()

	checksumA := types.MessageChecksum{0x01}

	// Add valid log on chain A
	mockA.AddLog(100, 10, 0, checksumA, types.BlockSeal{})
	mockA.SetLatestTimestamp(101)

	// Add INVALID executing message on chain B that references a non-existent log
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
		101,              // startTimestamp - matches what chains report
		time.Millisecond, // Short interval for test
		chains,
	)

	// Both chains should have no errors initially
	require.Nil(t, mockA.Error())
	require.Nil(t, mockB.Error())

	// First call triggers initialization (sets crossValidatedTs to minIngestedTs=101)
	cv.advanceValidation()

	// Simulate chains ingesting one more block (timestamp 102)
	mockA.SetLatestTimestamp(102)
	mockB.SetLatestTimestamp(102)

	// Update the invalid exec msg to have inclusionTimestamp=102
	mockB.ClearExecMsgs()
	mockB.AddExecMsg(IncludedMessage{
		ExecutingMessage: &types.ExecutingMessage{
			ChainID:   eth.ChainIDFromUInt64(testChainA),
			BlockNum:  999,
			LogIdx:    0,
			Timestamp: 50,
			Checksum:  types.MessageChecksum{0xFF},
		},
		InclusionBlockNum:  12,
		InclusionTimestamp: 102,
	})

	// This will try to validate timestamp 102, which will fail
	cv.advanceValidation()

	// Chain ingesters should NOT have errors - validation errors are tracked by cross-validator
	require.Nil(t, mockA.Error(), "chain ingesters should not have validation errors")
	require.Nil(t, mockB.Error(), "chain ingesters should not have validation errors")

	// Cross-validator should have an error
	require.NotNil(t, cv.Error(), "cross-validator should have error after validation failure")
	require.Contains(t, cv.Error().Message, "validation failed")

	// Cross-validated timestamp should NOT have advanced past 101
	ts, ok := cv.CrossValidatedTimestamp()
	require.True(t, ok)
	require.Equal(t, uint64(101), ts, "cross-validated timestamp should not advance after failure")
}

func TestCrossValidator_StartTimestampZeroStillAdvances(t *testing.T) {
	mock := newMockChainIngester()
	mock.SetLatestTimestamp(1)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 0)

	// First advancement initializes the validator at timestamp 0.
	cv.advanceValidation()

	ts, ok := cv.CrossValidatedTimestamp()
	require.True(t, ok)
	require.Equal(t, uint64(0), ts)

	// Second advancement should move past 0 once chains have ingested timestamp 1.
	cv.advanceValidation()

	ts, ok = cv.CrossValidatedTimestamp()
	require.True(t, ok)
	require.Equal(t, uint64(1), ts)
}

// =============================================================================
// ValidateAccessEntry Timestamp Ingestion Tests
// =============================================================================

func TestValidateAccessEntry_TimestampNotIngested(t *testing.T) {
	mock := newMockChainIngester()
	checksum := types.MessageChecksum{0x01}
	mock.AddLog(100, 10, 0, checksum, types.BlockSeal{})
	mock.SetLatestTimestamp(100) // Only ingested up to 100

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	// Access at timestamp 150, but we've only ingested up to 100
	access := makeAccess(testChainA, 150, 10, 0, checksum)
	exec := makeExecDescriptor(testChainA, 200, 0)

	err := cv.ValidateAccessEntry(access, types.LocalUnsafe, exec)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrOutOfScope)
	require.Contains(t, err.Error(), "not yet ingested")
}

// =============================================================================
// validateExecutingMessage Timing Tests
// =============================================================================

func TestValidateExecMsg_InitBeforeInclusion(t *testing.T) {
	mock := newMockChainIngester()
	checksum := types.MessageChecksum{0x01}
	mock.AddLog(100, 10, 0, checksum, types.BlockSeal{})
	mock.SetLatestTimestamp(200)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	// Init timestamp = 100, Inclusion timestamp = 100 (equal, not before)
	access := makeAccess(testChainA, 100, 10, 0, checksum)
	exec := makeExecDescriptor(testChainA, 100, 0) // Same as init timestamp

	err := cv.ValidateAccessEntry(access, types.LocalUnsafe, exec)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrConflict)
	require.Contains(t, err.Error(), "not before inclusion")
}

func TestValidateExecMsg_MessageExpired(t *testing.T) {
	mock := newMockChainIngester()
	checksum := types.MessageChecksum{0x01}
	mock.AddLog(100, 10, 0, checksum, types.BlockSeal{})
	mock.SetLatestTimestamp(300)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100) // expiry window = 100

	// Init timestamp = 100, expiry = 100, so expires at 200
	// Inclusion at 250 - message has expired
	access := makeAccess(testChainA, 100, 10, 0, checksum)
	exec := makeExecDescriptor(testChainA, 250, 0)

	err := cv.ValidateAccessEntry(access, types.LocalUnsafe, exec)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrConflict)
	require.Contains(t, err.Error(), "expired")
}

// =============================================================================
// advanceValidation Tests
// =============================================================================

func TestAdvanceValidation_WaitsForChainsReady(t *testing.T) {
	mock := newMockChainIngester()
	mock.SetReady(false) // Not ready
	mock.SetLatestTimestamp(100)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	// Should not initialize when chains not ready
	cv.advanceValidation()
	_, ok := cv.CrossValidatedTimestamp()
	require.False(t, ok, "should not initialize when chains not ready")

	// Once ready, should initialize
	mock.SetReady(true)
	cv.advanceValidation()
	ts, ok := cv.CrossValidatedTimestamp()
	require.True(t, ok)
	require.Equal(t, uint64(100), ts)
}

func TestAdvanceValidation_InitializesToStartTimestamp(t *testing.T) {
	mock := newMockChainIngester()
	mock.SetReady(true)
	mock.SetLatestTimestamp(150) // Chains have ingested up to 150

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100) // startTimestamp = 100

	// Before first advance, no cross-validated timestamp
	_, ok := cv.CrossValidatedTimestamp()
	require.False(t, ok)

	// First advance initializes to startTimestamp
	cv.advanceValidation()
	ts, ok := cv.CrossValidatedTimestamp()
	require.True(t, ok)
	require.Equal(t, uint64(100), ts, "should initialize to startTimestamp")
}

func TestAdvanceValidation_AdvancesTimestamp(t *testing.T) {
	mock := newMockChainIngester()
	mock.SetReady(true)
	mock.SetLatestTimestamp(100)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	// Initialize
	cv.advanceValidation()
	ts, _ := cv.CrossValidatedTimestamp()
	require.Equal(t, uint64(100), ts)

	// Advance chain
	mock.SetLatestTimestamp(103)

	// Should advance to 103 (validates 101, 102, 103)
	cv.advanceValidation()
	ts, _ = cv.CrossValidatedTimestamp()
	require.Equal(t, uint64(103), ts)
}

func TestAdvanceValidation_StopsAtMinIngested(t *testing.T) {
	mockA := newMockChainIngester()
	mockB := newMockChainIngester()
	mockA.SetReady(true)
	mockB.SetReady(true)
	mockA.SetLatestTimestamp(100)
	mockB.SetLatestTimestamp(100)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA):     mockA,
		eth.ChainIDFromUInt64(testChainA + 1): mockB,
	}
	cv := newTestCrossValidator(chains, testExpiryWindow, 100)

	// Initialize
	cv.advanceValidation()
	ts, _ := cv.CrossValidatedTimestamp()
	require.Equal(t, uint64(100), ts)

	// Only chain A advances
	mockA.SetLatestTimestamp(105)
	// Chain B stays at 100

	// Should not advance past 100 (min of both chains)
	cv.advanceValidation()
	ts, _ = cv.CrossValidatedTimestamp()
	require.Equal(t, uint64(100), ts, "should not advance past slow chain")
}

// =============================================================================
// validateMessageTiming Tests
// =============================================================================

func TestValidateMessageTiming(t *testing.T) {
	tests := []struct {
		name                string
		initTimestamp       uint64
		inclusionTimestamp  uint64
		messageExpiryWindow uint64
		timeout             uint64
		execTimestamp       uint64
		wantErr             bool
		errContains         string
	}{
		{
			name:                "valid: basic case without timeout",
			initTimestamp:       100,
			inclusionTimestamp:  150,
			messageExpiryWindow: 100,
			timeout:             0,
			execTimestamp:       0,
			wantErr:             false,
		},
		{
			name:                "valid: message expires exactly at inclusion",
			initTimestamp:       100,
			inclusionTimestamp:  200,
			messageExpiryWindow: 100, // expiresAt = 200
			timeout:             0,
			execTimestamp:       0,
			wantErr:             false,
		},
		{
			name:                "valid: with timeout, message expires after deadline",
			initTimestamp:       100,
			inclusionTimestamp:  150,
			messageExpiryWindow: 100, // expiresAt = 200
			timeout:             40,  // maxExecTs = 150 + 40 = 190
			execTimestamp:       150, // 200 >= 190, valid
			wantErr:             false,
		},
		{
			name:                "invalid: init timestamp equals inclusion",
			initTimestamp:       100,
			inclusionTimestamp:  100,
			messageExpiryWindow: 100,
			timeout:             0,
			execTimestamp:       0,
			wantErr:             true,
			errContains:         "not before inclusion",
		},
		{
			name:                "invalid: init timestamp after inclusion",
			initTimestamp:       150,
			inclusionTimestamp:  100,
			messageExpiryWindow: 100,
			timeout:             0,
			execTimestamp:       0,
			wantErr:             true,
			errContains:         "not before inclusion",
		},
		{
			name:                "invalid: overflow in expiry calculation",
			initTimestamp:       ^uint64(0) - 10, // near max uint64
			inclusionTimestamp:  ^uint64(0),
			messageExpiryWindow: 100, // will overflow
			timeout:             0,
			execTimestamp:       0,
			wantErr:             true,
			errContains:         "overflow in expiry calculation",
		},
		{
			name:                "invalid: message expired at inclusion",
			initTimestamp:       100,
			inclusionTimestamp:  250,
			messageExpiryWindow: 100, // expiresAt = 200 < 250
			timeout:             0,
			execTimestamp:       0,
			wantErr:             true,
			errContains:         "expired",
		},
		{
			name:                "invalid: timeout overflow",
			initTimestamp:       100,
			inclusionTimestamp:  150,
			messageExpiryWindow: 100,
			timeout:             ^uint64(0),      // max uint64
			execTimestamp:       ^uint64(0) - 10, // will overflow when added to timeout
			wantErr:             true,
			errContains:         "overflow in max exec timestamp",
		},
		{
			name:                "invalid: expires before timeout deadline",
			initTimestamp:       100,
			inclusionTimestamp:  150,
			messageExpiryWindow: 100, // expiresAt = 200
			timeout:             51,  // maxExecTs = 150 + 51 = 201
			execTimestamp:       150, // 200 < 201, invalid
			wantErr:             true,
			errContains:         "expire before timeout",
		},
		{
			name:                "valid: expires exactly at timeout deadline",
			initTimestamp:       100,
			inclusionTimestamp:  150,
			messageExpiryWindow: 100, // expiresAt = 200
			timeout:             50,  // maxExecTs = 150 + 50 = 200
			execTimestamp:       150, // 200 >= 200, valid (equal is ok)
			wantErr:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMessageTiming(
				tt.initTimestamp,
				tt.inclusionTimestamp,
				tt.messageExpiryWindow,
				tt.timeout,
				tt.execTimestamp,
			)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)
				require.ErrorIs(t, err, types.ErrConflict)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
