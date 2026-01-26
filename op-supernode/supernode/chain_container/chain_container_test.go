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
	"github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/virtual_node"
	"github.com/ethereum/go-ethereum"
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
}

func (m *mockEngineController) BlockAtTimestamp(ctx context.Context, ts uint64, label eth.BlockLabel) (eth.L2BlockRef, error) {
	return m.blockAtTimestampResult, m.blockAtTimestampErr
}

func (m *mockEngineController) OutputV0AtBlockNumber(ctx context.Context, num uint64) (*eth.OutputV0, error) {
	return nil, nil
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

// Test helpers
func createTestVNConfig() *opnodecfg.Config {
	return &opnodecfg.Config{
		Rollup: rollup.Config{
			L2ChainID: big.NewInt(420),
		},
	}
}

func createTestCLIConfig() config.CLIConfig {
	return config.CLIConfig{
		DataDir: "/tmp/test",
		RPCConfig: oprpc.CLIConfig{
			ListenAddr: "0.0.0.0",
			ListenPort: 8545,
		},
	}
}

func createTestLogger() gethlog.Logger {
	return gethlog.New()
}

// TestChainContainer_Constructor tests initialization and configuration
func TestChainContainer_Constructor(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	vncfg := createTestVNConfig()
	log := createTestLogger()
	cfg := createTestCLIConfig()
	initOverload := &rollupNode.InitializationOverrides{}

	t.Run("creates container with correct config", func(t *testing.T) {
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
		cfg := config.CLIConfig{
			DataDir: "/tmp/datadir",
		}
		container := NewChainContainer(eth.ChainIDFromUInt64(420), vncfg, log, cfg, initOverload, nil, nil, nil)

		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		expectedPath := filepath.Join("/tmp/datadir", "420", "safe_db")
		require.Equal(t, expectedPath, impl.vncfg.SafeDBPath)
	})

	t.Run("RPC config inherited from supernode config", func(t *testing.T) {
		cfg := config.CLIConfig{
			DataDir: "/tmp/test",
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		require.Equal(t, virtualNodeVersion, impl.appVersion)
	})

	t.Run("subPath combines DataDir, chainID, and path correctly", func(t *testing.T) {
		cfg := config.CLIConfig{
			DataDir: "/data",
		}
		container := NewChainContainer(eth.ChainIDFromUInt64(420), vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		result := impl.subPath("safe_db")
		expected := filepath.Join("/data", "420", "safe_db")
		require.Equal(t, expected, result)
	})

	t.Run("subPath works with various chain IDs", func(t *testing.T) {
		cfg := config.CLIConfig{
			DataDir: "/data",
		}

		testCases := []struct {
			chainID  eth.ChainID
			path     string
			expected string
		}{
			{eth.ChainIDFromUInt64(10), "safe_db", "/data/10/safe_db"},
			{eth.ChainIDFromUInt64(11155420), "safe_db", "/data/11155420/safe_db"},
			{eth.ChainIDFromUInt64(8453), "peerstore", "/data/8453/peerstore"},
		}

		for _, tc := range testCases {
			container := NewChainContainer(tc.chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
			impl, ok := container.(*simpleChainContainer)
			require.True(t, ok)

			result := impl.subPath(tc.path)
			expected := filepath.Join(cfg.DataDir, tc.chainID.String(), tc.path)
			require.Equal(t, expected, result, "subPath should work for chain %d", tc.chainID)
		}
	})
}

// TestChainContainer_Lifecycle tests Start/Stop behavior
func TestChainContainer_Lifecycle(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	vncfg := createTestVNConfig()
	log := createTestLogger()
	cfg := createTestCLIConfig()
	initOverload := &rollupNode.InitializationOverrides{}

	t.Run("Start respects stop flag", func(t *testing.T) {
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		require.False(t, impl.stop.Load())

		ctx := context.Background()
		_ = container.Stop(ctx)

		require.True(t, impl.stop.Load())
	})

	t.Run("signals stopped channel on exit", func(t *testing.T) {
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
	log := createTestLogger()
	cfg := createTestCLIConfig()
	initOverload := &rollupNode.InitializationOverrides{}

	t.Run("Pause sets pause flag", func(t *testing.T) {
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		ctx := context.Background()
		err := container.Pause(ctx)

		require.NoError(t, err)
		require.True(t, impl.pause.Load())
	})

	t.Run("Resume clears pause flag", func(t *testing.T) {
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

// TestChainContainer_VirtualNodeIntegration tests interaction with VirtualNode
func TestChainContainer_VirtualNodeIntegration(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	vncfg := createTestVNConfig()
	log := createTestLogger()
	cfg := createTestCLIConfig()
	initOverload := &rollupNode.InitializationOverrides{}

	t.Run("Start creates and starts virtual node", func(t *testing.T) {
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
	log := createTestLogger()
	cfg := createTestCLIConfig()
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
