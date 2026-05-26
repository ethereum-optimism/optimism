package chain_container

import (
	"context"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/virtual_node"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	dto "github.com/prometheus/client_model/go"
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

	// syncStatusOverride lets tests return a fully-formed eth.SyncStatus
	// (e.g. populated LocalSafeL2.Time / LocalFinalizedL2 / FinalizedL1)
	// instead of the synthesised default built from safeHeadL1/safeHeadL2.
	// When nil, the default synthesis below is used.
	syncStatusOverride func() (*eth.SyncStatus, error)

	// Decouples FirstSafeHeadEntry's result from safeHeadL1/L2 (read by
	// SyncStatus, SafeHeadAtL1, etc.).
	firstSafeHeadEntryOverride func() (eth.BlockID, eth.BlockID, error)

	// Order in which mock methods were invoked.
	methodCalls []string
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

// FirstSafeHeadEntry implements virtual_node.VirtualNode FirstSafeHeadEntry
func (m *mockVirtualNode) FirstSafeHeadEntry(ctx context.Context) (eth.BlockID, eth.BlockID, error) {
	m.mu.Lock()
	m.methodCalls = append(m.methodCalls, "FirstSafeHeadEntry")
	m.mu.Unlock()
	if m.firstSafeHeadEntryOverride != nil {
		return m.firstSafeHeadEntryOverride()
	}
	return m.safeHeadL1, m.safeHeadL2, m.safeHeadErr
}

// LastL1 implements virtual_node.VirtualNode LastL1
func (m *mockVirtualNode) LastL1(ctx context.Context) (eth.BlockID, error) {
	return m.safeHeadL1, m.safeHeadErr
}

// SyncStatus implements virtual_node.VirtualNode SyncStatus
func (m *mockVirtualNode) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	m.mu.Lock()
	m.methodCalls = append(m.methodCalls, "SyncStatus")
	m.mu.Unlock()
	if m.syncStatusOverride != nil {
		return m.syncStatusOverride()
	}
	if m.safeHeadErr != nil {
		return nil, m.safeHeadErr
	}
	return &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{},
		CurrentL1:   eth.L1BlockRef{Hash: m.safeHeadL1.Hash, Number: m.safeHeadL1.Number},
		LocalSafeL2: eth.L2BlockRef{Hash: m.safeHeadL2.Hash, Number: m.safeHeadL2.Number},
	}, nil
}

// SafeDB is not required by VirtualNode in these tests

// mockEngineController is a mock implementation of engine_controller.EngineController
type mockEngineController struct {
	rewindCalls              int
	rewindTarget             *eth.ExecutionPayloadEnvelope
	rewindErr                error
	rewindFunc               func(ctx context.Context, target *eth.ExecutionPayloadEnvelope) error // optional custom behavior
	l2BlockRefByNumberResult eth.L2BlockRef
	l2BlockRefByNumberErr    error
	payloadByHashResult      *eth.ExecutionPayloadEnvelope
	payloadByHashErr         error
	payloadByNumberResult    *eth.ExecutionPayloadEnvelope
	payloadByNumberErr       error
}

func (m *mockEngineController) BlockAtTimestamp(ctx context.Context, ts uint64, label eth.BlockLabel) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{}, nil
}

func (m *mockEngineController) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	return m.l2BlockRefByNumberResult, m.l2BlockRefByNumberErr
}

func (m *mockEngineController) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{}, nil
}

func (m *mockEngineController) OutputV0AtBlockNumber(ctx context.Context, num uint64) (*eth.OutputV0, error) {
	return nil, nil
}

func (m *mockEngineController) OutputV0ByBlockHash(ctx context.Context, blockHash common.Hash) (*eth.OutputV0, error) {
	return nil, nil
}

func (m *mockEngineController) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	return nil, nil, nil
}

func (m *mockEngineController) Close() error {
	return nil
}

var _ engine_controller.EngineController = (*mockEngineController)(nil)

