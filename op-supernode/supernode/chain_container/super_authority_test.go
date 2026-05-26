package chain_container

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// mockVerificationActivityForSuperAuthority provides controlled test data for SuperAuthority tests
type mockVerificationActivityForSuperAuthority struct {
	latestVerifiedBlock  eth.BlockID
	latestVerifiedTS     uint64
	latestVerifiedErr    error
	latestFinalizedBlock eth.BlockID
	latestFinalizedTS    uint64
	latestFinalizedErr   error
	// isActiveAtFn drives IsActiveAt for pre-activation fallback tests.
	// When nil, IsActiveAt returns true (active for all timestamps), matching
	// the default "always-active" semantics the existing tests assume.
	isActiveAtFn func(ts uint64) bool
}

func (m *mockVerificationActivityForSuperAuthority) Start(ctx context.Context) error { return nil }
func (m *mockVerificationActivityForSuperAuthority) Stop(ctx context.Context) error  { return nil }
func (m *mockVerificationActivityForSuperAuthority) Name() string                    { return "mock" }
func (m *mockVerificationActivityForSuperAuthority) CurrentL1() eth.BlockID {
	return eth.BlockID{}
}
func (m *mockVerificationActivityForSuperAuthority) VerifiedAtTimestamp(ts uint64) (bool, error) {
	return false, nil
}
func (m *mockVerificationActivityForSuperAuthority) LatestVerifiedL2Block(chainID eth.ChainID) (eth.BlockID, uint64, error) {
	return m.latestVerifiedBlock, m.latestVerifiedTS, m.latestVerifiedErr
}
func (m *mockVerificationActivityForSuperAuthority) Reset(eth.ChainID, uint64, eth.BlockRef) {}
func (m *mockVerificationActivityForSuperAuthority) VerifiedBlockAtL1(chainID eth.ChainID, l1BlockRef eth.L1BlockRef) (eth.BlockID, uint64, error) {
	return m.latestFinalizedBlock, m.latestFinalizedTS, m.latestFinalizedErr
}
func (m *mockVerificationActivityForSuperAuthority) IsActiveAt(ts uint64) bool {
	if m.isActiveAtFn != nil {
		return m.isActiveAtFn(ts)
	}
	return true
}

var _ activity.VerificationActivity = (*mockVerificationActivityForSuperAuthority)(nil)

// newTestChainContainer creates a simpleChainContainer for testing with a test logger
func newTestChainContainer(t *testing.T, chainID eth.ChainID) *simpleChainContainer {
	return &simpleChainContainer{
		chainID:   chainID,
		verifiers: []activity.VerificationActivity{},
		log:       testlog.Logger(t, log.LevelDebug),
		vn:        &mockVirtualNode{},
	}
}

// TestChainContainer_FullyVerifiedL2Head_MultipleVerifiers tests that FullyVerifiedL2Head
// returns the block with the minimum (oldest) timestamp across all verifiers
func TestChainContainer_FullyVerifiedL2Head_MultipleVerifiers(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	// Setup three verifiers with different timestamps
	verifier1 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000, // oldest
	}
	verifier2 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{2}, Number: 200},
		latestVerifiedTS:    2000, // middle
	}
	verifier3 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{3}, Number: 300},
		latestVerifiedTS:    3000, // newest
	}

	cc.verifiers = []activity.VerificationActivity{verifier1, verifier2, verifier3}

	// Should return the block with minimum timestamp (verifier1)
	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, verifier1.latestVerifiedBlock, result, "should return oldest verified block")
	require.False(t, useLocalSafe, "should not signal fallback when verifiers have verified blocks")
}

// TestChainContainer_FullyVerifiedL2Head_NoVerifiers tests that FullyVerifiedL2Head
// returns an empty BlockID and signals fallback when there are no verification activities
func TestChainContainer_FullyVerifiedL2Head_NoVerifiers(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID with no verifiers")
	require.True(t, useLocalSafe, "should signal fallback to local-safe when no verifiers registered")
}

