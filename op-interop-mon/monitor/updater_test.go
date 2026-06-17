package monitor

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// mockNumberAndHash implements eth.NumberAndHash for testing
type mockNumberAndHash struct {
	number uint64
}

func (m mockNumberAndHash) NumberU64() uint64 {
	return m.number
}

func (m mockNumberAndHash) Hash() common.Hash {
	return common.Hash{} // Return empty hash since it's not needed
}

// setupTestUpdater creates a new RPCUpdater instance for testing
func setupTestUpdater(t *testing.T) (*RPCUpdater, *mockClient) {
	logger := log.New()
	client := &mockClient{}
	expiry := locks.RWMapFromMap(map[eth.ChainID]eth.NumberAndHash{})
	updater := NewUpdater(eth.ChainIDFromUInt64(1), client, expiry, 604800, logger)
	return updater, client
}

// TestUpdaterJobExpiration tests the job expiration logic
func TestUpdaterJobExpiration(t *testing.T) {
	tests := []struct {
		name           string
		initiatingInfo *messages.Identifier
		executingInfo  eth.BlockID
		initExpiry     eth.NumberAndHash
		execExpiry     eth.NumberAndHash
		lastEvaluated  time.Time
		didMetrics     bool
		shouldExpire   bool
	}{
		{
			name: "job should expire - both blocks finalized and metrics counted",
			initiatingInfo: &messages.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
			},
			executingInfo: eth.BlockID{
				Number: 200,
			},
			initExpiry:    mockNumberAndHash{number: 150}, // initiating block is finalized
			execExpiry:    mockNumberAndHash{number: 250}, // executing block is finalized
			lastEvaluated: time.Now().Add(-time.Hour),
			didMetrics:    true,
			shouldExpire:  true,
		},
		{
			name: "job should not expire - initiating block not finalized",
			initiatingInfo: &messages.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
			},
			executingInfo: eth.BlockID{
				Number: 200,
			},
			initExpiry:    mockNumberAndHash{number: 50},  // initiating block not finalized
			execExpiry:    mockNumberAndHash{number: 250}, // executing block is finalized
			lastEvaluated: time.Now().Add(-time.Hour),
			didMetrics:    true,
			shouldExpire:  false,
		},
		{
			name: "job should not expire - executing block not finalized",
			initiatingInfo: &messages.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
			},
			executingInfo: eth.BlockID{
				Number: 200,
			},
			initExpiry:    mockNumberAndHash{number: 150}, // initiating block is finalized
			execExpiry:    mockNumberAndHash{number: 150}, // executing block not finalized
			lastEvaluated: time.Now().Add(-time.Hour),
			didMetrics:    true,
			shouldExpire:  false,
		},
		{
			name: "job should not expire - never evaluated",
			initiatingInfo: &messages.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
			},
			executingInfo: eth.BlockID{
				Number: 200,
			},
			initExpiry:    mockNumberAndHash{number: 150},
			execExpiry:    mockNumberAndHash{number: 250},
			lastEvaluated: time.Time{}, // never evaluated
			didMetrics:    true,
			shouldExpire:  false,
		},
		{
			name: "job should not expire - metrics not counted",
			initiatingInfo: &messages.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
			},
			executingInfo: eth.BlockID{
				Number: 200,
			},
			initExpiry:    mockNumberAndHash{number: 150},
			execExpiry:    mockNumberAndHash{number: 250},
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
			updater.finalized.Set(tt.initiatingInfo.ChainID, tt.initExpiry)
			updater.finalized.Set(job.executingChain, tt.execExpiry)

			// Check if job should expire
			shouldExpire := updater.ShouldExpire(job)
			require.Equal(t, tt.shouldExpire, shouldExpire, "job expiration check failed")
		})
	}
}

