package monitor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// mockNumberAndHash implements eth.NumberAndHash for testing
type mockNumberAndHash struct {
	number uint64
	hash   common.Hash
}

func (m mockNumberAndHash) NumberU64() uint64 {
	return m.number
}

func (m mockNumberAndHash) Hash() common.Hash {
	return m.hash
}

// mockUpdaterClient implements the UpdaterClient interface with configurable function implementations
type mockUpdaterClient struct {
	fetchReceiptsByNumberFn func(ctx context.Context, number uint64) (eth.BlockInfo, ethtypes.Receipts, error)
}

func (m *mockUpdaterClient) FetchReceiptsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, ethtypes.Receipts, error) {
	if m.fetchReceiptsByNumberFn != nil {
		return m.fetchReceiptsByNumberFn(ctx, number)
	}
	return nil, nil, nil
}

// setupTestUpdater creates a new RPCUpdater instance for testing
func setupTestUpdater(t *testing.T) (*RPCUpdater, *mockUpdaterClient) {
	logger := log.New()
	client := &mockUpdaterClient{}
	expiry := locks.RWMapFromMap(map[eth.ChainID]eth.NumberAndHash{})
	updater := NewUpdater(eth.ChainIDFromUInt64(1), client, expiry, logger)
	return updater, client
}

// TestUpdaterJobExpiration tests the job expiration logic
func TestUpdaterJobExpiration(t *testing.T) {
	tests := []struct {
		name           string
		initiatingInfo *supervisortypes.Identifier
		executingInfo  eth.BlockID
		initExpiry     eth.NumberAndHash
		execExpiry     eth.NumberAndHash
		lastEvaluated  time.Time
		didMetrics     bool
		shouldExpire   bool
	}{
		{
			name: "job should expire - both blocks finalized and metrics counted",
			initiatingInfo: &supervisortypes.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
			},
			executingInfo: eth.BlockID{
				Number: 200,
				Hash:   common.HexToHash("0x123"),
			},
			initExpiry:    mockNumberAndHash{number: 150, hash: common.HexToHash("0x456")}, // initiating block is finalized
			execExpiry:    mockNumberAndHash{number: 250, hash: common.HexToHash("0x789")}, // executing block is finalized
			lastEvaluated: time.Now().Add(-time.Hour),
			didMetrics:    true,
			shouldExpire:  true,
		},
		{
			name: "job should not expire - initiating block not finalized",
			initiatingInfo: &supervisortypes.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
			},
			executingInfo: eth.BlockID{
				Number: 200,
				Hash:   common.HexToHash("0x123"),
			},
			initExpiry:    mockNumberAndHash{number: 50, hash: common.HexToHash("0x456")},  // initiating block not finalized
			execExpiry:    mockNumberAndHash{number: 250, hash: common.HexToHash("0x789")}, // executing block is finalized
			lastEvaluated: time.Now().Add(-time.Hour),
			didMetrics:    true,
			shouldExpire:  false,
		},
		{
			name: "job should not expire - executing block not finalized",
			initiatingInfo: &supervisortypes.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
			},
			executingInfo: eth.BlockID{
				Number: 200,
				Hash:   common.HexToHash("0x123"),
			},
			initExpiry:    mockNumberAndHash{number: 150, hash: common.HexToHash("0x456")}, // initiating block is finalized
			execExpiry:    mockNumberAndHash{number: 150, hash: common.HexToHash("0x789")}, // executing block not finalized
			lastEvaluated: time.Now().Add(-time.Hour),
			didMetrics:    true,
			shouldExpire:  false,
		},
		{
			name: "job should not expire - never evaluated",
			initiatingInfo: &supervisortypes.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
			},
			executingInfo: eth.BlockID{
				Number: 200,
				Hash:   common.HexToHash("0x123"),
			},
			initExpiry:    mockNumberAndHash{number: 150, hash: common.HexToHash("0x456")},
			execExpiry:    mockNumberAndHash{number: 250, hash: common.HexToHash("0x789")},
			lastEvaluated: time.Time{}, // never evaluated
			didMetrics:    true,
			shouldExpire:  false,
		},
		{
			name: "job should not expire - metrics not counted",
			initiatingInfo: &supervisortypes.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
			},
			executingInfo: eth.BlockID{
				Number: 200,
				Hash:   common.HexToHash("0x123"),
			},
			initExpiry:    mockNumberAndHash{number: 150, hash: common.HexToHash("0x456")},
			execExpiry:    mockNumberAndHash{number: 250, hash: common.HexToHash("0x789")},
			lastEvaluated: time.Now().Add(-time.Hour),
			didMetrics:    false,
			shouldExpire:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater, _ := setupTestUpdater(t)

			// Create a test job
			job := &Job{
				id:             JobID(uuid.New().String()),
				initiating:     tt.initiatingInfo,
				executingBlock: tt.executingInfo,
				executingChain: eth.ChainIDFromUInt64(2),
			}

			// Set the last evaluated time if provided
			if !tt.lastEvaluated.IsZero() {
				job.UpdateLastEvaluated(tt.lastEvaluated)
			}

			// Set metrics flag if provided
			if tt.didMetrics {
				job.SetDidMetrics()
			}

			// Set expiry blocks
			updater.expiry.Set(tt.initiatingInfo.ChainID, tt.initExpiry)
			updater.expiry.Set(job.executingChain, tt.execExpiry)

			// Check if job should expire
			shouldExpire := updater.ShouldExpire(job)
			require.Equal(t, tt.shouldExpire, shouldExpire, "job expiration check failed")
		})
	}
}