// Test helpers
func createTestVNConfig() *opnodecfg.Config {
	return &opnodecfg.Config{
		Rollup: rollup.Config{
			L2ChainID: big.NewInt(420),
			BlockTime: 2, // Set a non-zero block time to avoid divide by zero
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
func (m *mockEngineController) Rewind(ctx context.Context, target *eth.ExecutionPayloadEnvelope) error {
	m.rewindCalls++
	m.rewindTarget = target
	if m.rewindFunc != nil {
		return m.rewindFunc(ctx, target)
	}
	return m.rewindErr
}

func (m *mockEngineController) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error) {
	return m.payloadByHashResult, m.payloadByHashErr
}

func (m *mockEngineController) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	return m.payloadByNumberResult, m.payloadByNumberErr
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)

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
		container := NewChainContainer(eth.ChainIDFromUInt64(420), vncfg, log, cfg, initOverload, nil, nil, nil, nil)

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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)

		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		require.Equal(t, cfg.RPCConfig, impl.vncfg.RPC)
	})

	t.Run("appVersion set correctly", func(t *testing.T) {
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		require.Equal(t, virtualNodeVersion, impl.appVersion)
	})

	t.Run("subPath combines DataDir, chainID, and path correctly", func(t *testing.T) {
		dataDir := t.TempDir()
		cfg := config.CLIConfig{
			DataDir: dataDir,
		}
		container := NewChainContainer(eth.ChainIDFromUInt64(420), vncfg, log, cfg, initOverload, nil, nil, nil, nil)
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
			container := NewChainContainer(tc.chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
			impl, ok := container.(*simpleChainContainer)
			require.True(t, ok)

			result := impl.subPath(tc.path)
			expected := filepath.Join(dataDir, tc.chainID.String(), tc.path)
			require.Equal(t, expected, result, "subPath should work for chain %d", tc.chainID)
		}
	})
}

