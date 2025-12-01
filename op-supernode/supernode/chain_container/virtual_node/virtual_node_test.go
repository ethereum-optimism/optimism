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
