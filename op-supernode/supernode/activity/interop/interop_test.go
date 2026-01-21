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

func (m *mockChainContainer) CurrentL1(ctx context.Context) (eth.BlockRef, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentL1, m.currentL1Err
}

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

var _ cc.ChainContainer = (*mockChainContainer)(nil)

// mockL1Client mocks the L1Client for testing L1 consistency checks
type mockL1Client struct {
	blockRefs map[uint64]eth.L1BlockRef
	err       error
}

func newMockL1Client() *mockL1Client {
	return &mockL1Client{
		blockRefs: make(map[uint64]eth.L1BlockRef),
	}
}

func (m *mockL1Client) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	if m.err != nil {
		return eth.L1BlockRef{}, m.err
	}
	ref, ok := m.blockRefs[num]
	if !ok {
		return eth.L1BlockRef{}, ethereum.NotFound
	}
	return ref, nil
}

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

	interop := New(testLogger(), 1000, chains, dataDir, nil)

	require.NotNil(t, interop)
	require.Equal(t, uint64(1000), interop.activationTimestamp)
	require.NotNil(t, interop.verifiedDB)
	require.NotNil(t, interop.currentL1s)
	require.Len(t, interop.chains, 1)
}

func TestNew_InvalidDataDir(t *testing.T) {
	t.Parallel()
	// Use a path that can't be written to
	invalidDir := "/nonexistent/path/that/cannot/exist/db"

	chains := map[eth.ChainID]cc.ChainContainer{}

	interop := New(testLogger(), 1000, chains, invalidDir, nil)

	// New returns nil when DB fails to open
	require.Nil(t, interop)
}

func TestNew_EmptyChains(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}

	interop := New(testLogger(), 0, chains, dataDir, nil)

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

	interop := New(testLogger(), 500, chains, dataDir, nil)

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
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- interop.Start(ctx)
	}()

	// Give it time to start the loop
	time.Sleep(100 * time.Millisecond)

	// Cancel and verify it exits
	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("Start should exit after context cancellation")
	}
}

func TestStart_AlreadyStarted(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start first instance
	go func() {
		_ = interop.Start(ctx)
	}()

	// Wait for it to mark as started
	time.Sleep(100 * time.Millisecond)

	// Try to start again - should block on context
	ctx2, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel2()

	err := interop.Start(ctx2)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestStop_ClosesVerifiedDB(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
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
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)

	ctx := context.Background()

	done := make(chan error, 1)
	go func() {
		done <- interop.Start(ctx)
	}()

	// Wait for start
	time.Sleep(100 * time.Millisecond)

	// Stop should cancel the internal context
	err := interop.Stop(context.Background())
	require.NoError(t, err)

	select {
	case <-done:
		// Success - Start exited
	case <-time.After(2 * time.Second):
		t.Fatal("Start should exit after Stop is called")
	}
}

func TestStop_NilCancel(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)

	// Stop without ever starting - cancel is nil
	err := interop.Stop(context.Background())
	require.NoError(t, err)
}

// =============================================================================
// collectCurrentL1s Tests
// =============================================================================

func TestCollectCurrentL1s_Success(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock1 := newMockChainContainer(10)
	mock1.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}

	mock2 := newMockChainContainer(8453)
	mock2.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x2")}

	chains := map[eth.ChainID]cc.ChainContainer{
		mock1.id: mock1,
		mock2.id: mock2,
	}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	l1s, err := interop.collectCurrentL1s()

	require.NoError(t, err)
	require.Len(t, l1s, 2)
	require.Equal(t, uint64(100), l1s[mock1.id].Number)
	require.Equal(t, uint64(200), l1s[mock2.id].Number)
}

func TestCollectCurrentL1s_ChainNotReady_Error(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.currentL1Err = errors.New("chain not synced")

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	l1s, err := interop.collectCurrentL1s()

	require.Error(t, err)
	require.Contains(t, err.Error(), "not ready")
	require.Nil(t, l1s)
}

func TestCollectCurrentL1s_EmptyBlockRef(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.currentL1 = eth.BlockRef{} // Empty - derivation not started

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	l1s, err := interop.collectCurrentL1s()

	require.Error(t, err)
	require.Contains(t, err.Error(), "not yet populated")
	require.Nil(t, l1s)
}