// TestChainContainer_EngineControllerNotInitInConstructor verifies that the
// engine controller is NOT initialized in NewChainContainer (it is deferred to
// the Start loop so that transient EL unavailability at startup is retried).
func TestChainContainer_EngineControllerNotInitInConstructor(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	vncfg := createTestVNConfig()
	log := createTestLogger(t)
	cfg := createTestCLIConfig(t.TempDir())
	initOverload := &rollupNode.InitializationOverrides{}

	container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
	impl, ok := container.(*simpleChainContainer)
	require.True(t, ok)
	require.Nil(t, impl.engine, "engine should not be initialized in constructor; it is deferred to Start loop")
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		mockVN := newMockVirtualNode()
		mockVN.blockOnStart = true
		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string, superAuthority rollup.SuperAuthority) virtual_node.VirtualNode {
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		mockVN := newMockVirtualNode()
		mockVN.startFunc = func(ctx context.Context) error {
			return nil // Exit immediately to trigger restart
		}

		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string, superAuthority rollup.SuperAuthority) virtual_node.VirtualNode {
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		mockVN := newMockVirtualNode()
		mockVN.startFunc = func(ctx context.Context) error {
			return nil // Exit immediately
		}

		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string, superAuthority rollup.SuperAuthority) virtual_node.VirtualNode {
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		var startedSignal = make(chan struct{})
		var totalStartCalls int
		var mu sync.Mutex

		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string, superAuthority rollup.SuperAuthority) virtual_node.VirtualNode {
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
	makeTarget := func(timestamp uint64) *eth.ExecutionPayloadEnvelope {
		return &eth.ExecutionPayloadEnvelope{
			ExecutionPayload: &eth.ExecutionPayload{
				BlockNumber: eth.Uint64Quantity(99),
				Timestamp:   eth.Uint64Quantity(timestamp),
				BlockHash:   common.Hash{0xaa},
				ParentHash:  common.Hash{0xab},
			},
		}
	}

	t.Run("calls engine Rewind with the supplied target and stops VN", func(t *testing.T) {
		mockVN := newMockVirtualNode()
		mockEngine := newMockEngineController()

		chainID := eth.ChainIDFromUInt64(420)
		log := createTestLogger(t)

		c := &simpleChainContainer{
			chainID: chainID,
			log:     log,
			engine:  mockEngine,
			vn:      mockVN,
		}

		ctx := context.Background()
		rewindTimestamp := uint64(1234567890)
		target := makeTarget(rewindTimestamp)
		invalidatedBlock := eth.BlockRef{Number: 100, Hash: common.Hash{0x1}, ParentHash: common.Hash{0x2}, Time: rewindTimestamp + 2}
		err := c.RewindEngine(ctx, target, invalidatedBlock)
		require.NoError(t, err)

		require.Equal(t, 1, mockEngine.rewindCalls, "engine.Rewind should be called once")
		require.Same(t, target, mockEngine.rewindTarget, "engine.Rewind should receive the supplied target envelope")

		mockVN.mu.Lock()
		require.Equal(t, 1, mockVN.stopCalled, "Virtual node should be stopped once")
		mockVN.mu.Unlock()

		require.False(t, c.pause.Load(), "Container should be resumed after rewind")
	})

	t.Run("rejects nil target without touching the engine", func(t *testing.T) {
		mockVN := newMockVirtualNode()
		mockEngine := newMockEngineController()

		c := &simpleChainContainer{
			chainID: eth.ChainIDFromUInt64(420),
			log:     createTestLogger(t),
			engine:  mockEngine,
			vn:      mockVN,
		}

		err := c.RewindEngine(context.Background(), nil, eth.BlockRef{})
		require.ErrorIs(t, err, engine_controller.ErrRewindNilTarget)
		require.Equal(t, 0, mockEngine.rewindCalls)
	})

	t.Run("retries transient errors and eventually fails", func(t *testing.T) {
		mockVN := newMockVirtualNode()
		mockEngine := newMockEngineController()
		mockEngine.rewindErr = engine_controller.ErrRewindFCUSyntheticFailed

		c := &simpleChainContainer{
			chainID: eth.ChainIDFromUInt64(420),
			log:     createTestLogger(t),
			engine:  mockEngine,
			vn:      mockVN,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		invalidatedBlock := eth.BlockRef{Number: 100, Hash: common.Hash{0x1}, ParentHash: common.Hash{0x2}, Time: 12347}
		err := c.RewindEngine(ctx, makeTarget(12345), invalidatedBlock)
		require.Error(t, err)
		require.ErrorIs(t, err, context.DeadlineExceeded)

		require.Greater(t, mockEngine.rewindCalls, 1, "engine.Rewind should be retried at least once")
		require.False(t, c.pause.Load(), "Container should be resumed (not stuck paused) after failed rewind")
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
			{"ErrRewindNilTarget", engine_controller.ErrRewindNilTarget},
			{"ErrRewindTargetMismatch", engine_controller.ErrRewindTargetMismatch},
		}

		for _, tc := range criticalErrors {
			t.Run(tc.name, func(t *testing.T) {
				mockVN := newMockVirtualNode()
				mockEngine := newMockEngineController()
				mockEngine.rewindErr = tc.err

				c := &simpleChainContainer{
					chainID: eth.ChainIDFromUInt64(420),
					log:     createTestLogger(t),
					engine:  mockEngine,
					vn:      mockVN,
				}

				invalidatedBlock := eth.BlockRef{Number: 100, Hash: common.Hash{0x1}, ParentHash: common.Hash{0x2}, Time: 12347}
				err := c.RewindEngine(context.Background(), makeTarget(12345), invalidatedBlock)
				require.Error(t, err)
				require.ErrorIs(t, err, tc.err)
				require.Equal(t, 1, mockEngine.rewindCalls, "engine.Rewind should not be retried for critical errors")
			})
		}
	})

	t.Run("returns error when VN stop fails", func(t *testing.T) {
		mockVN := newMockVirtualNode()
		mockVN.stopErr = context.DeadlineExceeded
		mockEngine := newMockEngineController()

		c := &simpleChainContainer{
			chainID: eth.ChainIDFromUInt64(420),
			log:     createTestLogger(t),
			engine:  mockEngine,
			vn:      mockVN,
		}

		invalidatedBlock := eth.BlockRef{Number: 100, Hash: common.Hash{0x1}, ParentHash: common.Hash{0x2}, Time: 12347}
		err := c.RewindEngine(context.Background(), makeTarget(12345), invalidatedBlock)
		require.Error(t, err)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Equal(t, 0, mockEngine.rewindCalls, "engine.Rewind should not be called when VN stop fails")
	})

	t.Run("succeeds after transient error on retry", func(t *testing.T) {
		mockVN := newMockVirtualNode()
		mockEngine := newMockEngineController()
		failCount := 0
		mockEngine.rewindFunc = func(ctx context.Context, target *eth.ExecutionPayloadEnvelope) error {
			failCount++
			if failCount < 3 {
				return engine_controller.ErrRewindFCUTargetFailed
			}
			return nil
		}

		c := &simpleChainContainer{
			chainID: eth.ChainIDFromUInt64(420),
			log:     createTestLogger(t),
			engine:  mockEngine,
			vn:      mockVN,
		}

		invalidatedBlock := eth.BlockRef{Number: 100, Hash: common.Hash{0x1}, ParentHash: common.Hash{0x2}, Time: 12347}
		err := c.RewindEngine(context.Background(), makeTarget(12345), invalidatedBlock)
		require.NoError(t, err)
		require.Equal(t, 3, mockEngine.rewindCalls, "engine.Rewind should be called 3 times (2 failures + 1 success)")
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		mockVN := newMockVirtualNode()
		mockVN.blockOnStart = true

		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string, superAuthority rollup.SuperAuthority) virtual_node.VirtualNode {
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
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

		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string, superAuthority rollup.SuperAuthority) virtual_node.VirtualNode {
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

	t.Run("restart increments VNRestarts metric", func(t *testing.T) {
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		metrics := resources.NewSupernodeMetrics()
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, metrics)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		restartCount := 0
		mockVN := &mockVirtualNode{startSignal: make(chan struct{})}
		mockVN.startFunc = func(ctx context.Context) error {
			restartCount++
			if restartCount < 3 {
				return nil
			}
			<-ctx.Done()
			return ctx.Err()
		}
		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string, superAuthority rollup.SuperAuthority) virtual_node.VirtualNode {
			return mockVN
		}

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		go func() { _ = container.Start(ctx) }()

		require.Eventually(t, func() bool { return restartCount >= 3 }, 1*time.Second, 10*time.Millisecond)
		// The first start is not a restart; restarts 2 and 3 should be counted.
		var dto dto.Metric
		require.NoError(t, metrics.VNRestarts.WithLabelValues(chainID.String()).Write(&dto))
		require.Equal(t, float64(2), dto.GetCounter().GetValue())
	})

	t.Run("Stop calls virtual node Stop", func(t *testing.T) {
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		mockVN := newMockVirtualNode()
		mockVN.blockOnStart = true

		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string, superAuthority rollup.SuperAuthority) virtual_node.VirtualNode {
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
		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, setHandler, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		mockVN := newMockVirtualNode()
		mockVN.blockOnStart = true
		impl.virtualNodeFactory = func(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string, superAuthority rollup.SuperAuthority) virtual_node.VirtualNode {
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

// TestChainContainer_OptimisticAt_ErrL1AtSafeHeadNotFound tests that
// ErrL1AtSafeHeadNotFound from the virtual node is mapped to ethereum.NotFound
// by OptimisticAt (via safeDBAtL2), so callers treat chain lag as "not ready".
func TestChainContainer_OptimisticAt_ErrL1AtSafeHeadNotFound(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	vncfg := createTestVNConfig()
	vncfg.Rollup.Genesis.L2Time = 1000
	vncfg.Rollup.BlockTime = 2
	log, logs := testlog.CaptureLogger(t, gethlog.LevelDebug)
	cfg := createTestCLIConfig(t.TempDir())
	initOverload := &rollupNode.InitializationOverrides{}

	container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
	impl, ok := container.(*simpleChainContainer)
	require.True(t, ok)

	// Set up engine that returns a valid block ref
	mockEngine := &mockEngineController{
		l2BlockRefByNumberResult: eth.L2BlockRef{
			Hash:   common.Hash{0x01},
			Number: 5,
			Time:   1010,
		},
	}
	impl.engine = mockEngine

	// Use a mock VN that returns valid SyncStatus (so LocalSafeBlockAtTimestamp succeeds)
	// but returns ErrL1AtSafeHeadNotFound from L1AtSafeHead (so safeDBAtL2 fails).
	mockVN := &mockVNForL1AtSafeHeadError{
		syncStatusResult: &eth.SyncStatus{
			CurrentL1:   eth.L1BlockRef{Hash: common.Hash{0x10}, Number: 50},
			LocalSafeL2: eth.L2BlockRef{Hash: common.Hash{0x20}, Number: 100},
		},
		l1AtSafeHeadErr: virtual_node.ErrL1AtSafeHeadNotFound,
	}
	impl.vn = mockVN

	ctx := context.Background()
	_, _, err := container.OptimisticAt(ctx, 1010)

	require.Error(t, err)
	require.True(t, errors.Is(err, ethereum.NotFound),
		"ErrL1AtSafeHeadNotFound should be mapped to ethereum.NotFound, got: %v", err)
	require.Nil(t, logs.FindLog(
		testlog.NewLevelFilter(slog.LevelError),
		testlog.NewMessageFilter("error determining l1 block number at which l2 block became safe"),
	))
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(slog.LevelDebug),
		testlog.NewMessageFilter("l1 block at which l2 block became safe is not available yet"),
	))
}

func TestChainContainer_OptimisticAt_LocalSafeTipNotFoundLogsDebug(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	vncfg := createTestVNConfig()
	vncfg.Rollup.Genesis.L2Time = 1000
	vncfg.Rollup.BlockTime = 2
	log, logs := testlog.CaptureLogger(t, gethlog.LevelDebug)
	cfg := createTestCLIConfig(t.TempDir())
	initOverload := &rollupNode.InitializationOverrides{}

	container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
	impl, ok := container.(*simpleChainContainer)
	require.True(t, ok)

	impl.engine = &mockEngineController{}
	impl.vn = &mockVNForL1AtSafeHeadError{
		syncStatusResult: &eth.SyncStatus{
			CurrentL1:   eth.L1BlockRef{Hash: common.Hash{0x10}, Number: 50},
			LocalSafeL2: eth.L2BlockRef{Hash: common.Hash{0x20}, Number: 100},
		},
	}

	_, _, err := container.OptimisticAt(context.Background(), 2000)

	require.ErrorIs(t, err, ethereum.NotFound)
	require.Nil(t, logs.FindLog(
		testlog.NewLevelFilter(slog.LevelError),
		testlog.NewMessageFilter("error determining l2 block at given timestamp"),
	))
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(slog.LevelDebug),
		testlog.NewMessageFilter("l2 block at timestamp is not local safe yet"),
	))
}

