package interop

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// mockChainContainer implements cc.ChainContainer for testing
type mockChainContainer struct {
	id eth.ChainID

	currentL1    eth.BlockRef
	currentL1Err error

	blockAtTimestamp    eth.L2BlockRef
	blockAtTimestampErr error

	mu sync.Mutex
}

func newMockChainContainer(id uint64) *mockChainContainer {
	return &mockChainContainer{
		id: eth.ChainIDFromUInt64(id),
	}
}

func (m *mockChainContainer) ID() eth.ChainID { return m.id }

func (m *mockChainContainer) Start(ctx context.Context) error  { return nil }
func (m *mockChainContainer) Stop(ctx context.Context) error   { return nil }
func (m *mockChainContainer) Pause(ctx context.Context) error  { return nil }
func (m *mockChainContainer) Resume(ctx context.Context) error { return nil }

func (m *mockChainContainer) RegisterVerifier(v activity.VerificationActivity) {
}
func (m *mockChainContainer) BlockAtTimestamp(ctx context.Context, ts uint64, label eth.BlockLabel) (eth.L2BlockRef, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.blockAtTimestamp, m.blockAtTimestampErr
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

var _ cc.ChainContainer = (*mockChainContainer)(nil)

// Helper to create a test logger
func testLogger() gethlog.Logger {
	return gethlog.New()
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNew_ValidInputs(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): newMockChainContainer(10),
	}

	interop := New(testLogger(), 1000, chains, dataDir)

	require.NotNil(t, interop)
	require.Equal(t, uint64(1000), interop.activationTimestamp)
	require.NotNil(t, interop.verifiedDB)
	require.Equal(t, eth.BlockID{}, interop.currentL1) // starts empty
	require.Len(t, interop.chains, 1)
}

func TestNew_InvalidDataDir(t *testing.T) {
	t.Parallel()
	// Use a path that can't be written to
	invalidDir := "/nonexistent/path/that/cannot/exist/db"

	chains := map[eth.ChainID]cc.ChainContainer{}

	interop := New(testLogger(), 1000, chains, invalidDir)

	// New returns nil when DB fails to open
	require.Nil(t, interop)
}

func TestNew_EmptyChains(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}

	interop := New(testLogger(), 0, chains, dataDir)

	require.NotNil(t, interop)
	require.Empty(t, interop.chains)
}

func TestNew_MultipleChains(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10):   newMockChainContainer(10),
		eth.ChainIDFromUInt64(8453): newMockChainContainer(8453),
		eth.ChainIDFromUInt64(420):  newMockChainContainer(420),
	}

	interop := New(testLogger(), 500, chains, dataDir)

	require.NotNil(t, interop)
	require.Len(t, interop.chains, 3)
}

// =============================================================================
// Lifecycle Tests
// =============================================================================

func TestStart_BlocksUntilContextCanceled(t *testing.T) {
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
	go func() {
		done <- interop.Start(ctx)
	}()

	// Wait for it to start the loop
	require.Eventually(t, func() bool {
		interop.mu.RLock()
		defer interop.mu.RUnlock()
		return interop.started
	}, 5*time.Second, 100*time.Millisecond, "Start should mark as started")

	// Cancel and verify it exits
	cancel()

	var err error
	require.Eventually(t, func() bool {
		select {
		case err = <-done:
			return true
		default:
			return false
		}
	}, 5*time.Second, 100*time.Millisecond, "Start should exit after context cancellation")

	require.ErrorIs(t, err, context.Canceled)
}

func TestStart_AlreadyStarted(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start first instance
	go func() {
		_ = interop.Start(ctx)
	}()

	// Wait for it to mark as started
	require.Eventually(t, func() bool {
		interop.mu.RLock()
		defer interop.mu.RUnlock()
		return interop.started
	}, 5*time.Second, 100*time.Millisecond, "Start should mark as started")

	// Try to start again - should block on context and return deadline exceeded
	ctx2, cancel2 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel2()

	err := interop.Start(ctx2)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestStop_ClosesVerifiedDB(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)

	err := interop.Stop(context.Background())
	require.NoError(t, err)

	// Verify DB is closed by trying to use it (should fail)
	_, err = interop.verifiedDB.Has(100)
	require.Error(t, err) // LevelDB returns error on closed DB
}