func TestCollectCurrentL1s_EmptyChains(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	l1s, err := interop.collectCurrentL1s()

	require.NoError(t, err)
	require.Empty(t, l1s)
}

// =============================================================================
// updateCurrentL1s Tests
// =============================================================================

// testableInterop wraps Interop with a mock L1 client for testing
type testableInterop struct {
	*Interop
	mockL1 *mockL1Client
}

func newTestableInterop(t *testing.T, chains map[eth.ChainID]cc.ChainContainer, activationTs uint64) *testableInterop {
	dataDir := t.TempDir()
	interop := New(testLogger(), activationTs, chains, dataDir, nil)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	mockL1 := newMockL1Client()
	return &testableInterop{
		Interop: interop,
		mockL1:  mockL1,
	}
}

// Override updateCurrentL1s to use mock L1 client
func (ti *testableInterop) updateCurrentL1sWithMock(currentL1s map[eth.ChainID]eth.BlockID) error {
	for _, l1Head := range currentL1s {
		ti.log.Info("updating current L1s", "l1Head", l1Head)
		if l1Head == (eth.BlockID{}) {
			continue
		}
		header, err := ti.mockL1.L1BlockRefByNumber(ti.ctx, l1Head.Number)
		if err != nil {
			return err
		}
		if header.ID() != l1Head {
			return ErrInconsistentL1Heads
		}
	}
	ti.currentL1s = currentL1s
	return nil
}

func TestUpdateCurrentL1s_Success(t *testing.T) {
	t.Parallel()

	chains := map[eth.ChainID]cc.ChainContainer{}
	ti := newTestableInterop(t, chains, 1000)

	blockHash := common.HexToHash("0xabc")
	ti.mockL1.blockRefs[100] = eth.L1BlockRef{
		Hash:   blockHash,
		Number: 100,
	}

	currentL1s := map[eth.ChainID]eth.BlockID{
		eth.ChainIDFromUInt64(10): {Hash: blockHash, Number: 100},
	}

	err := ti.updateCurrentL1sWithMock(currentL1s)

	require.NoError(t, err)
	require.Equal(t, currentL1s, ti.currentL1s)
}

func TestUpdateCurrentL1s_InconsistentHeads(t *testing.T) {
	t.Parallel()

	chains := map[eth.ChainID]cc.ChainContainer{}
	ti := newTestableInterop(t, chains, 1000)

	// L1 client returns different hash for same block number (reorg)
	ti.mockL1.blockRefs[100] = eth.L1BlockRef{
		Hash:   common.HexToHash("0xdifferent"),
		Number: 100,
	}

	currentL1s := map[eth.ChainID]eth.BlockID{
		eth.ChainIDFromUInt64(10): {Hash: common.HexToHash("0xoriginal"), Number: 100},
	}

	err := ti.updateCurrentL1sWithMock(currentL1s)

	require.ErrorIs(t, err, ErrInconsistentL1Heads)
}

func TestUpdateCurrentL1s_EmptyBlockID(t *testing.T) {
	t.Parallel()

	chains := map[eth.ChainID]cc.ChainContainer{}
	ti := newTestableInterop(t, chains, 1000)

	// Empty BlockID should be skipped
	currentL1s := map[eth.ChainID]eth.BlockID{
		eth.ChainIDFromUInt64(10): {}, // Empty
	}

	err := ti.updateCurrentL1sWithMock(currentL1s)

	require.NoError(t, err)
}

func TestUpdateCurrentL1s_L1ClientError(t *testing.T) {
	t.Parallel()

	chains := map[eth.ChainID]cc.ChainContainer{}
	ti := newTestableInterop(t, chains, 1000)

	ti.mockL1.err = errors.New("L1 RPC error")

	currentL1s := map[eth.ChainID]eth.BlockID{
		eth.ChainIDFromUInt64(10): {Hash: common.HexToHash("0x1"), Number: 100},
	}

	err := ti.updateCurrentL1sWithMock(currentL1s)

	require.Error(t, err)
	require.Contains(t, err.Error(), "L1 RPC error")
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
	interop := New(testLogger(), 1000, chains, dataDir, nil)
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
	interop := New(testLogger(), 1000, chains, dataDir, nil)
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
	interop := New(testLogger(), 1000, chains, dataDir, nil)
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

	interop := New(testLogger(), 1000, chains, dataDir, nil)
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
	interop := New(testLogger(), 5000, chains, dataDir, nil) // activation at 5000
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	err := interop.progressInterop()

	require.NoError(t, err)

	// Check that timestamp 5000 was committed
	has, err := interop.verifiedDB.Has(5000)
	require.NoError(t, err)
	require.True(t, has)
}