// mockVNForL1AtSafeHeadError is a VN mock that returns valid SyncStatus
// but can return specific errors from L1AtSafeHead.
type mockVNForL1AtSafeHeadError struct {
	syncStatusResult *eth.SyncStatus
	l1AtSafeHeadErr  error
}

func (m *mockVNForL1AtSafeHeadError) Start(ctx context.Context) error { return nil }
func (m *mockVNForL1AtSafeHeadError) Stop(ctx context.Context) error  { return nil }
func (m *mockVNForL1AtSafeHeadError) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *mockVNForL1AtSafeHeadError) L1AtSafeHead(ctx context.Context, target eth.BlockID) (eth.BlockID, error) {
	return eth.BlockID{}, m.l1AtSafeHeadErr
}
func (m *mockVNForL1AtSafeHeadError) FirstSafeHeadEntry(ctx context.Context) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *mockVNForL1AtSafeHeadError) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return m.syncStatusResult, nil
}

var _ virtual_node.VirtualNode = (*mockVNForL1AtSafeHeadError)(nil)

// ErrL1AtSafeHeadUnavailable from the VN must map to ErrHistoryUnavailable
// (and NOT to ethereum.NotFound) so interop halts instead of treating it as
// transient chain lag.
func TestChainContainer_OptimisticAt_ErrL1AtSafeHeadUnavailable(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	vncfg := createTestVNConfig()
	vncfg.Rollup.Genesis.L2Time = 1000
	vncfg.Rollup.BlockTime = 2
	log := createTestLogger(t)
	cfg := createTestCLIConfig(t.TempDir())
	initOverload := &rollupNode.InitializationOverrides{}

	container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
	impl, ok := container.(*simpleChainContainer)
	require.True(t, ok)

	mockEngine := &mockEngineController{
		l2BlockRefByNumberResult: eth.L2BlockRef{
			Hash:   common.Hash{0x01},
			Number: 5,
			Time:   1010,
		},
	}
	impl.engine = mockEngine

	mockVN := &mockVNForL1AtSafeHeadError{
		syncStatusResult: &eth.SyncStatus{
			CurrentL1:   eth.L1BlockRef{Hash: common.Hash{0x10}, Number: 50},
			LocalSafeL2: eth.L2BlockRef{Hash: common.Hash{0x20}, Number: 100},
		},
		l1AtSafeHeadErr: virtual_node.ErrL1AtSafeHeadUnavailable,
	}
	impl.vn = mockVN

	ctx := context.Background()
	_, _, err := container.OptimisticAt(ctx, 1010)

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrHistoryUnavailable),
		"ErrL1AtSafeHeadUnavailable should be mapped to ErrHistoryUnavailable, got: %v", err)
	require.False(t, errors.Is(err, ethereum.NotFound),
		"ErrL1AtSafeHeadUnavailable must NOT be mapped to ethereum.NotFound — that would make it look transient. got: %v", err)
}

