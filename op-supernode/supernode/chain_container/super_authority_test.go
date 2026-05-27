package chain_container

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
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
	activationTimestamp  uint64
	currentL1            eth.BlockID
	// isActiveAtFn drives IsActiveAt. When nil, IsActiveAt returns true for any
	// timestamp >= activationTimestamp (matching the production semantics).
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
	if m.latestVerifiedErr != nil {
		return eth.BlockID{}, 0, m.latestVerifiedErr
	}
	if m.latestVerifiedBlock == (eth.BlockID{}) {
		return eth.BlockID{}, m.activationCap(), nil
	}
	return m.latestVerifiedBlock, m.latestVerifiedTS, nil
}
func (m *mockVerificationActivityForSuperAuthority) Reset(eth.ChainID, uint64, eth.BlockRef) {}
func (m *mockVerificationActivityForSuperAuthority) VerifiedBlockAtL1(chainID eth.ChainID, l1BlockRef eth.L1BlockRef) (eth.BlockID, uint64, error) {
	if m.latestFinalizedErr != nil {
		return eth.BlockID{}, 0, m.latestFinalizedErr
	}
	if m.latestFinalizedBlock == (eth.BlockID{}) {
		return eth.BlockID{}, m.activationCap(), nil
	}
	return m.latestFinalizedBlock, m.latestFinalizedTS, nil
}
func (m *mockVerificationActivityForSuperAuthority) activationCap() uint64 {
	if m.activationTimestamp == 0 {
		return 0
	}
	return m.activationTimestamp - 1
}
func (m *mockVerificationActivityForSuperAuthority) IsActiveAt(ts uint64) bool {
	if m.isActiveAtFn != nil {
		return m.isActiveAtFn(ts)
	}
	return ts >= m.activationTimestamp
}

var _ activity.VerificationActivity = (*mockVerificationActivityForSuperAuthority)(nil)

// newTestChainContainer creates a simpleChainContainer with a test logger and a
// mock virtual node. Engine and vncfg are nil — tests that exercise the anchor
// path (Source == VerifierHeadAnchor) should use newTestChainContainerWithAnchor
// to install both.
func newTestChainContainer(t *testing.T, chainID eth.ChainID) *simpleChainContainer {
	return &simpleChainContainer{
		chainID: chainID,
		log:     testlog.Logger(t, log.LevelDebug),
		vn:      &mockVirtualNode{},
	}
}

// setSyncStatus configures the chain's mock virtual node to return the given SyncStatus.
func setSyncStatus(t *testing.T, cc *simpleChainContainer, ss *eth.SyncStatus) {
	t.Helper()
	mvn, ok := cc.vn.(*mockVirtualNode)
	require.True(t, ok, "newTestChainContainer must install a *mockVirtualNode")
	mvn.syncStatusOverride = func() (*eth.SyncStatus, error) { return ss, nil }
}

func TestChainContainer_RegisterVerifier_RejectsSecondVerifier(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{})

	require.Panics(t, func() {
		cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{})
	})
}

func TestChainContainer_VerifierCurrentL1s_ReturnsRegisteredVerifier(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	require.Nil(t, cc.VerifierCurrentL1s())

	currentL1 := eth.BlockID{Hash: [32]byte{0xaa}, Number: 10}
	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{currentL1: currentL1})

	require.Equal(t, []eth.BlockID{currentL1}, cc.VerifierCurrentL1s())
}

func TestChainContainer_FullyVerifiedL2Head_NoVerifiers_ReturnsPreActivation(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))

	head, ok := cc.FullyVerifiedL2Head(t.Context())
	require.True(t, ok)
	require.Equal(t, rollup.VerifierHeadPreActivation, head.Source,
		"no verifier registered -> PreActivation; caller uses local-safe")
	require.Equal(t, eth.BlockID{}, head.Block)
}

func TestChainContainer_FullyVerifiedL2Head_SingleVerifier(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	v := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000,
	}
	cc.RegisterVerifier(v)

	head, ok := cc.FullyVerifiedL2Head(t.Context())
	require.True(t, ok)
	require.Equal(t, rollup.VerifierHeadVerified, head.Source)
	require.Equal(t, v.latestVerifiedBlock, head.Block)
}

// =============================================================================
// FullyVerifiedL2Head — anchor contribution (replaces "empty BlockID" cases)
// =============================================================================

// Under the new contract, a verifier that is active but has no verified-DB
// entry for this chain contributes its activation-anchor timestamp, NOT an
// empty BlockID. This was bug B: a post-activation empty verifier caused
// SafeL2Head to drop to genesis.

// Chain_container does not resolve anchor blocks — it returns the cap timestamp
// via VerifierHead.Timestamp. These tests pin the timestamp pass-through.

