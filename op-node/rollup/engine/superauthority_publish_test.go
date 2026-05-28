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

// newSAController builds an EngineController wired to a SuperAuthority for the
// FCU-level publish tests below.
func newSAController(t *testing.T, mockEngine *testutils.MockEngine, sa rollup.SuperAuthority) (*EngineController, *testutils.MockEmitter) {
	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics,
		&rollup.Config{}, &sync.Config{}, &testutils.MockL1Source{}, emitter, sa)
	return ec, emitter
}

// Fix 3 / Fix 7(a): published finalized must NOT rewind across successive
// resolves when the published safe legitimately lowers. The resolver may lower
// safe (non-monotonic), but the FCU-level guard in reconcilePublished must keep
// the published finalized at its prior height (and never below it).
func TestSuperAuthorityPublished_FinalizedDoesNotRewindWhenSafeLowers(t *testing.T) {
	mockEngine := &testutils.MockEngine{}
	sa := &mockSuperAuthority{}
	ec, _ := newSAController(t, mockEngine, sa)

	localSafe := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 200}
	ec.SetLocalSafeHead(localSafe)
	ec.SetFinalizedHead(eth.L2BlockRef{Hash: common.Hash{0xdd}, Number: 100})

	// First resolve: verified safe 150, verified finalized 100.
	safe1 := eth.L2BlockRef{Hash: common.Hash{0xb1}, Number: 150}
	fin1 := eth.L2BlockRef{Hash: common.Hash{0xc1}, Number: 100}
	sa.fullyVerifiedL2Head = safe1.ID()
	sa.fullyVerifiedL2HeadSource = rollup.VerifierHeadVerified
	sa.finalizedL2Head = fin1.ID()
	sa.finalizedL2HeadSource = rollup.VerifierHeadVerified
	mockEngine.ExpectL2BlockRefByHash(safe1.Hash, safe1, nil)
	mockEngine.ExpectL2BlockRefByNumber(safe1.Number, safe1, nil) // canonicality
	mockEngine.ExpectL2BlockRefByHash(fin1.Hash, fin1, nil)

	ec.refreshSuperAuthorityPublished()
	require.Equal(t, safe1, ec.SafeL2Head())
	require.Equal(t, fin1, ec.FinalizedHead())

	// Second resolve: safe legitimately lowers to 120 (a verified tip behind the
	// cache, adopted), finalized verifier now holds-previous (transient). The
	// published finalized must stay at 100, never rewind below it.
	safe2 := eth.L2BlockRef{Hash: common.Hash{0xb2}, Number: 120}
	sa.fullyVerifiedL2Head = safe2.ID()
	sa.fullyVerifiedL2HeadSource = rollup.VerifierHeadVerified
	sa.holdPreviousFinalized = true
	mockEngine.ExpectL2BlockRefByHash(safe2.Hash, safe2, nil)
	mockEngine.ExpectL2BlockRefByNumber(safe2.Number, safe2, nil)

	ec.refreshSuperAuthorityPublished()
	require.Equal(t, safe2, ec.SafeL2Head(), "safe lowered to the adopted verified tip")
	require.Equal(t, fin1, ec.FinalizedHead(),
		"published finalized must not rewind when safe lowers (still >= last published finalized)")
}

// Fix 2 / Fix 7(b): a single refresh resolves the pair ONCE and SafeL2Head /
// FinalizedHead both return that same consistent snapshot — finalized <= safe.
func TestSuperAuthorityPublished_FCUPublishesConsistentPairFromSingleSnapshot(t *testing.T) {
	mockEngine := &testutils.MockEngine{}
	sa := &mockSuperAuthority{}
	ec, _ := newSAController(t, mockEngine, sa)

	ec.SetLocalSafeHead(eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 200})
	ec.SetFinalizedHead(eth.L2BlockRef{Hash: common.Hash{0xdd}, Number: 100})

	safe := eth.L2BlockRef{Hash: common.Hash{0xb1}, Number: 80}
	fin := eth.L2BlockRef{Hash: common.Hash{0xc1}, Number: 80}
	sa.fullyVerifiedL2Head = safe.ID()
	sa.fullyVerifiedL2HeadSource = rollup.VerifierHeadVerified
	sa.finalizedL2Head = fin.ID()
	sa.finalizedL2HeadSource = rollup.VerifierHeadVerified

	// Exactly one resolve -> exactly one set of EL lookups.
	mockEngine.ExpectL2BlockRefByHash(safe.Hash, safe, nil)
	mockEngine.ExpectL2BlockRefByNumber(safe.Number, safe, nil)
	mockEngine.ExpectL2BlockRefByHash(fin.Hash, fin, nil)

	ec.refreshSuperAuthorityPublished()
	gotSafe := ec.SafeL2Head()
	gotFin := ec.FinalizedHead()
	require.Equal(t, safe, gotSafe)
	require.Equal(t, fin, gotFin)
	require.LessOrEqual(t, gotFin.Number, gotSafe.Number, "published finalized <= published safe")

	// Repeated reads do not trigger further EL lookups (no side effects); the
	// mock would panic on an unexpected call.
	require.Equal(t, gotSafe, ec.SafeL2Head())
	require.Equal(t, gotFin, ec.FinalizedHead())
	mockEngine.AssertExpectations(t)
}