// TestChainContainer_LocalSafeBlockAtTimestamp tests the LocalSafeBlockAtTimestamp method
func TestChainContainer_LocalSafeBlockAtTimestamp(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name              string
		genesisTime       uint64
		blockTime         uint64
		targetTimestamp   uint64
		localSafeNumber   uint64
		engineResult      *eth.L2BlockRef
		engineError       error
		syncStatusError   error
		engineNil         bool
		expectError       error
		expectResult      *eth.L2BlockRef
		expectErrorString string
	}

	tests := []testCase{
		{
			name:            "returns block when target is before local safe head",
			genesisTime:     1000,
			blockTime:       2,
			targetTimestamp: 1010,
			localSafeNumber: 100,
			engineResult:    &eth.L2BlockRef{Hash: [32]byte{1}, Number: 5, Time: 1010},
			expectResult:    &eth.L2BlockRef{Hash: [32]byte{1}, Number: 5, Time: 1010},
		},
		{
			name:            "returns NotFound when target exceeds local safe head",
			genesisTime:     1000,
			blockTime:       2,
			targetTimestamp: 2000,
			localSafeNumber: 100,
			expectError:     ethereum.NotFound,
		},
		{
			name:            "returns error when engine is nil",
			genesisTime:     1000,
			blockTime:       2,
			targetTimestamp: 1000,
			engineNil:       true,
			expectError:     engine_controller.ErrNoEngineClient,
		},
		{
			name:            "returns block at exact timestamp match",
			genesisTime:     1000,
			blockTime:       2,
			targetTimestamp: 1020,
			localSafeNumber: 100,
			engineResult:    &eth.L2BlockRef{Hash: [32]byte{5}, Number: 10, Time: 1020},
			expectResult:    &eth.L2BlockRef{Hash: [32]byte{5}, Number: 10, Time: 1020},
		},
		{
			name:              "returns error when sync status fails",
			genesisTime:       1000,
			blockTime:         2,
			targetTimestamp:   1000,
			syncStatusError:   errors.New("sync status error"),
			expectErrorString: "sync status error",
		},
		{
			name:            "handles genesis block correctly",
			genesisTime:     1000,
			blockTime:       2,
			targetTimestamp: 1000,
			localSafeNumber: 10,
			engineResult:    &eth.L2BlockRef{Hash: [32]byte{0}, Number: 0, Time: 1000},
			expectResult:    &eth.L2BlockRef{Hash: [32]byte{0}, Number: 0, Time: 1000},
		},
	}

	runTest := func(t *testing.T, tc testCase) {
		chainID := eth.ChainIDFromUInt64(420)
		log := createTestLogger(t)
		cfg := createTestCLIConfig(t.TempDir())
		initOverload := &rollupNode.InitializationOverrides{}

		vncfg := createTestVNConfig()
		vncfg.Rollup.Genesis.L2Time = tc.genesisTime
		vncfg.Rollup.BlockTime = tc.blockTime

		container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
		impl, ok := container.(*simpleChainContainer)
		require.True(t, ok)

		// Setup engine
		if !tc.engineNil {
			mockEngine := &mockEngineController{
				l2BlockRefByNumberResult: eth.L2BlockRef{},
				l2BlockRefByNumberErr:    tc.engineError,
			}
			if tc.engineResult != nil {
				mockEngine.l2BlockRefByNumberResult = *tc.engineResult
			}
			impl.engine = mockEngine
		}

		// Setup virtual node
		mockVN := newMockVirtualNode()
		mockVN.safeHeadL2 = eth.BlockID{Number: tc.localSafeNumber}
		mockVN.safeHeadErr = tc.syncStatusError
		impl.vn = mockVN

		// Execute test
		result, err := container.LocalSafeBlockAtTimestamp(context.Background(), tc.targetTimestamp)

		// Verify results
		if tc.expectError != nil {
			require.Error(t, err)
			require.ErrorIs(t, err, tc.expectError)
		} else if tc.expectErrorString != "" {
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectErrorString)
		} else {
			require.NoError(t, err)
			require.Equal(t, *tc.expectResult, result)
		}
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestChainContainer_OptimisticOutputAtTimestamp_ReturnsDeniedOutput(t *testing.T) {
	t.Parallel()

	genesisTime := uint64(1000)
	blockTime := uint64(2)
	vncfg := createTestVNConfig()
	vncfg.Rollup.Genesis.L2Time = genesisTime
	vncfg.Rollup.BlockTime = blockTime
	log := createTestLogger(t)

	dl, err := OpenDenyList(filepath.Join(t.TempDir(), "denylist"))
	require.NoError(t, err)
	defer dl.Close()

	stateRoot := eth.Bytes32(common.HexToHash("0xabcd"))
	msgPasserRoot := eth.Bytes32(common.HexToHash("0x1234"))
	payloadHash := common.HexToHash("0xdead")

	// Block at height 5: timestamp = 1000 + 5*2 = 1010
	height := uint64(5)
	ts := genesisTime + height*blockTime
	require.NoError(t, dl.Add(height, payloadHash, 0, stateRoot, msgPasserRoot))

	container := &simpleChainContainer{
		vncfg:    vncfg,
		denyList: dl,
		log:      log,
	}

	out, err := container.OptimisticOutputAtTimestamp(context.Background(), ts)
	require.NoError(t, err)

	require.Equal(t, &eth.OutputV0{
		StateRoot:                stateRoot,
		MessagePasserStorageRoot: msgPasserRoot,
		BlockHash:                payloadHash,
	}, out)
}