func TestStop_CancelsRunningContext(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
	mock.blockAtTimestampErr = ethereum.NotFound // Keep it in "not ready" state

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)

	ctx := context.Background()

	done := make(chan error, 1)
	go func() {
		done <- interop.Start(ctx)
	}()

	// Wait for it to start
	require.Eventually(t, func() bool {
		interop.mu.RLock()
		defer interop.mu.RUnlock()
		return interop.started
	}, 5*time.Second, 100*time.Millisecond, "Start should mark as started")

	// Stop should cancel the internal context
	err := interop.Stop(context.Background())
	require.NoError(t, err)

	// Verify Start exited
	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, 5*time.Second, 100*time.Millisecond, "Start should exit after Stop is called")
}

func TestStop_NilCancel(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)

	// Stop without ever starting - cancel is nil
	err := interop.Stop(context.Background())
	require.NoError(t, err)
}

// =============================================================================
// collectCurrentL1 Tests
// =============================================================================

func TestCollectCurrentL1_ReturnsMinimum(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock1 := newMockChainContainer(10)
	mock1.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x2")}

	mock2 := newMockChainContainer(8453)
	mock2.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")} // minimum

	chains := map[eth.ChainID]cc.ChainContainer{
		mock1.id: mock1,
		mock2.id: mock2,
	}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	l1, err := interop.collectCurrentL1()

	require.NoError(t, err)
	require.Equal(t, uint64(100), l1.Number)
	require.Equal(t, common.HexToHash("0x1"), l1.Hash)
}

func TestCollectCurrentL1_ChainNotReady_Error(t *testing.T) {
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
}

func TestCollectCurrentL1_EmptyChains(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	l1, err := interop.collectCurrentL1()

	require.NoError(t, err)
	require.Equal(t, eth.BlockID{}, l1)
}

func TestCollectCurrentL1_SingleChain(t *testing.T) {
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
	require.Equal(t, common.HexToHash("0x5"), l1.Hash)
}

// =============================================================================
// checkChainsReady Tests
// =============================================================================

func TestCheckChainsReady_AllReady(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock1 := newMockChainContainer(10)
	mock1.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}

	mock2 := newMockChainContainer(8453)
	mock2.blockAtTimestamp = eth.L2BlockRef{Number: 200, Hash: common.HexToHash("0x2")}

	chains := map[eth.ChainID]cc.ChainContainer{
		mock1.id: mock1,
		mock2.id: mock2,
	}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	blocks, err := interop.checkChainsReady(1000)

	require.NoError(t, err)
	require.Len(t, blocks, 2)
	require.Equal(t, uint64(100), blocks[mock1.id].Number)
	require.Equal(t, uint64(200), blocks[mock2.id].Number)
}

func TestCheckChainsReady_OneNotReady(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock1 := newMockChainContainer(10)
	mock1.blockAtTimestamp = eth.L2BlockRef{Number: 100}

	mock2 := newMockChainContainer(8453)
	mock2.blockAtTimestampErr = ethereum.NotFound // Not ready

	chains := map[eth.ChainID]cc.ChainContainer{
		mock1.id: mock1,
		mock2.id: mock2,
	}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	blocks, err := interop.checkChainsReady(1000)

	require.Error(t, err)
	require.Nil(t, blocks)
}

func TestCheckChainsReady_EmptyChains(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	blocks, err := interop.checkChainsReady(1000)

	require.NoError(t, err)
	require.Empty(t, blocks)
}

