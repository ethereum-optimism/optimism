package chain_container

import (
	"context"
	"math/big"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/virtual_node"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// mockVirtualNode is a mock implementation of virtual_node.VirtualNode interface
type mockVirtualNode struct {
	mu           sync.Mutex
	startCalled  int
	stopCalled   int
	startErr     error
	stopErr      error
	startFunc    func(ctx context.Context) error
	stopFunc     func(ctx context.Context) error
	blockOnStart bool
	startSignal  chan struct{}
	// latest safe mock behavior
	latestSafe eth.BlockID
	latestErr  error

	// safe head mapping mock behavior
	safeHeadL1  eth.BlockID
	safeHeadL2  eth.BlockID
	safeHeadErr error
}

func newMockVirtualNode() *mockVirtualNode {
	return &mockVirtualNode{
		startSignal: make(chan struct{}),
	}
}

func (m *mockVirtualNode) Start(ctx context.Context) error {
	m.mu.Lock()
	m.startCalled++
	callCount := m.startCalled
	m.mu.Unlock()

	// Only close startSignal on first call to avoid panic
	if callCount == 1 {
		close(m.startSignal)
	}

	if m.startFunc != nil {
		return m.startFunc(ctx)
	}

	if m.blockOnStart {
		<-ctx.Done()
		return ctx.Err()
	}

	return m.startErr
}

func (m *mockVirtualNode) Stop(ctx context.Context) error {
	m.mu.Lock()
	m.stopCalled++
	m.mu.Unlock()

	if m.stopFunc != nil {
		return m.stopFunc(ctx)
	}
	return m.stopErr
}

// SafeTimestamp implements virtual_node.VirtualNode SafeTimestamp
func (m *mockVirtualNode) LatestSafe(ctx context.Context) (eth.BlockID, error) {
	return m.latestSafe, m.latestErr
}

// SafeHeadAtL1 implements virtual_node.VirtualNode SafeHeadAtL1
func (m *mockVirtualNode) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error) {
	return m.safeHeadL1, m.safeHeadL2, m.safeHeadErr
}

// L1AtSafeHead implements virtual_node.VirtualNode L1AtSafeHead
func (m *mockVirtualNode) L1AtSafeHead(ctx context.Context, target eth.BlockID) (eth.BlockID, error) {
	return m.safeHeadL1, m.safeHeadErr
}

// LastL1 implements virtual_node.VirtualNode LastL1
func (m *mockVirtualNode) LastL1(ctx context.Context) (eth.BlockID, error) {
	return m.safeHeadL1, m.safeHeadErr
}

// SyncStatus implements virtual_node.VirtualNode SyncStatus
func (m *mockVirtualNode) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	if m.safeHeadErr != nil {
		return nil, m.safeHeadErr
	}
	return &eth.SyncStatus{
		CurrentL1: eth.L1BlockRef{Hash: m.safeHeadL1.Hash, Number: m.safeHeadL1.Number},
	}, nil
}

// SafeDB is not required by VirtualNode in these tests

// mockEngineController is a mock implementation of engine_controller.EngineController
type mockEngineController struct {
	blockAtTimestampResult eth.L2BlockRef
	blockAtTimestampErr    error

	rewindToTimestampCalled int
	rewindTimestamp         uint64
	rewindErr               error
	rewindFunc              func(ctx context.Context, timestamp uint64) error // optional custom behavior
}

func (m *mockEngineController) BlockAtTimestamp(ctx context.Context, ts uint64, label eth.BlockLabel) (eth.L2BlockRef, error) {
	return m.blockAtTimestampResult, m.blockAtTimestampErr
}

func (m *mockEngineController) OutputV0AtBlockNumber(ctx context.Context, num uint64) (*eth.OutputV0, error) {
	return nil, nil
}

func (m *mockEngineController) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	return nil, nil, nil
}

func (m *mockEngineController) Close() error {
	return nil
}

var _ engine_controller.EngineController = (*mockEngineController)(nil)

// mockVerificationActivity is a mock implementation of activity.VerificationActivity
type mockVerificationActivity struct {
	name                      string
	currentL1Result           eth.BlockID
	verifiedAtTimestampResult bool
	verifiedAtTimestampErr    error
}

