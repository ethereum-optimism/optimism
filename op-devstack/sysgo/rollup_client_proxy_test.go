package sysgo

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// mockRollupClient is a mock implementation of RollupClient for testing
type mockRollupClient struct {
	syncStatus *eth.SyncStatus
	err        error
}

func (m *mockRollupClient) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return m.syncStatus, m.err
}

func (m *mockRollupClient) Close() {}

func (m *mockRollupClient) OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	return nil, nil
}

func (m *mockRollupClient) RollupConfig(ctx context.Context) (*rollup.Config, error) {
	return nil, nil
}

func (m *mockRollupClient) StartSequencer(ctx context.Context, unsafeHead common.Hash) error {
	return nil
}

func (m *mockRollupClient) SequencerActive(ctx context.Context) (bool, error) {
	return false, nil
}

// mockL2Client is a mock implementation of L2Client for testing
type mockL2Client struct {
	blocks map[uint64]*types.Block
}

func (m *mockL2Client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	if number == nil {
		return nil, nil
	}
	block, ok := m.blocks[bigs.Uint64Strict(number)]
	if !ok {
		return nil, nil
	}
	return block, nil
}

// newMockBlock creates a mock block for testing
func newMockBlock(number uint64) *types.Block {
	header := &types.Header{
		Number: big.NewInt(int64(number)),
		Time:   1000 + number,
	}
	// Set a deterministic hash based on block number
	header.Extra = []byte{byte(number)}
	return types.NewBlock(header, &types.Body{}, nil, trie.NewStackTrie(nil), types.DefaultBlockConfig)
}

func TestRollupClientProxy_Unpaused(t *testing.T) {
	ctx := context.Background()

	// Setup mock clients
	mockRollup := &mockRollupClient{
		syncStatus: &eth.SyncStatus{
			UnsafeL2: eth.L2BlockRef{
				Hash:   common.Hash{0x1},
				Number: 100,
			},
			SafeL2: eth.L2BlockRef{
				Hash:   common.Hash{0x2},
				Number: 95,
			},
			FinalizedL2: eth.L2BlockRef{
				Hash:   common.Hash{0x3},
				Number: 90,
			},
		},
	}
	mockL2 := &mockL2Client{blocks: make(map[uint64]*types.Block)}

	proxy := newRollupClientProxy(mockRollup, mockL2)

	// When not paused, should pass through unchanged
	status, err := proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Equal(t, uint64(100), status.UnsafeL2.Number)
	require.Equal(t, uint64(95), status.SafeL2.Number)
	require.Equal(t, uint64(90), status.FinalizedL2.Number)
}

func TestRollupClientProxy_PausedCapsUnsafeHead(t *testing.T) {
	ctx := context.Background()

	// Setup mock blocks
	blocks := make(map[uint64]*types.Block)
	for i := uint64(90); i <= 100; i++ {
		blocks[i] = newMockBlock(i)
	}

	mockRollup := &mockRollupClient{
		syncStatus: &eth.SyncStatus{
			UnsafeL2: eth.L2BlockRef{
				Hash:   blocks[100].Hash(),
				Number: 100,
			},
			SafeL2: eth.L2BlockRef{
				Hash:   blocks[95].Hash(),
				Number: 95,
			},
			FinalizedL2: eth.L2BlockRef{
				Hash:   blocks[90].Hash(),
				Number: 90,
			},
		},
	}
	mockL2 := &mockL2Client{blocks: blocks}

	proxy := newRollupClientProxy(mockRollup, mockL2)

	// Pause at block 98 (batcher should see up to and including 98)
	proxy.setPauseAtBlock(98)

	// Should cap unsafe head at 98 (pauseNum, inclusive)
	status, err := proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Equal(t, uint64(98), status.UnsafeL2.Number)
	require.Equal(t, blocks[98].Hash(), status.UnsafeL2.Hash)
	require.Equal(t, blocks[98].ParentHash(), status.UnsafeL2.ParentHash)
	require.Equal(t, blocks[98].Time(), status.UnsafeL2.Time)

	// Safe and finalized should be unchanged (they're below pause point)
	require.Equal(t, uint64(95), status.SafeL2.Number)
	require.Equal(t, uint64(90), status.FinalizedL2.Number)
}

func TestRollupClientProxy_PausedAtBlockStart(t *testing.T) {
	ctx := context.Background()

	// Setup mock blocks
	blocks := make(map[uint64]*types.Block)
	for i := uint64(1); i <= 10; i++ {
		blocks[i] = newMockBlock(i)
	}

	mockRollup := &mockRollupClient{
		syncStatus: &eth.SyncStatus{
			UnsafeL2: eth.L2BlockRef{
				Hash:   blocks[10].Hash(),
				Number: 10,
			},
			SafeL2: eth.L2BlockRef{
				Hash:   blocks[5].Hash(),
				Number: 5,
			},
			FinalizedL2: eth.L2BlockRef{
				Hash:   blocks[1].Hash(),
				Number: 1,
			},
		},
	}
	mockL2 := &mockL2Client{blocks: blocks}

	proxy := newRollupClientProxy(mockRollup, mockL2)

	// Pause at block 8 (batcher should see up to and including 8)
	proxy.setPauseAtBlock(8)

	status, err := proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(8), status.UnsafeL2.Number)
	require.Equal(t, blocks[8].Hash(), status.UnsafeL2.Hash)
}

