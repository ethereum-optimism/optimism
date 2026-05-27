package chain_container

import (
	"context"
	"errors"
	"math/big"
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
	activationTimestamp  uint64
	// isActiveAtFn drives IsActiveAt. When nil, IsActiveAt returns true for any
	// timestamp >= activationTimestamp (matching the production semantics).
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
	return ts >= m.activationTimestamp
}
func (m *mockVerificationActivityForSuperAuthority) ActivationTimestamp() uint64 {
	return m.activationTimestamp
}

var _ activity.VerificationActivity = (*mockVerificationActivityForSuperAuthority)(nil)

// newTestChainContainer creates a simpleChainContainer with a test logger and a
// mock virtual node. Engine and vncfg are nil — tests that exercise the anchor
// path (Source == VerifierHeadAnchor) should use newTestChainContainerWithAnchor
// to install both.
func newTestChainContainer(t *testing.T, chainID eth.ChainID) *simpleChainContainer {
	return &simpleChainContainer{
		chainID:   chainID,
		verifiers: []activity.VerificationActivity{},
		log:       testlog.Logger(t, log.LevelDebug),
		vn:        &mockVirtualNode{},
	}
}

// newTestChainContainerWithAnchor extends newTestChainContainer with a mock
// engine and a rollup config so the activation-anchor lookup
// (vncfg.Rollup.TargetBlockNumber + engine.L2BlockRefByNumber) can succeed in
// tests that exercise the Anchor source.
func newTestChainContainerWithAnchor(t *testing.T, chainID eth.ChainID) (*simpleChainContainer, *mockEngineController) {
	cc := newTestChainContainer(t, chainID)
	eng := newMockEngineController()
	cc.engine = eng
	cc.vncfg = &opnodecfg.Config{
		Rollup: rollup.Config{
			L2ChainID: big.NewInt(420),
			BlockTime: 2,
		},
	}
	return cc, eng
}

// setSyncStatus configures the chain's mock virtual node to return the given SyncStatus.
func setSyncStatus(t *testing.T, cc *simpleChainContainer, ss *eth.SyncStatus) {
	t.Helper()
	mvn, ok := cc.vn.(*mockVirtualNode)
	require.True(t, ok, "newTestChainContainer must install a *mockVirtualNode")
	mvn.syncStatusOverride = func() (*eth.SyncStatus, error) { return ss, nil }
}

// =============================================================================
// FullyVerifiedL2Head — happy paths
// =============================================================================

func TestChainContainer_FullyVerifiedL2Head_MultipleVerifiers_OldestWins(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	v1 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000, // oldest
	}
	v2 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{2}, Number: 200},
		latestVerifiedTS:    2000,
	}
	v3 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{3}, Number: 300},
		latestVerifiedTS:    3000,
	}
	cc.verifiers = []activity.VerificationActivity{v1, v2, v3}

	head, status := cc.FullyVerifiedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
	require.Equal(t, rollup.VerifierHeadVerified, head.Source)
	require.Equal(t, v1.latestVerifiedBlock, head.Block, "should return oldest verified block")
}

func TestChainContainer_FullyVerifiedL2Head_NoVerifiers_ReturnsPreActivation(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))

	head, status := cc.FullyVerifiedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
	require.Equal(t, rollup.VerifierHeadPreActivation, head.Source,
		"no verifiers registered → PreActivation; caller uses local-safe")
	require.Equal(t, eth.BlockID{}, head.Block)
}

func TestChainContainer_FullyVerifiedL2Head_SingleVerifier(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	v := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000,
	}
	cc.verifiers = []activity.VerificationActivity{v}

	head, status := cc.FullyVerifiedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
	require.Equal(t, rollup.VerifierHeadVerified, head.Source)
	require.Equal(t, v.latestVerifiedBlock, head.Block)
}

func TestChainContainer_FullyVerifiedL2Head_VerifiersDisagreeAtSameTimestamp_Panics(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	v1 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000,
	}
	v2 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{2}, Number: 100},
		latestVerifiedTS:    1000, // same TS, different hash → consensus violation
	}
	cc.verifiers = []activity.VerificationActivity{v1, v2}

	require.Panics(t, func() {
		_, _ = cc.FullyVerifiedL2Head()
	})
}

// =============================================================================
// FullyVerifiedL2Head — anchor contribution (replaces "empty BlockID" cases)
// =============================================================================

// Under the new contract, a verifier that is active but has no verified-DB
// entry for this chain contributes its activation-anchor block, NOT an empty
// BlockID. This was bug B: post-activation empty verifiers caused SafeL2Head
// to drop to genesis.

