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

// TestSafeL2Head_EmptyVerifier_FloorsAtFinalized exercises bug B.
//
// Under the previous boolean contract, an active verifier with no entries for
// this chain returned (BlockID{}, false), and engine_controller.SafeL2Head
// resolved the empty BlockID to L2 genesis via L2BlockRefByNumber(0). That
// dropped cross-safe to genesis once the chain hadn't yet been covered by the
// verifier's depset — see ethereum-optimism/optimism#20944.
//
// Under the tri-state contract, that scenario surfaces as Source = Anchor.
// SafeL2Head handles the anchor conservatively by flooring at FinalizedHead,
// never local-safe and never a synthetic genesis lookup.
func TestSafeL2Head_EmptyVerifier_FloorsAtFinalized(t *testing.T) {
	localSafe := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 600}
	localFinalized := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 50}

	cfg := &rollup.Config{BlockTime: 2}
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

	got := ec.SafeL2Head()
	require.NotEqual(t, uint64(0), got.Number,
		"SafeL2Head must not drop to genesis when local-safe (%d) and local-finalized (%d) are non-zero. "+
			"Pre-fix, empty verifier returned (BlockID{}, false) and SafeL2Head fetched L2BlockRefByNumber(0). "+
			"Post-fix, Anchor source floors at FinalizedHead (ethereum-optimism/optimism#20944).",
		localSafe.Number, localFinalized.Number)
	require.Equal(t, localFinalized, got, "SafeL2Head must floor at FinalizedHead for anchor results")
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

// TestSafeL2Head_HoldPrevious_UsesFinalized verifies the simplified
// conservative fallback: on HoldPrevious, return FinalizedHead rather than
// trying to preserve the previous cross-safe head with an extra cache.
func TestSafeL2Head_HoldPrevious_UsesFinalized(t *testing.T) {
	localSafe := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100}
	localFinalized := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 50}
	verifiedBlock := eth.BlockID{Hash: common.Hash{0xcc}, Number: 80}
	verifiedRef := eth.L2BlockRef{Hash: verifiedBlock.Hash, Number: verifiedBlock.Number}

	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	sa := &mockSuperAuthority{
		fullyVerifiedL2Head:       verifiedBlock,
		fullyVerifiedL2HeadSource: rollup.VerifierHeadVerified,
		// Finalized stays PreActivation so the floor resolves to localFinalized.
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

	// First call: verified path returns cross-safe.
	mockEngine.ExpectL2BlockRefByNumber(verifiedBlock.Number, verifiedRef, nil)
	got := ec.SafeL2Head()
	require.Equal(t, verifiedRef, got, "first call should resolve via the Verified path")

	// Verifier now returns HoldPrevious; the simplified path falls back to
	// finalized directly, without introducing a separate cross-safe cache.
	sa.holdPreviousVerified = true
	got = ec.SafeL2Head()
	require.Equal(t, localFinalized, got, "HoldPrevious must fall back to FinalizedHead")
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
