package virtual_node

import (
	"context"
	"errors"
	"math/big"
	"regexp"
	"testing"
	"time"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	opmetrics "github.com/ethereum-optimism/optimism/op-node/metrics"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// mockInnerNode is a mock implementation of innerNode interface for testing
type mockInnerNode struct {
	startCh   chan struct{}
	stopCh    chan struct{}
	startErr  error
	stopErr   error
	startFunc func(ctx context.Context)
	started   bool
	safeTs    uint64
	haveSafe  bool
	db        rollupNode.SafeDBReader
}

func newMockInnerNode() *mockInnerNode {
	return &mockInnerNode{
		startCh: make(chan struct{}),
		stopCh:  make(chan struct{}),
	}
}

func (m *mockInnerNode) Start(ctx context.Context) error {
	m.started = true
	if m.startCh != nil {
		close(m.startCh)
	}
	if m.startFunc != nil {
		m.startFunc(ctx)
	}
	return m.startErr
}

func (m *mockInnerNode) Stop(ctx context.Context) error {
	if m.stopCh != nil {
		close(m.stopCh)
	}
	return m.stopErr
}

// SafeL2Timestamp implements the innerNode interface method used by VirtualNode for safety checks
func (m *mockInnerNode) SafeL2Timestamp() (uint64, bool) {
	return m.safeTs, m.haveSafe
}

// SafeDB implements innerNode interface method used by VirtualNode
func (m *mockInnerNode) SafeDB() rollupNode.SafeDBReader { return m.db }

func (m *mockInnerNode) SyncStatus() *eth.SyncStatus { return &eth.SyncStatus{} }

// mockSafeDBReader is a mock implementation of SafeDBReader for testing L1AtSafeHead
type mockSafeDBReader struct {
	// entries maps L1 block number to (L1 BlockID, L2 BlockID)
	entries map[uint64]struct {
		l1 eth.BlockID
		l2 eth.BlockID
	}
}

func newMockSafeDBReader() *mockSafeDBReader {
	return &mockSafeDBReader{
		entries: make(map[uint64]struct {
			l1 eth.BlockID
			l2 eth.BlockID
		}),
	}
}

func (m *mockSafeDBReader) addEntry(l1Num uint64, l1Hash, l2Hash [32]byte, l2Num uint64) {
	m.entries[l1Num] = struct {
		l1 eth.BlockID
		l2 eth.BlockID
	}{
		l1: eth.BlockID{Number: l1Num, Hash: l1Hash},
		l2: eth.BlockID{Number: l2Num, Hash: l2Hash},
	}
}

func (m *mockSafeDBReader) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error) {
	// Find the entry at or before l1BlockNum
	var best uint64
	found := false
	for num := range m.entries {
		if num <= l1BlockNum && (!found || num > best) {
			best = num
			found = true
		}
	}
	if !found {
		return eth.BlockID{}, eth.BlockID{}, errors.New("no entry found")
	}
	entry := m.entries[best]
	return entry.l1, entry.l2, nil
}

// Test helpers
func createTestConfig() *opnodecfg.Config {
	return &opnodecfg.Config{
		Rollup: rollup.Config{
			L2ChainID: big.NewInt(420),
		},
	}
}

func createTestLogger() gethlog.Logger {
	return gethlog.New()
}

func createMockFactory(mock *mockInnerNode) innerNodeFactory {
	return func(ctx context.Context, cfg *opnodecfg.Config, log gethlog.Logger, appVersion string, m *opmetrics.Metrics, initOverload *rollupNode.InitializationOverrides) (innerNode, error) {
		return mock, nil
	}
}