func TestRollupClientProxy_UnsafeHeadAtOrBelowPause(t *testing.T) {
	ctx := context.Background()

	blocks := make(map[uint64]*types.Block)
	for i := uint64(1); i <= 100; i++ {
		blocks[i] = newMockBlock(i)
	}

	mockRollup := &mockRollupClient{
		syncStatus: &eth.SyncStatus{
			UnsafeL2: eth.L2BlockRef{
				Hash:   blocks[50].Hash(),
				Number: 50,
			},
			SafeL2: eth.L2BlockRef{
				Hash:   blocks[45].Hash(),
				Number: 45,
			},
			FinalizedL2: eth.L2BlockRef{
				Hash:   blocks[40].Hash(),
				Number: 40,
			},
		},
	}
	mockL2 := &mockL2Client{blocks: blocks}

	proxy := newRollupClientProxy(mockRollup, mockL2)

	// Pause at block 100 (well above current unsafe head)
	proxy.setPauseAtBlock(100)

	// Should pass through unchanged since unsafe head (50) <= pauseNum (100)
	status, err := proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(50), status.UnsafeL2.Number)
	require.Equal(t, blocks[50].Hash(), status.UnsafeL2.Hash)

	// Also test when unsafe head exactly equals pause point
	mockRollup.syncStatus.UnsafeL2.Hash = blocks[100].Hash()
	mockRollup.syncStatus.UnsafeL2.Number = 100

	status, err = proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), status.UnsafeL2.Number)
	require.Equal(t, blocks[100].Hash(), status.UnsafeL2.Hash)
}

func TestRollupClientProxy_ClearPause(t *testing.T) {
	ctx := context.Background()

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

	// Set pause at 50 (batcher should see up to and including 50)
	proxy.setPauseAtBlock(50)
	status, err := proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(50), status.UnsafeL2.Number)

	// Clear pause
	proxy.clearPause()
	status, err = proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), status.UnsafeL2.Number)
}

func TestRollupClientProxy_GetPause(t *testing.T) {
	mockRollup := &mockRollupClient{}
	mockL2 := &mockL2Client{blocks: make(map[uint64]*types.Block)}

	proxy := newRollupClientProxy(mockRollup, mockL2)

	// Initially no pause
	require.Equal(t, uint64(0), proxy.getPause())

	// Set pause
	proxy.setPauseAtBlock(42)
	require.Equal(t, uint64(42), proxy.getPause())

	// Clear pause
	proxy.clearPause()
	require.Equal(t, uint64(0), proxy.getPause())
}

func TestRollupClientProxy_SafeAndFinalizedCapping(t *testing.T) {
	ctx := context.Background()

	blocks := make(map[uint64]*types.Block)
	for i := uint64(1); i <= 100; i++ {
		blocks[i] = newMockBlock(i)
	}

	// Set up a scenario where safe and finalized are somehow ahead of pause
	// (this shouldn't normally happen, but we're defensive)
	mockRollup := &mockRollupClient{
		syncStatus: &eth.SyncStatus{
			UnsafeL2: eth.L2BlockRef{
				Hash:   blocks[100].Hash(),
				Number: 100,
			},
			SafeL2: eth.L2BlockRef{
				Hash:   blocks[100].Hash(),
				Number: 100,
			},
			FinalizedL2: eth.L2BlockRef{
				Hash:   blocks[100].Hash(),
				Number: 100,
			},
		},
	}
	mockL2 := &mockL2Client{blocks: blocks}

	proxy := newRollupClientProxy(mockRollup, mockL2)
	proxy.setPauseAtBlock(50)

	status, err := proxy.SyncStatus(ctx)
	require.NoError(t, err)

	// All heads should be capped to 50 (inclusive)
	require.Equal(t, uint64(50), status.UnsafeL2.Number)
	require.Equal(t, uint64(50), status.SafeL2.Number)
	require.Equal(t, uint64(50), status.FinalizedL2.Number)

	// They should all have the same (correct) hash
	require.Equal(t, blocks[50].Hash(), status.UnsafeL2.Hash)
	require.Equal(t, blocks[50].Hash(), status.SafeL2.Hash)
	require.Equal(t, blocks[50].Hash(), status.FinalizedL2.Hash)
}

func TestRollupClientProxy_ErrorPassthrough(t *testing.T) {
	ctx := context.Background()

	mockRollup := &mockRollupClient{
		err: context.DeadlineExceeded,
	}
	mockL2 := &mockL2Client{blocks: make(map[uint64]*types.Block)}

	proxy := newRollupClientProxy(mockRollup, mockL2)
	proxy.setPauseAtBlock(50)

	// Error should pass through unchanged
	_, err := proxy.SyncStatus(ctx)
	require.Error(t, err)
	require.Equal(t, context.DeadlineExceeded, err)
}

func TestRollupClientProxy_NilStatusPassthrough(t *testing.T) {
	ctx := context.Background()

	mockRollup := &mockRollupClient{
		syncStatus: nil,
		err:        nil,
	}
	mockL2 := &mockL2Client{blocks: make(map[uint64]*types.Block)}

	proxy := newRollupClientProxy(mockRollup, mockL2)
	proxy.setPauseAtBlock(50)

	// Nil status should pass through
	status, err := proxy.SyncStatus(ctx)
	require.NoError(t, err)
	require.Nil(t, status)
}

func TestRollupClientProxy_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()

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

	// Test concurrent access doesn't panic
	done := make(chan bool, 3)

	// Goroutine 1: Set/clear pause
	go func() {
		for i := 0; i < 100; i++ {
			proxy.setPauseAtBlock(50)
			proxy.clearPause()
		}
		done <- true
	}()

	// Goroutine 2: Read pause
	go func() {
		for i := 0; i < 100; i++ {
			_ = proxy.getPause()
		}
		done <- true
	}()

	// Goroutine 3: Call SyncStatus
	go func() {
		for i := 0; i < 100; i++ {
			_, _ = proxy.SyncStatus(ctx)
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done
}