func (m *mockVerificationActivity) Name() string {
	return m.name
}

func (m *mockVerificationActivity) CurrentL1() eth.BlockID {
	return m.currentL1Result
}

func (m *mockVerificationActivity) VerifiedAtTimestamp(ts uint64) (bool, error) {
	return m.verifiedAtTimestampResult, m.verifiedAtTimestampErr
}

func (m *mockVerificationActivity) Reset(chainID eth.ChainID, timestamp uint64) {}

// Test helpers
func createTestVNConfig() *opnodecfg.Config {
	return &opnodecfg.Config{
		Rollup: rollup.Config{
			L2ChainID: big.NewInt(420),
		},
	}
}

func createTestCLIConfig(dataDir string) config.CLIConfig {
	return config.CLIConfig{
		DataDir: dataDir,
		RPCConfig: oprpc.CLIConfig{
			ListenAddr: "0.0.0.0",
			ListenPort: 8545,
		},
	}
}

func newMockEngineController() *mockEngineController {
	return &mockEngineController{}
}
func (m *mockEngineController) SafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{}, nil
}
func (m *mockEngineController) RewindToTimestamp(ctx context.Context, timestamp uint64) error {
	m.rewindToTimestampCalled++
	m.rewindTimestamp = timestamp
	if m.rewindFunc != nil {
		return m.rewindFunc(ctx, timestamp)
	}
	return m.rewindErr
}

// Interface conformance assertion
var _ engine_controller.EngineController = (*mockEngineController)(nil)

func createTestLogger(t testing.TB) gethlog.Logger {
	return testlog.Logger(t, gethlog.LevelDebug)
}