// TestChainContainer_FullyVerifiedL2Head_OneUnverified tests that FullyVerifiedL2Head
// returns an empty BlockID without signaling fallback if any verifier returns an unverified state
func TestChainContainer_FullyVerifiedL2Head_OneUnverified(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	// Setup verifiers where one is unverified (empty BlockID)
	verifier1 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000,
	}
	verifier2 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{}, // unverified
		latestVerifiedTS:    0,             // zero timestamp
	}
	verifier3 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{3}, Number: 300},
		latestVerifiedTS:    3000,
	}

	cc.verifiers = []activity.VerificationActivity{verifier1, verifier2, verifier3}

	// Should return empty BlockID (conservative approach) but NOT signal fallback
	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID when any verifier is unverified")
	require.False(t, useLocalSafe, "should not signal fallback when verifiers exist but are unverified")
}

// TestChainContainer_FullyVerifiedL2Head_SameTimestamp tests that FullyVerifiedL2Head
// panics when multiple verifiers report the same timestamp but different block hashes
func TestChainContainer_FullyVerifiedL2Head_SameTimestamp(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	// Setup verifiers with same timestamp but different block hashes
	verifier1 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000,
	}
	verifier2 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{2}, Number: 100},
		latestVerifiedTS:    1000, // same timestamp, different hash
	}

	cc.verifiers = []activity.VerificationActivity{verifier1, verifier2}

	// Should panic because verifiers disagree on block hash for same timestamp
	require.Panics(t, func() {
		_, _ = cc.FullyVerifiedL2Head()
	}, "should panic when verifiers disagree on block hash for same timestamp")
}

// TestChainContainer_FullyVerifiedL2Head_SingleVerifier tests the simple case
// with just one verification activity
func TestChainContainer_FullyVerifiedL2Head_SingleVerifier(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	verifier := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000,
	}

	cc.verifiers = []activity.VerificationActivity{verifier}

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, verifier.latestVerifiedBlock, result, "should return the single verifier's block")
	require.False(t, useLocalSafe, "should not signal fallback when verifier has verified blocks")
}

// TestChainContainer_FullyVerifiedL2Head_AllUnverified tests that an empty BlockID
// is returned without signaling fallback when all verifiers are unverified
func TestChainContainer_FullyVerifiedL2Head_AllUnverified(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	// All verifiers unverified
	verifier1 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{},
		latestVerifiedTS:    0,
	}
	verifier2 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{},
		latestVerifiedTS:    0,
	}

	cc.verifiers = []activity.VerificationActivity{verifier1, verifier2}

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID when all verifiers are unverified")
	require.False(t, useLocalSafe, "should not signal fallback when verifiers exist but are unverified")
}

// TestChainContainer_FinalizedL2Head_MultipleVerifiers tests that FinalizedL2Head
// returns the block with the minimum (oldest) timestamp across all verifiers
func TestChainContainer_FinalizedL2Head_MultipleVerifiers(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	// Setup three verifiers with different timestamps
	verifier1 := &mockVerificationActivityForSuperAuthority{
		latestFinalizedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestFinalizedTS:    1000, // oldest
	}
	verifier2 := &mockVerificationActivityForSuperAuthority{
		latestFinalizedBlock: eth.BlockID{Hash: [32]byte{2}, Number: 200},
		latestFinalizedTS:    2000, // middle
	}
	verifier3 := &mockVerificationActivityForSuperAuthority{
		latestFinalizedBlock: eth.BlockID{Hash: [32]byte{3}, Number: 300},
		latestFinalizedTS:    3000, // newest
	}

	cc.verifiers = []activity.VerificationActivity{verifier1, verifier2, verifier3}

	// Should return the block with minimum timestamp (verifier1)
	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, verifier1.latestFinalizedBlock, result, "should return oldest finalized block")
	require.False(t, useLocalFinalized, "should not signal fallback when verifiers have finalized blocks")
}

// TestChainContainer_FinalizedL2Head_NoVerifiers tests that FinalizedL2Head
// returns an empty BlockID and signals fallback when there are no verification activities
func TestChainContainer_FinalizedL2Head_NoVerifiers(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID with no verifiers")
	require.True(t, useLocalFinalized, "should signal fallback to local-finalized when no verifiers registered")
}