func TestProgressInterop_Initialized_UsesNextTimestamp(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	// First progress - commits timestamp 1000
	err := interop.progressInterop()
	require.NoError(t, err)

	// Second progress - should commit timestamp 1001
	err = interop.progressInterop()
	require.NoError(t, err)

	has, err := interop.verifiedDB.Has(1001)
	require.NoError(t, err)
	require.True(t, has)
}

func TestProgressInterop_ChainsNotReady_ReturnsEarly(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.blockAtTimestampErr = ethereum.NotFound // Not ready

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	err := interop.progressInterop()

	require.NoError(t, err) // Returns nil when chains not ready

	// Nothing should be committed
	has, err := interop.verifiedDB.Has(1000)
	require.NoError(t, err)
	require.False(t, has)
}

func TestProgressInterop_ChainError(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.blockAtTimestampErr = errors.New("internal error")

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	err := interop.progressInterop()

	require.Error(t, err)
	require.Contains(t, err.Error(), "internal error")
}

// =============================================================================
// CurrentL1 Tests
// =============================================================================

func TestCurrentL1_ReturnsMinimum(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)

	interop.currentL1s = map[eth.ChainID]eth.BlockID{
		eth.ChainIDFromUInt64(10):   {Number: 300, Hash: common.HexToHash("0x3")},
		eth.ChainIDFromUInt64(8453): {Number: 100, Hash: common.HexToHash("0x1")}, // Minimum
		eth.ChainIDFromUInt64(420):  {Number: 200, Hash: common.HexToHash("0x2")},
	}

	result := interop.CurrentL1()

	require.Equal(t, uint64(100), result.Number)
	require.Equal(t, common.HexToHash("0x1"), result.Hash)
}

func TestCurrentL1_EmptyReturnsZero(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)

	result := interop.CurrentL1()

	require.Equal(t, eth.BlockID{}, result)
}

func TestCurrentL1_SingleChain(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)

	interop.currentL1s = map[eth.ChainID]eth.BlockID{
		eth.ChainIDFromUInt64(10): {Number: 500, Hash: common.HexToHash("0x5")},
	}

	result := interop.CurrentL1()

	require.Equal(t, uint64(500), result.Number)
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
	interop := New(testLogger(), 1000, chains, dataDir, nil)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	// Progress to commit timestamp 1000
	err := interop.progressInterop()
	require.NoError(t, err)

	verified, err := interop.VerifiedAtTimestamp(1000)

	require.NoError(t, err)
	require.True(t, verified)
}

func TestVerifiedAtTimestamp_NotExists(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chains := map[eth.ChainID]cc.ChainContainer{}
	interop := New(testLogger(), 1000, chains, dataDir, nil)
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
	interop := New(testLogger(), 1000, chains, dataDir, nil)
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
// Integration Tests
// =============================================================================

func TestInterop_FullCycle(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	mock := newMockChainContainer(10)
	mock.currentL1 = eth.BlockRef{Number: 1000, Hash: common.HexToHash("0xL1")}
	mock.blockAtTimestamp = eth.L2BlockRef{Number: 500, Hash: common.HexToHash("0xL2")}

	chains := map[eth.ChainID]cc.ChainContainer{mock.id: mock}
	interop := New(testLogger(), 100, chains, dataDir, nil)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	// Simulate multiple interop cycles
	for i := 0; i < 3; i++ {
		// Collect L1s
		l1s, err := interop.collectCurrentL1s()
		require.NoError(t, err)
		require.Len(t, l1s, 1)

		// Progress
		err = interop.progressInterop()
		require.NoError(t, err)
	}

	// Verify timestamps were committed sequentially
	for ts := uint64(100); ts <= 102; ts++ {
		has, err := interop.verifiedDB.Has(ts)
		require.NoError(t, err)
		require.True(t, has, "timestamp %d should be verified", ts)
	}
}