// TestChainContainer_Constructor tests initialization and configuration
func TestChainContainer_Constructor(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	vncfg := createTestVNConfig()
	log := createTestLogger(t)
	initOverload := &rollupNode.InitializationOverrides{}

	t.Run("creates container with correct config", func(t *testing.T) {
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)

		require.NotNil(t, container)

		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		require.Equal(t, chainID, impl.chainID)
		require.Equal(t, vncfg, impl.vncfg)
		require.Equal(t, cfg, impl.cfg)
		require.Equal(t, log, impl.log)
		require.NotNil(t, impl.stopped)
		require.Equal(t, 1, cap(impl.stopped))
	})

	t.Run("SafeDBPath uses subPath", func(t *testing.T) {
		dataDir := t.TempDir()
		cfg := config.CLIConfig{
			DataDir: dataDir,
		}
		container := NewChainContainer(eth.ChainIDFromUInt64(420), vncfg, log, cfg, initOverload, nil, nil, nil)

		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		expectedPath := filepath.Join(dataDir, "420", "safe_db")
		require.Equal(t, expectedPath, impl.vncfg.SafeDBPath)
	})

	t.Run("RPC config inherited from supernode config", func(t *testing.T) {
		cfg := config.CLIConfig{
			DataDir: t.TempDir(),
			RPCConfig: oprpc.CLIConfig{
				ListenAddr: "127.0.0.1",
				ListenPort: 9545,
			},
		}
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)

		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		require.Equal(t, cfg.RPCConfig, impl.vncfg.RPC)
	})

	t.Run("appVersion set correctly", func(t *testing.T) {
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		require.Equal(t, virtualNodeVersion, impl.appVersion)
	})

	t.Run("subPath combines DataDir, chainID, and path correctly", func(t *testing.T) {
		dataDir := t.TempDir()
		cfg := config.CLIConfig{
			DataDir: dataDir,
		}
		container := NewChainContainer(eth.ChainIDFromUInt64(420), vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		result := impl.subPath("safe_db")
		expected := filepath.Join(dataDir, "420", "safe_db")
		require.Equal(t, expected, result)
	})

	t.Run("subPath works with various chain IDs", func(t *testing.T) {
		dataDir := t.TempDir()
		cfg := config.CLIConfig{
			DataDir: dataDir,
		}

		testCases := []struct {
			chainID eth.ChainID
			path    string
		}{
			{eth.ChainIDFromUInt64(10), "safe_db"},
			{eth.ChainIDFromUInt64(11155420), "safe_db"},
			{eth.ChainIDFromUInt64(8453), "peerstore"},
		}

		for _, tc := range testCases {
			container := NewChainContainer(tc.chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
			impl, ok := container.(*simpleChainContainer)
			require.True(t, ok)

			result := impl.subPath(tc.path)
			expected := filepath.Join(dataDir, tc.chainID.String(), tc.path)
			require.Equal(t, expected, result, "subPath should work for chain %d", tc.chainID)
		}
	})
}

// TestChainContainer_Lifecycle tests Start/Stop behavior
func TestChainContainer_Lifecycle(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	vncfg := createTestVNConfig()
	initOverload := &rollupNode.InitializationOverrides{}

	t.Run("Start respects stop flag", func(t *testing.T) {
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		// Set stop flag before starting
		impl.stop.Store(true)

		ctx := context.Background()
		startDone := make(chan struct{})

		go func() {
			_ = container.Start(ctx)
			close(startDone)
		}()

		// Start should exit immediately due to stop flag
		select {
		case <-startDone:
			// Success
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Start should exit immediately when stop flag is set")
		}
	})

	t.Run("Stop sets stop flag", func(t *testing.T) {
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		require.False(t, impl.stop.Load())

		ctx := context.Background()
		_ = container.Stop(ctx)

		require.True(t, impl.stop.Load())
	})

	t.Run("signals stopped channel on exit", func(t *testing.T) {
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		mockVN := newMockVirtualNode()
		mockVN.blockOnStart = true
		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string) virtual_node.VirtualNode {
			return mockVN
		}

		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			_ = container.Start(ctx)
		}()

		<-mockVN.startSignal
		cancel()

		select {
		case <-impl.stopped:
			// Success
		case <-time.After(2 * time.Second):
			t.Fatal("Should receive signal on stopped channel")
		}
	})

	t.Run("context cancellation stops restart loop", func(t *testing.T) {
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		mockVN := newMockVirtualNode()
		mockVN.startFunc = func(ctx context.Context) error {
			return nil // Exit immediately to trigger restart
		}

		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string) virtual_node.VirtualNode {
			return mockVN
		}

		ctx, cancel := context.WithCancel(context.Background())

		startDone := make(chan struct{})
		go func() {
			_ = container.Start(ctx)
			close(startDone)
		}()

		// Wait for some restarts
		require.Eventually(t, func() bool {
			mockVN.mu.Lock()
			defer mockVN.mu.Unlock()
			return mockVN.startCalled >= 2
		}, 1*time.Second, 10*time.Millisecond)

		cancel()

		select {
		case <-startDone:
			// Success
		case <-time.After(2 * time.Second):
			t.Fatal("Start should exit after context cancellation")
		}
	})

	t.Run("Stop flag stops restart loop", func(t *testing.T) {
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		mockVN := newMockVirtualNode()
		mockVN.startFunc = func(ctx context.Context) error {
			return nil // Exit immediately
		}

		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string) virtual_node.VirtualNode {
			return mockVN
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = container.Start(ctx)
		}()

		// Wait for at least one start
		require.Eventually(t, func() bool {
			mockVN.mu.Lock()
			defer mockVN.mu.Unlock()
			return mockVN.startCalled >= 1
		}, 1*time.Second, 10*time.Millisecond)

		stopCtx := context.Background()
		_ = container.Stop(stopCtx)

		require.Eventually(t, func() bool {
			return impl.stop.Load()
		}, 1*time.Second, 10*time.Millisecond)
	})
}

