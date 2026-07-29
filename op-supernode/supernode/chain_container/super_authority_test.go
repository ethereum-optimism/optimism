package chain_container

import (
	"context"
	"errors"
	"testing"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
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
	currentL1            eth.BlockID
	activationTimestamp  uint64
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

// newTestChainContainer creates a simpleChainContainer for testing with a test logger
func newTestChainContainer(t *testing.T, chainID eth.ChainID) *simpleChainContainer {
	return newTestChainContainerWithGenesis(t, chainID, 0)
}

func newTestChainContainerWithGenesis(t *testing.T, chainID eth.ChainID, l2GenesisTime uint64) *simpleChainContainer {
	cfg := &opnodecfg.Config{}
	cfg.Rollup.Genesis.L2Time = l2GenesisTime
	return &simpleChainContainer{
		chainID: chainID,
		log:     testlog.Logger(t, log.LevelDebug),
		vn:      &mockVirtualNode{},
		vncfg:   cfg,
	}
}

// setSyncStatus configures the chain's mock virtual node to return the given SyncStatus.
func setSyncStatus(t *testing.T, cc *simpleChainContainer, ss *eth.SyncStatus) {
	t.Helper()
	mvn, ok := cc.vn.(*mockVirtualNode)
	require.True(t, ok, "newTestChainContainer must install a *mockVirtualNode")
	mvn.syncStatusOverride = func() (*eth.SyncStatus, error) { return ss, nil }
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

// =============================================================================
// FullyVerifiedL2Head — happy paths
// =============================================================================

func TestChainContainer_FullyVerifiedL2Head_NoVerifier_ReturnsPreActivation(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))

	head, ok := cc.FullyVerifiedL2Head(t.Context())
	require.True(t, ok)
	require.Equal(t, rollup.VerifierHeadPreActivation, head.Source,
		"no verifier registered → PreActivation; caller uses local-safe")
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
// FullyVerifiedL2Head — anchor contribution
// =============================================================================
//
// Under the new contract, a verifier that is active but has no verified-DB
// entry for this chain contributes its activation-anchor block, NOT an empty
// BlockID. This was bug B: post-activation empty verifiers caused SafeL2Head
// to drop to genesis.

func TestChainContainer_FullyVerifiedL2Head_PostActivation_EmptyVerifierContributesAnchor(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xbb}, Time: 1500},
	})

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{activationTimestamp: 1000})

	head, ok := cc.FullyVerifiedL2Head(t.Context())
	require.True(t, ok)
	require.Equal(t, rollup.VerifierHeadAnchor, head.Source,
		"empty verifier post-activation must contribute its activation anchor")
	require.Equal(t, eth.BlockID{}, head.Block, "anchor contribution carries no block")
	require.Equal(t, uint64(999), head.Timestamp,
		"anchor timestamp is activationTimestamp - 1; engine_controller resolves to a block")
}

// When interop activates at genesis the verifier's raw cap is pre-genesis;
// the super-authority must clamp the anchor timestamp up to L2 genesis so the
// engine controller can resolve a canonical block.
func TestChainContainer_FullyVerifiedL2Head_ActivationAtGenesis_ClampsAnchorToGenesis(t *testing.T) {
	t.Parallel()

	const genesisL2Time = uint64(1_700_000_000)
	const activationTime = genesisL2Time // interop activates at genesis

	cc := newTestChainContainerWithGenesis(t, eth.ChainIDFromUInt64(420), genesisL2Time)
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 5, Hash: [32]byte{0xbb}, Time: genesisL2Time + 10},
	})

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{activationTimestamp: activationTime})

	head, ok := cc.FullyVerifiedL2Head(t.Context())
	require.True(t, ok)
	require.Equal(t, rollup.VerifierHeadAnchor, head.Source)
	require.Equal(t, eth.BlockID{}, head.Block)
	require.Equal(t, genesisL2Time, head.Timestamp,
		"anchor cap must clamp to L2 genesis (raw cap %d is pre-genesis)", activationTime-1)
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

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{activationTimestamp: 1000})

	head, ok := cc.FullyVerifiedL2Head(t.Context())
	require.True(t, ok)
	require.Equal(t, rollup.VerifierHeadPreActivation, head.Source,
		"pre-activation → caller uses local-safe (Source=PreActivation)")
	require.Equal(t, eth.BlockID{}, head.Block)
}

// =============================================================================
// FullyVerifiedL2Head — verifier error → HoldPrevious
// =============================================================================

func TestChainContainer_FullyVerifiedL2Head_VerifierError_ReturnsHoldPrevious(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xbb}, Time: 1500},
	})

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{
		activationTimestamp: 1000,
		latestVerifiedErr:   errors.New("database not open"),
	})

	head, ok := cc.FullyVerifiedL2Head(t.Context())
	require.False(t, ok,
		"verifier read error must surface as HoldPrevious so the caller floors at finalized, "+
			"NOT use local-safe (that was the bug)")
	require.Equal(t, eth.BlockID{}, head.Block)
}

// =============================================================================
// FinalizedL2Head — same shape as FullyVerifiedL2Head
// =============================================================================

func TestChainContainer_FinalizedL2Head_NoVerifier_ReturnsPreActivation(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))

	head, ok := cc.FinalizedL2Head(t.Context())
	require.True(t, ok)
	require.Equal(t, rollup.VerifierHeadPreActivation, head.Source)
}

func TestChainContainer_FinalizedL2Head_SingleVerifier(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 400},
		LocalSafeL2: eth.L2BlockRef{Number: 600, Time: 1500},
	})

	v := &mockVerificationActivityForSuperAuthority{
		activationTimestamp:  1000,
		latestFinalizedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestFinalizedTS:    1000,
	}
	cc.RegisterVerifier(v)

	head, ok := cc.FinalizedL2Head(t.Context())
	require.True(t, ok)
	require.Equal(t, rollup.VerifierHeadVerified, head.Source)
	require.Equal(t, v.latestFinalizedBlock, head.Block)
}

func TestChainContainer_FinalizedL2Head_PostActivation_EmptyVerifierContributesAnchor(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 400},
		LocalSafeL2: eth.L2BlockRef{Number: 600, Time: 1500},
	})

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{activationTimestamp: 1000})

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

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{activationTimestamp: 1000})

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

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{
		activationTimestamp: 1000,
		latestFinalizedErr:  errors.New("database not open"),
	})

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

	cc.RegisterVerifier(&mockVerificationActivityForSuperAuthority{activationTimestamp: 1000})

	head, ok := cc.FinalizedL2Head(t.Context())
	require.False(t, ok)
	require.Equal(t, eth.BlockID{}, head.Block)
}
