package interop

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/reads"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Harness
// =============================================================================

// interopTestHarness provides a builder-pattern test setup for Interop tests.
// It reduces boilerplate by handling common setup: temp directories, mock chains,
// interop creation, context assignment, and cleanup.
type interopTestHarness struct {
	t              *testing.T
	interop        *Interop
	mocks          map[eth.ChainID]*mockChainContainer
	activationTime uint64
	dataDir        string
	skipBuild      bool // for tests that need custom construction
}

// newInteropTestHarness creates a new test harness with sensible defaults.
func newInteropTestHarness(t *testing.T) *interopTestHarness {
	t.Helper()
	t.Parallel()
	return &interopTestHarness{
		t:              t,
		mocks:          make(map[eth.ChainID]*mockChainContainer),
		activationTime: 1000,
		dataDir:        t.TempDir(),
	}
}

// WithActivation sets the interop activation timestamp.
func (h *interopTestHarness) WithActivation(ts uint64) *interopTestHarness {
	h.activationTime = ts
	return h
}

// WithDataDir sets a custom data directory (useful for error testing).
func (h *interopTestHarness) WithDataDir(dir string) *interopTestHarness {
	h.dataDir = dir
	return h
}

// WithChain adds a mock chain container with optional configuration.
func (h *interopTestHarness) WithChain(id uint64, configure func(*mockChainContainer)) *interopTestHarness {
	mock := newMockChainContainer(id)
	if configure != nil {
		configure(mock)
	}
	h.mocks[mock.id] = mock
	return h
}

// SkipBuild marks that Build() should not create an Interop instance.
// Useful for tests that need to test New() directly.
func (h *interopTestHarness) SkipBuild() *interopTestHarness {
	h.skipBuild = true
	return h
}

// Build creates the Interop instance from configured mocks.
// Sets up context and registers cleanup.
func (h *interopTestHarness) Build() *interopTestHarness {
	if h.skipBuild {
		return h
	}
	chains := make(map[eth.ChainID]cc.ChainContainer)
	for id, mock := range h.mocks {
		chains[id] = mock
	}
	h.interop = New(testLogger(), h.activationTime, chains, h.dataDir, nil)
	if h.interop != nil {
		h.interop.ctx = context.Background()
		h.t.Cleanup(func() { _ = h.interop.Stop(context.Background()) })
	}
	return h
}

// Chains returns the map of chain containers for use with New().
func (h *interopTestHarness) Chains() map[eth.ChainID]cc.ChainContainer {
	chains := make(map[eth.ChainID]cc.ChainContainer)
	for id, mock := range h.mocks {
		chains[id] = mock
	}
	return chains
}

// Mock returns the mock for a given chain ID.
func (h *interopTestHarness) Mock(id uint64) *mockChainContainer {
	return h.mocks[eth.ChainIDFromUInt64(id)]
}

// =============================================================================
// TestNew
// =============================================================================

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopTestHarness) *interopTestHarness
		run   func(t *testing.T, h *interopTestHarness)
	}{
		{
			name: "valid inputs initializes all components",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, nil).WithChain(8453, nil).SkipBuild()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				interop := New(testLogger(), h.activationTime, h.Chains(), h.dataDir, nil)
				require.NotNil(t, interop)
				t.Cleanup(func() { _ = interop.Stop(context.Background()) })

				require.Equal(t, uint64(1000), interop.activationTimestamp)
				require.NotNil(t, interop.verifiedDB)
				require.Len(t, interop.chains, 2)
				require.Len(t, interop.logsDBs, 2)
				require.NotNil(t, interop.verifyFn)
				require.NotNil(t, interop.cycleVerifyFn)

				for chainID := range h.Chains() {
					require.Contains(t, interop.logsDBs, chainID)
					require.NotNil(t, interop.logsDBs[chainID])
				}
			},
		},
		{
			name: "invalid dataDir returns nil",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithDataDir("/nonexistent/path").SkipBuild()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				interop := New(testLogger(), h.activationTime, h.Chains(), h.dataDir, nil)
				require.Nil(t, interop)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropTestHarness(t)
			tc.setup(h)
			tc.run(t, h)
		})
	}
}

// =============================================================================
// TestStartStop
// =============================================================================

func TestStartStop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopTestHarness) *interopTestHarness
		run   func(t *testing.T, h *interopTestHarness)
	}{
		{
			name: "Start blocks until context cancelled",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
					m.blockAtTimestamp = eth.L2BlockRef{Number: 50}
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				ctx, cancel := context.WithCancel(context.Background())
				done := make(chan error, 1)
				go func() { done <- h.interop.Start(ctx) }()

				require.Eventually(t, func() bool {
					h.interop.mu.RLock()
					defer h.interop.mu.RUnlock()
					return h.interop.started
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
			},
		},
		{
			name: "double Start blocked",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				go func() { _ = h.interop.Start(ctx) }()

				require.Eventually(t, func() bool {
					h.interop.mu.RLock()
					defer h.interop.mu.RUnlock()
					return h.interop.started
				}, 5*time.Second, 100*time.Millisecond)

				ctx2, cancel2 := context.WithTimeout(context.Background(), 500*time.Millisecond)
				defer cancel2()

				err := h.interop.Start(ctx2)
				require.ErrorIs(t, err, context.DeadlineExceeded)
			},
		},
		{
			name: "Stop cancels running Start and closes DB",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
					m.blockAtTimestampErr = ethereum.NotFound
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				done := make(chan error, 1)
				go func() { done <- h.interop.Start(context.Background()) }()

				require.Eventually(t, func() bool {
					h.interop.mu.RLock()
					defer h.interop.mu.RUnlock()
					return h.interop.started
				}, 5*time.Second, 100*time.Millisecond)

				err := h.interop.Stop(context.Background())
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
				_, err = h.interop.verifiedDB.Has(100)
				require.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropTestHarness(t)
			tc.setup(h)
			tc.run(t, h)
		})
	}
}

// =============================================================================
// TestCollectCurrentL1
// =============================================================================

func TestCollectCurrentL1(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(h *interopTestHarness) *interopTestHarness
		assert func(t *testing.T, l1 eth.BlockID, err error)
	}{
		{
			name: "returns minimum L1 across multiple chains",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
				}).WithChain(8453, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")} // minimum
				}).Build()
			},
			assert: func(t *testing.T, l1 eth.BlockID, err error) {
				require.NoError(t, err)
				require.Equal(t, uint64(100), l1.Number)
				require.Equal(t, common.HexToHash("0x1"), l1.Hash)
			},
		},
		{
			name: "single chain returns its L1",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 500, Hash: common.HexToHash("0x5")}
				}).Build()
			},
			assert: func(t *testing.T, l1 eth.BlockID, err error) {
				require.NoError(t, err)
				require.Equal(t, uint64(500), l1.Number)
			},
		},
		{
			name: "chain error propagated",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1Err = errors.New("chain not synced")
				}).Build()
			},
			assert: func(t *testing.T, l1 eth.BlockID, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "not ready")
				require.Equal(t, eth.BlockID{}, l1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropTestHarness(t)
			tc.setup(h)
			l1, err := h.interop.collectCurrentL1()
			tc.assert(t, l1, err)
		})
	}
}

// =============================================================================
// TestCheckChainsReady
// =============================================================================

func TestCheckChainsReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(h *interopTestHarness) *interopTestHarness
		assert func(t *testing.T, h *interopTestHarness, ready chainsReadyResult, err error)
	}{
		{
			name: "all chains ready returns blocks",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).WithChain(8453, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
				}).Build()
			},
			assert: func(t *testing.T, h *interopTestHarness, ready chainsReadyResult, err error) {
				require.NoError(t, err)
				require.Len(t, ready.blocks, 2)
				require.NotEqual(t, common.Hash{}, ready.blocks[h.Mock(10).id].Hash)
				require.NotEqual(t, common.Hash{}, ready.blocks[h.Mock(8453).id].Hash)
			},
		},
		{
			name: "one chain not ready returns error",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100}
				}).WithChain(8453, func(m *mockChainContainer) {
					m.blockAtTimestampErr = ethereum.NotFound
				}).Build()
			},
			assert: func(t *testing.T, h *interopTestHarness, ready chainsReadyResult, err error) {
				require.Error(t, err)
				require.Nil(t, ready.blocks)
			},
		},
		{
			name: "parallel execution works",
			setup: func(h *interopTestHarness) *interopTestHarness {
				for i := 0; i < 5; i++ {
					idx := i // capture loop var
					h.WithChain(uint64(10+idx), func(m *mockChainContainer) {
						m.blockAtTimestamp = eth.L2BlockRef{Number: uint64(100 + idx)}
					})
				}
				return h.Build()
			},
			assert: func(t *testing.T, h *interopTestHarness, ready chainsReadyResult, err error) {
				require.NoError(t, err)
				require.Len(t, ready.blocks, 5)
			},
		},
		{
			// Verify that checkChainsReady drains ALL goroutine results before returning,
			// even when one chain errors early. Without the drain, the slow chain's goroutine
			// would still be running concurrently when the next call spawns a new batch —
			// causing goroutine accumulation under repeated retries.
			name: "drains all goroutine results before returning on error",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					// Errors immediately, causing an early-return path.
					m.blockAtTimestampErr = ethereum.NotFound
				}).WithChain(8453, func(m *mockChainContainer) {
					// Slow chain: takes longer than the fast-error chain.
					// After checkChainsReady returns, callsCompleted must be 1,
					// proving the function waited for this goroutine to finish.
					m.blockAtTimestamp = eth.L2BlockRef{Number: 200}
					m.blockAtTimestampDelay = 30 * time.Millisecond
				}).Build()
			},
			assert: func(t *testing.T, h *interopTestHarness, ready chainsReadyResult, err error) {
				require.Error(t, err)
				require.Nil(t, ready.blocks)
				// Both goroutines must have completed before checkChainsReady returned.
				require.EqualValues(t, 1, h.Mock(10).callsCompleted.Load(), "chain 10 goroutine should have completed")
				require.EqualValues(t, 1, h.Mock(8453).callsCompleted.Load(), "chain 8453 goroutine should have completed before return")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropTestHarness(t)
			tc.setup(h)
			ready, err := h.interop.checkChainsReady(1000)
			tc.assert(t, h, ready, err)
		})
	}
}

// =============================================================================
// TestProgressInterop
// =============================================================================

func TestProgressInterop(t *testing.T) {
	t.Parallel()

	// Default verifyFn that passes through
	passThroughVerifyFn := func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
		return Result{Timestamp: ts, L1Inclusion: eth.BlockID{Number: 100}, L2Heads: blocks}, nil
	}

	tests := []struct {
		name     string
		setup    func(h *interopTestHarness) *interopTestHarness
		verifyFn func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error)
		assert   func(t *testing.T, result Result, err error)
		run      func(t *testing.T, h *interopTestHarness) // override for complex cases
	}{
		{
			name: "not initialized uses activation timestamp",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithActivation(5000).WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).Build()
			},
			verifyFn: passThroughVerifyFn,
			assert: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				require.Equal(t, uint64(5000), result.Timestamp)
			},
		},
		{
			name: "initialized uses next timestamp",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				h.interop.verifyFn = passThroughVerifyFn

				// First progress
				result1, err := progressInteropCompat(h.interop)
				require.NoError(t, err)
				require.Equal(t, uint64(1000), result1.Timestamp)

				// Commit
				err = applyResultCompat(h.interop, result1)
				require.NoError(t, err)

				// Second progress should use next timestamp
				result2, err := progressInteropCompat(h.interop)
				require.NoError(t, err)
				require.Equal(t, uint64(1001), result2.Timestamp)
			},
		},
		{
			name: "chains not ready returns empty result",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestampErr = ethereum.NotFound
				}).Build()
			},
			assert: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				require.True(t, result.IsEmpty())
			},
		},
		{
			name: "chain error propagated",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestampErr = errors.New("internal error")
				}).Build()
			},
			assert: func(t *testing.T, result Result, err error) {
				require.Error(t, err)
				require.True(t, result.IsEmpty())
			},
		},
		{
			name: "verifyFn error propagated",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithActivation(100).WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 1000, Hash: common.HexToHash("0xL1")}
					m.blockAtTimestamp = eth.L2BlockRef{Number: 500, Hash: common.HexToHash("0xL2")}
				}).Build()
			},
			verifyFn: func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
				return Result{}, errors.New("verification failed")
			},
			assert: func(t *testing.T, result Result, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "verification failed")
				require.True(t, result.IsEmpty())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropTestHarness(t)
			tc.setup(h)
			if tc.run != nil {
				tc.run(t, h)
				return
			}
			if tc.verifyFn != nil {
				h.interop.verifyFn = tc.verifyFn
			}
			result, err := progressInteropCompat(h.interop)
			tc.assert(t, result, err)
		})
	}
}

// =============================================================================
// TestProgressInteropWithCycleVerify
// =============================================================================

func TestProgressInteropWithCycleVerify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopTestHarness) *interopTestHarness
		run   func(t *testing.T, h *interopTestHarness)
	}{
		{
			name: "default cycleVerifyFn returns valid result",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				// Set verifyFn to return a valid result
				h.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
					return Result{Timestamp: ts, L2Heads: blocks}, nil
				}
				// cycleVerifyFn is overridden with this stub implementation.

				result, err := progressInteropCompat(h.interop)
				require.NoError(t, err)
				require.False(t, result.IsEmpty())
				require.True(t, result.IsValid())
			},
		},
		{
			name: "cycleVerifyFn called after verifyFn and results merged",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).WithChain(8453, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				verifyFnCalled := false
				cycleVerifyFnCalled := false
				chain10 := eth.ChainIDFromUInt64(10)
				chain8453 := eth.ChainIDFromUInt64(8453)

				// verifyFn returns valid result
				h.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
					verifyFnCalled = true
					return Result{Timestamp: ts, L2Heads: blocks}, nil
				}

				// cycleVerifyFn marks chain 8453 as invalid
				h.interop.cycleVerifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
					require.True(t, verifyFnCalled, "verifyFn should be called before cycleVerifyFn")
					cycleVerifyFnCalled = true
					return Result{
						Timestamp: ts,
						L2Heads:   blocks,
						InvalidHeads: map[eth.ChainID]eth.BlockID{
							chain8453: blocks[chain8453],
						},
					}, nil
				}

				result, err := progressInteropCompat(h.interop)
				require.NoError(t, err)
				require.True(t, verifyFnCalled, "verifyFn should be called")
				require.True(t, cycleVerifyFnCalled, "cycleVerifyFn should be called")
				require.False(t, result.IsValid(), "result should be invalid due to cycleVerifyFn")
				require.Contains(t, result.InvalidHeads, chain8453)
				require.NotContains(t, result.InvalidHeads, chain10)
			},
		},
		{
			name: "cycleVerifyFn error propagated",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				h.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
					return Result{Timestamp: ts, L2Heads: blocks}, nil
				}
				h.interop.cycleVerifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
					return Result{}, errors.New("cycle verification failed")
				}

				result, err := progressInteropCompat(h.interop)
				require.Error(t, err)
				require.Contains(t, err.Error(), "cycle verification")
				require.True(t, result.IsEmpty())
			},
		},
		{
			name: "both verifyFn and cycleVerifyFn invalid heads are merged",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).WithChain(8453, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				chain10 := eth.ChainIDFromUInt64(10)
				chain8453 := eth.ChainIDFromUInt64(8453)

				// verifyFn marks chain 10 as invalid
				h.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
					return Result{
						Timestamp: ts,
						L2Heads:   blocks,
						InvalidHeads: map[eth.ChainID]eth.BlockID{
							chain10: blocks[chain10],
						},
					}, nil
				}

				// cycleVerifyFn marks chain 8453 as invalid
				h.interop.cycleVerifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
					return Result{
						Timestamp: ts,
						L2Heads:   blocks,
						InvalidHeads: map[eth.ChainID]eth.BlockID{
							chain8453: blocks[chain8453],
						},
					}, nil
				}

				result, err := progressInteropCompat(h.interop)
				require.NoError(t, err)
				require.False(t, result.IsValid())
				// Both chains should be in InvalidHeads
				require.Contains(t, result.InvalidHeads, chain10, "chain10 from verifyFn should be invalid")
				require.Contains(t, result.InvalidHeads, chain8453, "chain8453 from cycleVerifyFn should be invalid")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropTestHarness(t)
			tc.setup(h)
			tc.run(t, h)
		})
	}
}

// =============================================================================
// TestVerifiedAtTimestamp
// =============================================================================

func TestVerifiedAtTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopTestHarness) *interopTestHarness
		run   func(t *testing.T, h *interopTestHarness)
	}{
		{
			name: "before activation always verified",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				verified, err := h.interop.VerifiedAtTimestamp(999)
				require.NoError(t, err)
				require.True(t, verified)

				verified, err = h.interop.VerifiedAtTimestamp(0)
				require.NoError(t, err)
				require.True(t, verified)
			},
		},
		{
			name: "at/after activation not verified until committed",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				verified, err := h.interop.VerifiedAtTimestamp(1000)
				require.NoError(t, err)
				require.False(t, verified)

				verified, err = h.interop.VerifiedAtTimestamp(9999)
				require.NoError(t, err)
				require.False(t, verified)
			},
		},
		{
			name: "committed timestamp verified",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100}
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				h.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
					return Result{Timestamp: ts, L1Inclusion: eth.BlockID{Number: 100}, L2Heads: blocks}, nil
				}

				result, err := progressInteropCompat(h.interop)
				require.NoError(t, err)

				err = applyResultCompat(h.interop, result)
				require.NoError(t, err)

				verified, err := h.interop.VerifiedAtTimestamp(1000)
				require.NoError(t, err)
				require.True(t, verified)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropTestHarness(t)
			tc.setup(h)
			tc.run(t, h)
		})
	}
}

// =============================================================================
// TestApplyResultCompat
// =============================================================================

func TestApplyResultCompat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopTestHarness) *interopTestHarness
		run   func(t *testing.T, h *interopTestHarness)
	}{
		{
			name: "empty result is no-op",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				err := applyResultCompat(h.interop, Result{})
				require.NoError(t, err)

				has, err := h.interop.verifiedDB.Has(0)
				require.NoError(t, err)
				require.False(t, has)
			},
		},
		{
			name: "valid result commits to DB with correct data",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, nil).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				mock := h.Mock(10)
				validResult := Result{
					Timestamp:   1000,
					L1Inclusion: eth.BlockID{Number: 100, Hash: common.HexToHash("0xL1")},
					L2Heads: map[eth.ChainID]eth.BlockID{
						mock.id: {Number: 500, Hash: common.HexToHash("0xL2")},
					},
				}

				err := applyResultCompat(h.interop, validResult)
				require.NoError(t, err)

				has, err := h.interop.verifiedDB.Has(1000)
				require.NoError(t, err)
				require.True(t, has)

				retrieved, err := h.interop.verifiedDB.Get(1000)
				require.NoError(t, err)
				require.Equal(t, validResult.Timestamp, retrieved.Timestamp)
				require.Equal(t, validResult.L1Inclusion, retrieved.L1Inclusion)
				require.Equal(t, validResult.L2Heads[mock.id], retrieved.L2Heads[mock.id])
			},
		},
		{
			name: "invalid result does not commit",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, nil).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				mock := h.Mock(10)
				invalidResult := Result{
					Timestamp:   1000,
					L1Inclusion: eth.BlockID{Number: 100, Hash: common.HexToHash("0xL1")},
					L2Heads: map[eth.ChainID]eth.BlockID{
						mock.id: {Number: 500, Hash: common.HexToHash("0xL2")},
					},
					InvalidHeads: map[eth.ChainID]eth.BlockID{
						mock.id: {Number: 500, Hash: common.HexToHash("0xBAD")},
					},
				}

				err := applyResultCompat(h.interop, invalidResult)
				require.NoError(t, err)

				has, err := h.interop.verifiedDB.Has(1000)
				require.NoError(t, err)
				require.False(t, has)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropTestHarness(t)
			tc.setup(h)
			tc.run(t, h)
		})
	}
}

// =============================================================================
// TestInvalidateBlock
// =============================================================================

// TestInvalidateBlock verifies the invalidateBlock method correctly calls
// ChainContainer.InvalidateBlock with the right parameters and handles errors.
func TestInvalidateBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopTestHarness) *interopTestHarness
		run   func(t *testing.T, h *interopTestHarness)
	}{
		{
			name: "calls chain.InvalidateBlock with correct args",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, nil).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				mock := h.Mock(10)
				blockID := eth.BlockID{Number: 500, Hash: common.HexToHash("0xBAD")}
				err := h.interop.invalidateBlock(mock.id, blockID, 0)
				require.NoError(t, err)

				require.Len(t, mock.invalidateBlockCalls, 1)
				require.Equal(t, uint64(500), mock.invalidateBlockCalls[0].height)
				require.Equal(t, common.HexToHash("0xBAD"), mock.invalidateBlockCalls[0].payloadHash)
			},
		},
		{
			name: "returns error when chain not found",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, nil).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				mock := h.Mock(10)
				unknownChain := eth.ChainIDFromUInt64(999)
				blockID := eth.BlockID{Number: 500, Hash: common.HexToHash("0xBAD")}
				err := h.interop.invalidateBlock(unknownChain, blockID, 0)

				require.Error(t, err)
				require.Contains(t, err.Error(), "not found")
				require.Len(t, mock.invalidateBlockCalls, 0)
			},
		},
		{
			name: "returns error when chain.InvalidateBlock fails",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.invalidateBlockErr = errors.New("engine failure")
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				mock := h.Mock(10)
				blockID := eth.BlockID{Number: 500, Hash: common.HexToHash("0xBAD")}
				err := h.interop.invalidateBlock(mock.id, blockID, 0)

				require.Error(t, err)
				require.Contains(t, err.Error(), "engine failure")
			},
		},
		{
			name: "handleResult calls invalidateBlock for each invalid head",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, nil).WithChain(8453, nil).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				mock1 := h.Mock(10)
				mock2 := h.Mock(8453)

				invalidResult := Result{
					Timestamp:   1000,
					L1Inclusion: eth.BlockID{Number: 100, Hash: common.HexToHash("0xL1")},
					L2Heads: map[eth.ChainID]eth.BlockID{
						mock1.id: {Number: 500, Hash: common.HexToHash("0xL2-1")},
						mock2.id: {Number: 600, Hash: common.HexToHash("0xL2-2")},
					},
					InvalidHeads: map[eth.ChainID]eth.BlockID{
						mock1.id: {Number: 500, Hash: common.HexToHash("0xBAD1")},
						mock2.id: {Number: 600, Hash: common.HexToHash("0xBAD2")},
					},
				}

				err := applyResultCompat(h.interop, invalidResult)
				require.NoError(t, err)

				require.Len(t, mock1.invalidateBlockCalls, 1)
				require.Equal(t, uint64(500), mock1.invalidateBlockCalls[0].height)
				require.Equal(t, common.HexToHash("0xBAD1"), mock1.invalidateBlockCalls[0].payloadHash)

				require.Len(t, mock2.invalidateBlockCalls, 1)
				require.Equal(t, uint64(600), mock2.invalidateBlockCalls[0].height)
				require.Equal(t, common.HexToHash("0xBAD2"), mock2.invalidateBlockCalls[0].payloadHash)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropTestHarness(t)
			tc.setup(h)
			tc.run(t, h)
		})
	}
}

// =============================================================================
// TestProgressAndRecord
// =============================================================================

func TestProgressAndRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopTestHarness) *interopTestHarness
		run   func(t *testing.T, h *interopTestHarness)
	}{
		{
			name: "empty result sets L1 to collected minimum",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
					m.blockAtTimestampErr = ethereum.NotFound
				}).WithChain(8453, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
					m.blockAtTimestampErr = ethereum.NotFound
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				require.Equal(t, eth.BlockID{}, h.interop.currentL1)

				madeProgress, err := h.interop.progressAndRecord()
				require.NoError(t, err)
				require.False(t, madeProgress, "empty result should not advance verified timestamp")

				require.Equal(t, uint64(100), h.interop.currentL1.Number)
				require.Equal(t, common.HexToHash("0x1"), h.interop.currentL1.Hash)
			},
		},
		{
			name: "valid result sets L1 to result L1Head",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x200")}
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0xL2")}
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				expectedL1Inclusion := eth.BlockID{Number: 150, Hash: common.HexToHash("0xL1Result")}
				h.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
					return Result{Timestamp: ts, L1Inclusion: expectedL1Inclusion, L2Heads: blocks}, nil
				}

				madeProgress, err := h.interop.progressAndRecord()
				require.NoError(t, err)
				require.True(t, madeProgress, "valid result should advance verified timestamp")

				require.Equal(t, expectedL1Inclusion.Number, h.interop.currentL1.Number)
				require.Equal(t, expectedL1Inclusion.Hash, h.interop.currentL1.Hash)
			},
		},
		{
			name: "valid result caps L1 at min node CurrentL1 when L1Inclusion exceeds it",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 1000, Hash: common.HexToHash("0xleading")}
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0xL2a")}
				}).WithChain(8453, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 990, Hash: common.HexToHash("0xlagging")}
					m.blockAtTimestamp = eth.L2BlockRef{Number: 200, Hash: common.HexToHash("0xL2b")}
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				// L1Inclusion is 1000 (from the leading chain) but chain 8453 is only at 990.
				// interop.currentL1 must be capped at 990 so it never exceeds any node's CurrentL1.
				h.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
					return Result{
						Timestamp:   ts,
						L1Inclusion: eth.BlockID{Number: 1000, Hash: common.HexToHash("0xleading")},
						L2Heads:     blocks,
					}, nil
				}

				madeProgress, err := h.interop.progressAndRecord()
				require.NoError(t, err)
				require.True(t, madeProgress)

				require.Equal(t, uint64(990), h.interop.currentL1.Number)
				require.Equal(t, common.HexToHash("0xlagging"), h.interop.currentL1.Hash)
			},
		},
		{
			name: "invalid result does not update L1",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x200")}
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0xL2")}
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				mock := h.Mock(10)
				initialL1 := eth.BlockID{Number: 50, Hash: common.HexToHash("0x50")}
				h.interop.currentL1 = initialL1

				h.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
					return Result{
						Timestamp:    ts,
						L1Inclusion:  eth.BlockID{Number: 999, Hash: common.HexToHash("0xShouldNotBeUsed")},
						L2Heads:      blocks,
						InvalidHeads: map[eth.ChainID]eth.BlockID{mock.id: {Number: 100}},
					}, nil
				}

				madeProgress, err := h.interop.progressAndRecord()
				require.NoError(t, err)
				require.False(t, madeProgress, "invalid result should not advance verified timestamp")

				require.Equal(t, initialL1.Number, h.interop.currentL1.Number)
				require.Equal(t, initialL1.Hash, h.interop.currentL1.Hash)
			},
		},
		{
			name: "errors propagated",
			setup: func(h *interopTestHarness) *interopTestHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestampErr = errors.New("chain sync error")
				}).Build()
			},
			run: func(t *testing.T, h *interopTestHarness) {
				madeProgress, err := h.interop.progressAndRecord()
				require.Error(t, err)
				require.False(t, madeProgress, "error should not advance verified timestamp")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropTestHarness(t)
			tc.setup(h)
			tc.run(t, h)
		})
	}
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
	interop := New(testLogger(), 100, chains, dataDir, nil)
	require.NotNil(t, interop)
	interop.ctx = context.Background()

	// Verify logsDB is empty initially
	_, hasBlocks := interop.logsDBs[mock.id].LatestSealedBlock()
	require.False(t, hasBlocks)

	// Stub verifyFn
	interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
		return Result{Timestamp: ts, L1Inclusion: eth.BlockID{Number: 100}, L2Heads: blocks}, nil
	}

	// Run 3 cycles
	for i := 0; i < 3; i++ {
		l1, err := interop.collectCurrentL1()
		require.NoError(t, err)
		require.Equal(t, uint64(1000), l1.Number)

		result, err := progressInteropCompat(interop)
		require.NoError(t, err)
		require.False(t, result.IsEmpty())

		err = applyResultCompat(interop, result)
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
		{"with L1Head", Result{Timestamp: 1000, L1Inclusion: eth.BlockID{Number: 100}}, false},
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

// callLog records the order of method calls across multiple mock chain containers.
// Tests use it to verify that operations happen in the expected sequence
// (e.g., all freezes before any invalidation).
type callLog struct {
	mu      sync.Mutex
	entries []callLogEntry
}

type callLogEntry struct {
	chainID eth.ChainID
	method  string
}

func (cl *callLog) record(chainID eth.ChainID, method string) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.entries = append(cl.entries, callLogEntry{chainID: chainID, method: method})
}

func (cl *callLog) snapshot() []callLogEntry {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	out := make([]callLogEntry, len(cl.entries))
	copy(out, cl.entries)
	return out
}

type mockChainContainer struct {
	id eth.ChainID

	currentL1    eth.BlockRef
	currentL1Err error

	blockAtTimestamp      eth.L2BlockRef
	blockAtTimestampErr   error
	blockAtTimestampDelay time.Duration // if set, sleeps this long before responding

	// callsCompleted is incremented atomically when LocalSafeBlockAtTimestamp returns,
	// allowing tests to verify all goroutines drained before checkChainsReady returned.
	callsCompleted atomic.Int32

	lastRequestedTimestamp uint64
	mu                     sync.Mutex

	// InvalidateBlock tracking
	invalidateBlockCalls []invalidateBlockCall
	invalidateBlockRet   bool
	invalidateBlockErr   error
	pruneDeniedResult    map[uint64][]common.Hash
	rewindEngineCalls    []uint64
	rewindEngineErr      error

	// OptimisticAt fields
	optimisticL2    eth.BlockID
	optimisticL1    eth.BlockID
	optimisticAtErr error

	// PauseAndStopVN / Resume tracking
	pauseAndStopVNCalls int
	pauseAndStopVNErr   error
	resumeCalls         int
	resumeErr           error
	callLog             *callLog // shared ordered call log across mocks
}

type invalidateBlockCall struct {
	height      uint64
	payloadHash common.Hash
}

func newMockChainContainer(id uint64) *mockChainContainer {
	return &mockChainContainer{id: eth.ChainIDFromUInt64(id)}
}

func (m *mockChainContainer) ID() eth.ChainID                 { return m.id }
func (m *mockChainContainer) Start(ctx context.Context) error { return nil }
func (m *mockChainContainer) Stop(ctx context.Context) error  { return nil }
func (m *mockChainContainer) Pause(ctx context.Context) error { return nil }
func (m *mockChainContainer) Resume(ctx context.Context) error {
	m.mu.Lock()
	m.resumeCalls++
	m.mu.Unlock()
	if m.callLog != nil {
		m.callLog.record(m.id, "Resume")
	}
	return m.resumeErr
}
func (m *mockChainContainer) PauseAndStopVN(ctx context.Context) error {
	m.mu.Lock()
	m.pauseAndStopVNCalls++
	m.mu.Unlock()
	if m.callLog != nil {
		m.callLog.record(m.id, "PauseAndStopVN")
	}
	return m.pauseAndStopVNErr
}
func (m *mockChainContainer) RegisterVerifier(v activity.VerificationActivity) {}
func (m *mockChainContainer) VerifierCurrentL1s() []eth.BlockID                { return nil }
func (m *mockChainContainer) LocalSafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	// Simulate slow chains. Sleep is outside the lock so it doesn't block other
	// concurrent mock operations during tests.
	if d := m.blockAtTimestampDelay; d > 0 {
		time.Sleep(d)
	}
	// Increment after any simulated delay so callers can verify the goroutine
	// has fully completed (not just started) by the time they observe the count.
	defer m.callsCompleted.Add(1)
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
	// Simulate slow chains. Sleep is outside the lock so it doesn't block other
	// concurrent mock operations during tests.
	if d := m.blockAtTimestampDelay; d > 0 {
		time.Sleep(d)
	}
	// Increment after any simulated delay so callers can verify the goroutine
	// has fully completed (not just started) by the time they observe the count.
	defer m.callsCompleted.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.optimisticAtErr != nil {
		return eth.BlockID{}, eth.BlockID{}, m.optimisticAtErr
	}
	// If explicit optimistic fields are set, use them
	if m.optimisticL2 != (eth.BlockID{}) || m.optimisticL1 != (eth.BlockID{}) {
		return m.optimisticL2, m.optimisticL1, nil
	}
	// Fall back to blockAtTimestamp-derived values (for checkChainsReady tests)
	if m.blockAtTimestampErr != nil {
		return eth.BlockID{}, eth.BlockID{}, m.blockAtTimestampErr
	}
	m.lastRequestedTimestamp = ts
	ref := m.blockAtTimestamp
	ref.Time = ts
	ref.Number = ts
	ref.Hash = common.BigToHash(big.NewInt(int64(ts)))
	return ref.ID(), eth.BlockID{}, nil
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
func (m *mockChainContainer) RewindEngine(ctx context.Context, timestamp uint64, invalidatedBlock eth.BlockRef) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rewindEngineCalls = append(m.rewindEngineCalls, timestamp)
	return m.rewindEngineErr
}
func (m *mockChainContainer) BlockTime() uint64 { return 1 }
func (m *mockChainContainer) InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash, decisionTimestamp uint64) (bool, error) {
	m.mu.Lock()
	m.invalidateBlockCalls = append(m.invalidateBlockCalls, invalidateBlockCall{height: height, payloadHash: payloadHash})
	m.mu.Unlock()
	if m.callLog != nil {
		m.callLog.record(m.id, "InvalidateBlock")
	}
	return m.invalidateBlockRet, m.invalidateBlockErr
}
func (m *mockChainContainer) PruneDeniedAtOrAfterTimestamp(timestamp uint64) (map[uint64][]common.Hash, error) {
	if m.pruneDeniedResult != nil {
		return m.pruneDeniedResult, nil
	}
	return nil, nil
}
func (m *mockChainContainer) IsDenied(height uint64, payloadHash common.Hash) (bool, error) {
	return false, nil
}
func (m *mockChainContainer) SetResetCallback(cb cc.ResetCallback) {}