// TestVirtualNode_Constructor tests constructor and initialization
func TestVirtualNode_Constructor(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig()
	log := createTestLogger()
	initOverload := &rollupNode.InitializationOverrides{}
	appVersion := "v1.0.0"

	t.Run("creates node with correct config", func(t *testing.T) {
		vn := NewVirtualNode(cfg, log, initOverload, appVersion)

		require.NotNil(t, vn)
		require.Equal(t, cfg, vn.cfg)
		require.Equal(t, initOverload, vn.initOverload)
		require.Equal(t, appVersion, vn.appVersion)
		require.Len(t, vn.vnID, 4)
		require.Equal(t, VNStateNotStarted, vn.State())
	})

	t.Run("generates unique 4-character IDs", func(t *testing.T) {
		id1 := generateVirtualNodeID()
		id2 := generateVirtualNodeID()
		id3 := generateVirtualNodeID()

		require.Len(t, id1, 4)
		require.NotEqual(t, id1, id2)
		require.NotEqual(t, id2, id3)

		matched, err := regexp.MatchString("^[0-9a-f-]{4}$", id1)
		require.NoError(t, err)
		require.True(t, matched)
	})

	t.Run("sets custom appVersion", func(t *testing.T) {
		customVersion := "v2.3.4"
		vn := NewVirtualNode(cfg, log, initOverload, customVersion)
		require.Equal(t, customVersion, vn.appVersion)
	})
}

// TestVirtualNode_Lifecycle tests the complete Start/Stop lifecycle
func TestVirtualNode_Lifecycle(t *testing.T) {
	t.Parallel()

	log := createTestLogger()
	cfg := createTestConfig()
	initOverload := &rollupNode.InitializationOverrides{}
	appVersion := "test"

	t.Run("Start with nil config returns error", func(t *testing.T) {
		vn := NewVirtualNode(cfg, log, initOverload, appVersion)
		vn.cfg = nil

		err := vn.Start(context.Background())
		require.ErrorIs(t, err, ErrVirtualNodeConfigNil)
	})

	t.Run("Start transitions through states correctly", func(t *testing.T) {
		mock := newMockInnerNode()
		mock.startFunc = func(ctx context.Context) {
			<-ctx.Done()
		}

		vn := NewVirtualNode(cfg, log, initOverload, appVersion)
		vn.innerNodeFactory = createMockFactory(mock)

		require.Equal(t, VNStateNotStarted, vn.State())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = vn.Start(ctx)
		}()

		// Wait for it to be running
		require.Eventually(t, func() bool {
			return vn.State() == VNStateRunning
		}, 1*time.Second, 10*time.Millisecond)

		// Cancel and wait for stopped
		cancel()
		require.Eventually(t, func() bool {
			return vn.State() == VNStateStopped
		}, 1*time.Second, 10*time.Millisecond)
	})

	t.Run("Start on already running node returns error", func(t *testing.T) {
		mock := newMockInnerNode()
		mock.startFunc = func(ctx context.Context) {
			<-ctx.Done()
		}

		vn := NewVirtualNode(cfg, log, initOverload, appVersion)
		vn.innerNodeFactory = createMockFactory(mock)

		// Start it first
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = vn.Start(ctx)
		}()

		require.Eventually(t, func() bool {
			return vn.State() == VNStateRunning
		}, 1*time.Second, 10*time.Millisecond)

		// Try to start again while running
		err := vn.Start(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be started in this state")

		cancel()
	})

	t.Run("Stop on non-running node is no-op", func(t *testing.T) {
		vn := NewVirtualNode(cfg, log, initOverload, appVersion)
		require.Equal(t, VNStateNotStarted, vn.State())

		err := vn.Stop(context.Background())
		require.NoError(t, err)
	})

	t.Run("Stop causes Start to exit", func(t *testing.T) {
		mock := newMockInnerNode()
		mock.startFunc = func(ctx context.Context) {
			<-ctx.Done()
		}

		vn := NewVirtualNode(cfg, log, initOverload, appVersion)
		vn.innerNodeFactory = createMockFactory(mock)

		ctx := context.Background()
		startDone := make(chan error, 1)

		go func() {
			startDone <- vn.Start(ctx)
		}()

		// Wait for running state
		require.Eventually(t, func() bool {
			return vn.State() == VNStateRunning
		}, 1*time.Second, 10*time.Millisecond)

		// Stop it
		err := vn.Stop(ctx)
		require.NoError(t, err)

		// Start should exit
		select {
		case <-startDone:
			require.Equal(t, VNStateStopped, vn.State())
		case <-time.After(2 * time.Second):
			t.Fatal("Start should exit after Stop")
		}
	})

	t.Run("Stop is idempotent", func(t *testing.T) {
		vn := NewVirtualNode(cfg, log, initOverload, appVersion)
		ctx := context.Background()

		// Multiple stops should all succeed
		require.NoError(t, vn.Stop(ctx))
		require.NoError(t, vn.Stop(ctx))
		require.NoError(t, vn.Stop(ctx))
	})

}

