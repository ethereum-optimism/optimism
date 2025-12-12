package filter

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestBackend_FailsafeEnabled(t *testing.T) {
	backend := newTestBackend(t)

	// Failsafe should be disabled by default
	require.False(t, backend.FailsafeEnabled())

	// Enable failsafe
	backend.SetFailsafeEnabled(true)
	require.True(t, backend.FailsafeEnabled())

	// Disable failsafe
	backend.SetFailsafeEnabled(false)
	require.False(t, backend.FailsafeEnabled())
}

func TestBackend_CheckAccessList_FailsafeEnabled(t *testing.T) {
	backend := newTestBackendWithMockChain(t)

	// Enable failsafe
	backend.SetFailsafeEnabled(true)

	// CheckAccessList should return ErrFailsafeEnabled
	err := backend.CheckAccessList(
		context.Background(),
		[]common.Hash{},
		types.LocalUnsafe,
		types.ExecutingDescriptor{},
	)
	require.ErrorIs(t, err, types.ErrFailsafeEnabled)
}

func TestBackend_CheckAccessList_NotReady(t *testing.T) {
	backend := newTestBackend(t)

	// Backend with no chains is not ready
	err := backend.CheckAccessList(
		context.Background(),
		[]common.Hash{},
		types.LocalUnsafe,
		types.ExecutingDescriptor{},
	)
	require.ErrorIs(t, err, types.ErrUninitialized)
}

func TestBackend_Ready_NoChains(t *testing.T) {
	backend := newTestBackend(t)

	// Backend with no chains should not be ready
	require.False(t, backend.Ready())
}

func TestBackend_Ready_WithChains(t *testing.T) {
	backend := newTestBackendWithMockChain(t)

	// Backend with mock ready chain should be ready
	require.True(t, backend.Ready())
}

func TestBackend_OnReorg_EnablesFailsafe(t *testing.T) {
	backend := newTestBackend(t)

	// Failsafe should be disabled initially
	require.False(t, backend.FailsafeEnabled())

	// Simulate a reorg callback
	backend.onReorg(eth.ChainIDFromUInt64(1))

	// Failsafe should now be enabled
	require.True(t, backend.FailsafeEnabled())
}

func TestBackend_CheckAccessList_UnsupportedSafetyLevel(t *testing.T) {
	backend := newTestBackendWithMockChain(t)

	// CrossUnsafe is not supported
	err := backend.CheckAccessList(
		context.Background(),
		[]common.Hash{},
		types.CrossUnsafe,
		types.ExecutingDescriptor{},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported safety level")
}

// Test helpers

func newTestBackend(t *testing.T) *Backend {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return &Backend{
		log:             log.New(),
		metrics:         metrics.NoopMetrics,
		cfg:             &Config{},
		chains:          make(map[eth.ChainID]*ChainIngester),
		pendingExecMsgs: make(map[eth.ChainID]map[uint64][]*types.ExecutingMessage),
		ctx:             ctx,
		cancel:          cancel,
	}
}

func newTestBackendWithMockChain(t *testing.T) *Backend {
	backend := newTestBackend(t)

	// Add a mock chain ingester that reports as ready
	chainID := eth.ChainIDFromUInt64(1)
	mockIngester := &mockChainIngester{ready: true}
	backend.chains[chainID] = mockIngester.asChainIngester()
	backend.pendingExecMsgs[chainID] = make(map[uint64][]*types.ExecutingMessage)

	return backend
}

// mockChainIngester provides a minimal mock for testing
type mockChainIngester struct {
	ready bool
}

// asChainIngester converts the mock into a real ChainIngester with only the ready field set
// This is a workaround since we can't easily mock the ChainIngester struct
func (m *mockChainIngester) asChainIngester() *ChainIngester {
	c := &ChainIngester{}
	if m.ready {
		c.ready.Store(true)
	}
	return c
}
