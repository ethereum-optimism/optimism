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
	currentL1            eth.BlockID
	// isActiveAtFn drives IsActiveAt for pre-activation fallback tests.
	// When nil, IsActiveAt returns true (active for all timestamps), matching
	// the default "always-active" semantics the existing tests assume.
	isActiveAtFn func(ts uint64) bool
}

func (m *mockVerificationActivityForSuperAuthority) Start(ctx context.Context) error { return nil }
func (m *mockVerificationActivityForSuperAuthority) Stop(ctx context.Context) error  { return nil }
func (m *mockVerificationActivityForSuperAuthority) Name() string                    { return "mock" }
func (m *mockVerificationActivityForSuperAuthority) CurrentL1() eth.BlockID {
	return m.currentL1
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
		chainID: chainID,
		log:     testlog.Logger(t, log.LevelDebug),
		vn:      &mockVirtualNode{},
	}
}

// TestChainContainer_RegisterVerifier_RejectsSecondVerifier pins the contract
// that the chain container supports only one verifier.
func TestChainContainer_RegisterVerifier_RejectsSecondVerifier(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{})

	require.Panics(t, func() {
		cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{})
	})
}

// TestChainContainer_VerifierCurrentL1_ReturnsRegisteredVerifier verifies the
// single-verifier projection of VerifierCurrentL1: (zero, false) when none
// registered, the verifier's L1 plus true once a verifier is registered.
func TestChainContainer_VerifierCurrentL1_ReturnsRegisteredVerifier(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	l1, ok := cc.VerifierCurrentL1()
	require.False(t, ok)
	require.Equal(t, eth.BlockID{}, l1)

	currentL1 := eth.BlockID{Hash: [32]byte{0xaa}, Number: 10}
	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{currentL1: currentL1})

	l1, ok = cc.VerifierCurrentL1()
	require.True(t, ok)
	require.Equal(t, currentL1, l1)
}

// TestChainContainer_FullyVerifiedL2Head_NoVerifier tests that FullyVerifiedL2Head
// returns an empty BlockID and signals fallback when no verifier is registered
func TestChainContainer_FullyVerifiedL2Head_NoVerifier(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID with no verifier")
	require.True(t, useLocalSafe, "should signal fallback to local-safe when no verifier registered")
}

// TestChainContainer_FullyVerifiedL2Head_SingleVerifier tests the happy path
// where a single verifier reports a verified block.
func TestChainContainer_FullyVerifiedL2Head_SingleVerifier(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	verifier := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000,
	}
	cc.RegisterVerifier(verifier)

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, verifier.latestVerifiedBlock, result, "should return the verifier's block")
	require.False(t, useLocalSafe, "should not signal fallback when verifier has a verified block")
}

// TestChainContainer_FullyVerifiedL2Head_Unverified tests that an empty BlockID
// is returned without signaling fallback when the verifier is operational but
// has nothing verified yet.
func TestChainContainer_FullyVerifiedL2Head_Unverified(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{},
		latestVerifiedTS:    0,
	})

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID when verifier is unverified")
	require.False(t, useLocalSafe, "should not signal fallback when verifier exists but is unverified")
}

// TestChainContainer_FinalizedL2Head_NoVerifier tests that FinalizedL2Head
// returns an empty BlockID and signals fallback when no verifier is registered
func TestChainContainer_FinalizedL2Head_NoVerifier(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID with no verifier")
	require.True(t, useLocalFinalized, "should signal fallback to local-finalized when no verifier registered")
}

// TestChainContainer_FinalizedL2Head_SingleVerifier tests the happy path
// where a single verifier reports a finalized block.
func TestChainContainer_FinalizedL2Head_SingleVerifier(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	verifier := &mockVerificationActivityForSuperAuthority{
		latestFinalizedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestFinalizedTS:    1000,
	}
	cc.RegisterVerifier(verifier)

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, verifier.latestFinalizedBlock, result, "should return the verifier's block")
	require.False(t, useLocalFinalized, "should not signal fallback when verifier has a finalized block")
}

// TestChainContainer_FinalizedL2Head_Unfinalized tests that an empty BlockID
// is returned without signaling fallback when the verifier is operational but
// has nothing finalized yet.
func TestChainContainer_FinalizedL2Head_Unfinalized(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{
		latestFinalizedBlock: eth.BlockID{},
		latestFinalizedTS:    0,
	})

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID when verifier is unfinalized")
	require.False(t, useLocalFinalized, "should not signal fallback when verifier exists but is unfinalized")
}