func TestChainContainer_OptimisticOutputAtTimestamp_UsesLatestDeniedRecord(t *testing.T) {
	t.Parallel()

	genesisTime := uint64(1000)
	blockTime := uint64(2)
	vncfg := createTestVNConfig()
	vncfg.Rollup.Genesis.L2Time = genesisTime
	vncfg.Rollup.BlockTime = blockTime
	log := createTestLogger(t)

	dl, err := OpenDenyList(filepath.Join(t.TempDir(), "denylist"))
	require.NoError(t, err)
	defer dl.Close()

	height := uint64(5)
	ts := genesisTime + height*blockTime

	// Add two denied records at the same height — the latest should win
	firstHash := common.HexToHash("0x1111")
	require.NoError(t, dl.Add(height, firstHash, 100, eth.Bytes32{0x01}, eth.Bytes32{0x02}))

	latestHash := common.HexToHash("0x2222")
	latestState := eth.Bytes32(common.HexToHash("0xlatest"))
	latestMsgPasser := eth.Bytes32(common.HexToHash("0xlatestmp"))
	require.NoError(t, dl.Add(height, latestHash, 200, latestState, latestMsgPasser))

	container := &simpleChainContainer{
		vncfg:    vncfg,
		denyList: dl,
		log:      log,
	}

	out, err := container.OptimisticOutputAtTimestamp(context.Background(), ts)
	require.NoError(t, err)
	require.Equal(t, latestHash, out.BlockHash)
	require.Equal(t, latestState, out.StateRoot)
	require.Equal(t, latestMsgPasser, out.MessagePasserStorageRoot)
}