func TestChainContainer_FullyVerifiedL2Head_PostActivation_EmptyVerifierContributesAnchor(t *testing.T) {
	t.Parallel()

	cc, eng := newTestChainContainerWithAnchor(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xbb}, Time: 1500},
	})

	// Active verifier with no entries for this chain — contributes its anchor.
	v := &mockVerificationActivityForSuperAuthority{
		activationTimestamp: 1000,
	}
	cc.verifiers = []activity.VerificationActivity{v}

	// Engine returns a real block for the anchor lookup
	// (block at timestamp activationTS - 1 = 999 → block number 499 with BlockTime=2).
	anchorBlock := eth.L2BlockRef{Hash: [32]byte{0xa1}, Number: 499, Time: 999}
	eng.l2BlockRefByNumberResult = anchorBlock

	head, status := cc.FullyVerifiedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
	require.Equal(t, rollup.VerifierHeadAnchor, head.Source,
		"empty verifier post-activation must contribute its activation anchor, not empty")
	require.Equal(t, anchorBlock.ID(), head.Block,
		"anchor block must be the L2 block at activationTimestamp - 1")
}

func TestChainContainer_FullyVerifiedL2Head_AllUnverified_ContributeAnchors(t *testing.T) {
	t.Parallel()

	cc, eng := newTestChainContainerWithAnchor(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 600, Time: 3000},
	})

	v1 := &mockVerificationActivityForSuperAuthority{activationTimestamp: 1000}
	v2 := &mockVerificationActivityForSuperAuthority{activationTimestamp: 1000}
	cc.verifiers = []activity.VerificationActivity{v1, v2}

	anchorBlock := eth.L2BlockRef{Hash: [32]byte{0xa1}, Number: 499, Time: 999}
	eng.l2BlockRefByNumberResult = anchorBlock

	head, status := cc.FullyVerifiedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
	require.Equal(t, rollup.VerifierHeadAnchor, head.Source)
	require.Equal(t, anchorBlock.ID(), head.Block)
}

func TestChainContainer_FullyVerifiedL2Head_MixedAnchorAndVerified_OldestWins(t *testing.T) {
	t.Parallel()

	cc, eng := newTestChainContainerWithAnchor(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 1000, Time: 3000},
	})

	// v1: empty → contributes anchor at TS 999 (older).
	v1 := &mockVerificationActivityForSuperAuthority{activationTimestamp: 1000}
	// v2: has a verified tip at TS 2500 (newer).
	v2 := &mockVerificationActivityForSuperAuthority{
		activationTimestamp: 1000,
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{0xbb}, Number: 750},
		latestVerifiedTS:    2500,
	}
	cc.verifiers = []activity.VerificationActivity{v1, v2}

	anchorBlock := eth.L2BlockRef{Hash: [32]byte{0xa1}, Number: 499, Time: 999}
	eng.l2BlockRefByNumberResult = anchorBlock

	head, status := cc.FullyVerifiedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
	require.Equal(t, rollup.VerifierHeadAnchor, head.Source,
		"oldest contribution is v1's anchor at TS 999; v2's verified tip at TS 2500 is newer")
	require.Equal(t, anchorBlock.ID(), head.Block)
}

func TestChainContainer_FullyVerifiedL2Head_OneEmptyOneVerified_AnchorOlder(t *testing.T) {
	t.Parallel()

	cc, eng := newTestChainContainerWithAnchor(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 1000, Time: 3000},
	})

	v1 := &mockVerificationActivityForSuperAuthority{
		activationTimestamp: 1000,
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    2000, // newer than anchor
	}
	v2 := &mockVerificationActivityForSuperAuthority{
		activationTimestamp: 1000, // empty → anchor at TS 999
	}
	v3 := &mockVerificationActivityForSuperAuthority{
		activationTimestamp: 1000,
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{3}, Number: 300},
		latestVerifiedTS:    3000,
	}
	cc.verifiers = []activity.VerificationActivity{v1, v2, v3}

	anchorBlock := eth.L2BlockRef{Hash: [32]byte{0xa1}, Number: 499, Time: 999}
	eng.l2BlockRefByNumberResult = anchorBlock

	head, status := cc.FullyVerifiedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
	require.Equal(t, rollup.VerifierHeadAnchor, head.Source,
		"v2's anchor at TS 999 is the oldest contribution")
	require.Equal(t, anchorBlock.ID(), head.Block)
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
	cc.verifiers = []activity.VerificationActivity{v}

	head, status := cc.FullyVerifiedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
	require.Equal(t, rollup.VerifierHeadPreActivation, head.Source,
		"pre-activation → caller uses local-safe (Source=PreActivation)")
	require.Equal(t, eth.BlockID{}, head.Block)
}

func TestChainContainer_FullyVerifiedL2Head_AllPreActivation_ReturnsPreActivationSource(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 50, Hash: [32]byte{0xaa}, Time: 999},
	})

	v1 := &mockVerificationActivityForSuperAuthority{activationTimestamp: 1000}
	v2 := &mockVerificationActivityForSuperAuthority{activationTimestamp: 2000}
	cc.verifiers = []activity.VerificationActivity{v1, v2}

	head, status := cc.FullyVerifiedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
	require.Equal(t, rollup.VerifierHeadPreActivation, head.Source)
}