var _ cc.ChainContainer = (*mockChainContainer)(nil)

func testLogger() gethlog.Logger {
	return gethlog.New()
}

// =============================================================================
// TestWAL_PreservedOnInvalidationFailure
// =============================================================================

// TestWAL_PreservedOnInvalidationFailure verifies that when invalidateBlock fails,
// the pending transition is NOT cleared, allowing retry on restart.
func TestWAL_PreservedOnInvalidationFailure(t *testing.T) {
	h := newInteropTestHarness(t). // newInteropTestHarness calls t.Parallel()
					WithChain(10, func(m *mockChainContainer) {
			m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
			m.invalidateBlockErr = errors.New("engine failure") // InvalidateBlock will fail
		}).
		Build()

	mock := h.Mock(10)

	// Create an invalid result that will trigger DecisionInvalidate
	invalidResult := Result{
		Timestamp:   1000,
		L1Inclusion: eth.BlockID{Number: 100, Hash: common.HexToHash("0xL1")},
		L2Heads: map[eth.ChainID]eth.BlockID{
			mock.id: {Number: 500, Hash: common.HexToHash("0xL2")},
		},
		InvalidHeads: map[eth.ChainID]eth.BlockID{
			mock.id: {Number: 500, Hash: common.HexToHash("0xBAD")},
		},
	}

	// Execute the invalidation decision
	pending, err := h.interop.buildPendingTransition(
		StepOutput{Decision: DecisionInvalidate, Result: invalidResult},
		RoundObservation{},
	)
	require.NoError(t, err)
	require.NoError(t, h.interop.verifiedDB.SetPendingTransition(pending))
	_, err = h.interop.applyPendingTransition(pending)

	// Should return an error because invalidateBlock failed
	require.Error(t, err)
	require.Contains(t, err.Error(), "transition preserved")

	// The pending transition should still be in the WAL (not cleared)
	storedPending, err := h.interop.verifiedDB.GetPendingTransition()
	require.NoError(t, err)
	require.NotNil(t, storedPending, "transition should be preserved when invalidation fails")
	require.Equal(t, DecisionInvalidate, storedPending.Decision)
	require.NotNil(t, storedPending.Result)
	require.Equal(t, common.HexToHash("0xBAD"), storedPending.Result.InvalidHeads[mock.id].Hash)
}

// =============================================================================
// TestWAL_ReplayPreservedOnFailure
// =============================================================================

// TestPendingTransition_RecoverInvalidatePreservedOnFailure verifies that the
// generic pending transition recovery path preserves the transition when
// invalidation fails during replay.
func TestPendingTransition_RecoverInvalidatePreservedOnFailure(t *testing.T) {
	h := newInteropTestHarness(t). // newInteropTestHarness calls t.Parallel()
					WithChain(10, func(m *mockChainContainer) {
			m.invalidateBlockErr = errors.New("engine unavailable") // Will fail on replay
		}).
		Build()

	mock := h.Mock(10)

	// Manually set a pending invalidation transition (simulating a crash mid-apply).
	pending := PendingTransition{
		Decision: DecisionInvalidate,
		Result: &Result{
			Timestamp:   1000,
			L1Inclusion: eth.BlockID{Hash: common.HexToHash("0xL1"), Number: 100},
			InvalidHeads: map[eth.ChainID]eth.BlockID{
				mock.id: {Hash: common.HexToHash("0xBAD"), Number: 500},
			},
		},
	}
	err := h.interop.verifiedDB.SetPendingTransition(pending)
	require.NoError(t, err)

	// Call progressAndRecord — it should recover through applyPendingTransition
	// before attempting a normal round, and preserve the transition on failure.
	_, err = h.interop.progressAndRecord()
	require.Error(t, err)
	require.Contains(t, err.Error(), "transition preserved")

	// The transition should NOT be cleared — it must survive for next restart.
	got, err := h.interop.verifiedDB.GetPendingTransition()
	require.NoError(t, err)
	require.NotNil(t, got, "transition should be preserved when replay fails")
	require.Equal(t, pending.Decision, got.Decision)
	require.NotNil(t, got.Result)
	require.Equal(t, pending.Result.InvalidHeads, got.Result.InvalidHeads)
}

func TestPendingTransition_RecoverRewindPreservedOnFailure(t *testing.T) {
	h := newInteropTestHarness(t). // newInteropTestHarness calls t.Parallel()
					WithChain(10, func(m *mockChainContainer) {
			m.rewindEngineErr = errors.New("rewind failed")
		}).
		Build()

	mock := h.Mock(10)

	require.NoError(t, h.interop.verifiedDB.Commit(VerifiedResult{
		Timestamp:   1000,
		L1Inclusion: eth.BlockID{Number: 50, Hash: common.HexToHash("0xL1a")},
		L2Heads:     map[eth.ChainID]eth.BlockID{mock.id: {Number: 100, Hash: common.HexToHash("0x1")}},
	}))
	require.NoError(t, h.interop.verifiedDB.Commit(VerifiedResult{
		Timestamp:   1001,
		L1Inclusion: eth.BlockID{Number: 51, Hash: common.HexToHash("0xL1b")},
		L2Heads:     map[eth.ChainID]eth.BlockID{mock.id: {Number: 101, Hash: common.HexToHash("0x2")}},
	}))

	lastTS := uint64(1001)
	pending, err := h.interop.buildPendingTransition(
		StepOutput{Decision: DecisionRewind},
		RoundObservation{LastVerifiedTS: &lastTS},
	)
	require.NoError(t, err)
	require.NoError(t, h.interop.verifiedDB.SetPendingTransition(pending))
	_, err = h.interop.applyPendingTransition(pending)
	require.Error(t, err)
	require.Contains(t, err.Error(), "reset chain engine on rewind")

	storedPending, err := h.interop.verifiedDB.GetPendingTransition()
	require.NoError(t, err)
	require.NotNil(t, storedPending, "rewind transition should be preserved when apply fails")
	require.Equal(t, DecisionRewind, storedPending.Decision)
	require.NotNil(t, storedPending.Rewind)
	require.Equal(t, uint64(1001), storedPending.Rewind.RewindAtOrAfter)
	require.Len(t, mock.rewindEngineCalls, 1)
	require.Equal(t, uint64(1000), mock.rewindEngineCalls[0])
}

func TestPendingTransition_RecoverRewindReportsAllFailures(t *testing.T) {
	h := newInteropTestHarness(t). // newInteropTestHarness calls t.Parallel()
					WithChain(10, func(m *mockChainContainer) {
			m.rewindEngineErr = errors.New("rewind failed a")
		}).
		WithChain(8453, func(m *mockChainContainer) {
			m.rewindEngineErr = errors.New("rewind failed b")
		}).
		Build()

	mockA := h.Mock(10)
	mockB := h.Mock(8453)

	require.NoError(t, h.interop.verifiedDB.Commit(VerifiedResult{
		Timestamp:   1000,
		L1Inclusion: eth.BlockID{Number: 50, Hash: common.HexToHash("0xL1a")},
		L2Heads: map[eth.ChainID]eth.BlockID{
			mockA.id: {Number: 100, Hash: common.HexToHash("0x1")},
			mockB.id: {Number: 200, Hash: common.HexToHash("0x2")},
		},
	}))
	require.NoError(t, h.interop.verifiedDB.Commit(VerifiedResult{
		Timestamp:   1001,
		L1Inclusion: eth.BlockID{Number: 51, Hash: common.HexToHash("0xL1b")},
		L2Heads: map[eth.ChainID]eth.BlockID{
			mockA.id: {Number: 101, Hash: common.HexToHash("0x3")},
			mockB.id: {Number: 201, Hash: common.HexToHash("0x4")},
		},
	}))

	lastTS := uint64(1001)
	pending, err := h.interop.buildPendingTransition(
		StepOutput{Decision: DecisionRewind},
		RoundObservation{LastVerifiedTS: &lastTS},
	)
	require.NoError(t, err)
	require.NoError(t, h.interop.verifiedDB.SetPendingTransition(pending))
	_, err = h.interop.applyPendingTransition(pending)
	require.Error(t, err)
	require.Contains(t, err.Error(), "chain 10: reset chain engine on rewind")
	require.Contains(t, err.Error(), "chain 8453: reset chain engine on rewind")
}