func TestCheckChainsReady_ParallelQueries(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	// Create multiple chains to test parallel execution
	var mocks []*mockChainContainer
	chains := make(map[eth.ChainID]cc.ChainContainer)

	for i := 0; i < 5; i++ {
		mock := newMockChainContainer(uint64(10 + i))
		mock.blockAtTimestamp = eth.L2BlockRef{Number: uint64(100 + i)}
		mocks = append(mocks, mock)
		chains[mock.id] = mock
	}

	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	blocks, err := interop.checkChainsReady(1000)

	require.NoError(t, err)
	require.Len(t, blocks, 5)

	// Verify all chains were queried
	for _, mock := range mocks {
		require.Contains(t, blocks, mock.id)
	}
}

// =============================================================================
// progressInterop Tests
// =============================================================================

func TestProgressInterop_NotInitialized_UsesActivationTimestamp(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 5000, chains, dataDir) // activation at 5000
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	result, err := interop.progressInterop()

	require.NoError(t, err)
	require.False(t, result.IsEmpty())
	require.Equal(t, uint64(5000), result.Timestamp)
	require.True(t, result.IsValid())
}

func TestProgressInterop_Initialized_UsesNextTimestamp(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	// First progress - returns result for timestamp 1000
	result1, err := interop.progressInterop()
	require.NoError(t, err)
	require.Equal(t, uint64(1000), result1.Timestamp)

	// Commit the result so DB is initialized
	err = interop.handleResult(result1)
	require.NoError(t, err)

	// Second progress - should return result for timestamp 1001
	result2, err := interop.progressInterop()
	require.NoError(t, err)
	require.Equal(t, uint64(1001), result2.Timestamp)
}

func TestProgressInterop_ChainsNotReady_ReturnsEmptyResult(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.blockAtTimestampErr = ethereum.NotFound // Not ready

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	result, err := interop.progressInterop()

	require.NoError(t, err) // Returns nil error when chains not ready
	require.True(t, result.IsEmpty())
}

func TestProgressInterop_ChainError(t *testing.T) {
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
	require.Contains(t, err.Error(), "internal error")
	require.True(t, result.IsEmpty())
}

// =============================================================================
// CurrentL1 Tests
// =============================================================================

func TestCurrentL1_ReturnsStoredValue(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)

	interop.currentL1 = eth.BlockID{Number: 100, Hash: common.HexToHash("0x1")}

	result := interop.CurrentL1()

	require.Equal(t, uint64(100), result.Number)
	require.Equal(t, common.HexToHash("0x1"), result.Hash)
}

func TestCurrentL1_EmptyReturnsZero(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)

	result := interop.CurrentL1()

	require.Equal(t, eth.BlockID{}, result)
}

// =============================================================================
// VerifiedAtTimestamp Tests
// =============================================================================

func TestVerifiedAtTimestamp_Exists(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.blockAtTimestamp = eth.L2BlockRef{Number: 100}

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	// Progress to get result for timestamp 1000
	result, err := interop.progressInterop()
	require.NoError(t, err)

	// Commit the result to DB
	err = interop.handleResult(result)
	require.NoError(t, err)

	verified, err := interop.VerifiedAtTimestamp(1000)

	require.NoError(t, err)
	require.True(t, verified)
}

func TestVerifiedAtTimestamp_NotExists(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)

	verified, err := interop.VerifiedAtTimestamp(9999)

	require.NoError(t, err)
	require.False(t, verified)
}

// =============================================================================
// verifyInteropMessages Tests
// =============================================================================