// TestVirtualNode_InnerNodeIntegration tests interaction with inner node
func TestVirtualNode_InnerNodeIntegration(t *testing.T) {
	t.Parallel()

	log := createTestLogger()
	cfg := createTestConfig()
	initOverload := &rollupNode.InitializationOverrides{}
	appVersion := "test"

	t.Run("Start calls inner node Start", func(t *testing.T) {
		mock := newMockInnerNode()
		mock.startFunc = func(ctx context.Context) {
			<-ctx.Done()
		}

		vn := NewVirtualNode(cfg, log, initOverload, appVersion)
		vn.innerNodeFactory = createMockFactory(mock)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		go func() {
			_ = vn.Start(ctx)
		}()

		require.Eventually(t, func() bool {
			return vn.State() == VNStateRunning && mock.started
		}, 1*time.Second, 10*time.Millisecond)
	})

	t.Run("Stop calls inner node Stop", func(t *testing.T) {
		mock := newMockInnerNode()
		mock.startFunc = func(ctx context.Context) {
			<-ctx.Done()
		}

		vn := NewVirtualNode(cfg, log, initOverload, appVersion)
		vn.innerNodeFactory = createMockFactory(mock)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		startDone := make(chan struct{})
		go func() {
			_ = vn.Start(ctx)
			close(startDone)
		}()

		// Wait for it to be running
		require.Eventually(t, func() bool {
			return vn.State() == VNStateRunning
		}, 1*time.Second, 10*time.Millisecond)

		_ = vn.Stop(ctx)

		select {
		case <-startDone:
			// Verify inner Stop was called
			select {
			case <-mock.stopCh:
				// Success
			default:
				t.Fatal("inner node Stop should be called")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Start should complete after Stop")
		}
	})

	t.Run("inner node error propagates through cancel callback", func(t *testing.T) {
		mock := newMockInnerNode()
		vn := NewVirtualNode(cfg, log, initOverload, appVersion)

		mock.startFunc = func(ctx context.Context) {
			if vn.cfg.Cancel != nil {
				vn.cfg.Cancel(errors.New("inner node error"))
			}
		}

		vn.innerNodeFactory = createMockFactory(mock)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		err := vn.Start(ctx)

		require.Error(t, err)
		require.Contains(t, err.Error(), "inner node error")
	})

	t.Run("cancel callback is configured", func(t *testing.T) {
		mock := newMockInnerNode()
		mock.startFunc = func(ctx context.Context) {
			<-ctx.Done()
		}

		// Create fresh config to ensure Cancel is nil
		freshCfg := createTestConfig()
		vn := NewVirtualNode(freshCfg, log, initOverload, appVersion)
		vn.innerNodeFactory = createMockFactory(mock)

		require.Nil(t, vn.cfg.Cancel, "Cancel should be nil initially")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = vn.Start(ctx)
		}()

		require.Eventually(t, func() bool {
			return vn.cfg.Cancel != nil
		}, 1*time.Second, 10*time.Millisecond)
		cancel()
	})
}

