package interop

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TestNew
// =============================================================================

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("valid inputs initializes all components", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		chains := map[eth.ChainID]cc.ChainContainer{
			eth.ChainIDFromUInt64(10):   newMockChainContainer(10),
			eth.ChainIDFromUInt64(8453): newMockChainContainer(8453),
		}

		interop := New(testLogger(), 1000, chains, dataDir)

		require.NotNil(t, interop)
		require.Equal(t, uint64(1000), interop.activationTimestamp)
		require.NotNil(t, interop.verifiedDB)
		require.Len(t, interop.chains, 2)
		require.Len(t, interop.logsDBs, 2)
		require.NotNil(t, interop.verifyFn)

		// Verify logsDBs populated for each chain
		for chainID := range chains {
			require.Contains(t, interop.logsDBs, chainID)
			require.NotNil(t, interop.logsDBs[chainID])
		}
	})

	t.Run("invalid dataDir returns nil", func(t *testing.T) {
		t.Parallel()

		interop := New(testLogger(), 1000, map[eth.ChainID]cc.ChainContainer{}, "/nonexistent/path")

		require.Nil(t, interop)
	})
}

// =============================================================================
// TestStartStop
// =============================================================================

func TestStartStop(t *testing.T) {
	t.Parallel()

	t.Run("Start blocks until context cancelled", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
		mock.blockAtTimestamp = eth.L2BlockRef{Number: 50}

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- interop.Start(ctx) }()

		// Wait for start
		require.Eventually(t, func() bool {
			interop.mu.RLock()
			defer interop.mu.RUnlock()
			return interop.started
		}, 5*time.Second, 100*time.Millisecond)

		cancel()

		var err error
		require.Eventually(t, func() bool {
			select {
			case err = <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("double Start blocked", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() { _ = interop.Start(ctx) }()

		require.Eventually(t, func() bool {
			interop.mu.RLock()
			defer interop.mu.RUnlock()
			return interop.started
		}, 5*time.Second, 100*time.Millisecond)

		ctx2, cancel2 := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel2()

		err := interop.Start(ctx2)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("Stop cancels running Start and closes DB", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
		mock.blockAtTimestampErr = ethereum.NotFound

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)

		done := make(chan error, 1)
		go func() { done <- interop.Start(context.Background()) }()

		require.Eventually(t, func() bool {
			interop.mu.RLock()
			defer interop.mu.RUnlock()
			return interop.started
		}, 5*time.Second, 100*time.Millisecond)

		err := interop.Stop(context.Background())
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		// Verify DB is closed
		_, err = interop.verifiedDB.Has(100)
		require.Error(t, err)
	})
}

// =============================================================================
// TestCollectCurrentL1
// =============================================================================

func TestCollectCurrentL1(t *testing.T) {
	t.Parallel()

	t.Run("returns minimum L1 across multiple chains", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock1 := newMockChainContainer(10)
		mock1.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x2")}

		mock2 := newMockChainContainer(8453)
		mock2.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")} // minimum

		chains := map[eth.ChainID]cc.ChainContainer{mock1.id: mock1, mock2.id: mock2}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()

		l1, err := interop.collectCurrentL1()

		require.NoError(t, err)
		require.Equal(t, uint64(100), l1.Number)
		require.Equal(t, common.HexToHash("0x1"), l1.Hash)
	})

	t.Run("single chain returns its L1", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.currentL1 = eth.BlockRef{Number: 500, Hash: common.HexToHash("0x5")}

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()

		l1, err := interop.collectCurrentL1()

		require.NoError(t, err)
		require.Equal(t, uint64(500), l1.Number)
	})

	t.Run("chain error propagated", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.currentL1Err = errors.New("chain not synced")

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()

		l1, err := interop.collectCurrentL1()

		require.Error(t, err)
		require.Contains(t, err.Error(), "not ready")
		require.Equal(t, eth.BlockID{}, l1)
	})
}

// =============================================================================
// TestCheckChainsReady
// =============================================================================

