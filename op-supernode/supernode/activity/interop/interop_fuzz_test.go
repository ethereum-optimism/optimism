package interop

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Harness
// =============================================================================

// interopFuzzHarness provides a builder-pattern test setup for Interop tests.
// It reduces boilerplate by handling common setup: temp directories, mock chains,
// interop creation, context assignment, and cleanup.
type interopFuzzHarness struct {
	t              *testing.T
	interop        *Interop
	seed           uint64
	randomChain    cc.RandomChain
	mocks          map[eth.ChainID]*cc.RandomChainContainer
	activationTime uint64
	dataDir        string
	skipBuild      bool // for tests that need custom construction
}

// newInteropFuzzHarness creates a new test harness with sensible defaults.
func newInteropFuzzHarness(t *testing.T) *interopFuzzHarness {
	t.Helper()
	t.Parallel()
	return &interopFuzzHarness{
		t:              t,
		mocks:          make(map[eth.ChainID]*cc.RandomChainContainer),
		activationTime: 1000,
		dataDir:        t.TempDir(),
	}
}

// WithActivation sets the interop activation timestamp.
func (h *interopFuzzHarness) WithActivation(ts uint64) *interopFuzzHarness {
	h.activationTime = ts
	return h
}

// WithDataDir sets a custom data directory (useful for error testing).
func (h *interopFuzzHarness) WithDataDir(dir string) *interopFuzzHarness {
	h.dataDir = dir
	return h
}

// WithChain adds a mock chain container with optional configuration.
func (h *interopFuzzHarness) WithChain(id uint64, configure func(*mockChainContainer)) *interopFuzzHarness {
	mock := newMockChainContainer(id)
	if configure != nil {
		configure(mock)
	}
	h.mocks[mock.id] = mock
	return h
}

// SkipBuild marks that Build() should not create an Interop instance.
// Useful for tests that need to test New() directly.
func (h *interopFuzzHarness) SkipBuild() *interopFuzzHarness {
	h.skipBuild = true
	return h
}

// Build creates the Interop instance from configured mocks.
// Sets up context and registers cleanup.
func (h *interopFuzzHarness) Build() *interopFuzzHarness {
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
func (h *interopFuzzHarness) Chains() map[eth.ChainID]cc.ChainContainer {
	chains := make(map[eth.ChainID]cc.ChainContainer)
	for id, mock := range h.mocks {
		chains[id] = mock
	}
	return chains
}

// Mock returns the mock for a given chain ID.
func (h *interopFuzzHarness) Mock(id uint64) *mockChainContainer {
	return h.mocks[eth.ChainIDFromUInt64(id)]
}

// =============================================================================
// TestNew
// =============================================================================

func _TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopFuzzHarness) *interopFuzzHarness
		run   func(t *testing.T, h *interopFuzzHarness)
	}{
		{
			name: "valid inputs initializes all components",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, nil).WithChain(8453, nil).SkipBuild()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithDataDir("/nonexistent/path").SkipBuild()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
				interop := New(testLogger(), h.activationTime, h.Chains(), h.dataDir)
				require.Nil(t, interop)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropFuzzHarness(t)
			tc.setup(h)
			tc.run(t, h)
		})
	}
}

// =============================================================================
// TestStartStop
// =============================================================================

func _TestStartStop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopFuzzHarness) *interopFuzzHarness
		run   func(t *testing.T, h *interopFuzzHarness)
	}{
		{
			name: "Start blocks until context cancelled",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
					m.blockAtTimestamp = eth.L2BlockRef{Number: 50}
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
					m.blockAtTimestampErr = ethereum.NotFound
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			h := newInteropFuzzHarness(t)
			tc.setup(h)
			tc.run(t, h)
		})
	}
}

// =============================================================================
// TestCollectCurrentL1
// =============================================================================

func _TestCollectCurrentL1(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(h *interopFuzzHarness) *interopFuzzHarness
		assert func(t *testing.T, l1 eth.BlockID, err error)
	}{
		{
			name: "returns minimum L1 across multiple chains",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
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
			h := newInteropFuzzHarness(t)
			tc.setup(h)
			l1, err := h.interop.collectCurrentL1()
			tc.assert(t, l1, err)
		})
	}
}

// =============================================================================
// TestCheckChainsReady
// =============================================================================