func TestPendingTransition_RecoverAdvanceAfterCommitClearsPendingTransition(t *testing.T) {
	h := newInteropTestHarness(t). // newInteropTestHarness calls t.Parallel()
					WithChain(10, nil).
					Build()

	mock := h.Mock(10)
	result := Result{
		Timestamp:   1000,
		L1Inclusion: eth.BlockID{Hash: common.HexToHash("0xL1"), Number: 100},
		L2Heads: map[eth.ChainID]eth.BlockID{
			mock.id: {Hash: common.HexToHash("0xaaa"), Number: 500},
		},
	}

	require.NoError(t, h.interop.verifiedDB.SetPendingTransition(PendingTransition{
		Decision: DecisionAdvance,
		Result:   &result,
	}))
	require.NoError(t, h.interop.verifiedDB.Commit(result.ToVerifiedResult()))

	madeProgress, err := h.interop.progressAndRecord()
	require.NoError(t, err)
	require.True(t, madeProgress, "replay should finish the already-applied advance")

	pending, err := h.interop.verifiedDB.GetPendingTransition()
	require.NoError(t, err)
	require.Nil(t, pending, "idempotent commit should let replay clear the pending transition")
}

// =============================================================================
// TestL1CanonicalityCheckErrorPropagates
// =============================================================================

// TestL1CanonicalityCheckErrorPropagates verifies that when the L1 canonicality
// checker returns an error, observeRound propagates it (does not silently proceed).
func TestL1CanonicalityCheckErrorPropagates(t *testing.T) {
	h := newInteropTestHarness(t). // newInteropTestHarness calls t.Parallel()
					WithChain(10, func(m *mockChainContainer) {
			m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
		}).
		Build()

	mock := h.Mock(10)

	// Commit a verified result so observeRound has a LastVerified to check
	err := h.interop.verifiedDB.Commit(VerifiedResult{
		Timestamp:   1000,
		L1Inclusion: eth.BlockID{Number: 50, Hash: common.HexToHash("0xL1")},
		L2Heads:     map[eth.ChainID]eth.BlockID{mock.id: {Number: 100, Hash: common.HexToHash("0x1")}},
	})
	require.NoError(t, err)

	// Set up a failing L1 checker using a mock that returns errors for all lookups
	h.interop.l1Checker = newByNumberConsistencyChecker(&errorL1Source{
		err: errors.New("L1 RPC unavailable"),
	})

	// Call observeRound — should propagate the L1 checker error
	_, err = h.interop.observeRound()
	require.Error(t, err)
	require.Contains(t, err.Error(), "L1 RPC unavailable")
}

// =============================================================================
// TestRewindAccepted
// =============================================================================

func TestRewindAccepted(t *testing.T) {
	t.Run("rewinds verifiedDB and logsDB to previous frontier", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithChain(10, func(m *mockChainContainer) {
				m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
			}).
			Build()

		mock := h.Mock(10)
		chainID := mock.id

		// Stub verifyFn
		h.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
			return Result{Timestamp: ts, L1Inclusion: eth.BlockID{Number: ts}, L2Heads: blocks}, nil
		}

		// Commit two verified results: T=1000 and T=1001
		err := h.interop.verifiedDB.Commit(VerifiedResult{
			Timestamp:   1000,
			L1Inclusion: eth.BlockID{Number: 50, Hash: common.HexToHash("0xL1a")},
			L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Number: 100, Hash: common.HexToHash("0x1")}},
		})
		require.NoError(t, err)
		err = h.interop.verifiedDB.Commit(VerifiedResult{
			Timestamp:   1001,
			L1Inclusion: eth.BlockID{Number: 51, Hash: common.HexToHash("0xL1b")},
			L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Number: 101, Hash: common.HexToHash("0x2")}},
		})
		require.NoError(t, err)

		// Verify both exist
		has, _ := h.interop.verifiedDB.Has(1001)
		require.True(t, has)

		// Replace logsDB with a tracking mock
		trackingDB := &mockLogsDBWithState{
			latestBlock: eth.BlockID{Hash: common.HexToHash("0x2"), Number: 101},
			hasBlocks:   true,
		}
		h.interop.logsDBs[chainID] = trackingDB

		// Rewind timestamp 1001
		plan, err := h.interop.buildRewindPlan(1001)
		require.NoError(t, err)
		err = h.interop.applyRewindPlan(plan)
		require.NoError(t, err)

		// verifiedDB should only have T=1000
		has, _ = h.interop.verifiedDB.Has(1001)
		require.False(t, has, "T=1001 should be removed")
		has, _ = h.interop.verifiedDB.Has(1000)
		require.True(t, has, "T=1000 should remain")

		// logsDB should have been rewound (not cleared)
		require.True(t, trackingDB.rewindCalled, "logsDB should be rewound to previous frontier")
		require.Equal(t, 0, trackingDB.clearCalled, "logsDB should not be cleared")
	})

	t.Run("clears logsDB when rewinding to empty", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithChain(10, nil).
			Build()

		chainID := h.Mock(10).id

		// Commit one result at activation timestamp
		err := h.interop.verifiedDB.Commit(VerifiedResult{
			Timestamp:   1000,
			L1Inclusion: eth.BlockID{Number: 50},
			L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Number: 100}},
		})
		require.NoError(t, err)

		trackingDB := &mockLogsDBWithState{
			latestBlock: eth.BlockID{Number: 100},
			hasBlocks:   true,
		}
		h.interop.logsDBs[chainID] = trackingDB

		// Rewind the only entry — verifiedDB becomes empty
		plan, err := h.interop.buildRewindPlan(1000)
		require.NoError(t, err)
		err = h.interop.applyRewindPlan(plan)
		require.NoError(t, err)

		// logsDB should be cleared (no previous frontier to rewind to)
		require.True(t, trackingDB.clearCalled > 0, "logsDB should be cleared when rewinding to empty")
	})
}

// =============================================================================
// Mock types for new tests
// =============================================================================

// mockLogsDBWithState extends mockLogsDBForInterop with state tracking for trim tests.
type mockLogsDBWithState struct {
	latestBlock  eth.BlockID
	hasBlocks    bool
	rewindCalled bool
	clearCalled  int
}

func (m *mockLogsDBWithState) LatestSealedBlock() (eth.BlockID, bool) {
	return m.latestBlock, m.hasBlocks
}
func (m *mockLogsDBWithState) FirstSealedBlock() (suptypes.BlockSeal, error) {
	return suptypes.BlockSeal{}, nil
}
func (m *mockLogsDBWithState) FindSealedBlock(number uint64) (suptypes.BlockSeal, error) {
	return suptypes.BlockSeal{}, nil
}
func (m *mockLogsDBWithState) OpenBlock(blockNum uint64) (eth.BlockRef, uint32, map[uint32]*suptypes.ExecutingMessage, error) {
	return eth.BlockRef{}, 0, nil, nil
}
func (m *mockLogsDBWithState) Contains(query suptypes.ContainsQuery) (suptypes.BlockSeal, error) {
	return suptypes.BlockSeal{}, nil
}
func (m *mockLogsDBWithState) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *suptypes.ExecutingMessage) error {
	return nil
}
func (m *mockLogsDBWithState) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	return nil
}
func (m *mockLogsDBWithState) Rewind(inv reads.Invalidator, newHead eth.BlockID) error {
	m.rewindCalled = true
	return nil
}
func (m *mockLogsDBWithState) Clear(inv reads.Invalidator) error {
	m.clearCalled++
	return nil
}
func (m *mockLogsDBWithState) Close() error { return nil }

var _ LogsDB = (*mockLogsDBWithState)(nil)

// errorL1Source implements l1ByNumberSource and always returns an error.
// This is separate from mockL1Source in checker_test.go which uses a map lookup.
type errorL1Source struct {
	err error
}

func (m *errorL1Source) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	return eth.L1BlockRef{}, m.err
}