func TestCheckChainsReady(t *testing.T) {
	t.Parallel()

	t.Run("all chains ready returns blocks", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock1 := newMockChainContainer(10)
		mock1.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}

		mock2 := newMockChainContainer(8453)
		mock2.blockAtTimestamp = eth.L2BlockRef{Number: 200, Hash: common.HexToHash("0x2")}

		chains := map[eth.ChainID]cc.ChainContainer{mock1.id: mock1, mock2.id: mock2}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()

		blocks, err := interop.checkChainsReady(1000)

		require.NoError(t, err)
		require.Len(t, blocks, 2)
		require.NotEqual(t, common.Hash{}, blocks[mock1.id].Hash)
		require.NotEqual(t, common.Hash{}, blocks[mock2.id].Hash)
	})

	t.Run("one chain not ready returns error", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock1 := newMockChainContainer(10)
		mock1.blockAtTimestamp = eth.L2BlockRef{Number: 100}

		mock2 := newMockChainContainer(8453)
		mock2.blockAtTimestampErr = ethereum.NotFound

		chains := map[eth.ChainID]cc.ChainContainer{mock1.id: mock1, mock2.id: mock2}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()

		blocks, err := interop.checkChainsReady(1000)

		require.Error(t, err)
		require.Nil(t, blocks)
	})

	t.Run("parallel execution works", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		chains := make(map[eth.ChainID]cc.ChainContainer)
		for i := 0; i < 5; i++ {
			mock := newMockChainContainer(uint64(10 + i))
			mock.blockAtTimestamp = eth.L2BlockRef{Number: uint64(100 + i)}
			chains[mock.id] = mock
		}

		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()

		blocks, err := interop.checkChainsReady(1000)

		require.NoError(t, err)
		require.Len(t, blocks, 5)
	})
}

// =============================================================================
// TestProgressInterop
// =============================================================================

func TestProgressInterop(t *testing.T) {
	t.Parallel()

	t.Run("not initialized uses activation timestamp", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 5000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()

		var capturedTimestamp uint64
		interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
			capturedTimestamp = ts
			return Result{Timestamp: ts, L2Heads: blocks}, nil
		}

		result, err := interop.progressInterop()

		require.NoError(t, err)
		require.Equal(t, uint64(5000), result.Timestamp)
		require.Equal(t, uint64(5000), capturedTimestamp)
	})

	t.Run("initialized uses next timestamp", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()
		interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
			return Result{Timestamp: ts, L2Heads: blocks}, nil
		}

		// First progress
		result1, err := interop.progressInterop()
		require.NoError(t, err)
		require.Equal(t, uint64(1000), result1.Timestamp)

		// Commit
		err = interop.handleResult(result1)
		require.NoError(t, err)

		// Second progress should use next timestamp
		result2, err := interop.progressInterop()
		require.NoError(t, err)
		require.Equal(t, uint64(1001), result2.Timestamp)
	})

	t.Run("chains not ready returns empty result", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.blockAtTimestampErr = ethereum.NotFound

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()

		result, err := interop.progressInterop()

		require.NoError(t, err)
		require.True(t, result.IsEmpty())
	})

	t.Run("chain error propagated", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.blockAtTimestampErr = errors.New("internal error")

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()

		result, err := interop.progressInterop()

		require.Error(t, err)
		require.True(t, result.IsEmpty())
	})

	t.Run("verifyFn error propagated", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.currentL1 = eth.BlockRef{Number: 1000, Hash: common.HexToHash("0xL1")}
		mock.blockAtTimestamp = eth.L2BlockRef{Number: 500, Hash: common.HexToHash("0xL2")}

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 100, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()
		interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
			return Result{}, errors.New("verification failed")
		}

		result, err := interop.progressInterop()

		require.Error(t, err)
		require.Contains(t, err.Error(), "verification failed")
		require.True(t, result.IsEmpty())
	})
}

// =============================================================================
// TestVerifiedAtTimestamp
// =============================================================================