func TestChainContainer_OptimisticOutputAtTimestamp_FallsThroughWhenNoDenied(t *testing.T) {
	t.Parallel()

	genesisTime := uint64(1000)
	blockTime := uint64(2)
	vncfg := createTestVNConfig()
	vncfg.Rollup.Genesis.L2Time = genesisTime
	vncfg.Rollup.BlockTime = blockTime
	log := createTestLogger(t)

	// Empty deny list — no denied records at any height
	dl, err := OpenDenyList(filepath.Join(t.TempDir(), "denylist"))
	require.NoError(t, err)
	defer dl.Close()

	container := &simpleChainContainer{
		vncfg:    vncfg,
		denyList: dl,
		log:      log,
		// No engine set, so the fallback path will error — proving we reached it
	}

	_, err = container.OptimisticOutputAtTimestamp(context.Background(), genesisTime+5*blockTime)
	require.Error(t, err)
	require.ErrorIs(t, err, engine_controller.ErrNoEngineClient)
}

func TestChainContainer_SyncStatus_UninitializedVirtualNode(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	log := createTestLogger(t)
	cfg := createTestCLIConfig(t.TempDir())
	initOverload := &rollupNode.InitializationOverrides{}

	container := NewChainContainer(chainID, createTestVNConfig(), log, cfg, initOverload, nil, nil, nil, nil)

	status, err := container.SyncStatus(context.Background())
	require.Nil(t, status)
	require.ErrorIs(t, err, virtual_node.ErrVirtualNodeNotRunning)
}

func TestChainContainer_BlockNumberToTimestamp_RespectsGenesisBlockNumber(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	log := createTestLogger(t)
	cfg := createTestCLIConfig(t.TempDir())
	initOverload := &rollupNode.InitializationOverrides{}

	vncfg := createTestVNConfig()
	vncfg.Rollup.Genesis.L2Time = 1000
	vncfg.Rollup.Genesis.L2.Number = 100
	vncfg.Rollup.BlockTime = 2

	container := NewChainContainer(chainID, vncfg, log, cfg, initOverload, nil, nil, nil, nil)
	impl, ok := container.(*simpleChainContainer)
	require.True(t, ok)

	timestamp, err := impl.BlockNumberToTimestamp(context.Background(), 104)
	require.NoError(t, err)
	require.Equal(t, uint64(1008), timestamp)

	_, err = impl.BlockNumberToTimestamp(context.Background(), 99)
	require.Error(t, err)
	require.Contains(t, err.Error(), "before genesis 100")
}

