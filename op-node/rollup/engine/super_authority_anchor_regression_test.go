package engine

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// newAnchorTestController builds an EngineController with BlockTime=2 and
// genesis at time 0, so an anchor timestamp of 999 resolves to block 499.
func newAnchorTestController(t *testing.T, mockEngine *testutils.MockEngine, sa rollup.SuperAuthority) *EngineController {
	t.Helper()
	return NewEngineController(
		context.Background(),
		mockEngine,
		testlog.Logger(t, 0),
		metrics.NoopMetrics,
		&rollup.Config{BlockTime: 2},
		&sync.Config{},
		&testutils.MockL1Source{},
		&testutils.MockEmitter{},
		sa,
	)
}

// TestSafeL2Head_Anchor_DoesNotRegressCachedCrossSafe covers the restart shape
// of the interop-reorg-4 incident: a process whose verifier state was lost
// contributes the activation anchor while the chain (and the last published
// cross-safe, held in the cross-safe cache) is far past it. Anchor resolution
// must hold the cached cross-safe instead of regressing to the anchor — which
// would poison the EL's safe label and turn the next engine reset into a full
// re-derivation from the anchor.
func TestSafeL2Head_Anchor_DoesNotRegressCachedCrossSafe(t *testing.T) {
	localSafe := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 600}
	crossSafe := eth.L2BlockRef{Hash: common.Hash{0xcc}, Number: 580}
	anchorRef := eth.L2BlockRef{Hash: common.Hash{0xa1}, Number: 499}

	mockEngine := &testutils.MockEngine{}
	sa := &mockSuperAuthority{
		fullyVerifiedL2HeadSource: rollup.VerifierHeadAnchor,
		fullyVerifiedTimestamp:    999,
	}
	ec := newAnchorTestController(t, mockEngine, sa)
	ec.SetLocalSafeHead(localSafe)
	ec.crossSafeCache.Store(crossSafe)

	// Anchor resolution looks up block 499; the guard then re-validates the
	// cached cross-safe canonicality at block 580.
	mockEngine.ExpectL2BlockRefByNumber(anchorRef.Number, anchorRef, nil)
	mockEngine.ExpectL2BlockRefByNumber(crossSafe.Number, crossSafe, nil)

	got := ec.SafeL2Head()
	require.Equal(t, crossSafe, got,
		"anchor below the cached cross-safe must hold the cache, not regress to the anchor")
}

// TestSafeL2Head_Anchor_GenesisBootstrap_StillResolvesAnchor verifies the
// downward guard does not break a genuine bootstrap: when the last published
// cross-safe is at (or below) the anchor, the anchor still resolves.
func TestSafeL2Head_Anchor_GenesisBootstrap_StillResolvesAnchor(t *testing.T) {
	genesis := eth.L2BlockRef{Hash: common.Hash{0x99}, Number: 0}

	mockEngine := &testutils.MockEngine{}
	sa := &mockSuperAuthority{
		fullyVerifiedL2HeadSource: rollup.VerifierHeadAnchor,
		fullyVerifiedTimestamp:    0, // resolves to block 0
	}
	ec := newAnchorTestController(t, mockEngine, sa)
	ec.SetLocalSafeHead(genesis)
	ec.crossSafeCache.Store(genesis)

	// Anchor lookup at block 0, then cache canonicality re-validation at block 0.
	mockEngine.ExpectL2BlockRefByNumber(uint64(0), genesis, nil)
	mockEngine.ExpectL2BlockRefByNumber(uint64(0), genesis, nil)

	got := ec.SafeL2Head()
	require.Equal(t, genesis, got, "genesis bootstrap must still resolve the anchor")
}