// progressInteropCompat replicates the old progressInterop() behavior for test compatibility.
// It runs observeRound + verify, returning the same Result that the old method would have returned.
func progressInteropCompat(i *Interop) (Result, error) {
	obs, err := i.observeRound()
	if err != nil {
		return Result{}, err
	}
	if obs.Paused || !obs.ChainsReady {
		return Result{}, nil
	}
	return i.verify(obs.NextTimestamp, obs.BlocksAtTS)
}

// applyResultCompat replicates the old result-application behavior for test compatibility.
func applyResultCompat(i *Interop, result Result) error {
	if result.IsEmpty() {
		return nil
	}
	obs := RoundObservation{}
	var output StepOutput
	if !result.IsValid() {
		output = StepOutput{Decision: DecisionInvalidate, Result: result}
	} else {
		output = StepOutput{Decision: DecisionAdvance, Result: result}
	}
	pending, err := i.buildPendingTransition(output, obs)
	if err != nil {
		return err
	}
	if err := i.verifiedDB.SetPendingTransition(pending); err != nil {
		return err
	}
	_, err = i.applyPendingTransition(pending)
	return err
}

// mockLogsDBForInterop implements LogsDB for interop tests
type mockLogsDBForInterop struct {
	openBlockRef     eth.BlockRef
	openBlockLogCnt  uint32
	openBlockExecMsg map[uint32]*suptypes.ExecutingMessage
	openBlockErr     error
	containsSeal     suptypes.BlockSeal
	containsErr      error

	// Track calls for verification
	rewindCalls []eth.BlockID
	clearCalls  int
	addLogCalls int
	sealCalls   int

	// Configurable return value for FirstSealedBlock
	firstSealedBlock suptypes.BlockSeal
}

func (m *mockLogsDBForInterop) LatestSealedBlock() (eth.BlockID, bool) { return eth.BlockID{}, false }
func (m *mockLogsDBForInterop) FirstSealedBlock() (suptypes.BlockSeal, error) {
	return m.firstSealedBlock, nil
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
	m.addLogCalls++
	return nil
}
func (m *mockLogsDBForInterop) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	m.sealCalls++
	return nil
}
func (m *mockLogsDBForInterop) Rewind(inv reads.Invalidator, newHead eth.BlockID) error {
	m.rewindCalls = append(m.rewindCalls, newHead)
	return nil
}
func (m *mockLogsDBForInterop) Clear(inv reads.Invalidator) error {
	m.clearCalls++
	return nil
}
func (m *mockLogsDBForInterop) Close() error { return nil }

var _ LogsDB = (*mockLogsDBForInterop)(nil)

func TestVerify_DoesNotPersistFrontierLogs(t *testing.T) {
	h := newInteropTestHarness(t). // newInteropTestHarness calls t.Parallel()
					WithChain(10, func(m *mockChainContainer) {
			m.blockAtTimestamp = eth.L2BlockRef{Number: 1000, Hash: common.HexToHash("0x1")}
		}).
		Build()

	mock := h.Mock(10)
	trackingDB := &mockLogsDBForInterop{}
	h.interop.logsDBs[mock.id] = trackingDB

	obs, err := h.interop.observeRound()
	require.NoError(t, err)
	require.True(t, obs.ChainsReady)

	result, err := h.interop.verify(obs.NextTimestamp, obs.BlocksAtTS)
	require.NoError(t, err)
	require.False(t, result.IsEmpty())
	require.Zero(t, trackingDB.addLogCalls, "verify must not write logs into logsDB")
	require.Zero(t, trackingDB.sealCalls, "verify must not seal frontier blocks into logsDB")
}

func TestResetIsNoOp(t *testing.T) {
	h := newInteropTestHarness(t).
		WithChain(10, nil).
		Build()

	mock := h.Mock(10)
	err := h.interop.verifiedDB.Commit(VerifiedResult{
		Timestamp:   1000,
		L1Inclusion: eth.BlockID{Number: 50},
		L2Heads:     map[eth.ChainID]eth.BlockID{mock.id: {Number: 100}},
	})
	require.NoError(t, err)

	trackingDB := &mockLogsDBWithState{
		latestBlock: eth.BlockID{Hash: common.HexToHash("0x1"), Number: 100},
		hasBlocks:   true,
	}
	h.interop.logsDBs[mock.id] = trackingDB
	h.interop.currentL1 = eth.BlockID{Number: 999, Hash: common.HexToHash("0xL1")}

	h.interop.Reset(mock.id, 1000, eth.BlockRef{Number: 100, ParentHash: common.HexToHash("0xparent")})

	has, err := h.interop.verifiedDB.Has(1000)
	require.NoError(t, err)
	require.True(t, has, "reset callback should not mutate verifiedDB")
	require.False(t, trackingDB.rewindCalled, "reset callback should not rewind logsDB")
	require.Zero(t, trackingDB.clearCalled, "reset callback should not clear logsDB")
	require.Equal(t, eth.BlockID{Number: 999, Hash: common.HexToHash("0xL1")}, h.interop.currentL1)
}

// =============================================================================
// TestVerifiedBlockAtL1
// =============================================================================

func TestVerifiedBlockAtL1(t *testing.T) {
	t.Run("zero l1Block returns empty immediately", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithChain(10, nil).
			Build()

		// Commit some verified results so the DB is non-empty
		for ts := uint64(100); ts <= 110; ts++ {
			err := h.interop.verifiedDB.Commit(VerifiedResult{
				Timestamp:   ts,
				L1Inclusion: eth.BlockID{Number: ts + 1000},
				L2Heads:     map[eth.ChainID]eth.BlockID{h.Mock(10).id: {Number: ts}},
			})
			require.NoError(t, err)
		}

		// Call with zero L1BlockRef — should return empty without scanning the DB
		blockID, ts := h.interop.VerifiedBlockAtL1(h.Mock(10).id, eth.L1BlockRef{})
		require.Equal(t, eth.BlockID{}, blockID)
		require.Equal(t, uint64(0), ts)
	})

	t.Run("non-zero l1Block finds matching entry", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithChain(10, nil).
			Build()

		chainID := h.Mock(10).id
		// Use timestamps at/above activation (1000) so VerifiedBlockAtL1 scan finds them
		expectedL2 := eth.BlockID{Hash: common.Hash{0xaa}, Number: 1005}

		for ts := uint64(1000); ts <= 1010; ts++ {
			l2Head := eth.BlockID{Hash: common.Hash{byte(ts)}, Number: ts}
			if ts == 1005 {
				l2Head = expectedL2
			}
			err := h.interop.verifiedDB.Commit(VerifiedResult{
				Timestamp:   ts,
				L1Inclusion: eth.BlockID{Number: ts * 10}, // L1 inclusion grows with timestamp
				L2Heads:     map[eth.ChainID]eth.BlockID{chainID: l2Head},
			})
			require.NoError(t, err)
		}

		// Query for L1 block 10059 — should match timestamp 1005 (L1Inclusion.Number=10050 <= 10059)
		// but not timestamp 1006 (L1Inclusion.Number=10060 > 10059)
		l1Block := eth.L1BlockRef{Hash: common.Hash{0x01}, Number: 10059, Time: 999}
		blockID, ts := h.interop.VerifiedBlockAtL1(chainID, l1Block)
		require.Equal(t, expectedL2, blockID)
		require.Equal(t, uint64(1005), ts)
	})

	t.Run("empty DB returns empty", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithChain(10, nil).
			Build()

		l1Block := eth.L1BlockRef{Hash: common.Hash{0x01}, Number: 1000, Time: 999}
		blockID, ts := h.interop.VerifiedBlockAtL1(h.Mock(10).id, l1Block)
		require.Equal(t, eth.BlockID{}, blockID)
		require.Equal(t, uint64(0), ts)
	})
}

// =============================================================================
// TestFreezeAllBeforeRewind
// =============================================================================