// Fix 4 / Fix 7(c): on the resolver error/hold fallback the published pair must
// never have finalized > safe. We arrange a same-height/different-hash conflict
// (resolver errors) while the held caches would, if returned raw, have finalized
// ahead of safe. The fallback re-applies the clamp.
func TestSuperAuthorityPublished_ErrorFallbackNeverPublishesFinalizedAheadOfSafe(t *testing.T) {
	mockEngine := &testutils.MockEngine{}
	sa := &mockSuperAuthority{}
	ec, _ := newSAController(t, mockEngine, sa)

	ec.SetLocalSafeHead(eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 200})
	ec.SetFinalizedHead(eth.L2BlockRef{Hash: common.Hash{0xff}, Number: 200})

	// Per-head caches deliberately VIOLATE finalized<=safe (safe 50, finalized
	// 80). The OLD error path returned these raw and could publish finalized >
	// safe. The published pair is the last CONSISTENT pair (safe 80, finalized
	// 80).
	ec.superAuthoritySafeHead = eth.L2BlockRef{Hash: common.Hash{0x51}, Number: 50}
	ec.superAuthorityFinalizedHead = eth.L2BlockRef{Hash: common.Hash{0x81}, Number: 80}
	ec.superAuthorityPublishedSafeHead = eth.L2BlockRef{Hash: common.Hash{0x80}, Number: 80}
	ec.superAuthorityPublishedFinalizedHead = eth.L2BlockRef{Hash: common.Hash{0x81}, Number: 80}

	// Force a resolver error: verified finalized at the cache height but a
	// different hash -> errCrossHeadConflict -> error fallback.
	conflict := eth.L2BlockRef{Hash: common.Hash{0x82}, Number: 80}
	sa.finalizedL2Head = conflict.ID()
	sa.finalizedL2HeadSource = rollup.VerifierHeadVerified
	mockEngine.ExpectL2BlockRefByHash(conflict.Hash, conflict, nil)

	ec.refreshSuperAuthorityPublished()
	require.LessOrEqual(t, ec.FinalizedHead().Number, ec.SafeL2Head().Number,
		"error fallback must hold the last CONSISTENT published pair, never the raw per-head caches (safe 50 < finalized 80)")
}

// Fix 5 / Fix 7(d): a use-local resolve followed by a hold-previous resolve must
// not rewind the published finalized. The first resolve publishes the local head
// and records it in the cache; the second (verifier transiently unavailable)
// holds that cached head rather than a stale/zero value.
func TestSuperAuthorityPublished_UseLocalThenHoldPreviousDoesNotRewindFinalized(t *testing.T) {
	mockEngine := &testutils.MockEngine{}
	sa := &mockSuperAuthority{}
	ec, _ := newSAController(t, mockEngine, sa)

	localSafe := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100}
	localFinalized := eth.L2BlockRef{Hash: common.Hash{0xdd}, Number: 70}
	ec.SetLocalSafeHead(localSafe)
	ec.SetFinalizedHead(localFinalized)

	// First resolve: both pre-activation -> use-local. Publishes local heads and
	// records them in the caches.
	sa.fullyVerifiedL2HeadSource = rollup.VerifierHeadPreActivation
	sa.finalizedL2HeadSource = rollup.VerifierHeadPreActivation
	ec.refreshSuperAuthorityPublished()
	require.Equal(t, localSafe, ec.SafeL2Head())
	require.Equal(t, localFinalized, ec.FinalizedHead())
	require.Equal(t, localFinalized, ec.superAuthorityFinalizedHead,
		"use-local must record the published finalized in the cache (Fix 5)")

	// Second resolve: verifier transiently unavailable for both -> hold-previous.
	// Must hold the previously published heads, NOT rewind to zero.
	sa.holdPreviousVerified = true
	sa.holdPreviousFinalized = true
	ec.refreshSuperAuthorityPublished()
	require.Equal(t, localSafe, ec.SafeL2Head(),
		"hold-previous must not rewind published safe after a use-local resolve")
	require.Equal(t, localFinalized, ec.FinalizedHead(),
		"hold-previous must not rewind published finalized after a use-local resolve")
}

// guard: the error fallback also bumps the conflict metric (non-fatal). We use a
// counting metricer to confirm Fix 6 wiring fires on the resolver error path.
func TestSuperAuthorityPublished_ConflictRecordsMetric(t *testing.T) {
	mockEngine := &testutils.MockEngine{}
	sa := &mockSuperAuthority{}
	cm := newCountingMetricer()
	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), cm,
		&rollup.Config{}, &sync.Config{}, &testutils.MockL1Source{}, emitter, sa)

	ec.SetLocalSafeHead(eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 200})
	ec.SetFinalizedHead(eth.L2BlockRef{Hash: common.Hash{0xff}, Number: 60})
	cached := eth.L2BlockRef{Hash: common.Hash{0xdd}, Number: 50}
	ec.superAuthorityFinalizedHead = cached
	ec.superAuthorityPublishedFinalizedHead = cached
	ec.superAuthoritySafeHead = eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 200}
	ec.superAuthorityPublishedSafeHead = eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 200}

	conflict := eth.L2BlockRef{Hash: common.Hash{0xee}, Number: 50}
	sa.finalizedL2Head = conflict.ID()
	sa.finalizedL2HeadSource = rollup.VerifierHeadVerified
	mockEngine.ExpectL2BlockRefByHash(conflict.Hash, conflict, nil)

	ec.refreshSuperAuthorityPublished()
	require.GreaterOrEqual(t, cm.conflicts, 1, "resolver conflict must increment the conflict metric (Fix 6)")
	require.Equal(t, cached, ec.FinalizedHead(), "published finalized held on conflict")
}

// countingMetricer wraps NoopMetrics and counts SuperAuthority conflict events.
type countingMetricer struct {
	metrics.Metricer
	conflicts int
}

func newCountingMetricer() *countingMetricer {
	return &countingMetricer{Metricer: metrics.NoopMetrics}
}

func (c *countingMetricer) RecordSuperAuthorityForkchoiceConflict() {
	c.conflicts++
}
