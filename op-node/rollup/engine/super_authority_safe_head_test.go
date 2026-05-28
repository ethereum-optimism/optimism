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

// TestSafeL2Head_EmptyVerifier_DoesNotDropToGenesis exercises bug B.
//
// Under the previous boolean contract, an active verifier with no entries for
// this chain returned (BlockID{}, false), and engine_controller.SafeL2Head
// resolved the empty BlockID to L2 genesis via L2BlockRefByNumber(0). That
// dropped cross-safe to genesis once the chain hadn't yet been covered by the
// verifier's depset — see ethereum-optimism/optimism#20944.
//
// Under the tri-state contract, that scenario surfaces as Source = Anchor with
// a concrete activation-anchor block. SafeL2Head bounds by local-safe and
// returns the verifier's contribution. SafeL2Head must never return the L2
// genesis block when local-safe and local-finalized are non-zero.
func TestSafeL2Head_EmptyVerifier_DoesNotDropToGenesis(t *testing.T) {
	// Realistic shape: local-safe has crossed activation. With BlockTime=2 and
	// genesis at time 0, timestamp 999 maps to block 499. Local-safe must be
	// at or past the anchor block (otherwise the chain hasn't crossed
	// activation and the PreActivation path would have fired upstream).
	localSafe := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 600}
	localFinalized := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 50}

	cfg := &rollup.Config{BlockTime: 2}
	anchorRef := eth.L2BlockRef{Hash: common.Hash{0xa1}, Number: 499}

	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	sa := &mockSuperAuthority{
		fullyVerifiedL2HeadSource: rollup.VerifierHeadAnchor,
		fullyVerifiedTimestamp:    999,
	}
	ec := NewEngineController(
		context.Background(),
		mockEngine,
		testlog.Logger(t, 0),
		metrics.NoopMetrics,
		cfg,
		&sync.Config{},
		&testutils.MockL1Source{},
		emitter,
		sa,
	)
	ec.SetLocalSafeHead(localSafe)
	ec.SetFinalizedHead(localFinalized)

	mockEngine.ExpectL2BlockRefByNumber(uint64(499), anchorRef, nil)

	ec.refreshSuperAuthorityPublished()
	got := ec.SafeL2Head()
	require.NotEqual(t, uint64(0), got.Number,
		"SafeL2Head must not drop to genesis when local-safe (%d) and local-finalized (%d) are non-zero. "+
			"Pre-fix, empty verifier returned (BlockID{}, false) and SafeL2Head fetched L2BlockRefByNumber(0). "+
			"Post-fix, Anchor source carries the pre-activation cap timestamp and the engine controller "+
			"resolves the canonical L2 block at that timestamp (ethereum-optimism/optimism#20944).",
		localSafe.Number, localFinalized.Number)
	require.Equal(t, anchorRef, got, "SafeL2Head must return the canonical anchor block at the cap timestamp")
}

// TestSafeL2Head_VerifierError_HoldsCachedCrossSafe exercises bug A and the
// verifier-error portion of bug D under the resolver contract.
//
// Under the previous boolean contract, a verifier read error returned
// (BlockID{}, true) which SafeL2Head interpreted as "fall back to local-safe",
// advancing cross-safe past verification. Under the resolver contract, errors
// surface as ok=false (HoldPrevious) and the resolver holds the PREVIOUS cached
// cross-safe — never local-safe, never a finalized floor.
func TestSafeL2Head_VerifierError_HoldsCachedCrossSafe(t *testing.T) {
	localSafe := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100}
	localFinalized := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 50}
	cachedCrossSafe := eth.L2BlockRef{Hash: common.Hash{0x77}, Number: 70}

	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	sa := &mockSuperAuthority{
		holdPreviousVerified:  true,
		finalizedL2HeadSource: rollup.VerifierHeadPreActivation,
	}
	ec := NewEngineController(
		context.Background(),
		mockEngine,
		testlog.Logger(t, 0),
		metrics.NoopMetrics,
		&rollup.Config{},
		&sync.Config{},
		&testutils.MockL1Source{},
		emitter,
		sa,
	)
	ec.SetLocalSafeHead(localSafe)
	ec.SetFinalizedHead(localFinalized)
	// Seed the SuperAuthority resolver's safe cache + published safe head.
	ec.superAuthoritySafeHead = cachedCrossSafe
	ec.superAuthorityPublishedSafeHead = cachedCrossSafe

	ec.refreshSuperAuthorityPublished()
	got := ec.SafeL2Head()
	require.NotEqual(t, localSafe, got,
		"SafeL2Head must not return localSafeHead on verifier error (bug A).")
	require.Equal(t, cachedCrossSafe, got,
		"SafeL2Head must hold the cached cross-safe on verifier error (HoldPrevious semantics).")
}

