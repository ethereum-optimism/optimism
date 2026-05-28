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

	got := ec.SafeL2Head()
	require.NotEqual(t, uint64(0), got.Number,
		"SafeL2Head must not drop to genesis when local-safe (%d) and local-finalized (%d) are non-zero. "+
			"Pre-fix, empty verifier returned (BlockID{}, false) and SafeL2Head fetched L2BlockRefByNumber(0). "+
			"Post-fix, Anchor source carries the pre-activation cap timestamp and the engine controller "+
			"resolves the canonical L2 block at that timestamp (ethereum-optimism/optimism#20944).",
		localSafe.Number, localFinalized.Number)
	require.Equal(t, anchorRef, got, "SafeL2Head must return the canonical anchor block at the cap timestamp")
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
		holdPreviousVerified: true,
		// FinalizedHead is also consulted; configure it to PreActivation so the
		// floor resolves to localFinalizedHead.
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

	got := ec.SafeL2Head()
	require.NotEqual(t, localSafe, got,
		"SafeL2Head must not return localSafeHead on verifier error; "+
			"the previous (BlockID{}, true) signal advanced cross-safe past verification (bug A).")
	require.Equal(t, localFinalized, got,
		"SafeL2Head must floor at localFinalizedHead on verifier error (HoldPrevious semantics).")
}

// TestSafeL2Head_HoldPrevious_UsesCanonicalCache verifies that on HoldPrevious
// the cross-safe cache is consulted (and re-validated for canonicality) before
// falling back to FinalizedHead, so a transient verifier outage doesn't drop
// cross-safe.
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
		finalizedL2HeadSource:     rollup.VerifierHeadPreActivation,
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

	// First call: Verified path populates the cache.
	mockEngine.ExpectL2BlockRefByHash(verifiedBlock.Hash, verifiedRef, nil)
	mockEngine.ExpectL2BlockRefByNumber(verifiedBlock.Number, verifiedRef, nil)
	got := ec.SafeL2Head()
	require.Equal(t, verifiedRef, got, "first call should resolve via the Verified path")

	// Verifier returns HoldPrevious; cache canonicality re-validates and is
	// returned in preference to flooring at finalized.
	sa.holdPreviousVerified = true
	mockEngine.ExpectL2BlockRefByNumber(verifiedBlock.Number, verifiedRef, nil)
	got = ec.SafeL2Head()
	require.Equal(t, verifiedRef, got,
		"HoldPrevious must return the canonicality-validated cache, not drop to localFinalized")
}

// TestSafeL2Head_HoldPrevious_CacheBelowFinalized_FloorsAtFinalized verifies
// that when finalized advances past the cached cross-safe (the cache hasn't
// been refreshed because the verifier has been HoldPrevious throughout), the
// cache is cleared and the fallback returns finalized rather than a cached
// cross-safe that violates the safe >= finalized engine invariant.
func TestSafeL2Head_HoldPrevious_CacheBelowFinalized_FloorsAtFinalized(t *testing.T) {
	localSafe := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 200}
	initialFinalized := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 50}
	verifiedBlock := eth.BlockID{Hash: common.Hash{0xcc}, Number: 80}
	verifiedRef := eth.L2BlockRef{Hash: verifiedBlock.Hash, Number: verifiedBlock.Number}
	advancedFinalized := eth.L2BlockRef{Hash: common.Hash{0xee}, Number: 100}

	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	sa := &mockSuperAuthority{
		fullyVerifiedL2Head:       verifiedBlock,
		fullyVerifiedL2HeadSource: rollup.VerifierHeadVerified,
		finalizedL2HeadSource:     rollup.VerifierHeadPreActivation,
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
	ec.SetFinalizedHead(initialFinalized)

	// Populate the cache via the Verified path.
	mockEngine.ExpectL2BlockRefByHash(verifiedBlock.Hash, verifiedRef, nil)
	mockEngine.ExpectL2BlockRefByNumber(verifiedBlock.Number, verifiedRef, nil)
	_ = ec.SafeL2Head()

	// Finalized advances past the cached cross-safe while the verifier is
	// HoldPrevious. The cache must not be returned (would publish
	// safe=80 < finalized=100), and instead the fallback returns finalized.
	ec.SetFinalizedHead(advancedFinalized)
	sa.holdPreviousVerified = true
	// Cache canonicality re-validates against the EL; the cached block is still
	// canonical, but the fallback prefers finalized because cached < finalized.
	mockEngine.ExpectL2BlockRefByNumber(verifiedBlock.Number, verifiedRef, nil)
	got := ec.SafeL2Head()
	require.Equal(t, advancedFinalized, got,
		"cached cross-safe below finalized must not be returned; fallback floors at finalized")
}