// TestFinalizedHead_Anchor_DoesNotRegressSeededCache verifies that with the
// finalized cache seeded (as initializeUnknowns now does from the EL label), a
// fresh process cannot be re-seeded down to the activation anchor.
func TestFinalizedHead_Anchor_DoesNotRegressSeededCache(t *testing.T) {
	localFinalized := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 580}
	anchorRef := eth.L2BlockRef{Hash: common.Hash{0xa1}, Number: 499}

	mockEngine := &testutils.MockEngine{}
	sa := &mockSuperAuthority{
		finalizedL2HeadSource: rollup.VerifierHeadAnchor,
		finalizedTimestamp:    999,
	}
	ec := newAnchorTestController(t, mockEngine, sa)
	ec.SetFinalizedHead(localFinalized)
	ec.superAuthorityFinalizedHead = localFinalized // seeded from the EL label at init

	mockEngine.ExpectL2BlockRefByNumber(anchorRef.Number, anchorRef, nil)

	got := ec.FinalizedHead()
	require.Equal(t, localFinalized, got,
		"anchor below the seeded finalized cache must hold the cache, not regress finalized")
}

// TestInitializeUnknowns_SeedsSuperAuthorityStateFromELLabels verifies that a
// fresh controller loads the EL's own labels BEFORE consulting the
// SuperAuthority, and seeds the finalized cache and cross-safe cache from
// them. Pre-fix, initializeUnknowns consulted FinalizedHead() (i.e. the super
// authority) first: with an empty verifier the anchor resolved, seeded the
// caches at the anchor, and the EL finalized label was never loaded at all —
// so every subsequent FCU published the anchor as safe/finalized and the next
// reset re-derived from it.
func TestInitializeUnknowns_SeedsSuperAuthorityStateFromELLabels(t *testing.T) {
	elUnsafe := eth.L2BlockRef{Hash: common.Hash{0x01}, Number: 700}
	elSafe := eth.L2BlockRef{Hash: common.Hash{0x02}, Number: 600}
	elFinalized := eth.L2BlockRef{Hash: common.Hash{0x03}, Number: 580}
	anchorRef := eth.L2BlockRef{Hash: common.Hash{0xa1}, Number: 499}

	mockEngine := &testutils.MockEngine{}
	sa := &mockSuperAuthority{
		fullyVerifiedL2HeadSource: rollup.VerifierHeadAnchor,
		fullyVerifiedTimestamp:    999,
		finalizedL2HeadSource:     rollup.VerifierHeadAnchor,
		finalizedTimestamp:        999,
	}
	ec := newAnchorTestController(t, mockEngine, sa)

	mockEngine.ExpectL2BlockRefByLabel(eth.Unsafe, elUnsafe, nil)
	mockEngine.ExpectL2BlockRefByLabel(eth.Finalized, elFinalized, nil)
	mockEngine.ExpectL2BlockRefByLabel(eth.Safe, elSafe, nil)
	// The cross-unsafe branch calls SafeL2Head() twice (set + log); each call
	// resolves the anchor at 499 and then holds the seeded cross-safe at 600.
	for i := 0; i < 2; i++ {
		mockEngine.ExpectL2BlockRefByNumber(anchorRef.Number, anchorRef, nil)
		mockEngine.ExpectL2BlockRefByNumber(elSafe.Number, elSafe, nil)
	}

	require.NoError(t, ec.initializeUnknowns(context.Background()))

	require.Equal(t, elFinalized, ec.localFinalizedHead,
		"the EL finalized label must be loaded even when the super authority resolves an anchor")
	require.Equal(t, elFinalized, ec.superAuthorityFinalizedHead,
		"the finalized cache must be seeded from the EL label")
	require.Equal(t, elSafe, ec.crossUnsafeHead,
		"cross-unsafe initializes to the held cross-safe (the EL safe label), not the anchor")

	// With the caches seeded, neither published head regresses to the anchor.
	mockEngine.ExpectL2BlockRefByNumber(anchorRef.Number, anchorRef, nil)
	require.Equal(t, elFinalized, ec.FinalizedHead())
	mockEngine.ExpectL2BlockRefByNumber(anchorRef.Number, anchorRef, nil)
	mockEngine.ExpectL2BlockRefByNumber(elSafe.Number, elSafe, nil)
	require.Equal(t, elSafe, ec.SafeL2Head())
}