func TestVerifiedAtTimestamp(t *testing.T) {
	t.Parallel()

	t.Run("before activation always verified", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		interop := New(testLogger(), 1000, map[eth.ChainID]cc.ChainContainer{}, dataDir)
		require.NotNil(t, interop)

		verified, err := interop.VerifiedAtTimestamp(999)
		require.NoError(t, err)
		require.True(t, verified)

		verified, err = interop.VerifiedAtTimestamp(0)
		require.NoError(t, err)
		require.True(t, verified)
	})

	t.Run("at/after activation not verified until committed", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		interop := New(testLogger(), 1000, map[eth.ChainID]cc.ChainContainer{}, dataDir)
		require.NotNil(t, interop)

		verified, err := interop.VerifiedAtTimestamp(1000)
		require.NoError(t, err)
		require.False(t, verified)

		verified, err = interop.VerifiedAtTimestamp(9999)
		require.NoError(t, err)
		require.False(t, verified)
	})

	t.Run("committed timestamp verified", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.blockAtTimestamp = eth.L2BlockRef{Number: 100}

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()
		interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
			return Result{Timestamp: ts, L2Heads: blocks}, nil
		}

		result, err := interop.progressInterop()
		require.NoError(t, err)

		err = interop.handleResult(result)
		require.NoError(t, err)

		verified, err := interop.VerifiedAtTimestamp(1000)
		require.NoError(t, err)
		require.True(t, verified)
	})
}

// =============================================================================
// TestHandleResult
// =============================================================================

func TestHandleResult(t *testing.T) {
	t.Parallel()

	t.Run("empty result is no-op", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		interop := New(testLogger(), 1000, map[eth.ChainID]cc.ChainContainer{}, dataDir)
		require.NotNil(t, interop)

		err := interop.handleResult(Result{})
		require.NoError(t, err)

		has, err := interop.verifiedDB.Has(0)
		require.NoError(t, err)
		require.False(t, has)
	})

	t.Run("valid result commits to DB with correct data", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)

		validResult := Result{
			Timestamp: 1000,
			L1Head:    eth.BlockID{Number: 100, Hash: common.HexToHash("0xL1")},
			L2Heads: map[eth.ChainID]eth.BlockID{
				mock.id: {Number: 500, Hash: common.HexToHash("0xL2")},
			},
		}

		err := interop.handleResult(validResult)
		require.NoError(t, err)

		has, err := interop.verifiedDB.Has(1000)
		require.NoError(t, err)
		require.True(t, has)

		retrieved, err := interop.verifiedDB.Get(1000)
		require.NoError(t, err)
		require.Equal(t, validResult.Timestamp, retrieved.Timestamp)
		require.Equal(t, validResult.L1Head, retrieved.L1Head)
		require.Equal(t, validResult.L2Heads[mock.id], retrieved.L2Heads[mock.id])
	})

	t.Run("invalid result does not commit", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)

		invalidResult := Result{
			Timestamp: 1000,
			L1Head:    eth.BlockID{Number: 100, Hash: common.HexToHash("0xL1")},
			L2Heads: map[eth.ChainID]eth.BlockID{
				mock.id: {Number: 500, Hash: common.HexToHash("0xL2")},
			},
			InvalidHeads: map[eth.ChainID]eth.BlockID{
				mock.id: {Number: 500, Hash: common.HexToHash("0xBAD")},
			},
		}

		err := interop.handleResult(invalidResult)
		require.NoError(t, err)

		has, err := interop.verifiedDB.Has(1000)
		require.NoError(t, err)
		require.False(t, has)
	})
}

// =============================================================================
// TestProgressAndRecord
// =============================================================================