// TestUpdaterJobStatusUpdate tests the job status update functionality
func TestUpdaterJobStatusUpdate(t *testing.T) {
	tests := []struct {
		name           string
		initiatingInfo *supervisortypes.Identifier
		executingInfo  eth.BlockID
		receipts       ethtypes.Receipts
		expectedStatus []jobStatus
	}{
		{
			name: "valid log found and hash matches",
			initiatingInfo: &supervisortypes.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
				LogIndex:    0,
			},
			executingInfo: eth.BlockID{
				Number: 200,
				Hash:   common.HexToHash("0x123"),
			},
			receipts: ethtypes.Receipts{
				{
					Logs: []*ethtypes.Log{
						{
							Index: 0,
							Data:  []byte{0x01, 0x02, 0x03},
						},
					},
				},
			},
			expectedStatus: []jobStatus{jobStatusValid},
		},
		{
			name: "log not found",
			initiatingInfo: &supervisortypes.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
				LogIndex:    1, // Log index 1 doesn't exist in receipts
			},
			executingInfo: eth.BlockID{
				Number: 200,
				Hash:   common.HexToHash("0x123"),
			},
			receipts: ethtypes.Receipts{
				{
					Logs: []*ethtypes.Log{
						{
							Index: 0,
							Data:  []byte{0x01, 0x02, 0x03},
						},
					},
				},
			},
			expectedStatus: []jobStatus{jobStatusInvalid},
		},
		{
			name: "log hash mismatch",
			initiatingInfo: &supervisortypes.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
				LogIndex:    0,
			},
			executingInfo: eth.BlockID{
				Number: 200,
				Hash:   common.HexToHash("0x123"),
			},
			receipts: ethtypes.Receipts{
				{
					Logs: []*ethtypes.Log{
						{
							Index: 0,
							Data:  []byte{0x04, 0x05, 0x06}, // Different data will result in different hash
						},
					},
				},
			},
			expectedStatus: []jobStatus{jobStatusInvalid},
		},
		{
			name: "error fetching receipts",
			initiatingInfo: &supervisortypes.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
				LogIndex:    0,
			},
			executingInfo: eth.BlockID{
				Number: 200,
				Hash:   common.HexToHash("0x123"),
			},
			receipts:       nil, // Will cause error in mock
			expectedStatus: []jobStatus{jobStatusUnknown},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater, client := setupTestUpdater(t)

			// Create a test job
			job := &Job{
				id:             JobID(uuid.New().String()),
				initiating:     tt.initiatingInfo,
				executingBlock: tt.executingInfo,
				executingChain: eth.ChainIDFromUInt64(2),
			}

			// If we have valid receipts, calculate the expected hash
			if len(tt.receipts) > 0 && len(tt.receipts[0].Logs) > 0 {
				if tt.name == "log hash mismatch" {
					// For the mismatch case, set a different hash than what we expect from the log
					job.executingPayload = common.HexToHash("0x1234567890abcdef")
				} else {
					expectedHash := crypto.Keccak256Hash(supervisortypes.LogToMessagePayload(tt.receipts[0].Logs[0]))
					job.executingPayload = expectedHash
				}
			}

			// Configure mock client
			client.fetchReceiptsByNumberFn = func(ctx context.Context, number uint64) (eth.BlockInfo, ethtypes.Receipts, error) {
				if tt.receipts == nil {
					return nil, nil, errors.New("mock error")
				}
				return nil, tt.receipts, nil
			}

			// Update job status
			updater.UpdateJobStatus(job)

			// Verify status
			require.Equal(t, tt.expectedStatus, job.status, "job status mismatch")
		})
	}
}
