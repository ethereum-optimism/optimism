package sysgo

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestBatcherTestControl_PauseAtBlock(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	// Setup mock clients
	blocks := make(map[uint64]*types.Block)
	for i := uint64(1); i <= 100; i++ {
		blocks[i] = newMockBlock(i)
	}

	mockRollup := &mockRollupClient{
		syncStatus: &eth.SyncStatus{
			UnsafeL2: eth.L2BlockRef{
				Hash:   blocks[100].Hash(),
				Number: 100,
			},
		},
	}
	mockL2 := &mockL2Client{blocks: blocks}

	proxy := newRollupClientProxy(mockRollup, mockL2)
	testControl := &batcherTestControl{
		proxy: proxy,
		log:   logger,
	}

	// Pause at block 50
	result := testControl.PauseAtBlock(50)

	// Should return 50 (the highest block the batcher will see)
	require.Equal(t, uint64(50), result)

	// Verify the proxy is actually paused at 50
	require.Equal(t, uint64(50), proxy.getPause())

	// Verify the proxy caps the unsafe head
	ctx := context.Background()
	status, err := proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(50), status.UnsafeL2.Number)
}

func TestBatcherTestControl_Unpause(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	// Setup mock clients
	blocks := make(map[uint64]*types.Block)
	for i := uint64(1); i <= 100; i++ {
		blocks[i] = newMockBlock(i)
	}

	mockRollup := &mockRollupClient{
		syncStatus: &eth.SyncStatus{
			UnsafeL2: eth.L2BlockRef{
				Hash:   blocks[100].Hash(),
				Number: 100,
			},
		},
	}
	mockL2 := &mockL2Client{blocks: blocks}

	proxy := newRollupClientProxy(mockRollup, mockL2)
	testControl := &batcherTestControl{
		proxy: proxy,
		log:   logger,
	}

	// Pause at block 50
	testControl.PauseAtBlock(50)
	require.Equal(t, uint64(50), proxy.getPause())

	// Unpause
	testControl.Unpause()

	// Verify the proxy is cleared
	require.Equal(t, uint64(0), proxy.getPause())

	// Verify the proxy no longer caps the unsafe head
	ctx := context.Background()
	status, err := proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), status.UnsafeL2.Number)
}

func TestBatcherTestControl_PauseAndUnpauseCycle(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	blocks := make(map[uint64]*types.Block)
	for i := uint64(1); i <= 100; i++ {
		blocks[i] = newMockBlock(i)
	}

	mockRollup := &mockRollupClient{
		syncStatus: &eth.SyncStatus{
			UnsafeL2: eth.L2BlockRef{
				Hash:   blocks[100].Hash(),
				Number: 100,
			},
		},
	}
	mockL2 := &mockL2Client{blocks: blocks}

	proxy := newRollupClientProxy(mockRollup, mockL2)
	testControl := &batcherTestControl{
		proxy: proxy,
		log:   logger,
	}

	ctx := context.Background()

	// Initially unpaused
	status, err := proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), status.UnsafeL2.Number)

	// Pause at 30
	result := testControl.PauseAtBlock(30)
	require.Equal(t, uint64(30), result)

	status, err = proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(30), status.UnsafeL2.Number)

	// Pause at 60 (change pause point)
	result = testControl.PauseAtBlock(60)
	require.Equal(t, uint64(60), result)

	status, err = proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(60), status.UnsafeL2.Number)

	// Unpause
	testControl.Unpause()

	status, err = proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), status.UnsafeL2.Number)
}

func TestL2Batcher_TestControl(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	blocks := make(map[uint64]*types.Block)
	for i := uint64(1); i <= 100; i++ {
		blocks[i] = newMockBlock(i)
	}

	mockRollup := &mockRollupClient{
		syncStatus: &eth.SyncStatus{
			UnsafeL2: eth.L2BlockRef{
				Hash:   blocks[100].Hash(),
				Number: 100,
			},
		},
	}
	mockL2 := &mockL2Client{blocks: blocks}

	proxy := newRollupClientProxy(mockRollup, mockL2)
	testControl := &batcherTestControl{
		proxy: proxy,
		log:   logger,
	}

	// Create L2Batcher with test control
	batcher := &L2Batcher{
		testControl: testControl,
	}

	// Verify TestControl() returns the test control
	tc := batcher.TestControl()
	require.NotNil(t, tc)

	// Verify it's the same instance
	require.Equal(t, testControl, tc)

	// Verify we can use it to pause/unpause
	ctx := context.Background()
	result := tc.PauseAtBlock(50)
	require.Equal(t, uint64(50), result)

	status, err := proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(50), status.UnsafeL2.Number)

	tc.Unpause()
	status, err = proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), status.UnsafeL2.Number)
}