func TestProgressAndRecord(t *testing.T) {
	t.Parallel()

	t.Run("empty result sets L1 to collected minimum", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock1 := newMockChainContainer(10)
		mock1.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
		mock1.blockAtTimestampErr = ethereum.NotFound

		mock2 := newMockChainContainer(8453)
		mock2.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
		mock2.blockAtTimestampErr = ethereum.NotFound

		chains := map[eth.ChainID]cc.ChainContainer{mock1.id: mock1, mock2.id: mock2}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()

		require.Equal(t, eth.BlockID{}, interop.currentL1)

		madeProgress, err := interop.progressAndRecord()
		require.NoError(t, err)
		require.False(t, madeProgress, "empty result should not advance verified timestamp")

		require.Equal(t, uint64(100), interop.currentL1.Number)
		require.Equal(t, common.HexToHash("0x1"), interop.currentL1.Hash)
	})

	t.Run("valid result sets L1 to result L1Head", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x200")}
		mock.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0xL2")}

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()

		expectedL1Head := eth.BlockID{Number: 150, Hash: common.HexToHash("0xL1Result")}
		interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
			return Result{Timestamp: ts, L1Head: expectedL1Head, L2Heads: blocks}, nil
		}

		madeProgress, err := interop.progressAndRecord()
		require.NoError(t, err)
		require.True(t, madeProgress, "valid result should advance verified timestamp")

		require.Equal(t, expectedL1Head.Number, interop.currentL1.Number)
		require.Equal(t, expectedL1Head.Hash, interop.currentL1.Hash)
	})

	t.Run("invalid result does not update L1", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x200")}
		mock.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0xL2")}

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()

		initialL1 := eth.BlockID{Number: 50, Hash: common.HexToHash("0x50")}
		interop.currentL1 = initialL1

		interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
			return Result{
				Timestamp:    ts,
				L1Head:       eth.BlockID{Number: 999, Hash: common.HexToHash("0xShouldNotBeUsed")},
				L2Heads:      blocks,
				InvalidHeads: map[eth.ChainID]eth.BlockID{mock.id: {Number: 100}},
			}, nil
		}

		madeProgress, err := interop.progressAndRecord()
		require.NoError(t, err)
		require.False(t, madeProgress, "invalid result should not advance verified timestamp")

		require.Equal(t, initialL1.Number, interop.currentL1.Number)
		require.Equal(t, initialL1.Hash, interop.currentL1.Hash)
	})

	t.Run("errors propagated", func(t *testing.T) {
		t.Parallel()
		dataDir := t.TempDir()

		mock := newMockChainContainer(10)
		mock.currentL1Err = errors.New("L1 sync error")

		chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
		interop := New(testLogger(), 1000, chains, dataDir)
		require.NotNil(t, interop)
		interop.ctx = context.Background()

		madeProgress, err := interop.progressAndRecord()
		require.Error(t, err)
		require.False(t, madeProgress, "error should not advance verified timestamp")
	})
}

// =============================================================================
// TestInterop_FullCycle
// =============================================================================

func TestInterop_FullCycle(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.currentL1 = eth.BlockRef{Number: 1000, Hash: common.HexToHash("0xL1")}
	mock.blockAtTimestamp = eth.L2BlockRef{Number: 500, Hash: common.HexToHash("0xL2")}

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 100, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	// Verify logsDB is empty initially
	_, hasBlocks := interop.logsDBs[mock.id].LatestSealedBlock()
	require.False(t, hasBlocks)

	// Stub verifyFn
	interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
		return Result{Timestamp: ts, L2Heads: blocks}, nil
	}

	// Run 3 cycles
	for i := 0; i < 3; i++ {
		l1, err := interop.collectCurrentL1()
		require.NoError(t, err)
		require.Equal(t, uint64(1000), l1.Number)

		result, err := interop.progressInterop()
		require.NoError(t, err)
		require.False(t, result.IsEmpty())

		err = interop.handleResult(result)
		require.NoError(t, err)
	}

	// Verify timestamps committed with correct L2Heads
	for ts := uint64(100); ts <= 102; ts++ {
		has, err := interop.verifiedDB.Has(ts)
		require.NoError(t, err)
		require.True(t, has)

		retrieved, err := interop.verifiedDB.Get(ts)
		require.NoError(t, err)
		require.Equal(t, ts, retrieved.Timestamp)
		require.Contains(t, retrieved.L2Heads, mock.id)
		require.Equal(t, ts, retrieved.L2Heads[mock.id].Number)
	}

	// Verify logsDB populated
	latestBlock, hasBlocks := interop.logsDBs[mock.id].LatestSealedBlock()
	require.True(t, hasBlocks)
	require.Equal(t, uint64(102), latestBlock.Number)
}

