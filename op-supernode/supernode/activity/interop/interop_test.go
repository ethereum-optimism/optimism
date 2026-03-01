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
	h.interop = New(testLogger(), h.activationTime, chains, h.dataDir)
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
				interop := New(testLogger(), h.activationTime, h.Chains(), h.dataDir)
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
				interop := New(testLogger(), h.activationTime, h.Chains(), h.dataDir)
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
		assert func(t *testing.T, h *interopTestHarness, blocks map[eth.ChainID]eth.BlockID, err error)
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
			assert: func(t *testing.T, h *interopTestHarness, blocks map[eth.ChainID]eth.BlockID, err error) {
				require.NoError(t, err)
				require.Len(t, blocks, 2)
				require.NotEqual(t, common.Hash{}, blocks[h.Mock(10).id].Hash)
				require.NotEqual(t, common.Hash{}, blocks[h.Mock(8453).id].Hash)
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
			assert: func(t *testing.T, h *interopTestHarness, blocks map[eth.ChainID]eth.BlockID, err error) {
				require.Error(t, err)
				require.Nil(t, blocks)
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
			assert: func(t *testing.T, h *interopTestHarness, blocks map[eth.ChainID]eth.BlockID, err error) {
				require.NoError(t, err)
				require.Len(t, blocks, 5)
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
			assert: func(t *testing.T, h *interopTestHarness, blocks map[eth.ChainID]eth.BlockID, err error) {
				require.Error(t, err)
				require.Nil(t, blocks)
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
			blocks, err := h.interop.checkChainsReady(1000)
			tc.assert(t, h, blocks, err)
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
				result1, err := h.interop.progressInterop()
				require.NoError(t, err)
				require.Equal(t, uint64(1000), result1.Timestamp)

				// Commit
				err = h.interop.handleResult(result1)
				require.NoError(t, err)

				// Second progress should use next timestamp
				result2, err := h.interop.progressInterop()
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
			result, err := h.interop.progressInterop()
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

				result, err := h.interop.progressInterop()
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

				result, err := h.interop.progressInterop()
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

				result, err := h.interop.progressInterop()
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

				result, err := h.interop.progressInterop()
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

				result, err := h.interop.progressInterop()
				require.NoError(t, err)

				err = h.interop.handleResult(result)
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
// TestHandleResult
// =============================================================================

func TestHandleResult(t *testing.T) {
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
				err := h.interop.handleResult(Result{})
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

				err := h.interop.handleResult(validResult)
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

				err := h.interop.handleResult(invalidResult)
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
				err := h.interop.invalidateBlock(mock.id, blockID)
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
				err := h.interop.invalidateBlock(unknownChain, blockID)

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
				err := h.interop.invalidateBlock(mock.id, blockID)

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

				err := h.interop.handleResult(invalidResult)
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
					m.currentL1Err = errors.New("L1 sync error")
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
	interop := New(testLogger(), 100, chains, dataDir)
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

	// OptimisticAt fields
	optimisticL2    eth.BlockID
	optimisticL1    eth.BlockID
	optimisticAtErr error
}

type invalidateBlockCall struct {
	height      uint64
	payloadHash common.Hash
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
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.optimisticAtErr != nil {
		return eth.BlockID{}, eth.BlockID{}, m.optimisticAtErr
	}
	return m.optimisticL2, m.optimisticL1, nil
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
	return nil
}
func (m *mockChainContainer) BlockTime() uint64 { return 1 }
func (m *mockChainContainer) InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invalidateBlockCalls = append(m.invalidateBlockCalls, invalidateBlockCall{height: height, payloadHash: payloadHash})
	return m.invalidateBlockRet, m.invalidateBlockErr
}
func (m *mockChainContainer) IsDenied(height uint64, payloadHash common.Hash) (bool, error) {
	return false, nil
}
func (m *mockChainContainer) SetResetCallback(cb cc.ResetCallback) {}

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

	// Track calls for verification
	rewindCalls []eth.BlockID
	clearCalls  int

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
	return nil
}
func (m *mockLogsDBForInterop) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
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

// =============================================================================
// TestReset
// =============================================================================

func TestReset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopTestHarness) (*interopTestHarness, *mockLogsDBForInterop)
		run   func(t *testing.T, h *interopTestHarness, mockLogsDB *mockLogsDBForInterop)
	}{
		{
			name: "rewinds logsDB to parent of invalidated block",
			setup: func(h *interopTestHarness) (*interopTestHarness, *mockLogsDBForInterop) {
				h.WithChain(10, nil).Build()
				mockLogsDB := &mockLogsDBForInterop{}
				h.interop.logsDBs[h.Mock(10).id] = mockLogsDB
				return h, mockLogsDB
			},
			run: func(t *testing.T, h *interopTestHarness, mockLogsDB *mockLogsDBForInterop) {
				// BlockRef provides the target block info directly (no RPC call needed)
				// logsDB rewinds to parent of invalidated block (Number-1, ParentHash)
				invalidatedBlock := eth.BlockRef{Number: 100, ParentHash: common.HexToHash("0xPARENT")}
				h.interop.Reset(h.Mock(10).id, 100, invalidatedBlock)

				// Should rewind to block 99 (parent of invalidated block 100)
				require.Len(t, mockLogsDB.rewindCalls, 1)
				require.Equal(t, uint64(99), mockLogsDB.rewindCalls[0].Number)
				require.Equal(t, common.HexToHash("0xPARENT"), mockLogsDB.rewindCalls[0].Hash)
				require.Equal(t, 0, mockLogsDB.clearCalls)
			},
		},
		{
			name: "clears logsDB when timestamp at or before blockTime",
			setup: func(h *interopTestHarness) (*interopTestHarness, *mockLogsDBForInterop) {
				h.WithChain(10, nil).Build()
				mockLogsDB := &mockLogsDBForInterop{
					firstSealedBlock: suptypes.BlockSeal{Number: 5},
				}
				h.interop.logsDBs[h.Mock(10).id] = mockLogsDB
				return h, mockLogsDB
			},
			run: func(t *testing.T, h *interopTestHarness, mockLogsDB *mockLogsDBForInterop) {
				// Reset at timestamp 1 with block 1 invalidated; target is block 0
				// Since firstSealedBlock.Number (5) > targetBlock.Number (0), Clear is called
				invalidatedBlock := eth.BlockRef{Number: 1, ParentHash: common.Hash{}}
				h.interop.Reset(h.Mock(10).id, 1, invalidatedBlock)

				require.Len(t, mockLogsDB.rewindCalls, 0)
				require.Equal(t, 1, mockLogsDB.clearCalls)
			},
		},
		{
			name: "rewinds verifiedDB",
			setup: func(h *interopTestHarness) (*interopTestHarness, *mockLogsDBForInterop) {
				h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 99}
				}).Build()
				mockLogsDB := &mockLogsDBForInterop{}
				h.interop.logsDBs[h.Mock(10).id] = mockLogsDB
				return h, mockLogsDB
			},
			run: func(t *testing.T, h *interopTestHarness, mockLogsDB *mockLogsDBForInterop) {
				mock := h.Mock(10)
				// Add some verified results
				for ts := uint64(98); ts <= 102; ts++ {
					err := h.interop.verifiedDB.Commit(VerifiedResult{
						Timestamp:   ts,
						L1Inclusion: eth.BlockID{Number: ts},
						L2Heads:     map[eth.ChainID]eth.BlockID{mock.id: {Number: ts}},
					})
					require.NoError(t, err)
				}

				// Reset at timestamp 100 (timestamp 100 is first NOT removed, so 101, 102 are removed)
				invalidatedBlock := eth.BlockRef{Number: 100, ParentHash: common.Hash{}}
				h.interop.Reset(mock.id, 100, invalidatedBlock)

				// Verify results at 98, 99, 100 still exist (100 is first NOT removed)
				has, _ := h.interop.verifiedDB.Has(98)
				require.True(t, has)
				has, _ = h.interop.verifiedDB.Has(99)
				require.True(t, has)
				has, _ = h.interop.verifiedDB.Has(100)
				require.True(t, has)

				// Verify results at 101, 102 are gone (after reset timestamp)
				has, _ = h.interop.verifiedDB.Has(101)
				require.False(t, has)
				has, _ = h.interop.verifiedDB.Has(102)
				require.False(t, has)
			},
		},
		{
			name: "resets currentL1",
			setup: func(h *interopTestHarness) (*interopTestHarness, *mockLogsDBForInterop) {
				h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 99}
				}).Build()
				mockLogsDB := &mockLogsDBForInterop{}
				h.interop.logsDBs[h.Mock(10).id] = mockLogsDB
				return h, mockLogsDB
			},
			run: func(t *testing.T, h *interopTestHarness, mockLogsDB *mockLogsDBForInterop) {
				h.interop.currentL1 = eth.BlockID{Number: 500, Hash: common.HexToHash("0xL1")}

				invalidatedBlock := eth.BlockRef{Number: 100, ParentHash: common.Hash{}}
				h.interop.Reset(h.Mock(10).id, 100, invalidatedBlock)

				require.Equal(t, eth.BlockID{}, h.interop.currentL1)
			},
		},
		{
			name: "handles unknown chain gracefully",
			setup: func(h *interopTestHarness) (*interopTestHarness, *mockLogsDBForInterop) {
				h.WithChain(10, nil).Build()
				return h, nil
			},
			run: func(t *testing.T, h *interopTestHarness, mockLogsDB *mockLogsDBForInterop) {
				// Reset on unknown chain (should not panic)
				unknownChain := eth.ChainIDFromUInt64(999)
				invalidatedBlock := eth.BlockRef{Number: 100, ParentHash: common.Hash{}}
				h.interop.Reset(unknownChain, 100, invalidatedBlock)
				// Just verify it didn't panic
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropTestHarness(t)
			h, mockLogsDB := tc.setup(h)
			tc.run(t, h, mockLogsDB)
		})
	}
}
