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
	localSafe := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100}
	localFinalized := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 50}
	anchorBlock := eth.BlockID{Hash: common.Hash{0xa1}, Number: 80}

	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	sa := &mockSuperAuthority{
		// Empty-verifier post-activation case under the new contract: the
		// supernode contributes its activation anchor as a concrete block.
		fullyVerifiedL2Head:       anchorBlock,
		fullyVerifiedL2HeadSource: rollup.VerifierHeadAnchor,
		fullyVerifiedStatus:       rollup.VerifierHeadOk,
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

	// Anchor block 80 is bounded above by local-safe 100, so we expect the EL
	// canonicality lookups to be exercised.
	anchorRef := eth.L2BlockRef{Hash: anchorBlock.Hash, Number: anchorBlock.Number}
	mockEngine.ExpectL2BlockRefByHash(anchorBlock.Hash, anchorRef, nil)
	mockEngine.ExpectL2BlockRefByNumber(anchorBlock.Number, anchorRef, nil)

	got := ec.SafeL2Head()
	require.NotEqual(t, uint64(0), got.Number,
		"SafeL2Head must not drop to genesis when local-safe (%d) and local-finalized (%d) are non-zero. "+
			"Pre-fix, empty verifier returned (BlockID{}, false) and SafeL2Head fetched L2BlockRefByNumber(0). "+
			"Post-fix, Anchor source carries a concrete activation-anchor block (ethereum-optimism/optimism#20944).",
		localSafe.Number, localFinalized.Number)
	require.Equal(t, anchorRef, got, "SafeL2Head must return the anchor block when verifier is active but empty")
}

// TestSafeL2Head_VerifierError_FloorsAtFinalized exercises bug A and the
// verifier-error portion of bug D.
//
// Under the previous boolean contract, a verifier read error returned
// (BlockID{}, true) which engine_controller.SafeL2Head interpreted as
// "fall back to local-safe", advancing cross-safe past verification. Under the
// tri-state contract, errors surface as VerifierHeadHoldPrevious and the caller
// floors at FinalizedHead — never below.
func TestSafeL2Head_VerifierError_FloorsAtFinalized(t *testing.T) {
	localSafe := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100}
	localFinalized := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 50}

	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	sa := &mockSuperAuthority{
		fullyVerifiedStatus: rollup.VerifierHeadHoldPrevious,
		// FinalizedHead is also consulted; configure it to PreActivation so the
		// floor resolves to localFinalizedHead.
		finalizedL2HeadSource: rollup.VerifierHeadPreActivation,
		finalizedStatus:       rollup.VerifierHeadOk,
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

	got := ec.SafeL2Head()
	require.NotEqual(t, localSafe, got,
		"SafeL2Head must not return localSafeHead on verifier error; "+
			"the previous (BlockID{}, true) signal advanced cross-safe past verification (bug A).")
	require.Equal(t, localFinalized, got,
		"SafeL2Head must floor at localFinalizedHead on verifier error (HoldPrevious semantics).")
}

// TestSafeL2Head_HoldPrevious_UsesCanonicalCache verifies the cross-safe
// cache: on HoldPrevious, return the previously-resolved cross-safe head if
// it is still canonical on the EL, rather than dropping all the way to
// FinalizedHead. Preserves cross-safe progress across transient verifier
// outages.
func TestSafeL2Head_HoldPrevious_UsesCanonicalCache(t *testing.T) {
	localSafe := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100}
	localFinalized := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 50}
	verifiedBlock := eth.BlockID{Hash: common.Hash{0xcc}, Number: 80}
	verifiedRef := eth.L2BlockRef{Hash: verifiedBlock.Hash, Number: verifiedBlock.Number}

	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	sa := &mockSuperAuthority{
		fullyVerifiedL2Head:       verifiedBlock,
		fullyVerifiedL2HeadSource: rollup.VerifierHeadVerified,
		fullyVerifiedStatus:       rollup.VerifierHeadOk,
		// Finalized stays PreActivation so the floor would resolve to
		// localFinalized — distinguishable from the cache hit at block 80.
		finalizedL2HeadSource: rollup.VerifierHeadPreActivation,
		finalizedStatus:       rollup.VerifierHeadOk,
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

	// First call: verified path populates the cache.
	mockEngine.ExpectL2BlockRefByHash(verifiedBlock.Hash, verifiedRef, nil)
	mockEngine.ExpectL2BlockRefByNumber(verifiedBlock.Number, verifiedRef, nil)
	got := ec.SafeL2Head()
	require.Equal(t, verifiedRef, got, "first call should resolve via the Verified path")

	// Verifier now returns HoldPrevious; cache canonicality re-validates and is
	// returned in preference to flooring at finalized.
	sa.fullyVerifiedStatus = rollup.VerifierHeadHoldPrevious
	mockEngine.ExpectL2BlockRefByNumber(verifiedBlock.Number, verifiedRef, nil)
	got = ec.SafeL2Head()
	require.Equal(t, verifiedRef, got,
		"HoldPrevious must return the canonicality-validated cache, not drop to localFinalized")
}

// TestSafeL2Head_HoldPrevious_NonCanonicalCache_FloorsAtFinalized verifies
// that the cross-safe cache is cleared when the cached block is no longer
// canonical (reorg), and the caller then floors at FinalizedHead.
func TestSafeL2Head_HoldPrevious_NonCanonicalCache_FloorsAtFinalized(t *testing.T) {
	localSafe := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100}
	localFinalized := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 50}
	verifiedBlock := eth.BlockID{Hash: common.Hash{0xcc}, Number: 80}
	verifiedRef := eth.L2BlockRef{Hash: verifiedBlock.Hash, Number: verifiedBlock.Number}

	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	sa := &mockSuperAuthority{
		fullyVerifiedL2Head:       verifiedBlock,
		fullyVerifiedL2HeadSource: rollup.VerifierHeadVerified,
		fullyVerifiedStatus:       rollup.VerifierHeadOk,
		finalizedL2HeadSource:     rollup.VerifierHeadPreActivation,
		finalizedStatus:           rollup.VerifierHeadOk,
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

	// First call: verified path populates the cache.
	mockEngine.ExpectL2BlockRefByHash(verifiedBlock.Hash, verifiedRef, nil)
	mockEngine.ExpectL2BlockRefByNumber(verifiedBlock.Number, verifiedRef, nil)
	_ = ec.SafeL2Head()
	require.Equal(t, verifiedRef, ec.superAuthoritySafeHead, "cache must be populated after successful resolution")

	// Simulate a reorg: the EL now reports a different canonical block at the
	// cached number. HoldPrevious must clear the cache and floor at finalized.
	sa.fullyVerifiedStatus = rollup.VerifierHeadHoldPrevious
	mockEngine.ExpectL2BlockRefByNumber(verifiedBlock.Number,
		eth.L2BlockRef{Hash: common.Hash{0xdd}, Number: verifiedBlock.Number}, nil)
	got := ec.SafeL2Head()
	require.Equal(t, localFinalized, got, "non-canonical cache must clear and floor at finalized")
	require.Equal(t, eth.L2BlockRef{}, ec.superAuthoritySafeHead, "cache must be cleared after non-canonical reorg")
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
		finalizedStatus: rollup.VerifierHeadHoldPrevious,
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

	got := ec.FinalizedHead()
	require.Equal(t, eth.L2BlockRef{}, got,
		"FinalizedHead on cold-start HoldPrevious must return zero L2BlockRef, not garbage")
	require.Equal(t, common.Hash{}, got.Hash,
		"resulting ForkchoiceUpdate sends a zero finalized hash, preserving the EL's own label")
}