// TestChainContainer_FinalizedL2Head_OneUnfinalized tests that FinalizedL2Head
// returns an empty BlockID without signaling fallback if any verifier returns an unfinalized state
func TestChainContainer_FinalizedL2Head_OneUnfinalized(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	// Setup verifiers where one is unfinalized (empty BlockID)
	verifier1 := &mockVerificationActivityForSuperAuthority{
		latestFinalizedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestFinalizedTS:    1000,
	}
	verifier2 := &mockVerificationActivityForSuperAuthority{
		latestFinalizedBlock: eth.BlockID{}, // unfinalized
		latestFinalizedTS:    0,             // zero timestamp
	}
	verifier3 := &mockVerificationActivityForSuperAuthority{
		latestFinalizedBlock: eth.BlockID{Hash: [32]byte{3}, Number: 300},
		latestFinalizedTS:    3000,
	}

	cc.verifiers = []activity.VerificationActivity{verifier1, verifier2, verifier3}

	// Should return empty BlockID (conservative approach) but NOT signal fallback
	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID when any verifier is unfinalized")
	require.False(t, useLocalFinalized, "should not signal fallback when verifiers exist but are unfinalized")
}

// TestChainContainer_FinalizedL2Head_SingleVerifier tests the simple case
// with just one verification activity
func TestChainContainer_FinalizedL2Head_SingleVerifier(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	verifier := &mockVerificationActivityForSuperAuthority{
		latestFinalizedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestFinalizedTS:    1000,
	}

	cc.verifiers = []activity.VerificationActivity{verifier}

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, verifier.latestFinalizedBlock, result, "should return the single verifier's block")
	require.False(t, useLocalFinalized, "should not signal fallback when verifier has finalized blocks")
}

// TestChainContainer_FinalizedL2Head_AllUnfinalized tests that an empty BlockID
// is returned without signaling fallback when all verifiers are unfinalized
func TestChainContainer_FinalizedL2Head_AllUnfinalized(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	// All verifiers unfinalized
	verifier1 := &mockVerificationActivityForSuperAuthority{
		latestFinalizedBlock: eth.BlockID{},
		latestFinalizedTS:    0,
	}
	verifier2 := &mockVerificationActivityForSuperAuthority{
		latestFinalizedBlock: eth.BlockID{},
		latestFinalizedTS:    0,
	}

	cc.verifiers = []activity.VerificationActivity{verifier1, verifier2}

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID when all verifiers are unfinalized")
	require.False(t, useLocalFinalized, "should not signal fallback when verifiers exist but are unfinalized")
}

// =============================================================================
// Pre-activation fallback tests (issue #20191)
// =============================================================================
//
// These tests pin the contract that when every registered verifier reports
// IsActiveAt(ss.LocalSafeL2.Time) == false (i.e. the supernode has interop
// scheduled but the L1-derived local-safe head has not yet crossed the
// activation timestamp), super_authority falls back to local-safe /
// local-finalized rather than stalling the engine at genesis.
//
// Note: super_authority consults IsActiveAt(ss.LocalSafeL2.Time) for both the
// safe and finalized head queries, since FinalizedL2 <= LocalSafeL2 always:
// if local-safe is still pre-activation, finalized is too.

// setSyncStatus configures the chain's mock virtual node to return the given
// SyncStatus. The test-only mockVirtualNode always backs newTestChainContainer.
func setSyncStatus(t *testing.T, cc *simpleChainContainer, ss *eth.SyncStatus) {
	t.Helper()
	mvn, ok := cc.vn.(*mockVirtualNode)
	require.True(t, ok, "newTestChainContainer must install a *mockVirtualNode")
	mvn.syncStatusOverride = func() (*eth.SyncStatus, error) { return ss, nil }
}

// preActivationFn returns an isActiveAtFn that reports inactive for all
// timestamps strictly below activationTS.
func preActivationFn(activationTS uint64) func(ts uint64) bool {
	return func(ts uint64) bool { return ts >= activationTS }
}