// TestChainContainer_PauseResume tests pause/resume functionality
func TestChainContainer_PauseResume(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	vncfg := createTestVNConfig()
	initOverload := &rollupNode.InitializationOverrides{}

	t.Run("Pause sets pause flag", func(t *testing.T) {
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		ctx := context.Background()
		err := container.Pause(ctx)

		require.NoError(t, err)
		require.True(t, impl.pause.Load())
	})

	t.Run("Resume clears pause flag", func(t *testing.T) {
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		impl.pause.Store(true)

		ctx := context.Background()
		err := container.Resume(ctx)

		require.NoError(t, err)
		require.False(t, impl.pause.Load())
	})

	t.Run("paused container doesn't start VN, resumed does", func(t *testing.T) {
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		var startedSignal = make(chan struct{})
		var totalStartCalls int
		var mu sync.Mutex

		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string) virtual_node.VirtualNode {
			mockVN := newMockVirtualNode()
			mockVN.blockOnStart = true
			mockVN.startFunc = func(ctx context.Context) error {
				mu.Lock()
				totalStartCalls++
				mu.Unlock()
				select {
				case startedSignal <- struct{}{}:
				default:
				}
				<-ctx.Done()
				return ctx.Err()
			}
			return mockVN
		}

		// Pause the container
		impl.pause.Store(true)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		go func() {
			_ = container.Start(ctx)
		}()

		// Wait for VN to be created
		require.Eventually(t, func() bool {
			return impl.vn != nil
		}, 1*time.Second, 10*time.Millisecond)

		// VN should be created but not started
		mu.Lock()
		require.Equal(t, 0, totalStartCalls)
		mu.Unlock()

		// Now resume
		impl.pause.Store(false)

		select {
		case <-startedSignal:
			// Success
		case <-time.After(2 * time.Second):
			mu.Lock()
			calls := totalStartCalls
			mu.Unlock()
			t.Fatalf("VN should be started after resume (got %d start calls)", calls)
		}

		mu.Lock()
		require.Equal(t, 1, totalStartCalls)
		mu.Unlock()
	})
}