func _TestCheckChainsReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(h *interopFuzzHarness) *interopFuzzHarness
		assert func(t *testing.T, h *interopFuzzHarness, blocks map[eth.ChainID]eth.BlockID, err error)
	}{
		{
			name: "all chains ready returns blocks",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).WithChain(8453, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
				}).Build()
			},
			assert: func(t *testing.T, h *interopFuzzHarness, blocks map[eth.ChainID]eth.BlockID, err error) {
				require.NoError(t, err)
				require.Len(t, blocks, 2)
				require.NotEqual(t, common.Hash{}, blocks[h.Mock(10).id].Hash)
				require.NotEqual(t, common.Hash{}, blocks[h.Mock(8453).id].Hash)
			},
		},
		{
			name: "one chain not ready returns error",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100}
				}).WithChain(8453, func(m *mockChainContainer) {
					m.blockAtTimestampErr = ethereum.NotFound
				}).Build()
			},
			assert: func(t *testing.T, h *interopFuzzHarness, blocks map[eth.ChainID]eth.BlockID, err error) {
				require.Error(t, err)
				require.Nil(t, blocks)
			},
		},
		{
			name: "parallel execution works",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				for i := 0; i < 5; i++ {
					idx := i // capture loop var
					h.WithChain(uint64(10+idx), func(m *mockChainContainer) {
						m.blockAtTimestamp = eth.L2BlockRef{Number: uint64(100 + idx)}
					})
				}
				return h.Build()
			},
			assert: func(t *testing.T, h *interopFuzzHarness, blocks map[eth.ChainID]eth.BlockID, err error) {
				require.NoError(t, err)
				require.Len(t, blocks, 5)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropFuzzHarness(t)
			tc.setup(h)
			blocks, err := h.interop.checkChainsReady(1000)
			tc.assert(t, h, blocks, err)
		})
	}
}

// =============================================================================
// TestProgressInterop
// =============================================================================

func _TestProgressInterop(t *testing.T) {
	t.Parallel()

	// Default verifyFn that passes through
	passThroughVerifyFn := func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
		return Result{Timestamp: ts, L1Inclusion: eth.BlockID{Number: 100}, L2Heads: blocks}, nil
	}

	tests := []struct {
		name     string
		setup    func(h *interopFuzzHarness) *interopFuzzHarness
		verifyFn func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error)
		assert   func(t *testing.T, result Result, err error)
		run      func(t *testing.T, h *interopFuzzHarness) // override for complex cases
	}{
		{
			name: "not initialized uses activation timestamp",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
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
			h := newInteropFuzzHarness(t)
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

func _TestProgressInteropWithCycleVerify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopFuzzHarness) *interopFuzzHarness
		run   func(t *testing.T, h *interopFuzzHarness)
	}{
		{
			name: "default cycleVerifyFn returns valid result",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).WithChain(8453, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
				}).WithChain(8453, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			h := newInteropFuzzHarness(t)
			tc.setup(h)
			tc.run(t, h)
		})
	}
}

// =============================================================================
// TestVerifiedAtTimestamp
// =============================================================================

func _TestVerifiedAtTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopFuzzHarness) *interopFuzzHarness
		run   func(t *testing.T, h *interopFuzzHarness)
	}{
		{
			name: "before activation always verified",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100}
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			h := newInteropFuzzHarness(t)
			tc.setup(h)
			tc.run(t, h)
		})
	}
}

// =============================================================================
// TestHandleResult
// =============================================================================

func _TestHandleResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopFuzzHarness) *interopFuzzHarness
		run   func(t *testing.T, h *interopFuzzHarness)
	}{
		{
			name: "empty result is no-op",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
				err := h.interop.handleResult(Result{})
				require.NoError(t, err)

				has, err := h.interop.verifiedDB.Has(0)
				require.NoError(t, err)
				require.False(t, has)
			},
		},
		{
			name: "valid result commits to DB with correct data",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, nil).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, nil).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			h := newInteropFuzzHarness(t)
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
func _TestInvalidateBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopFuzzHarness) *interopFuzzHarness
		run   func(t *testing.T, h *interopFuzzHarness)
	}{
		{
			name: "calls chain.InvalidateBlock with correct args",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, nil).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, nil).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.invalidateBlockErr = errors.New("engine failure")
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
				mock := h.Mock(10)
				blockID := eth.BlockID{Number: 500, Hash: common.HexToHash("0xBAD")}
				err := h.interop.invalidateBlock(mock.id, blockID)

				require.Error(t, err)
				require.Contains(t, err.Error(), "engine failure")
			},
		},
		{
			name: "handleResult calls invalidateBlock for each invalid head",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, nil).WithChain(8453, nil).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			h := newInteropFuzzHarness(t)
			tc.setup(h)
			tc.run(t, h)
		})
	}
}