// =============================================================================
// Pre-activation fallback tests (issue #20191)
// =============================================================================
//
// These tests pin the contract that when the registered verifier reports
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
// verifies that a verifier reporting pre-activation for the current LocalSafeL2
// timestamp causes a fallback to local-safe.
func TestChainContainer_FullyVerifiedL2Head_PreActivation_FallsBackToLocalSafe(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 50, Hash: [32]byte{0xaa}, Time: 999},
	})

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{
		isActiveAtFn: preActivationFn(1000),
	})

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "pre-activation should return empty BlockID")
	require.True(t, useLocalSafe, "pre-activation should signal fallback to local-safe")
}

// TestChainContainer_FullyVerifiedL2Head_PostActivation_EmptyVerifierNoFallback
// verifies that once activation has been reached, the existing "verifier
// present but nothing verified yet" behavior is preserved: no fallback even
// though the verifier returns empty.
func TestChainContainer_FullyVerifiedL2Head_PostActivation_EmptyVerifierNoFallback(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xbb}, Time: 1500},
	})

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{
		isActiveAtFn: preActivationFn(1000),
	})

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "post-activation empty verifier returns empty")
	require.False(t, useLocalSafe, "post-activation empty verifier must not signal fallback")
}

// TestChainContainer_FullyVerifiedL2Head_PostActivation_VerifiedBlockReturned
// verifies that the existing "happy path" of returning the verifier's
// LatestVerifiedL2Block is preserved after activation.
func TestChainContainer_FullyVerifiedL2Head_PostActivation_VerifiedBlockReturned(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xbb}, Time: 1500},
	})

	verifiedBlock := eth.BlockID{Hash: [32]byte{0x11}, Number: 100}
	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: verifiedBlock,
		latestVerifiedTS:    1500,
		isActiveAtFn:        preActivationFn(1000),
	})

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, verifiedBlock, result, "post-activation should return verified block")
	require.False(t, useLocalSafe, "post-activation with a verified block must not signal fallback")
}

// TestChainContainer_FinalizedL2Head_PreActivation_FallsBackToLocalFinalized
// mirrors the FullyVerified pre-activation case for the finalized head.
func TestChainContainer_FinalizedL2Head_PreActivation_FallsBackToLocalFinalized(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 40},
		LocalSafeL2: eth.L2BlockRef{Number: 50, Hash: [32]byte{0xcc}, Time: 999},
	})

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{
		isActiveAtFn: preActivationFn(1000),
	})

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, eth.BlockID{}, result, "pre-activation should return empty BlockID")
	require.True(t, useLocalFinalized, "pre-activation should signal fallback to local-finalized")
}

// TestChainContainer_FinalizedL2Head_PostActivation_EmptyVerifierNoFallback —
// post-activation, verifier empty → empty, no fallback (unchanged).
func TestChainContainer_FinalizedL2Head_PostActivation_EmptyVerifierNoFallback(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 400},
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xdd}, Time: 1500},
	})

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{
		isActiveAtFn: preActivationFn(1000),
	})

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, eth.BlockID{}, result)
	require.False(t, useLocalFinalized, "post-activation empty verifier must not signal fallback")
}

// TestChainContainer_FinalizedL2Head_PostActivation_FinalizedBlockReturned —
// post-activation, verifier has a finalized block → returned (unchanged).
func TestChainContainer_FinalizedL2Head_PostActivation_FinalizedBlockReturned(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 400},
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xdd}, Time: 1500},
	})

	finalizedBlock := eth.BlockID{Hash: [32]byte{0x22}, Number: 100}
	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{
		latestFinalizedBlock: finalizedBlock,
		latestFinalizedTS:    1500,
		isActiveAtFn:         preActivationFn(1000),
	})

	result, useLocalFinalized := cc.FinalizedL2Head()
	require.Equal(t, finalizedBlock, result, "post-activation should return finalized block")
	require.False(t, useLocalFinalized)
}

func TestChainContainer_FinalizedL2Head_VerifierError_SignalsLocalFinalizedFallback(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 400},
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xdd}, Time: 1500},
	})

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{
		isActiveAtFn:       preActivationFn(1000),
		latestFinalizedErr: errors.New("database not open"),
	})

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

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{
		isActiveAtFn:      preActivationFn(1000),
		latestVerifiedErr: errors.New("database not open"),
	})

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result)
	require.True(t, useLocalSafe)
}