// TestChainContainer_RewindEngine tests the RewindEngine method
func TestChainContainer_RewindEngine(t *testing.T) {
	t.Run("calls RewindToTimestamp on engine controller and stops VN", func(t *testing.T) {
		// Setup
		mockVN := newMockVirtualNode()
		mockEngine := newMockEngineController()

		chainID := eth.ChainIDFromUInt64(420)
		log := createTestLogger(t)

		// Create container with mocks directly injected (no Start loop needed)
		c := &simpleChainContainer{
			chainID: chainID,
			log:     log,
			engine:  mockEngine,
			vn:      mockVN,
		}

		// Call RewindEngine
		ctx := context.Background()
		rewindTimestamp := uint64(1234567890)
		err := c.RewindEngine(ctx, rewindTimestamp)
		require.NoError(t, err)

		// Verify RewindToTimestamp was called with correct timestamp
		require.Equal(t, 1, mockEngine.rewindToTimestampCalled, "RewindToTimestamp should be called once")
		require.Equal(t, rewindTimestamp, mockEngine.rewindTimestamp, "RewindToTimestamp should be called with correct timestamp")

		// Verify the virtual node was stopped
		mockVN.mu.Lock()
		require.Equal(t, 1, mockVN.stopCalled, "Virtual node should be stopped once")
		mockVN.mu.Unlock()

		// Verify container state: paused should be false (resumed), allowing new VN to start
		require.False(t, c.pause.Load(), "Container should be resumed after rewind")
	})

	t.Run("retries transient errors and eventually fails", func(t *testing.T) {
		// Setup - transient error should be retried
		mockVN := newMockVirtualNode()
		mockEngine := newMockEngineController()
		mockEngine.rewindErr = engine_controller.ErrRewindFCUSyntheticFailed

		chainID := eth.ChainIDFromUInt64(420)
		log := createTestLogger(t)

		c := &simpleChainContainer{
			chainID: chainID,
			log:     log,
			engine:  mockEngine,
			vn:      mockVN,
		}

		// Call RewindEngine - should retry and eventually fail
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) // this will prevent infinite retries
		defer cancel()
		err := c.RewindEngine(ctx, 12345)
		require.Error(t, err)
		require.ErrorIs(t, err, context.DeadlineExceeded)

		// Verify RewindToTimestamp was called multiple times (retry attempts)
		require.Greater(t, mockEngine.rewindToTimestampCalled, 1, "RewindToTimestamp should be retried at least once")

		// Container should still be paused since rewind failed
		require.True(t, c.pause.Load(), "Container should remain paused after failed rewind")
	})

	t.Run("does not retry critical errors", func(t *testing.T) {
		criticalErrors := []struct {
			name string
			err  error
		}{
			{"ErrNoEngineClient", engine_controller.ErrNoEngineClient},
			{"ErrNoRollupConfig", engine_controller.ErrNoRollupConfig},
			{"ErrRewindComputeTargetsFailed", engine_controller.ErrRewindComputeTargetsFailed},
			{"ErrRewindTimestampToBlockConversion", engine_controller.ErrRewindTimestampToBlockConversion},
		}

		for _, tc := range criticalErrors {
			t.Run(tc.name, func(t *testing.T) {
				// Setup - critical error should not be retried
				mockVN := newMockVirtualNode()
				mockEngine := newMockEngineController()
				mockEngine.rewindErr = tc.err

				chainID := eth.ChainIDFromUInt64(420)
				log := createTestLogger(t)

				c := &simpleChainContainer{
					chainID: chainID,
					log:     log,
					engine:  mockEngine,
					vn:      mockVN,
				}

				// Call RewindEngine - should fail immediately without retry
				ctx := context.Background()
				err := c.RewindEngine(ctx, 12345)
				require.Error(t, err)
				require.ErrorIs(t, err, tc.err)

				// Verify RewindToTimestamp was called only once (no retry for critical errors)
				require.Equal(t, 1, mockEngine.rewindToTimestampCalled, "RewindToTimestamp should not be retried for critical errors")
			})
		}
	})

	t.Run("returns error when VN stop fails", func(t *testing.T) {
		// Setup
		mockVN := newMockVirtualNode()
		mockVN.stopErr = context.DeadlineExceeded
		mockEngine := newMockEngineController()

		chainID := eth.ChainIDFromUInt64(420)
		log := createTestLogger(t)

		c := &simpleChainContainer{
			chainID: chainID,
			log:     log,
			engine:  mockEngine,
			vn:      mockVN,
		}

		// Call RewindEngine - should fail on VN stop
		ctx := context.Background()
		err := c.RewindEngine(ctx, 12345)
		require.Error(t, err)
		require.ErrorIs(t, err, context.DeadlineExceeded)

		// Verify RewindToTimestamp was NOT called since VN stop failed
		require.Equal(t, 0, mockEngine.rewindToTimestampCalled, "RewindToTimestamp should not be called when VN stop fails")
	})

	t.Run("succeeds after transient error on retry", func(t *testing.T) {
		// Setup - fail first 2 attempts, succeed on 3rd
		mockVN := newMockVirtualNode()
		mockEngine := newMockEngineController()
		failCount := 0
		mockEngine.rewindFunc = func(ctx context.Context, timestamp uint64) error {
			failCount++
			if failCount < 3 {
				return engine_controller.ErrRewindFCUTargetFailed
			}
			return nil
		}

		chainID := eth.ChainIDFromUInt64(420)
		log := createTestLogger(t)

		c := &simpleChainContainer{
			chainID: chainID,
			log:     log,
			engine:  mockEngine,
			vn:      mockVN,
		}

		// Call RewindEngine - should succeed after retries
		ctx := context.Background()
		err := c.RewindEngine(ctx, 12345)
		require.NoError(t, err)

		// Verify RewindToTimestamp was called 3 times (2 failures + 1 success)
		require.Equal(t, 3, mockEngine.rewindToTimestampCalled, "RewindToTimestamp should be called 3 times")

		// Container should be resumed after successful rewind
		require.False(t, c.pause.Load(), "Container should be resumed after successful rewind")
	})
}