// Returns ErrSafeDBNotReady until the deriver's currentL1 has moved past
// firstEntry.L1; only then is the entry's L2 final.
func TestChainContainer_FirstSafeHeadTimestamp_StableSnapshot(t *testing.T) {
	t.Parallel()

	const blockTime = 2
	const genesisL2Time = 1000

	type tc struct {
		name       string
		firstEntry func() (eth.BlockID, eth.BlockID, error)
		syncStatus func() (*eth.SyncStatus, error)
		wantErr    error
		wantTS     uint64
	}
	for _, c := range []tc{
		{
			name: "empty SafeDB -> ErrSafeDBNotReady",
			firstEntry: func() (eth.BlockID, eth.BlockID, error) {
				return eth.BlockID{}, eth.BlockID{}, safedb.ErrNotFound
			},
			syncStatus: func() (*eth.SyncStatus, error) {
				return &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 5}}, nil
			},
			wantErr: ErrSafeDBNotReady,
		},
		{
			name: "deriver still on firstEntry's L1 -> ErrSafeDBNotReady",
			firstEntry: func() (eth.BlockID, eth.BlockID, error) {
				return eth.BlockID{Number: 4}, eth.BlockID{Number: 23}, nil
			},
			syncStatus: func() (*eth.SyncStatus, error) {
				return &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 4}}, nil
			},
			wantErr: ErrSafeDBNotReady,
		},
		{
			name: "deriver below firstEntry's L1 (impossible but defensive) -> ErrSafeDBNotReady",
			firstEntry: func() (eth.BlockID, eth.BlockID, error) {
				return eth.BlockID{Number: 4}, eth.BlockID{Number: 23}, nil
			},
			syncStatus: func() (*eth.SyncStatus, error) {
				return &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 3}}, nil
			},
			wantErr: ErrSafeDBNotReady,
		},
		{
			name: "deriver past firstEntry's L1 -> returns timestamp",
			firstEntry: func() (eth.BlockID, eth.BlockID, error) {
				return eth.BlockID{Number: 4}, eth.BlockID{Number: 23}, nil
			},
			syncStatus: func() (*eth.SyncStatus, error) {
				return &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 5}}, nil
			},
			wantTS: genesisL2Time + 23*blockTime,
		},
		{
			name: "SyncStatus error propagates",
			firstEntry: func() (eth.BlockID, eth.BlockID, error) {
				return eth.BlockID{Number: 4}, eth.BlockID{Number: 23}, nil
			},
			syncStatus: func() (*eth.SyncStatus, error) {
				return nil, errors.New("rpc down")
			},
			wantErr: nil, // checked via Contains below
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			vncfg := createTestVNConfig()
			vncfg.Rollup.Genesis.L2Time = genesisL2Time
			vncfg.Rollup.BlockTime = blockTime
			log := createTestLogger(t)

			mockVN := newMockVirtualNode()
			mockVN.firstSafeHeadEntryOverride = c.firstEntry
			mockVN.syncStatusOverride = c.syncStatus

			impl := &simpleChainContainer{
				chainID: eth.ChainIDFromUInt64(420),
				log:     log,
				vncfg:   vncfg,
				vn:      mockVN,
			}

			ts, err := impl.FirstSafeHeadTimestamp(context.Background())
			if c.wantErr != nil {
				require.ErrorIs(t, err, c.wantErr)
				require.Zero(t, ts)
				return
			}
			if c.name == "SyncStatus error propagates" {
				require.Error(t, err)
				require.Contains(t, err.Error(), "sync status")
				require.Contains(t, err.Error(), "rpc down")
				require.Zero(t, ts)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.wantTS, ts)
		})
	}
}

// SyncStatus must be sampled before FirstSafeHeadEntry; the reverse order
// admits a race where the deriver finishes firstEntry.L1 between reads.
func TestChainContainer_FirstSafeHeadTimestamp_SamplesSyncStatusFirst(t *testing.T) {
	t.Parallel()

	vncfg := createTestVNConfig()
	vncfg.Rollup.Genesis.L2Time = 1000
	vncfg.Rollup.BlockTime = 2
	log := createTestLogger(t)

	mockVN := newMockVirtualNode()
	mockVN.firstSafeHeadEntryOverride = func() (eth.BlockID, eth.BlockID, error) {
		return eth.BlockID{Number: 4}, eth.BlockID{Number: 23}, nil
	}
	mockVN.syncStatusOverride = func() (*eth.SyncStatus, error) {
		return &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 5}}, nil
	}

	impl := &simpleChainContainer{
		chainID: eth.ChainIDFromUInt64(420),
		log:     log,
		vncfg:   vncfg,
		vn:      mockVN,
	}

	_, err := impl.FirstSafeHeadTimestamp(context.Background())
	require.NoError(t, err)

	mockVN.mu.Lock()
	defer mockVN.mu.Unlock()
	require.GreaterOrEqual(t, len(mockVN.methodCalls), 2)
	syncIdx, firstIdx := -1, -1
	for i, name := range mockVN.methodCalls {
		if name == "SyncStatus" && syncIdx == -1 {
			syncIdx = i
		}
		if name == "FirstSafeHeadEntry" && firstIdx == -1 {
			firstIdx = i
		}
	}
	require.NotEqual(t, -1, syncIdx, "SyncStatus should have been called")
	require.NotEqual(t, -1, firstIdx, "FirstSafeHeadEntry should have been called")
	require.Less(t, syncIdx, firstIdx,
		"SyncStatus must be sampled before FirstSafeHeadEntry — the reverse order admits a race; methodCalls=%v", mockVN.methodCalls)
}