// TestForceReset_SeedsCrossSafeCache verifies that a forced engine reset
// replaces any pre-reset cached cross-safe with the reset crossSafe value, so
// the fallback path returns the reset value (not pre-reset state, and not
// finalized) on a subsequent HoldPrevious.
func TestForceReset_SeedsCrossSafeCache(t *testing.T) {
	cachedRef := eth.L2BlockRef{Hash: common.Hash{0xcc}, Number: 80}
	resetRef := eth.L2BlockRef{Hash: common.Hash{0xa1}, Number: 70}
	resetFinalized := eth.L2BlockRef{Hash: common.Hash{0xb1}, Number: 40}

	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(
		context.Background(),
		mockEngine,
		testlog.Logger(t, 0),
		metrics.NoopMetrics,
		&rollup.Config{},
		&sync.Config{},
		&testutils.MockL1Source{},
		emitter,
		nil,
	)

	// Seed the cache directly.
	ec.crossSafeCache.Store(cachedRef)

	// ForceReset performs a ForkchoiceUpdate; satisfy it with a noop expectation.
	mockEngine.ExpectForkchoiceUpdate(
		&eth.ForkchoiceState{
			HeadBlockHash:      resetRef.Hash,
			SafeBlockHash:      resetRef.Hash,
			FinalizedBlockHash: resetFinalized.Hash,
		},
		nil,
		&eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}},
		nil,
	)
	emitter.ExpectOnceType("ForkchoiceUpdateEvent")
	emitter.ExpectOnceType("EngineResetConfirmedEvent")

	ec.ForceReset(context.Background(), resetRef, resetRef, resetRef, resetRef, resetFinalized)

	// Cache must hold the reset crossSafe, not the pre-reset value. Get
	// re-validates canonicality against the EL; expect the lookup at the reset
	// block number to return the reset ref.
	mockEngine.ExpectL2BlockRefByNumber(resetRef.Number, resetRef, nil)
	got, ok := ec.crossSafeCache.Get(context.Background(), mockEngine,
		eth.L2BlockRef{Number: 200})
	require.True(t, ok, "forced reset must seed the cross-safe cache")
	require.Equal(t, resetRef, got,
		"forced reset must replace stale cached cross-safe with the reset value")
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
		finalizedL2HeadSource:     rollup.VerifierHeadPreActivation,
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

	// First call: Verified path populates the cache.
	mockEngine.ExpectL2BlockRefByHash(verifiedBlock.Hash, verifiedRef, nil)
	mockEngine.ExpectL2BlockRefByNumber(verifiedBlock.Number, verifiedRef, nil)
	_ = ec.SafeL2Head()

	// Simulate a reorg: the EL now reports a different canonical block at the
	// cached number. HoldPrevious must clear the cache and floor at finalized.
	sa.holdPreviousVerified = true
	mockEngine.ExpectL2BlockRefByNumber(verifiedBlock.Number,
		eth.L2BlockRef{Hash: common.Hash{0xdd}, Number: verifiedBlock.Number}, nil)
	got := ec.SafeL2Head()
	require.Equal(t, localFinalized, got, "non-canonical cache must clear and floor at finalized")
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

	got := ec.FinalizedHead()
	require.Equal(t, eth.L2BlockRef{}, got,
		"FinalizedHead on cold-start HoldPrevious must return zero L2BlockRef, not garbage")
	require.Equal(t, common.Hash{}, got.Hash,
		"resulting ForkchoiceUpdate sends a zero finalized hash, preserving the EL's own label")
}