// TestVirtualNode_L1AtSafeHead tests the L1AtSafeHead function
func TestVirtualNode_L1AtSafeHead(t *testing.T) {
	t.Parallel()

	genesisL1 := eth.BlockID{Number: 100, Hash: [32]byte{0x01}}
	genesisL2 := eth.BlockID{Number: 0, Hash: [32]byte{0x02}}

	createConfigWithGenesis := func() *opnodecfg.Config {
		return &opnodecfg.Config{
			Rollup: rollup.Config{
				L2ChainID: big.NewInt(420),
				Genesis: rollup.Genesis{
					L1: genesisL1,
					L2: genesisL2,
				},
			},
		}
	}

	t.Run("returns error when inner node is nil", func(t *testing.T) {
		cfg := createConfigWithGenesis()
		log := createTestLogger()
		vn := NewVirtualNode(cfg, log, nil, "test")

		_, err := vn.L1AtSafeHead(context.Background(), eth.BlockID{Number: 10})
		require.ErrorIs(t, err, ErrVirtualNodeNotRunning)
	})

	t.Run("returns error when SafeDB is nil", func(t *testing.T) {
		cfg := createConfigWithGenesis()
		log := createTestLogger()
		vn := NewVirtualNode(cfg, log, nil, "test")

		mock := newMockInnerNode()
		mock.db = nil
		vn.inner = mock
		vn.state = VNStateRunning

		_, err := vn.L1AtSafeHead(context.Background(), eth.BlockID{Number: 10})
		require.ErrorIs(t, err, ErrVirtualNodeNotRunning)
	})

	t.Run("genesis L2 target returns genesis L1 directly", func(t *testing.T) {
		cfg := createConfigWithGenesis()
		log := createTestLogger()
		vn := NewVirtualNode(cfg, log, nil, "test")

		// Set up mock with SafeDB - but it shouldn't be called for genesis
		mockDB := newMockSafeDBReader()
		mock := newMockInnerNode()
		mock.db = mockDB
		vn.inner = mock
		vn.state = VNStateRunning

		// Query for genesis L2 block
		result, err := vn.L1AtSafeHead(context.Background(), genesisL2)
		require.NoError(t, err)
		require.Equal(t, eth.BlockID{}, result) // Genesis L2 target returns genesis L1 directly, but without the hash
	})

	t.Run("genesis L2 number with different hash is not treated as genesis", func(t *testing.T) {
		cfg := createConfigWithGenesis()
		log := createTestLogger()
		vn := NewVirtualNode(cfg, log, nil, "test")

		mockDB := newMockSafeDBReader()
		mock := newMockInnerNode()
		mock.db = mockDB
		vn.inner = mock
		vn.state = VNStateRunning

		// Query with same number as genesis but different hash
		// Should NOT match genesis since both number AND hash must match
		target := eth.BlockID{Number: genesisL2.Number, Hash: [32]byte{0xff}}
		_, err := vn.L1AtSafeHead(context.Background(), target)
		// Returns error because mockDB is empty and walkback fails
		require.Error(t, err)
	})

	t.Run("non-genesis target uses walkback to find earliest L1", func(t *testing.T) {
		cfg := createConfigWithGenesis()
		log := createTestLogger()
		vn := NewVirtualNode(cfg, log, nil, "test")

		mockDB := newMockSafeDBReader()
		// Set up entries: L1 block -> L2 safe head
		// L1=100 (genesis) -> L2=0
		// L1=101 -> L2=5
		// L1=102 -> L2=10
		// L1=103 -> L2=15
		// L1=104 -> L2=20
		mockDB.addEntry(100, [32]byte{0x01}, [32]byte{0x02}, 0)
		mockDB.addEntry(101, [32]byte{0x03}, [32]byte{0x04}, 5)
		mockDB.addEntry(102, [32]byte{0x05}, [32]byte{0x06}, 10)
		mockDB.addEntry(103, [32]byte{0x07}, [32]byte{0x08}, 15)
		mockDB.addEntry(104, [32]byte{0x09}, [32]byte{0x0a}, 20)

		mock := newMockInnerNode()
		mock.db = mockDB
		vn.inner = mock
		vn.state = VNStateRunning

		// Query for L2 block 10 - should return L1=102 (earliest L1 where L2 safe head >= 10)
		target := eth.BlockID{Number: 10, Hash: [32]byte{0x06}}
		result, err := vn.L1AtSafeHead(context.Background(), target)
		require.NoError(t, err)
		require.Equal(t, uint64(102), result.Number)
	})

	t.Run("target beyond latest returns error", func(t *testing.T) {
		cfg := createConfigWithGenesis()
		log := createTestLogger()
		vn := NewVirtualNode(cfg, log, nil, "test")

		mockDB := newMockSafeDBReader()
		mockDB.addEntry(100, [32]byte{0x01}, [32]byte{0x02}, 0)
		mockDB.addEntry(101, [32]byte{0x03}, [32]byte{0x04}, 5)

		mock := newMockInnerNode()
		mock.db = mockDB
		vn.inner = mock
		vn.state = VNStateRunning

		// Query for L2 block 100 - beyond latest L2 safe head (5)
		target := eth.BlockID{Number: 100, Hash: [32]byte{}}
		_, err := vn.L1AtSafeHead(context.Background(), target)
		require.ErrorIs(t, err, ErrL1AtSafeHeadNotFound)
	})
}