// =============================================================================
// TestResult_IsEmpty
// =============================================================================

func TestResult_IsEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		result  Result
		isEmpty bool
	}{
		{"zero value", Result{}, true},
		{"only timestamp", Result{Timestamp: 1000}, true},
		{"with L1Head", Result{Timestamp: 1000, L1Head: eth.BlockID{Number: 100}}, false},
		{"with L2Heads", Result{Timestamp: 1000, L2Heads: map[eth.ChainID]eth.BlockID{eth.ChainIDFromUInt64(10): {Number: 50}}}, false},
		{"with InvalidHeads", Result{Timestamp: 1000, InvalidHeads: map[eth.ChainID]eth.BlockID{eth.ChainIDFromUInt64(10): {Number: 50}}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isEmpty, tt.result.IsEmpty())
		})
	}
}

// =============================================================================
// Mock Types
// =============================================================================

type mockBlockInfo struct {
	hash       common.Hash
	parentHash common.Hash
	number     uint64
	timestamp  uint64
}

func (m *mockBlockInfo) Hash() common.Hash                                    { return m.hash }
func (m *mockBlockInfo) ParentHash() common.Hash                              { return m.parentHash }
func (m *mockBlockInfo) Coinbase() common.Address                             { return common.Address{} }
func (m *mockBlockInfo) Root() common.Hash                                    { return common.Hash{} }
func (m *mockBlockInfo) NumberU64() uint64                                    { return m.number }
func (m *mockBlockInfo) Time() uint64                                         { return m.timestamp }
func (m *mockBlockInfo) MixDigest() common.Hash                               { return common.Hash{} }
func (m *mockBlockInfo) BaseFee() *big.Int                                    { return big.NewInt(1) }
func (m *mockBlockInfo) BlobBaseFee(chainConfig *params.ChainConfig) *big.Int { return big.NewInt(1) }
func (m *mockBlockInfo) ExcessBlobGas() *uint64                               { return nil }
func (m *mockBlockInfo) ReceiptHash() common.Hash                             { return common.Hash{} }
func (m *mockBlockInfo) GasUsed() uint64                                      { return 0 }
func (m *mockBlockInfo) GasLimit() uint64                                     { return 30000000 }
func (m *mockBlockInfo) BlobGasUsed() *uint64                                 { return nil }
func (m *mockBlockInfo) ParentBeaconRoot() *common.Hash                       { return nil }
func (m *mockBlockInfo) WithdrawalsRoot() *common.Hash                        { return nil }
func (m *mockBlockInfo) HeaderRLP() ([]byte, error)                           { return nil, nil }
func (m *mockBlockInfo) Header() *types.Header                                { return nil }
func (m *mockBlockInfo) ID() eth.BlockID                                      { return eth.BlockID{Hash: m.hash, Number: m.number} }

var _ eth.BlockInfo = (*mockBlockInfo)(nil)

type mockChainContainer struct {
	id eth.ChainID

	currentL1    eth.BlockRef
	currentL1Err error

	blockAtTimestamp    eth.L2BlockRef
	blockAtTimestampErr error

	lastRequestedTimestamp uint64
	mu                     sync.Mutex
}

func newMockChainContainer(id uint64) *mockChainContainer {
	return &mockChainContainer{id: eth.ChainIDFromUInt64(id)}
}

