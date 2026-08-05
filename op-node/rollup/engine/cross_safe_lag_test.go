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

func newLagTestController(t *testing.T, sa rollup.SuperAuthority, mockEngine *testutils.MockEngine) *EngineController {
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

// TestCrossSafeLagExceeded_Disabled: maxLag 0 disables the gate regardless of
// how far local-safe has run ahead. No verifier or EL reads are made.
func TestCrossSafeLagExceeded_Disabled(t *testing.T) {
	sa := &mockSuperAuthority{
		fullyVerifiedL2Head:       eth.BlockID{Hash: common.Hash{0xcc}, Number: 10},
		fullyVerifiedL2HeadSource: rollup.VerifierHeadVerified,
	}
	ec := newLagTestController(t, sa, &testutils.MockEngine{})
	ec.SetLocalSafeHead(eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100_000})
	require.False(t, ec.CrossSafeLagExceeded(0))
}

// TestCrossSafeLagExceeded_NoSuperAuthority: without a registered verifier
// (standalone op-node) the gate is inert.
func TestCrossSafeLagExceeded_NoSuperAuthority(t *testing.T) {
	ec := newLagTestController(t, nil, &testutils.MockEngine{})
	ec.SetLocalSafeHead(eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100_000})
	require.False(t, ec.CrossSafeLagExceeded(50))
}

// TestCrossSafeLagExceeded_VerifiedBoundary: with a verified cross-safe head,
// the gate engages strictly beyond crossSafe+maxLag and not at the boundary.
func TestCrossSafeLagExceeded_VerifiedBoundary(t *testing.T) {
	verifiedBlock := eth.BlockID{Hash: common.Hash{0xcc}, Number: 100}
	verifiedRef := eth.L2BlockRef{Hash: verifiedBlock.Hash, Number: verifiedBlock.Number}

	for _, tc := range []struct {
		name      string
		localSafe uint64
		exceeded  bool
	}{
		{"at_window", 150, false},
		{"beyond_window", 151, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockEngine := &testutils.MockEngine{}
			sa := &mockSuperAuthority{
				fullyVerifiedL2Head:       verifiedBlock,
				fullyVerifiedL2HeadSource: rollup.VerifierHeadVerified,
				finalizedL2HeadSource:     rollup.VerifierHeadPreActivation,
			}
			ec := newLagTestController(t, sa, mockEngine)
			ec.SetLocalSafeHead(eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: tc.localSafe})

			// Verified branch of SafeL2Head validates against the EL.
			mockEngine.ExpectL2BlockRefByHash(verifiedBlock.Hash, verifiedRef, nil)
			mockEngine.ExpectL2BlockRefByNumber(verifiedBlock.Number, verifiedRef, nil)

			require.Equal(t, tc.exceeded, ec.CrossSafeLagExceeded(50))
		})
	}
}

// TestCrossSafeLagExceeded_Anchor: a fresh verifier DB (Anchor source) gates
// relative to the resolved activation-anchor block, so a cold-start backfill
// cannot sprint past anchor+window before verification begins.
func TestCrossSafeLagExceeded_Anchor(t *testing.T) {
	// BlockTime 2, genesis time 0: cap timestamp 999 -> anchor block 499.
	anchorRef := eth.L2BlockRef{Hash: common.Hash{0xa1}, Number: 499}

	mockEngine := &testutils.MockEngine{}
	sa := &mockSuperAuthority{
		fullyVerifiedL2HeadSource: rollup.VerifierHeadAnchor,
		fullyVerifiedTimestamp:    999,
	}
	ec := newLagTestController(t, sa, mockEngine)
	ec.SetLocalSafeHead(eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 600})

	mockEngine.ExpectL2BlockRefByNumber(anchorRef.Number, anchorRef, nil)
	require.True(t, ec.CrossSafeLagExceeded(50), "local-safe 600 vs anchor 499 exceeds window 50")

	mockEngine.ExpectL2BlockRefByNumber(anchorRef.Number, anchorRef, nil)
	require.False(t, ec.CrossSafeLagExceeded(101), "local-safe 600 vs anchor 499 within window 101")
}

// TestCrossSafeLagExceeded_VerifierOutage_StaysGated: on a transient verifier
// read failure, SafeL2Head floors at finalized (no cache populated), so a
// node far ahead of verification stays gated during a verifier outage rather
// than resuming its sprint exactly when verification is blind.
func TestCrossSafeLagExceeded_VerifierOutage_StaysGated(t *testing.T) {
	mockEngine := &testutils.MockEngine{}
	sa := &mockSuperAuthority{
		holdPreviousVerified:  true,
		finalizedL2HeadSource: rollup.VerifierHeadPreActivation,
	}
	ec := newLagTestController(t, sa, mockEngine)
	ec.SetLocalSafeHead(eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 1000})
	ec.SetFinalizedHead(eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 100})

	require.True(t, ec.CrossSafeLagExceeded(50),
		"verifier outage must not release the gate: fallback floors at finalized (100), local-safe 1000 exceeds 100+50")
}