// TestFreezeAllBeforeRewind verifies the freeze-all-then-resume behavior
// introduced in applyPendingTransition for DecisionInvalidate:
//   - All chains (not just invalidated ones) are frozen via PauseAndStopVN
//     before any invalidateBlock call
//   - Only non-invalidated chains are resumed after the invalidation loop
//   - Invalidated chains are NOT resumed (RewindEngine handles that internally)
func TestFreezeAllBeforeRewind(t *testing.T) {
	t.Parallel()

	t.Run("all chains frozen before any invalidation", func(t *testing.T) {
		cl := &callLog{}
		h := newInteropTestHarness(t).
			WithChain(10, func(m *mockChainContainer) {
				m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				m.callLog = cl
			}).
			WithChain(8453, func(m *mockChainContainer) {
				m.blockAtTimestamp = eth.L2BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
				m.callLog = cl
			}).
			WithChain(42, func(m *mockChainContainer) {
				m.blockAtTimestamp = eth.L2BlockRef{Number: 300, Hash: common.HexToHash("0x3")}
				m.callLog = cl
			}).
			Build()

		chain10 := h.Mock(10).id
		chain8453 := h.Mock(8453).id

		// Only chain 10 is invalidated; chains 8453 and 42 are valid.
		invalidResult := Result{
			Timestamp:   1000,
			L1Inclusion: eth.BlockID{Number: 100, Hash: common.HexToHash("0xL1")},
			L2Heads: map[eth.ChainID]eth.BlockID{
				chain10:   {Number: 500, Hash: common.HexToHash("0xL2-10")},
				chain8453: {Number: 600, Hash: common.HexToHash("0xL2-8453")},
			},
			InvalidHeads: map[eth.ChainID]eth.BlockID{
				chain10: {Number: 500, Hash: common.HexToHash("0xBAD")},
			},
		}

		pending, err := h.interop.buildPendingTransition(
			StepOutput{Decision: DecisionInvalidate, Result: invalidResult},
			RoundObservation{},
		)
		require.NoError(t, err)
		require.NoError(t, h.interop.verifiedDB.SetPendingTransition(pending))
		_, err = h.interop.applyPendingTransition(pending)
		require.NoError(t, err)

		entries := cl.snapshot()

		// All three chains must have PauseAndStopVN called.
		require.Equal(t, 1, h.Mock(10).pauseAndStopVNCalls, "chain 10 should be frozen")
		require.Equal(t, 1, h.Mock(8453).pauseAndStopVNCalls, "chain 8453 should be frozen")
		require.Equal(t, 1, h.Mock(42).pauseAndStopVNCalls, "chain 42 should be frozen")

		// Find the index of the first InvalidateBlock call.
		firstInvalidateIdx := -1
		for i, e := range entries {
			if e.method == "InvalidateBlock" {
				firstInvalidateIdx = i
				break
			}
		}
		require.NotEqual(t, -1, firstInvalidateIdx, "should have at least one InvalidateBlock call")

		// Every PauseAndStopVN must come before the first InvalidateBlock.
		for i, e := range entries {
			if e.method == "PauseAndStopVN" {
				require.Less(t, i, firstInvalidateIdx,
					"PauseAndStopVN on chain %s (index %d) must precede first InvalidateBlock (index %d)",
					e.chainID, i, firstInvalidateIdx)
			}
		}
	})

	t.Run("only non-invalidated chains are resumed", func(t *testing.T) {
		cl := &callLog{}
		h := newInteropTestHarness(t).
			WithChain(10, func(m *mockChainContainer) {
				m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				m.callLog = cl
			}).
			WithChain(8453, func(m *mockChainContainer) {
				m.blockAtTimestamp = eth.L2BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
				m.callLog = cl
			}).
			WithChain(42, func(m *mockChainContainer) {
				m.blockAtTimestamp = eth.L2BlockRef{Number: 300, Hash: common.HexToHash("0x3")}
				m.callLog = cl
			}).
			Build()

		chain10 := h.Mock(10).id
		chain8453 := h.Mock(8453).id
		chain42 := h.Mock(42).id

		// Chains 10 and 8453 are invalidated; chain 42 is valid.
		invalidResult := Result{
			Timestamp:   1000,
			L1Inclusion: eth.BlockID{Number: 100, Hash: common.HexToHash("0xL1")},
			L2Heads: map[eth.ChainID]eth.BlockID{
				chain10:   {Number: 500, Hash: common.HexToHash("0xL2-10")},
				chain8453: {Number: 600, Hash: common.HexToHash("0xL2-8453")},
				chain42:   {Number: 700, Hash: common.HexToHash("0xL2-42")},
			},
			InvalidHeads: map[eth.ChainID]eth.BlockID{
				chain10:   {Number: 500, Hash: common.HexToHash("0xBAD10")},
				chain8453: {Number: 600, Hash: common.HexToHash("0xBAD8453")},
			},
		}

		pending, err := h.interop.buildPendingTransition(
			StepOutput{Decision: DecisionInvalidate, Result: invalidResult},
			RoundObservation{},
		)
		require.NoError(t, err)
		require.NoError(t, h.interop.verifiedDB.SetPendingTransition(pending))
		_, err = h.interop.applyPendingTransition(pending)
		require.NoError(t, err)

		// Only chain 42 (non-invalidated) should have Resume called.
		require.Equal(t, 0, h.Mock(10).resumeCalls, "invalidated chain 10 should NOT be resumed")
		require.Equal(t, 0, h.Mock(8453).resumeCalls, "invalidated chain 8453 should NOT be resumed")
		require.Equal(t, 1, h.Mock(42).resumeCalls, "non-invalidated chain 42 should be resumed")
	})

	t.Run("resume happens after all invalidations", func(t *testing.T) {
		cl := &callLog{}
		h := newInteropTestHarness(t).
			WithChain(10, func(m *mockChainContainer) {
				m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				m.callLog = cl
			}).
			WithChain(8453, func(m *mockChainContainer) {
				m.blockAtTimestamp = eth.L2BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
				m.callLog = cl
			}).
			Build()

		chain10 := h.Mock(10).id
		chain8453 := h.Mock(8453).id

		// Only chain 10 is invalidated.
		invalidResult := Result{
			Timestamp:   1000,
			L1Inclusion: eth.BlockID{Number: 100, Hash: common.HexToHash("0xL1")},
			L2Heads: map[eth.ChainID]eth.BlockID{
				chain10:   {Number: 500, Hash: common.HexToHash("0xL2-10")},
				chain8453: {Number: 600, Hash: common.HexToHash("0xL2-8453")},
			},
			InvalidHeads: map[eth.ChainID]eth.BlockID{
				chain10: {Number: 500, Hash: common.HexToHash("0xBAD")},
			},
		}

		pending, err := h.interop.buildPendingTransition(
			StepOutput{Decision: DecisionInvalidate, Result: invalidResult},
			RoundObservation{},
		)
		require.NoError(t, err)
		require.NoError(t, h.interop.verifiedDB.SetPendingTransition(pending))
		_, err = h.interop.applyPendingTransition(pending)
		require.NoError(t, err)

		entries := cl.snapshot()

		// Find the last InvalidateBlock index.
		lastInvalidateIdx := -1
		for i, e := range entries {
			if e.method == "InvalidateBlock" {
				lastInvalidateIdx = i
			}
		}
		require.NotEqual(t, -1, lastInvalidateIdx)

		// Every Resume must come after the last InvalidateBlock.
		for i, e := range entries {
			if e.method == "Resume" {
				require.Greater(t, i, lastInvalidateIdx,
					"Resume on chain %s (index %d) must follow last InvalidateBlock (index %d)",
					e.chainID, i, lastInvalidateIdx)
			}
		}
	})

	t.Run("single chain invalidated freezes and does not resume", func(t *testing.T) {
		cl := &callLog{}
		h := newInteropTestHarness(t).
			WithChain(10, func(m *mockChainContainer) {
				m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				m.callLog = cl
			}).
			Build()

		chain10 := h.Mock(10).id

		// The only chain is invalidated — no chain should be resumed.
		invalidResult := Result{
			Timestamp:   1000,
			L1Inclusion: eth.BlockID{Number: 100, Hash: common.HexToHash("0xL1")},
			L2Heads: map[eth.ChainID]eth.BlockID{
				chain10: {Number: 500, Hash: common.HexToHash("0xL2-10")},
			},
			InvalidHeads: map[eth.ChainID]eth.BlockID{
				chain10: {Number: 500, Hash: common.HexToHash("0xBAD")},
			},
		}

		pending, err := h.interop.buildPendingTransition(
			StepOutput{Decision: DecisionInvalidate, Result: invalidResult},
			RoundObservation{},
		)
		require.NoError(t, err)
		require.NoError(t, h.interop.verifiedDB.SetPendingTransition(pending))
		_, err = h.interop.applyPendingTransition(pending)
		require.NoError(t, err)

		require.Equal(t, 1, h.Mock(10).pauseAndStopVNCalls, "chain should be frozen")
		require.Equal(t, 0, h.Mock(10).resumeCalls, "invalidated chain should NOT be resumed")

		entries := cl.snapshot()
		for _, e := range entries {
			require.NotEqual(t, "Resume", e.method, "no Resume calls expected when all chains are invalidated")
		}
	})
}