func (m *mockChainContainer) ID() eth.ChainID                  { return m.id }
func (m *mockChainContainer) Start(ctx context.Context) error  { return nil }
func (m *mockChainContainer) Stop(ctx context.Context) error   { return nil }
func (m *mockChainContainer) Pause(ctx context.Context) error  { return nil }
func (m *mockChainContainer) Resume(ctx context.Context) error { return nil }
func (m *mockChainContainer) RegisterVerifier(v activity.VerificationActivity) {
}
func (m *mockChainContainer) BlockAtTimestamp(ctx context.Context, ts uint64, label eth.BlockLabel) (eth.L2BlockRef, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.blockAtTimestampErr != nil {
		return eth.L2BlockRef{}, m.blockAtTimestampErr
	}
	m.lastRequestedTimestamp = ts
	ref := m.blockAtTimestamp
	ref.Time = ts
	ref.Number = ts
	ref.Hash = common.BigToHash(big.NewInt(int64(ts)))
	return ref, nil
}
func (m *mockChainContainer) VerifiedAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *mockChainContainer) L1ForL2(ctx context.Context, l2Block eth.BlockID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}
func (m *mockChainContainer) OptimisticAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *mockChainContainer) OutputRootAtL2BlockNumber(ctx context.Context, l2BlockNum uint64) (eth.Bytes32, error) {
	return eth.Bytes32{}, nil
}
func (m *mockChainContainer) OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputResponse, error) {
	return nil, nil
}
func (m *mockChainContainer) FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, types.Receipts, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ts := m.lastRequestedTimestamp
	var parentHash common.Hash
	if ts > 0 {
		parentHash = common.BigToHash(big.NewInt(int64(ts - 1)))
	}
	blockInfo := &mockBlockInfo{
		hash:       blockID.Hash,
		parentHash: parentHash,
		number:     blockID.Number,
		timestamp:  ts,
	}
	return blockInfo, types.Receipts{}, nil
}
func (m *mockChainContainer) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.currentL1Err != nil {
		return nil, m.currentL1Err
	}
	return &eth.SyncStatus{CurrentL1: m.currentL1}, nil
}
func (m *mockChainContainer) RewindEngine(ctx context.Context, timestamp uint64) error {
	return nil
}
func (m *mockChainContainer) BlockTime() uint64 { return 1 }

var _ cc.ChainContainer = (*mockChainContainer)(nil)

func testLogger() gethlog.Logger {
	return gethlog.New()
}

// mockLogsDBForInterop implements LogsDB for interop tests
type mockLogsDBForInterop struct {
	openBlockRef     eth.BlockRef
	openBlockLogCnt  uint32
	openBlockExecMsg map[uint32]*suptypes.ExecutingMessage
	openBlockErr     error
	containsSeal     suptypes.BlockSeal
	containsErr      error
}

func (m *mockLogsDBForInterop) LatestSealedBlock() (eth.BlockID, bool) { return eth.BlockID{}, false }
func (m *mockLogsDBForInterop) FirstSealedBlock() (suptypes.BlockSeal, error) {
	return suptypes.BlockSeal{}, nil
}
func (m *mockLogsDBForInterop) FindSealedBlock(number uint64) (suptypes.BlockSeal, error) {
	return suptypes.BlockSeal{}, nil
}
func (m *mockLogsDBForInterop) OpenBlock(blockNum uint64) (eth.BlockRef, uint32, map[uint32]*suptypes.ExecutingMessage, error) {
	if m.openBlockErr != nil {
		return eth.BlockRef{}, 0, nil, m.openBlockErr
	}
	return m.openBlockRef, m.openBlockLogCnt, m.openBlockExecMsg, nil
}
func (m *mockLogsDBForInterop) Contains(query suptypes.ContainsQuery) (suptypes.BlockSeal, error) {
	if m.containsErr != nil {
		return suptypes.BlockSeal{}, m.containsErr
	}
	return m.containsSeal, nil
}
func (m *mockLogsDBForInterop) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *suptypes.ExecutingMessage) error {
	return nil
}
func (m *mockLogsDBForInterop) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	return nil
}
func (m *mockLogsDBForInterop) Close() error { return nil }

var _ LogsDB = (*mockLogsDBForInterop)(nil)