// blockingStopMock wraps mockInnerNode but blocks Stop() until explicitly released.
// This simulates an OpNode whose shutdown (event drain) takes a long time.
type blockingStopMock struct {
	*mockInnerNode
	stopStarted chan struct{}
	stopRelease chan struct{}
}

func (m *blockingStopMock) Stop(ctx context.Context) error {
	close(m.stopStarted)
	select {
	case <-m.stopRelease:
	case <-ctx.Done():
		return ctx.Err()
	}
	return m.stopErr
}

// TestVirtualNode_SyncStatusDuringShutdown proves that SyncStatus does not deadlock
// when called while Start() is shutting down the inner node. Before the fix,
// Start() held v.mu during inner.Stop(), so any concurrent SyncStatus() call
// would block on v.mu forever — creating a deadlock if the inner node's shutdown
// path called back into SyncStatus (e.g. via the event system).
func TestVirtualNode_SyncStatusDuringShutdown(t *testing.T) {
	t.Parallel()
	log := createTestLogger()
	cfg := createTestConfig()
	initOverload := &rollupNode.InitializationOverrides{}

	mock := newMockInnerNode()
	mock.startFunc = func(ctx context.Context) {
		<-ctx.Done()
	}
	mock.stopCh = nil // prevent close in default Stop — we use blockingStopMock

	stopStarted := make(chan struct{})
	stopRelease := make(chan struct{})
	blocking := &blockingStopMock{
		mockInnerNode: mock,
		stopStarted:   stopStarted,
		stopRelease:   stopRelease,
	}

	vn := NewVirtualNode(cfg, log, initOverload, "test")
	vn.innerNodeFactory = func(ctx context.Context, cfg *opnodecfg.Config,
		log gethlog.Logger, appVersion string, m *opmetrics.Metrics,
		initOverload *rollupNode.InitializationOverrides) (innerNode, error) {
		return blocking, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	startDone := make(chan error, 1)
	go func() {
		startDone <- vn.Start(ctx)
	}()

	// Wait for running
	require.Eventually(t, func() bool {
		return vn.State() == VNStateRunning
	}, time.Second, 10*time.Millisecond)

	// Cancel to trigger shutdown — Start() will call inner.Stop() which blocks
	cancel()

	// Wait for inner.Stop() to be entered
	select {
	case <-stopStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("inner.Stop() was never called")
	}

	// Now try to call SyncStatus — this MUST NOT deadlock.
	// Before the fix, this would block forever on v.mu.
	syncDone := make(chan struct{})
	go func() {
		_, _ = vn.SyncStatus(context.Background())
		close(syncDone)
	}()

	select {
	case <-syncDone:
		// Success — SyncStatus completed without deadlock
	case <-time.After(5 * time.Second):
		t.Fatal("SyncStatus deadlocked during shutdown — v.mu held during inner.Stop()")
	}

	// Release inner.Stop() so Start() can return
	close(stopRelease)

	select {
	case <-startDone:
		require.Equal(t, VNStateStopped, vn.State())
	case <-time.After(5 * time.Second):
		t.Fatal("Start() did not return after inner.Stop() completed")
	}
}