func TestChainContainer_FullyVerifiedL2Head_PostActivation_EmptyVerifierContributesAnchor(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xbb}, Time: 1500},
	})

	v := &mockVerificationActivityForSuperAuthority{activationTimestamp: 1000}
	cc.RegisterVerifier(v)

	head, ok := cc.FullyVerifiedL2Head(t.Context())
	require.True(t, ok)
	require.Equal(t, rollup.VerifierHeadAnchor, head.Source,
		"empty verifier post-activation must contribute its activation anchor")
	require.Equal(t, eth.BlockID{}, head.Block, "anchor contribution carries no block")
	require.Equal(t, uint64(999), head.Timestamp,
		"anchor timestamp is activationTimestamp - 1; engine_controller resolves to a block")
}

// =============================================================================
// FullyVerifiedL2Head — pre-activation
// =============================================================================

func TestChainContainer_FullyVerifiedL2Head_PreActivation_ReturnsPreActivationSource(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 50, Hash: [32]byte{0xaa}, Time: 999},
	})

	v := &mockVerificationActivityForSuperAuthority{activationTimestamp: 1000}
	cc.RegisterVerifier(v)

	head, ok := cc.FullyVerifiedL2Head(t.Context())
	require.True(t, ok)
	require.Equal(t, rollup.VerifierHeadPreActivation, head.Source,
		"pre-activation -> caller uses local-safe (Source=PreActivation)")
	require.Equal(t, eth.BlockID{}, head.Block)
}

// =============================================================================
// FullyVerifiedL2Head — verifier error → HoldPrevious (was: useLocalSafe=true)
// =============================================================================

func TestChainContainer_FullyVerifiedL2Head_VerifierError_ReturnsHoldPrevious(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xbb}, Time: 1500},
	})

	v := &mockVerificationActivityForSuperAuthority{
		activationTimestamp: 1000,
		latestVerifiedErr:   errors.New("database not open"),
	}
	cc.RegisterVerifier(v)

	head, ok := cc.FullyVerifiedL2Head(t.Context())
	require.False(t, ok,
		"verifier read error must surface as HoldPrevious so the caller floors at finalized, "+
			"NOT use local-safe (that was the bug)")
	require.Equal(t, eth.BlockID{}, head.Block)
}

func TestChainContainer_FinalizedL2Head_NoVerifiers_ReturnsPreActivation(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))

	head, ok := cc.FinalizedL2Head(t.Context())
	require.True(t, ok)
	require.Equal(t, rollup.VerifierHeadPreActivation, head.Source)
}

func TestChainContainer_FinalizedL2Head_PostActivation_EmptyVerifierContributesAnchor(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 400},
		LocalSafeL2: eth.L2BlockRef{Number: 600, Time: 1500},
	})

	v := &mockVerificationActivityForSuperAuthority{activationTimestamp: 1000}
	cc.RegisterVerifier(v)

	head, ok := cc.FinalizedL2Head(t.Context())
	require.True(t, ok)
	require.Equal(t, rollup.VerifierHeadAnchor, head.Source,
		"empty verifier post-activation contributes anchor (fixes the safeDB-to-genesis bug, #20944)")
	require.Equal(t, uint64(999), head.Timestamp,
		"anchor timestamp is activationTimestamp - 1; engine_controller resolves to a block")
}

func TestChainContainer_FinalizedL2Head_PreActivation_ReturnsPreActivationSource(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 40},
		LocalSafeL2: eth.L2BlockRef{Number: 50, Time: 999},
	})

	v := &mockVerificationActivityForSuperAuthority{activationTimestamp: 1000}
	cc.RegisterVerifier(v)

	head, ok := cc.FinalizedL2Head(t.Context())
	require.True(t, ok)
	require.Equal(t, rollup.VerifierHeadPreActivation, head.Source)
}

func TestChainContainer_FinalizedL2Head_VerifierError_ReturnsHoldPrevious(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 400},
		LocalSafeL2: eth.L2BlockRef{Number: 600, Time: 1500},
	})

	v := &mockVerificationActivityForSuperAuthority{
		activationTimestamp: 1000,
		latestFinalizedErr:  errors.New("database not open"),
	}
	cc.RegisterVerifier(v)

	head, ok := cc.FinalizedL2Head(t.Context())
	require.False(t, ok,
		"verifier read error must surface as HoldPrevious, NOT use local-finalized")
	require.Equal(t, eth.BlockID{}, head.Block)
}

// SyncStatus error path: under the new contract this returns HoldPrevious so the
// caller floors at finalized instead of silently advancing.
func TestChainContainer_FinalizedL2Head_SyncStatusError_ReturnsHoldPrevious(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	mvn := cc.vn.(*mockVirtualNode)
	mvn.syncStatusOverride = func() (*eth.SyncStatus, error) {
		return nil, errors.New("vn not ready")
	}

	v := &mockVerificationActivityForSuperAuthority{activationTimestamp: 1000}
	cc.RegisterVerifier(v)

	head, ok := cc.FinalizedL2Head(t.Context())
	require.False(t, ok)
	require.Equal(t, eth.BlockID{}, head.Block)
}