func TestVerifyInteropMessages_CopiesBlocks(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock1 := newMockChainContainer(10)
	mock2 := newMockChainContainer(8453)

	chains := map[eth.ChainID]cc.ChainContainer{
		mock1.id: mock1,
		mock2.id: mock2,
	}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)

	blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
		mock1.id: {Number: 100, Hash: common.HexToHash("0x1")},
		mock2.id: {Number: 200, Hash: common.HexToHash("0x2")},
	}

	result, err := interop.verifyInteropMessages(1000, blocksAtTimestamp)

	require.NoError(t, err)
	require.Equal(t, uint64(1000), result.Timestamp)
	require.Len(t, result.L2Heads, 2)
	require.Equal(t, blocksAtTimestamp[mock1.id], result.L2Heads[mock1.id])
	require.Equal(t, blocksAtTimestamp[mock2.id], result.L2Heads[mock2.id])
	require.True(t, result.IsValid()) // No invalid heads in stub implementation
}

// =============================================================================
// handleResult Tests
// =============================================================================

func TestHandleResult_EmptyResult_ReturnsNil(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)

	emptyResult := Result{}
	require.True(t, emptyResult.IsEmpty())

	err := interop.handleResult(emptyResult)

	require.NoError(t, err)
	// Empty result should not commit anything to DB
	has, err := interop.verifiedDB.Has(0)
	require.NoError(t, err)
	require.False(t, has)
}

func TestHandleResult_ValidResult_CommitsToDb(t *testing.T) {
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
		InvalidHeads: nil, // No invalid heads = valid result
	}
	require.True(t, validResult.IsValid())
	require.False(t, validResult.IsEmpty())

	err := interop.handleResult(validResult)

	require.NoError(t, err)
	// Valid result should be committed to DB
	has, err := interop.verifiedDB.Has(1000)
	require.NoError(t, err)
	require.True(t, has)
}

func TestHandleResult_InvalidResult_DoesNotCommitToDb(t *testing.T) {
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
			mock.id: {Number: 500, Hash: common.HexToHash("0xBAD")}, // Has invalid heads
		},
	}
	require.False(t, invalidResult.IsValid())
	require.False(t, invalidResult.IsEmpty())

	err := interop.handleResult(invalidResult)

	require.NoError(t, err)
	// Invalid results trigger block invalidation but are NOT committed to the verified DB
	has, err := interop.verifiedDB.Has(1000)
	require.NoError(t, err)
	require.False(t, has)
}

func TestHandleResult_InvalidResult_MultipleInvalidHeads(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock1 := newMockChainContainer(10)
	mock2 := newMockChainContainer(8453)
	chains := map[eth.ChainID]cc.ChainContainer{
		mock1.id: mock1,
		mock2.id: mock2,
	}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)

	invalidResult := Result{
		Timestamp: 1000,
		L1Head:    eth.BlockID{Number: 100, Hash: common.HexToHash("0xL1")},
		L2Heads: map[eth.ChainID]eth.BlockID{
			mock1.id: {Number: 500, Hash: common.HexToHash("0xL2a")},
			mock2.id: {Number: 600, Hash: common.HexToHash("0xL2b")},
		},
		InvalidHeads: map[eth.ChainID]eth.BlockID{
			mock1.id: {Number: 500, Hash: common.HexToHash("0xBAD1")},
			mock2.id: {Number: 600, Hash: common.HexToHash("0xBAD2")},
		},
	}

	err := interop.handleResult(invalidResult)

	require.NoError(t, err)
	// Invalid results trigger block invalidation but are NOT committed to the verified DB
	has, err := interop.verifiedDB.Has(1000)
	require.NoError(t, err)
	require.False(t, has)
}

// =============================================================================
// progressAndRecord L1 Update Tests
// =============================================================================

func TestProgressAndRecord_EmptyResult_SetsL1ToCollectedMinimum(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock1 := newMockChainContainer(10)
	mock1.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
	mock1.blockAtTimestampErr = ethereum.NotFound // Chains not ready -> empty result

	mock2 := newMockChainContainer(8453)
	mock2.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")} // This is minimum
	mock2.blockAtTimestampErr = ethereum.NotFound

	chains := map[eth.ChainID]cc.ChainContainer{
		mock1.id: mock1,
		mock2.id: mock2,
	}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	// Verify currentL1 starts empty
	require.Equal(t, eth.BlockID{}, interop.currentL1)

	err := interop.progressAndRecord()

	require.NoError(t, err)
	// When result is empty, currentL1 should be set to the collected minimum
	require.Equal(t, uint64(100), interop.currentL1.Number)
	require.Equal(t, common.HexToHash("0x1"), interop.currentL1.Hash)
}