// TestUpdaterJobStatusUpdate tests the job status update functionality
func TestUpdaterJobStatusUpdate(t *testing.T) {
	// Create test data
	validLog := &ethtypes.Log{
		Index: 0,
		Data:  []byte{0x01, 0x02, 0x03},
	}
	validHash := crypto.Keccak256Hash(messages.LogToMessagePayload(validLog))

	invalidLog := &ethtypes.Log{
		Index: 0,
		Data:  []byte{0x04, 0x05, 0x06}, // Different data will result in different hash
	}

	tests := []struct {
		name           string
		initiatingInfo *messages.Identifier
		executingInfo  eth.BlockID
		receipts       ethtypes.Receipts
		expectedHash   common.Hash
		expectedStatus []jobStatus
	}{
		{
			name: "valid log found and hash matches",
			initiatingInfo: &messages.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
				LogIndex:    0,
			},
			executingInfo: eth.BlockID{
				Number: 200,
			},
			receipts: ethtypes.Receipts{
				{
					Logs: []*ethtypes.Log{validLog},
				},
			},
			expectedHash:   validHash,
			expectedStatus: []jobStatus{jobStatusValid},
		},
		{
			name: "log not found - index out of bounds",
			initiatingInfo: &messages.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
				LogIndex:    1, // Log index 1 doesn't exist in receipts
			},
			executingInfo: eth.BlockID{
				Number: 200,
			},
			receipts: ethtypes.Receipts{
				{
					Logs: []*ethtypes.Log{validLog},
				},
			},
			expectedHash:   validHash,
			expectedStatus: []jobStatus{jobStatusInvalid},
		},
		{
			name: "log hash mismatch",
			initiatingInfo: &messages.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
				LogIndex:    0,
			},
			executingInfo: eth.BlockID{
				Number: 200,
			},
			receipts: ethtypes.Receipts{
				{
					Logs: []*ethtypes.Log{invalidLog},
				},
			},
			expectedHash:   validHash, // Expecting the valid hash but got invalid log
			expectedStatus: []jobStatus{jobStatusInvalid},
		},
		{
			name: "empty receipts",
			initiatingInfo: &messages.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
				LogIndex:    0,
			},
			executingInfo: eth.BlockID{
				Number: 200,
			},
			receipts:       ethtypes.Receipts{},
			expectedHash:   validHash,
			expectedStatus: []jobStatus{jobStatusInvalid},
		},
		{
			name: "fetch receipts error",
			initiatingInfo: &messages.Identifier{
				ChainID:     eth.ChainIDFromUInt64(1),
				BlockNumber: 100,
				LogIndex:    0,
			},
			executingInfo: eth.BlockID{
				Number: 200,
			},
			receipts:       nil, // Will cause error in mock
			expectedHash:   validHash,
			expectedStatus: []jobStatus{jobStatusUnknown},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater, client := setupTestUpdater(t)

			// Create a test job
			job := &Job{
				initiating:       tt.initiatingInfo,
				executingBlock:   tt.executingInfo,
				executingChain:   eth.ChainIDFromUInt64(2),
				executingPayload: tt.expectedHash,
			}

			// Configure mock client to return the test receipts
			client.fetchReceiptsByNumber = func(ctx context.Context, number uint64) (eth.BlockInfo, ethtypes.Receipts, error) {
				if tt.receipts == nil {
					return nil, nil, errors.New("mock error")
				}
				return eth.HeaderBlockInfo(&ethtypes.Header{}), tt.receipts, nil
			}

			// Update job status
			updater.UpdateJobStatus(job)

			// Verify status
			require.Equal(t, tt.expectedStatus, job.status, "job status mismatch")
		})
	}
}

// TestUpdaterValidityInvariants exercises the current interop validity model:
// origin binding, initiating-timestamp binding, payload hash, and message expiry.
func TestUpdaterValidityInvariants(t *testing.T) {
	validLog := &ethtypes.Log{Index: 0, Address: common.HexToAddress("0xabc"), Data: []byte{0x01, 0x02, 0x03}}
	validHash := crypto.Keccak256Hash(messages.LogToMessagePayload(validLog))

	tests := []struct {
		name           string
		origin         common.Address
		initTimestamp  uint64
		execTimestamp  uint64
		blockTime      uint64
		expiryWindow   uint64
		payload        common.Hash
		expectedStatus []jobStatus
	}{
		{
			name:           "valid within expiry window",
			origin:         common.HexToAddress("0xabc"),
			initTimestamp:  1000,
			execTimestamp:  1100,
			blockTime:      1000,
			expiryWindow:   604800,
			payload:        validHash,
			expectedStatus: []jobStatus{jobStatusValid},
		},
		{
			name:           "origin mismatch is invalid",
			origin:         common.HexToAddress("0xdead"),
			initTimestamp:  1000,
			execTimestamp:  1100,
			blockTime:      1000,
			expiryWindow:   604800,
			payload:        validHash,
			expectedStatus: []jobStatus{jobStatusInvalid},
		},
		{
			name:           "block timestamp mismatch",
			origin:         common.HexToAddress("0xabc"),
			initTimestamp:  1000,
			execTimestamp:  1100,
			blockTime:      999,
			expiryWindow:   604800,
			payload:        validHash,
			expectedStatus: []jobStatus{jobStatusTimestampMismatch},
		},
		{
			name:           "expired beyond window",
			origin:         common.HexToAddress("0xabc"),
			initTimestamp:  1000,
			execTimestamp:  1000 + 604800 + 1,
			blockTime:      1000,
			expiryWindow:   604800,
			payload:        validHash,
			expectedStatus: []jobStatus{jobStatusExpired},
		},
		{
			name:           "executing before initiating is invalid",
			origin:         common.HexToAddress("0xabc"),
			initTimestamp:  1000,
			execTimestamp:  999,
			blockTime:      1000,
			expiryWindow:   604800,
			payload:        validHash,
			expectedStatus: []jobStatus{jobStatusInvalid},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.New()
			client := &mockClient{}
			expiry := locks.RWMapFromMap(map[eth.ChainID]eth.NumberAndHash{})
			updater := NewUpdater(eth.ChainIDFromUInt64(1), client, expiry, tt.expiryWindow, logger)

			client.fetchReceiptsByNumber = func(ctx context.Context, number uint64) (eth.BlockInfo, ethtypes.Receipts, error) {
				blk := eth.HeaderBlockInfo(&ethtypes.Header{Number: big.NewInt(100), Time: tt.blockTime})
				return blk, ethtypes.Receipts{{Logs: []*ethtypes.Log{validLog}}}, nil
			}

			job := &Job{
				initiating: &messages.Identifier{
					ChainID:     eth.ChainIDFromUInt64(1),
					BlockNumber: 100,
					LogIndex:    0,
					Origin:      tt.origin,
					Timestamp:   tt.initTimestamp,
				},
				executingBlock:     eth.BlockID{Number: 200},
				executingChain:     eth.ChainIDFromUInt64(2),
				executingPayload:   tt.payload,
				executingTimestamp: tt.execTimestamp,
			}

			updater.UpdateJobStatus(job)
			require.Equal(t, tt.expectedStatus, job.status)
		})
	}
}
