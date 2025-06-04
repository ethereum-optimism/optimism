package syncnode

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// mockResetBackend implements the resetBackend interface for testing
type mockResetBackend struct {
	// nodeBlocks represents blocks known to the node
	nodeBlocks map[uint64]eth.BlockID
	// safeBlocks represents blocks marked as safe in the local DB
	safeBlocks map[uint64]eth.BlockID
}

func (m *mockResetBackend) reset() {
	m.nodeBlocks = make(map[uint64]eth.BlockID)
	m.safeBlocks = make(map[uint64]eth.BlockID)
}

func (m *mockResetBackend) BlockIDByNumber(ctx context.Context, n uint64) (eth.BlockID, error) {
	if block, ok := m.nodeBlocks[n]; ok {
		return block, nil
	}
	return eth.BlockID{}, ethereum.NotFound
}

func (m *mockResetBackend) IsLocalSafe(ctx context.Context, block eth.BlockID) error {
	if safeBlock, ok := m.safeBlocks[block.Number]; ok {
		if safeBlock == block {
			return nil
		}
		return types.ErrConflict
	}
	return types.ErrFuture
}

func TestResetTracker(t *testing.T) {
	logger := testlog.Logger(t, log.LvlDebug)
	backend := new(mockResetBackend)
	tracker := newResetTracker(logger, backend)
	ctx := context.Background()

	// Helper to create a block ID with a specific hash
	mkBlock := func(n uint64, nodeDivHash bool) eth.BlockID {
		hash := common.Hash{byte(n)}
		if nodeDivHash {
			hash[1] = 0xff
		}
		return eth.BlockID{Number: n, Hash: hash}
	}

	// Helper to set up a range of blocks
	// start: first block number in range
	// endNode: last block number in node
	// endLocal: last block number in local DB
	// divergence: block number at which node and safe DB hashes start to differ
	setupRange := func(start, endNode, endLocal, divergence uint64) {
		for i := start; i <= endNode; i++ {
			backend.nodeBlocks[i] = mkBlock(i, i >= divergence)
		}

		for i := start; i <= endLocal; i++ {
			backend.safeBlocks[i] = mkBlock(i, false)
		}
	}

	t.Run("pre-interop start block not found in node", func(t *testing.T) {
		backend.reset()
		target, err := tracker.FindResetTarget(ctx, mkBlock(1, false), mkBlock(10, false))
		require.NoError(t, err)
		require.True(t, target.PreInterop, "target is instead %v", target.Target)
	})

	t.Run("pre-interop start block inconsistent", func(t *testing.T) {
		backend.reset()
		setupRange(1, 10, 10, 1) // divergence at start, so all blocks are inconsistent
		target, err := tracker.FindResetTarget(ctx, mkBlock(1, false), mkBlock(10, false))
		require.NoError(t, err)
		require.True(t, target.PreInterop, "target is instead %v", target.Target)
	})

	t.Run("target found when end block is consistent", func(t *testing.T) {
		backend.reset()
		setupRange(1, 10, 10, 11) // divergence after range, so all blocks are consistent
		target, err := tracker.FindResetTarget(ctx, mkBlock(1, false), mkBlock(10, false))
		require.NoError(t, err)
		require.False(t, target.PreInterop)
		require.Equal(t, uint64(10), target.Target.Number)
		require.Equal(t, common.Hash{0x0a}, target.Target.Hash)
	})

	t.Run("bisection finds last consistent block", func(t *testing.T) {
		const rangeEnd = uint64(17)
		for divergence := uint64(2); divergence <= rangeEnd; divergence++ {
			t.Run(fmt.Sprintf("divergence at %d", divergence), func(t *testing.T) {
				backend.reset()
				setupRange(1, rangeEnd, rangeEnd, divergence)
				target, err := tracker.FindResetTarget(ctx, mkBlock(1, false), mkBlock(rangeEnd, false))
				require.NoError(t, err)
				require.False(t, target.PreInterop)
				require.Equal(t, divergence-1, target.Target.Number)
				require.Equal(t, common.Hash{byte(divergence - 1)}, target.Target.Hash)
			})
		}
	})

	t.Run("converges to start when range is small", func(t *testing.T) {
		backend.reset()
		// Set up a small range where only the start is consistent
		setupRange(1, 2, 2, 2)
		target, err := tracker.FindResetTarget(ctx, mkBlock(1, false), mkBlock(2, false))
		require.NoError(t, err)
		require.False(t, target.PreInterop)
		require.Equal(t, uint64(1), target.Target.Number)
		require.Equal(t, common.Hash{0x01}, target.Target.Hash)
	})

	t.Run("handles node ahead of local DB", func(t *testing.T) {
		backend.reset()
		// Node has more blocks than local DB
		setupRange(1, 10, 5, 11) // node has 1-10, local has 1-5
		target, err := tracker.FindResetTarget(ctx, mkBlock(1, false), mkBlock(5, false))
		require.NoError(t, err)
		require.False(t, target.PreInterop)
		require.Equal(t, uint64(5), target.Target.Number)
		require.Equal(t, common.Hash{0x05}, target.Target.Hash)
	})

	t.Run("handles local DB ahead of node", func(t *testing.T) {
		backend.reset()
		// Local DB has more blocks than node
		setupRange(1, 5, 10, 11) // node has 1-5, local has 1-10
		target, err := tracker.FindResetTarget(ctx, mkBlock(1, false), mkBlock(10, false))
		require.NoError(t, err)
		require.False(t, target.PreInterop)
		require.Equal(t, uint64(5), target.Target.Number)
		require.Equal(t, common.Hash{0x05}, target.Target.Hash)
	})
}