func TestProgressAndRecord_ValidResult_SetsL1ToResultL1Head(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x200")}
	mock.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0xL2")}

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	// Override verifyFn to return a valid result with a specific L1Head
	expectedL1Head := eth.BlockID{Number: 150, Hash: common.HexToHash("0xL1Result")}
	interop.verifyFn = func(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error) {
		return Result{
			Timestamp: ts,
			L1Head:    expectedL1Head,
			L2Heads: map[eth.ChainID]eth.BlockID{
				mock.id: blocksAtTimestamp[mock.id],
			},
			InvalidHeads: nil, // valid result
		}, nil
	}

	// Verify currentL1 starts empty
	require.Equal(t, eth.BlockID{}, interop.currentL1)

	err := interop.progressAndRecord()

	require.NoError(t, err)
	// When result is valid (non-empty), currentL1 should be set to result.L1Head
	require.Equal(t, expectedL1Head.Number, interop.currentL1.Number)
	require.Equal(t, expectedL1Head.Hash, interop.currentL1.Hash)
}

func TestProgressAndRecord_InvalidResult_DoesNotUpdateL1(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x200")}
	mock.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0xL2")}

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	// Set an initial currentL1 value
	initialL1 := eth.BlockID{Number: 50, Hash: common.HexToHash("0x50")}
	interop.currentL1 = initialL1

	// Override verifyFn to return an invalid result (has InvalidHeads)
	interop.verifyFn = func(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error) {
		return Result{
			Timestamp: ts,
			L1Head:    eth.BlockID{Number: 999, Hash: common.HexToHash("0xShouldNotBeUsed")},
			L2Heads: map[eth.ChainID]eth.BlockID{
				mock.id: blocksAtTimestamp[mock.id],
			},
			InvalidHeads: map[eth.ChainID]eth.BlockID{
				mock.id: {Number: 100, Hash: common.HexToHash("0xBAD")}, // marks result as invalid
			},
		}, nil
	}

	err := interop.progressAndRecord()

	require.NoError(t, err)
	// When result is invalid, currentL1 should NOT be updated (remains at initial value)
	require.Equal(t, initialL1.Number, interop.currentL1.Number)
	require.Equal(t, initialL1.Hash, interop.currentL1.Hash)
}

func TestProgressAndRecord_CollectL1Error_ReturnsError(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.currentL1Err = errors.New("L1 sync error")

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	err := interop.progressAndRecord()

	require.Error(t, err)
	require.Contains(t, err.Error(), "not ready")
}

func TestProgressAndRecord_ProgressInteropError_ReturnsError(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
	mock.blockAtTimestampErr = errors.New("internal chain error")

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	err := interop.progressAndRecord()

	require.Error(t, err)
	require.Contains(t, err.Error(), "internal chain error")
}

// =============================================================================
// Integration Tests
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

	// Simulate multiple interop cycles
	for i := 0; i < 3; i++ {
		// Collect L1 (returns minimum across chains)
		l1, err := interop.collectCurrentL1()
		require.NoError(t, err)
		require.Equal(t, uint64(1000), l1.Number)

		// Progress and get result
		result, err := interop.progressInterop()
		require.NoError(t, err)
		require.False(t, result.IsEmpty())

		// Handle the result (commits to DB)
		err = interop.handleResult(result)
		require.NoError(t, err)
	}

	// Verify timestamps were committed sequentially
	for ts := uint64(100); ts <= 102; ts++ {
		has, err := interop.verifiedDB.Has(ts)
		require.NoError(t, err)
		require.True(t, has, "timestamp %d should be verified", ts)
	}
}
