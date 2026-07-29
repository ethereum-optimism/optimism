package sources

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFollowClient_GetFollowStatus(t *testing.T) {
	t.Run("CopiesLocalSafeL2", func(t *testing.T) {
		ctx := context.Background()
		rpc := new(mockRPC)
		defer rpc.AssertExpectations(t)

		client, err := NewFollowClient(rpc)
		require.NoError(t, err)

		// Create a mock sync status with distinct values for each field
		// to ensure we're copying the right fields
		mockSyncStatus := &eth.SyncStatus{
			CurrentL1: eth.L1BlockRef{
				Hash:   common.Hash{0x01},
				Number: 100,
			},
			SafeL2: eth.L2BlockRef{
				Hash:   common.Hash{0x02},
				Number: 50,
			},
			LocalSafeL2: eth.L2BlockRef{
				Hash:   common.Hash{0x03},
				Number: 45, // LocalSafe can be different from (cross) Safe
			},
			FinalizedL2: eth.L2BlockRef{
				Hash:   common.Hash{0x04},
				Number: 40,
			},
		}

		rpc.On("CallContext", ctx, mock.AnythingOfType("**eth.SyncStatus"),
			"optimism_syncStatus", []any(nil)).Run(func(args mock.Arguments) {
			// Set the result pointer to our mock sync status
			*args[1].(**eth.SyncStatus) = mockSyncStatus
		}).Return([]error{nil})

		status, err := client.GetFollowStatus(ctx)
		require.NoError(t, err)

		// Verify all fields are correctly copied
		require.Equal(t, mockSyncStatus.CurrentL1, status.CurrentL1, "CurrentL1 should be copied")
		require.Equal(t, mockSyncStatus.SafeL2, status.SafeL2, "SafeL2 should be copied")
		require.Equal(t, mockSyncStatus.FinalizedL2, status.FinalizedL2, "FinalizedL2 should be copied")
		require.Equal(t, mockSyncStatus.LocalSafeL2, status.LocalSafeL2, "LocalSafeL2 should be copied")
	})

	t.Run("Error", func(t *testing.T) {
		ctx := context.Background()
		rpc := new(mockRPC)
		defer rpc.AssertExpectations(t)

		client, err := NewFollowClient(rpc)
		require.NoError(t, err)

		rpc.On("CallContext", ctx, mock.AnythingOfType("**eth.SyncStatus"),
			"optimism_syncStatus", []any(nil)).Return([]error{context.DeadlineExceeded})

		_, err = client.GetFollowStatus(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to fetch external syncStatus")
	})
}