// =============================================================================
// TestProgressAndRecord
// =============================================================================

func _TestProgressAndRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopFuzzHarness) *interopFuzzHarness
		run   func(t *testing.T, h *interopFuzzHarness)
	}{
		{
			name: "empty result sets L1 to collected minimum",
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x2")}
					m.blockAtTimestampErr = ethereum.NotFound
				}).WithChain(8453, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
					m.blockAtTimestampErr = ethereum.NotFound
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x200")}
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0xL2")}
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1 = eth.BlockRef{Number: 200, Hash: common.HexToHash("0x200")}
					m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0xL2")}
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
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
			setup: func(h *interopFuzzHarness) *interopFuzzHarness {
				return h.WithChain(10, func(m *mockChainContainer) {
					m.currentL1Err = errors.New("L1 sync error")
				}).Build()
			},
			run: func(t *testing.T, h *interopFuzzHarness) {
				madeProgress, err := h.interop.progressAndRecord()
				require.Error(t, err)
				require.False(t, madeProgress, "error should not advance verified timestamp")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newInteropFuzzHarness(t)
			tc.setup(h)
			tc.run(t, h)
		})
	}
}

// =============================================================================
// TestInterop_FullCycle
// =============================================================================

func _TestInterop_FullCycle(t *testing.T) {
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

func _TestResult_IsEmpty(t *testing.T) {
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
// TestReset
// =============================================================================

func _TestReset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(h *interopFuzzHarness) (*interopFuzzHarness, *mockLogsDBForInterop)
		run   func(t *testing.T, h *interopFuzzHarness, mockLogsDB *mockLogsDBForInterop)
	}{
		{
			name: "rewinds logsDB to parent of invalidated block",
			setup: func(h *interopFuzzHarness) (*interopFuzzHarness, *mockLogsDBForInterop) {
				h.WithChain(10, nil).Build()
				mockLogsDB := &mockLogsDBForInterop{}
				h.interop.logsDBs[h.Mock(10).id] = mockLogsDB
				return h, mockLogsDB
			},
			run: func(t *testing.T, h *interopFuzzHarness, mockLogsDB *mockLogsDBForInterop) {
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
			setup: func(h *interopFuzzHarness) (*interopFuzzHarness, *mockLogsDBForInterop) {
				h.WithChain(10, nil).Build()
				mockLogsDB := &mockLogsDBForInterop{
					firstSealedBlock: suptypes.BlockSeal{Number: 5},
				}
				h.interop.logsDBs[h.Mock(10).id] = mockLogsDB
				return h, mockLogsDB
			},
			run: func(t *testing.T, h *interopFuzzHarness, mockLogsDB *mockLogsDBForInterop) {
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
			setup: func(h *interopFuzzHarness) (*interopFuzzHarness, *mockLogsDBForInterop) {
				h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 99}
				}).Build()
				mockLogsDB := &mockLogsDBForInterop{}
				h.interop.logsDBs[h.Mock(10).id] = mockLogsDB
				return h, mockLogsDB
			},
			run: func(t *testing.T, h *interopFuzzHarness, mockLogsDB *mockLogsDBForInterop) {
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
			setup: func(h *interopFuzzHarness) (*interopFuzzHarness, *mockLogsDBForInterop) {
				h.WithChain(10, func(m *mockChainContainer) {
					m.blockAtTimestamp = eth.L2BlockRef{Number: 99}
				}).Build()
				mockLogsDB := &mockLogsDBForInterop{}
				h.interop.logsDBs[h.Mock(10).id] = mockLogsDB
				return h, mockLogsDB
			},
			run: func(t *testing.T, h *interopFuzzHarness, mockLogsDB *mockLogsDBForInterop) {
				h.interop.currentL1 = eth.BlockID{Number: 500, Hash: common.HexToHash("0xL1")}

				invalidatedBlock := eth.BlockRef{Number: 100, ParentHash: common.Hash{}}
				h.interop.Reset(h.Mock(10).id, 100, invalidatedBlock)

				require.Equal(t, eth.BlockID{}, h.interop.currentL1)
			},
		},
		{
			name: "handles unknown chain gracefully",
			setup: func(h *interopFuzzHarness) (*interopFuzzHarness, *mockLogsDBForInterop) {
				h.WithChain(10, nil).Build()
				return h, nil
			},
			run: func(t *testing.T, h *interopFuzzHarness, mockLogsDB *mockLogsDBForInterop) {
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
			h := newInteropFuzzHarness(t)
			h, mockLogsDB := tc.setup(h)
			tc.run(t, h, mockLogsDB)
		})
	}
}
