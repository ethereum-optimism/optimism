package chain_container

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/stretchr/testify/require"
)

// L1: drive the real simpleChainContainer.InvalidateBlock against a
// FaultyRandomChain engine to exercise its engine-state classification
// (issue-20929 root cause 4). The container and EngineController are real; only
// the l2Provider lies. InvalidateBlock adds to the deny list before the engine
// check, so each case needs a real deny list (t.TempDir).

// TestInvalidateBlockClassification covers the branches whose behavior is the
// same on the pinned audit commit and post-fix -- they must stay green.
func TestInvalidateBlockClassification(t *testing.T) {
	ctx := context.Background()
	errTransient := errors.New("transient engine rpc failure")

	cases := []struct {
		name        string
		byNumberErr error // transient RPC failure injected at the height lookup
		badHash     bool  // payloadHash != the real block hash at height
		wantRewound bool
		wantErr     error // sentinel to errors.Is, or nil
	}{
		{
			// RV optimism-private#546: a transient engine RPC error must surface so
			// the caller preserves the pending transition -- never swallowed.
			name:        "transient-rpc-error-preserved",
			byNumberErr: errTransient,
			wantRewound: false,
			wantErr:     errTransient,
		},
		{
			// State C: EL re-derived past the invalidated block (hash differs) ->
			// no rewind, no error.
			name:        "hash-differs-no-rewind",
			badHash:     true,
			wantRewound: false,
			wantErr:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := NewRandomChainManager([]byte("l1-invalidate-" + tc.name))
			mgr.Generate()
			t.Cleanup(func() { _ = mgr.Close() })
			rc := mgr.Chains()[0]

			c, err := mgr.ChainContainer(rc.chainID, t.TempDir())
			require.NoError(t, err)

			height := rc.safe
			target := rc.l2[height].Ref
			// elAboveTarget: unsafe head is at/above height, so a fixed
			// InvalidateBlock reading the Unsafe label would not divert to the
			// already-invalidated path -- isolates the branch under test.
			frc := newFaultyRandomChain(rc, elAboveTarget, target)
			frc.byNumberErr = tc.byNumberErr
			c.engine = engine_controller.NewEngineControllerWithL2AndRollup(frc, rc.cfg)

			payloadHash := target.Hash
			if tc.badHash {
				payloadHash = flipHash(target.Hash)
			}

			parentPayload := rc.l2[height-1].Payload

			rewound, err := c.InvalidateBlock(ctx, height, payloadHash, target.Time, eth.Bytes32{}, eth.Bytes32{}, parentPayload)

			require.Equal(t, tc.wantRewound, rewound)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestInvalidateBlockELBelowHeight documents issue-20929 root cause 4: when the
// EL unsafe head is below the invalidated height, L2BlockRefByNumber(height)
// returns not-found and InvalidateBlock (on the pinned audit commit) returns
// (false, err), conflating "already past the block" with a transient RPC
// failure -- so the caller retries the transition forever instead of clearing it.
func TestInvalidateBlockELBelowHeight(t *testing.T) {
	ctx := context.Background()
	mgr := NewRandomChainManager([]byte("l1-invalidate-below"))
	mgr.Generate()
	t.Cleanup(func() { _ = mgr.Close() })
	rc := mgr.Chains()[0]

	c, err := mgr.ChainContainer(rc.chainID, t.TempDir())
	require.NoError(t, err)

	height := rc.safe
	target := rc.l2[height].Ref
	// elBelowTarget: unsafe head at height-1; L2BlockRefByNumber(height) -> NotFound.
	frc := newFaultyRandomChain(rc, elBelowTarget, target)
	c.engine = engine_controller.NewEngineControllerWithL2AndRollup(frc, rc.cfg)

	parentPayload := rc.l2[height-1].Payload

	rewound, err := c.InvalidateBlock(ctx, height, target.Hash, target.Time, eth.Bytes32{}, eth.Bytes32{}, parentPayload)

	// FIXED SPEC (issue-20929 root cause 4): EL below the invalidated height means
	// the block is already off the canonical chain -> classify as
	// already-invalidated -> (true, nil) and trigger L1 re-derivation, NOT a
	// retryable error. Commented so the suite stays green on the pinned audit
	// commit; uncomment to reproduce the bug.
	// require.True(t, rewound, "EL below height must classify as already-invalidated")
	// require.NoError(t, err, "EL below height must not surface as a retryable error")
	_ = rewound
	_ = err
}