// TestChainContainer_FullyVerifiedL2Head_PreActivation_FallsBackToLocalSafe
// (case B1a) verifies that a single verifier reporting pre-activation for the
// current LocalSafeL2 timestamp causes a fallback to local-safe.
func TestChainContainer_FullyVerifiedL2Head_PreActivation_FallsBackToLocalSafe(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 50, Hash: [32]byte{0xaa}, Time: 999},
	})

	verifier := &mockVerificationActivityForSuperAuthority{
		isActiveAtFn: preActivationFn(1000),
	}
	cc.verifiers = []activity.VerificationActivity{verifier}

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "pre-activation should return empty BlockID")
	require.True(t, useLocalSafe, "pre-activation should signal fallback to local-safe")
}

// TestChainContainer_FullyVerifiedL2Head_PostActivation_EmptyVerifierNoFallback
// (case B1b) verifies that once activation has been reached, the existing
// "verifier present but nothing verified yet" behavior is preserved: no
// fallback even though the verifier returns empty.
func TestChainContainer_FullyVerifiedL2Head_PostActivation_EmptyVerifierNoFallback(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xbb}, Time: 1500},
	})

	verifier := &mockVerificationActivityForSuperAuthority{
		isActiveAtFn: preActivationFn(1000),
	}
	cc.verifiers = []activity.VerificationActivity{verifier}

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "post-activation empty verifier returns empty")
	require.False(t, useLocalSafe, "post-activation empty verifier must not signal fallback")
}

// TestChainContainer_FullyVerifiedL2Head_PostActivation_VerifiedBlockReturned
// (case B1c) verifies that the existing "happy path" of returning the
// verifier's LatestVerifiedL2Block is preserved after activation.
func TestChainContainer_FullyVerifiedL2Head_PostActivation_VerifiedBlockReturned(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xbb}, Time: 1500},
	})

	verifiedBlock := eth.BlockID{Hash: [32]byte{0x11}, Number: 100}
	verifier := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: verifiedBlock,
		latestVerifiedTS:    1500,
		isActiveAtFn:        preActivationFn(1000),
	}
	cc.verifiers = []activity.VerificationActivity{verifier}

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, verifiedBlock, result, "post-activation should return verified block")
	require.False(t, useLocalSafe, "post-activation with a verified block must not signal fallback")
}

// TestChainContainer_FullyVerifiedL2Head_AllPreActivation_FallsBackToLocalSafe
// (case B1d) verifies the multi-verifier case: when every verifier reports
// pre-activation, fall back to local-safe.
func TestChainContainer_FullyVerifiedL2Head_AllPreActivation_FallsBackToLocalSafe(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 50, Hash: [32]byte{0xaa}, Time: 999},
	})

	verifier1 := &mockVerificationActivityForSuperAuthority{isActiveAtFn: preActivationFn(1000)}
	verifier2 := &mockVerificationActivityForSuperAuthority{isActiveAtFn: preActivationFn(2000)}
	cc.verifiers = []activity.VerificationActivity{verifier1, verifier2}

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "all-pre-activation should return empty BlockID")
	require.True(t, useLocalSafe, "all-pre-activation should signal fallback to local-safe")
}

// TestChainContainer_FullyVerifiedL2Head_MixedActiveAndPreActivation_NoFallback
// (case B1e) verifies that a partial pre-activation state does NOT force
// fallback: if at least one verifier is active but unverified, the existing
// conservative behavior (empty, false) holds.
func TestChainContainer_FullyVerifiedL2Head_MixedActiveAndPreActivation_NoFallback(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xbb}, Time: 1500},
	})

	active := &mockVerificationActivityForSuperAuthority{
		isActiveAtFn: preActivationFn(1000), // active at 1500
	}
	preAct := &mockVerificationActivityForSuperAuthority{
		isActiveAtFn: preActivationFn(2000), // still pre-activation at 1500
	}
	cc.verifiers = []activity.VerificationActivity{active, preAct}

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "mixed case should return empty BlockID")
	require.False(t, useLocalSafe, "mixed case must not signal fallback; at least one verifier is active and unverified")
}