// TestChainContainer_VirtualNodeIntegration tests interaction with VirtualNode
func TestChainContainer_VirtualNodeIntegration(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	vncfg := createTestVNConfig()
	initOverload := &rollupNode.InitializationOverrides{}

	t.Run("Start creates and starts virtual node", func(t *testing.T) {
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		mockVN := newMockVirtualNode()
		mockVN.blockOnStart = true

		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string) virtual_node.VirtualNode {
			return mockVN
		}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		go func() {
			_ = container.Start(ctx)
		}()

		select {
		case <-mockVN.startSignal:
			// Success
		case <-time.After(500 * time.Millisecond):
			t.Fatal("VN Start should have been called")
		}

		require.Equal(t, 1, mockVN.startCalled)
	})

	t.Run("auto-restart virtual node on exit", func(t *testing.T) {
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		restartCount := 0
		mockVN := &mockVirtualNode{
			startSignal: make(chan struct{}),
		}

		mockVN.startFunc = func(ctx context.Context) error {
			restartCount++
			if restartCount < 3 {
				return nil // Exit immediately to trigger restart
			}
			<-ctx.Done()
			return ctx.Err()
		}

		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string) virtual_node.VirtualNode {
			return mockVN
		}

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		go func() {
			_ = container.Start(ctx)
		}()

		require.Eventually(t, func() bool {
			return restartCount >= 3
		}, 1*time.Second, 10*time.Millisecond)
	})

	t.Run("Stop calls virtual node Stop", func(t *testing.T) {
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		mockVN := newMockVirtualNode()
		mockVN.blockOnStart = true

		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string) virtual_node.VirtualNode {
			return mockVN
		}

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			_ = container.Start(ctx)
		}()

		<-mockVN.startSignal

		// Ensure VN is set in container
		require.Eventually(t, func() bool {
			return impl.vn != nil
		}, 1*time.Second, 10*time.Millisecond)

		stopCtx := context.Background()
		_ = container.Stop(stopCtx)

		require.Eventually(t, func() bool {
			mockVN.mu.Lock()
			defer mockVN.mu.Unlock()
			return mockVN.stopCalled >= 1
		}, 2*time.Second, 10*time.Millisecond)

		cancel()
	})

	t.Run("registers handler with reverse proxy", func(t *testing.T) {
		var setHandlerCalled bool
		var calledChainID string

		setHandler := func(id string, h http.Handler) {
			setHandlerCalled = true
			calledChainID = id
		}

		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, setHandler, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		mockVN := newMockVirtualNode()
		mockVN.blockOnStart = true
		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string) virtual_node.VirtualNode {
			return mockVN
		}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		go func() {
			_ = container.Start(ctx)
		}()

		<-mockVN.startSignal

		require.Eventually(t, func() bool {
			return setHandlerCalled && calledChainID == "420"
		}, 1*time.Second, 10*time.Millisecond)
	})
}

// TestChainContainer_VerifiedAt tests the VerifiedAt method
func TestChainContainer_VerifiedAt(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	vncfg := createTestVNConfig()
	log := createTestLogger(t)
	cfg := createTestCLIConfig(t.TempDir())
	initOverload := &rollupNode.InitializationOverrides{}

	t.Run("returns error when verification activity reports not verified", func(t *testing.T) {
		// Create a mock verification activity that returns verified=false
		mockVerifier := &mockVerificationActivity{
			name:                      "test-verifier",
			verifiedAtTimestampResult: false, // not verified
			verifiedAtTimestampErr:    nil,
		}

		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		container.RegisterVerifier(mockVerifier)

		// Set up mock engine controller
		mockEngine := &mockEngineController{
			blockAtTimestampResult: eth.L2BlockRef{
				Hash:   [32]byte{1},
				Number: 100,
			},
			blockAtTimestampErr: nil,
		}
		impl.engine = mockEngine

		// Set up mock virtual node for safeDBAtL2
		mockVN := newMockVirtualNode()
		mockVN.safeHeadL1 = eth.BlockID{Hash: [32]byte{2}, Number: 50}
		mockVN.safeHeadErr = nil
		impl.vn = mockVN

		ctx := context.Background()
		l2, l1, err := container.VerifiedAt(ctx, 1000)

		// Should return an error when verification fails
		require.Error(t, err)
		require.ErrorIs(t, err, ethereum.NotFound)
		require.Equal(t, eth.BlockID{}, l2)
		require.Equal(t, eth.BlockID{}, l1)
	})
}