// TestFinalizedHead_HoldPrevious_NoCache_ReturnsZero documents the
// error-after-startup trace: verifier errors on the first call, no cached
// super-authority finalized head yet, localSafeHead and localFinalizedHead are
// both zero. The expected behavior is a zero L2BlockRef — not garbage, not
// genesis, not localFinalizedHead. This is the cold-start safety property the
// design discussion explicitly called for.
func TestFinalizedHead_HoldPrevious_NoCache_ReturnsZero(t *testing.T) {
	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	sa := &mockSuperAuthority{
		holdPreviousFinalized: true,
	}
	ec := NewEngineController(
		context.Background(),
		mockEngine,
		testlog.Logger(t, 0),
		metrics.NoopMetrics,
		&rollup.Config{},
		&sync.Config{},
		&testutils.MockL1Source{},
		emitter,
		sa,
	)
	// localSafeHead / localFinalizedHead deliberately left as zero.

	ec.refreshSuperAuthorityPublished()
	got := ec.FinalizedHead()
	require.Equal(t, eth.L2BlockRef{}, got,
		"FinalizedHead on cold-start HoldPrevious must return zero L2BlockRef, not garbage")
	require.Equal(t, common.Hash{}, got.Hash,
		"resulting ForkchoiceUpdate sends a zero finalized hash, preserving the EL's own label")
}

// TestPromoteFinalized_WithSuperAuthorityDoesNotPublishLocalFinalityAheadOfSafe
// is the local-finality gating property (fix 0a2d/d725): under a SuperAuthority
// the L1-derived finalizer advances localFinalizedHead UNCONDITIONALLY (so cross
// finality is never pinned at genesis), but the PUBLISHED finalized head remains
// the resolver's output — it must never run ahead of the cross-safe head.
func TestPromoteFinalized_WithSuperAuthorityDoesNotPublishLocalFinalityAheadOfSafe(t *testing.T) {
	oldPublished := eth.L2BlockRef{Hash: common.Hash{0x99}, Number: 99}
	localSafe := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 12_374}
	localFinalized := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 12_105}

	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	sa := &mockSuperAuthority{
		fullyVerifiedL2Head:       eth.BlockID{Hash: oldPublished.Hash, Number: oldPublished.Number},
		fullyVerifiedL2HeadSource: rollup.VerifierHeadVerified,
		holdPreviousFinalized:     true,
	}
	ec := NewEngineController(
		context.Background(),
		mockEngine,
		testlog.Logger(t, 0),
		metrics.NoopMetrics,
		&rollup.Config{},
		&sync.Config{},
		&testutils.MockL1Source{},
		emitter,
		sa,
	)
	ec.SetLocalSafeHead(localSafe)
	ec.SetFinalizedHead(oldPublished)
	ec.superAuthorityFinalizedHead = oldPublished
	ec.superAuthorityPublishedFinalizedHead = oldPublished
	// Seed the safe cache/published so the held safe is the verified tip.
	ec.superAuthoritySafeHead = oldPublished
	ec.superAuthorityPublishedSafeHead = oldPublished

	mockEngine.ExpectL2BlockRefByHash(oldPublished.Hash, oldPublished, nil)
	mockEngine.ExpectL2BlockRefByNumber(oldPublished.Number, oldPublished, nil)

	ec.PromoteFinalized(context.Background(), localFinalized)

	require.Equal(t, localFinalized, ec.localFinalizedHead,
		"local finality should still advance and bound future super-authority finality")
	require.Equal(t, oldPublished, ec.superAuthorityPublishedFinalizedHead,
		"published/FCU finality must remain at the super-authority finalized head")
	mockEngine.AssertExpectations(t)
}