func TestChainContainer_FullyVerifiedL2Head_MixedActiveAndPreActivation_SkipsInactive(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Number: 600, Hash: [32]byte{0xbb}, Time: 1500},
	})

	// active: has a verified tip
	active := &mockVerificationActivityForSuperAuthority{
		activationTimestamp: 1000,
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{0x11}, Number: 250},
		latestVerifiedTS:    1500,
	}
	// preAct: not yet active — must be skipped, must NOT try to resolve an anchor
	// for a block that doesn't exist on this chain yet.
	preAct := &mockVerificationActivityForSuperAuthority{activationTimestamp: 2000}
	cc.verifiers = []activity.VerificationActivity{active, preAct}

	head, status := cc.FullyVerifiedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status,
		"not-yet-active verifier must be skipped, not surface as HoldPrevious")
	require.Equal(t, rollup.VerifierHeadVerified, head.Source)
	require.Equal(t, active.latestVerifiedBlock, head.Block)
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
	cc.verifiers = []activity.VerificationActivity{v}

	head, status := cc.FullyVerifiedL2Head()
	require.Equal(t, rollup.VerifierHeadHoldPrevious, status,
		"verifier read error must surface as HoldPrevious so the caller floors at finalized, "+
			"NOT use local-safe (that was the bug)")
	require.Equal(t, eth.BlockID{}, head.Block)
}

// =============================================================================
// FinalizedL2Head — same shape as FullyVerifiedL2Head
// =============================================================================

func TestChainContainer_FinalizedL2Head_MultipleVerifiers_OldestWins(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 400},
		LocalSafeL2: eth.L2BlockRef{Number: 600, Time: 1500},
	})

	v1 := &mockVerificationActivityForSuperAuthority{
		activationTimestamp:  1000,
		latestFinalizedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestFinalizedTS:    1000, // oldest
	}
	v2 := &mockVerificationActivityForSuperAuthority{
		activationTimestamp:  1000,
		latestFinalizedBlock: eth.BlockID{Hash: [32]byte{2}, Number: 200},
		latestFinalizedTS:    2000,
	}
	v3 := &mockVerificationActivityForSuperAuthority{
		activationTimestamp:  1000,
		latestFinalizedBlock: eth.BlockID{Hash: [32]byte{3}, Number: 300},
		latestFinalizedTS:    3000,
	}
	cc.verifiers = []activity.VerificationActivity{v1, v2, v3}

	head, status := cc.FinalizedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
	require.Equal(t, rollup.VerifierHeadVerified, head.Source)
	require.Equal(t, v1.latestFinalizedBlock, head.Block)
}

func TestChainContainer_FinalizedL2Head_NoVerifiers_ReturnsPreActivation(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))

	head, status := cc.FinalizedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
	require.Equal(t, rollup.VerifierHeadPreActivation, head.Source)
}

func TestChainContainer_FinalizedL2Head_PostActivation_EmptyVerifierContributesAnchor(t *testing.T) {
	t.Parallel()

	cc, eng := newTestChainContainerWithAnchor(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 400},
		LocalSafeL2: eth.L2BlockRef{Number: 600, Time: 1500},
	})

	v := &mockVerificationActivityForSuperAuthority{activationTimestamp: 1000}
	cc.verifiers = []activity.VerificationActivity{v}

	anchorBlock := eth.L2BlockRef{Hash: [32]byte{0xa1}, Number: 499, Time: 999}
	eng.l2BlockRefByNumberResult = anchorBlock

	head, status := cc.FinalizedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
	require.Equal(t, rollup.VerifierHeadAnchor, head.Source,
		"empty verifier post-activation contributes anchor (fixes the safeDB-to-genesis bug, #20944)")
	require.Equal(t, anchorBlock.ID(), head.Block)
}

func TestChainContainer_FinalizedL2Head_PreActivation_ReturnsPreActivationSource(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 40},
		LocalSafeL2: eth.L2BlockRef{Number: 50, Time: 999},
	})

	v := &mockVerificationActivityForSuperAuthority{activationTimestamp: 1000}
	cc.verifiers = []activity.VerificationActivity{v}

	head, status := cc.FinalizedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
	require.Equal(t, rollup.VerifierHeadPreActivation, head.Source)
}

func TestChainContainer_FinalizedL2Head_AllPreActivation_ReturnsPreActivationSource(t *testing.T) {
	t.Parallel()

	cc := newTestChainContainer(t, eth.ChainIDFromUInt64(420))
	setSyncStatus(t, cc, &eth.SyncStatus{
		FinalizedL1: eth.L1BlockRef{Number: 40},
		LocalSafeL2: eth.L2BlockRef{Number: 50, Time: 999},
	})

	v1 := &mockVerificationActivityForSuperAuthority{activationTimestamp: 1000}
	v2 := &mockVerificationActivityForSuperAuthority{activationTimestamp: 2000}
	cc.verifiers = []activity.VerificationActivity{v1, v2}

	head, status := cc.FinalizedL2Head()
	require.Equal(t, rollup.VerifierHeadOk, status)
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
	cc.verifiers = []activity.VerificationActivity{v}

	head, status := cc.FinalizedL2Head()
	require.Equal(t, rollup.VerifierHeadHoldPrevious, status,
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
	cc.verifiers = []activity.VerificationActivity{v}

	head, status := cc.FinalizedL2Head()
	require.Equal(t, rollup.VerifierHeadHoldPrevious, status)
	require.Equal(t, eth.BlockID{}, head.Block)
}