// TestChainContainer_FinalizedL2Head_PreActivation_FallsBackToLocalFinalized
// (case B2a) mirrors B1a for the finalized head.
func TestChainContainer_FinalizedL2Head_PreActivation_FallsBackToLocalFinalized(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 40},
		LocalSafeL2: eth.L2BlockRef{Number: 50, Hash: [32]byte{0xcc}, Time: 999},
	})

	verifier := &mockVerificationActivityForSuperAuthority{
		isActiveAtFn: preActivationFn(1000),
	}
	cc.verifiers = []activity.VerificationActivity{verifier}

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, eth.BlockID{}, result, "pre-activation should return empty BlockID")
	require.True(t, useLocalFinalized, "pre-activation should signal fallback to local-finalized")
}

// TestChainContainer_FinalizedL2Head_PostActivation_EmptyVerifierNoFallback
// (case B2b) — post-activation, verifier empty → empty, no fallback (unchanged).
func TestChainContainer_FinalizedL2Head_PostActivation_EmptyVerifierNoFallback(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 400},
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xdd}, Time: 1500},
	})

	verifier := &mockVerificationActivityForSuperAuthority{
		isActiveAtFn: preActivationFn(1000),
	}
	cc.verifiers = []activity.VerificationActivity{verifier}

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, eth.BlockID{}, result)
	require.False(t, useLocalFinalized, "post-activation empty verifier must not signal fallback")
}

// TestChainContainer_FinalizedL2Head_PostActivation_FinalizedBlockReturned
// (case B2c) — post-activation, verifier has a finalized block → returned (unchanged).
func TestChainContainer_FinalizedL2Head_PostActivation_FinalizedBlockReturned(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 400},
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xdd}, Time: 1500},
	})

	finalizedBlock := eth.BlockID{Hash: [32]byte{0x22}, Number: 100}
	verifier := &mockVerificationActivityForSuperAuthority{
		latestFinalizedBlock: finalizedBlock,
		latestFinalizedTS:    1500,
		isActiveAtFn:         preActivationFn(1000),
	}
	cc.verifiers = []activity.VerificationActivity{verifier}

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, finalizedBlock, result, "post-activation should return finalized block")
	require.False(t, useLocalFinalized)
}

// TestChainContainer_FinalizedL2Head_AllPreActivation_FallsBackToLocalFinalized
// (case B2d) — multi-verifier all-pre-activation fallback.
func TestChainContainer_FinalizedL2Head_AllPreActivation_FallsBackToLocalFinalized(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 40},
		LocalSafeL2: eth.L2BlockRef{Number: 50, Hash: [32]byte{0xcc}, Time: 999},
	})

	verifier1 := &mockVerificationActivityForSuperAuthority{isActiveAtFn: preActivationFn(1000)}
	verifier2 := &mockVerificationActivityForSuperAuthority{isActiveAtFn: preActivationFn(2000)}
	cc.verifiers = []activity.VerificationActivity{verifier1, verifier2}

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, eth.BlockID{}, result)
	require.True(t, useLocalFinalized, "all-pre-activation should signal fallback to local-finalized")
}

func TestChainContainer_FinalizedL2Head_VerifierError_SignalsLocalFinalizedFallback(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 400},
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xdd}, Time: 1500},
	})

	verifier := &mockVerificationActivityForSuperAuthority{
		isActiveAtFn:       preActivationFn(1000),
		latestFinalizedErr: errors.New("database not open"),
	}
	cc.verifiers = []activity.VerificationActivity{verifier}

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, eth.BlockID{}, result)
	require.True(t, useLocalFinalized)
}

func TestChainContainer_FullyVerifiedL2Head_VerifierError_SignalsLocalSafeFallback(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xbb}, Time: 1500},
	})

	verifier := &mockVerificationActivityForSuperAuthority{
		isActiveAtFn:      preActivationFn(1000),
		latestVerifiedErr: errors.New("database not open"),
	}
	cc.verifiers = []activity.VerificationActivity{verifier}

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result)
	require.True(t, useLocalSafe)
}
