package engine

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func ref(n uint64, h byte) eth.L2BlockRef {
	if n == 0 && h == 0 {
		return eth.L2BlockRef{}
	}
	return eth.L2BlockRef{Hash: common.Hash{h}, Number: n}
}

// TestResolveCrossHeads exhaustively table-tests the pure resolver. Each of the
// four bug scenarios that motivated the refactor is a table row; with the
// coupled resolver each one is a one-liner.
func TestResolveCrossHeads(t *testing.T) {
	tests := []struct {
		name            string
		safeCand        headCandidate
		finalizedCand   headCandidate
		cachedSafe      eth.L2BlockRef
		cachedFinalized eth.L2BlockRef

		wantSafe            eth.L2BlockRef
		wantFinalized       eth.L2BlockRef
		wantCachedSafe      eth.L2BlockRef
		wantCachedFinalized eth.L2BlockRef
		wantErr             bool
	}{
		{
			// BUG 1 — finalized pinned at genesis. local finalized is 0 (use-local),
			// cross-safe is a real verified tip. Published finalized follows local
			// (0) and does NOT get dragged up; published safe is the verified tip.
			name:          "bug1: local-finalized zero with verified cross-safe -> finalized 0",
			safeCand:      headCandidate{kind: crossVerified, ref: ref(44643, 0xaa)},
			finalizedCand: headCandidate{kind: crossUseLocal, ref: ref(0, 0)},
			wantSafe:      ref(44643, 0xaa),
			wantFinalized: ref(0, 0),
			// safe verified adopts and caches; finalized use-local records the
			// published head, which here is zero (local-finalized is genesis).
			wantCachedSafe:      ref(44643, 0xaa),
			wantCachedFinalized: ref(0, 0),
		},
		{
			// BUG 2 — verifier cold start. Both signals hold previous (cold-start
			// returns unavailable, NOT an anchor). The resolver returns the cached
			// values verbatim, preserving the EL's persisted heads.
			name:                "bug2: cold-start hold-previous -> cached values",
			safeCand:            headCandidate{kind: crossHoldPrevious},
			finalizedCand:       headCandidate{kind: crossHoldPrevious},
			cachedSafe:          ref(80, 0xbb),
			cachedFinalized:     ref(50, 0xcc),
			wantSafe:            ref(80, 0xbb),
			wantFinalized:       ref(50, 0xcc),
			wantCachedSafe:      ref(80, 0xbb),
			wantCachedFinalized: ref(50, 0xcc),
		},
		{
			// BUG 3 — finalized ahead of safe. The resolver clamps finalized DOWN
			// to safe (it must never publish finalized > safe).
			name:                "bug3: finalized ahead of safe -> clamp finalized to safe",
			safeCand:            headCandidate{kind: crossVerified, ref: ref(50, 0xaa)},
			finalizedCand:       headCandidate{kind: crossVerified, ref: ref(100, 0xdd)},
			wantSafe:            ref(50, 0xaa),
			wantFinalized:       ref(50, 0xaa),
			wantCachedSafe:      ref(50, 0xaa),
			wantCachedFinalized: ref(100, 0xdd), // cache records what the verifier proved
		},
		{
			// BUG 4 — clamp finalized to verified safe (now resolver-owned). A
			// finalized anchor while safe is a verified tip resolves to: safe is
			// the verified tip, finalized is the anchor block, both within the
			// coupled bound.
			name:                "bug4: finalized anchor with verified safe -> bounded by safe",
			safeCand:            headCandidate{kind: crossVerified, ref: ref(60, 0xaa)},
			finalizedCand:       headCandidate{kind: crossAnchor, ref: ref(40, 0xee)},
			wantSafe:            ref(60, 0xaa),
			wantFinalized:       ref(40, 0xee),
			wantCachedSafe:      ref(60, 0xaa),
			wantCachedFinalized: ref(40, 0xee),
		},
		{
			// Anchor must not rewind a stronger cached finalized.
			name:                "anchor behind cache does not regress finalized",
			safeCand:            headCandidate{kind: crossHoldPrevious},
			finalizedCand:       headCandidate{kind: crossAnchor, ref: ref(499, 0xa1)},
			cachedSafe:          ref(2000, 0xaa),
			cachedFinalized:     ref(1000, 0xdd),
			wantSafe:            ref(2000, 0xaa),
			wantFinalized:       ref(1000, 0xdd),
			wantCachedSafe:      ref(2000, 0xaa),
			wantCachedFinalized: ref(1000, 0xdd),
		},
		{
			// Verified safe behind a startup-seeded cache IS adopted (lowered):
			// the cache was seeded from local-safe and the verifier reports the
			// true, lower cross-safe.
			name:           "verified safe behind startup-seeded cache is adopted",
			safeCand:       headCandidate{kind: crossVerified, ref: ref(80, 0xbb)},
			finalizedCand:  headCandidate{kind: crossUseLocal, ref: ref(50, 0xcc)},
			cachedSafe:     ref(120, 0xaa), // seeded from local-safe at startup
			wantSafe:       ref(80, 0xbb),
			wantFinalized:  ref(50, 0xcc),
			wantCachedSafe: ref(80, 0xbb),
			// use-local now records the published head in the cache (Fix 5) so a
			// subsequent hold-previous cannot rewind below it.
			wantCachedFinalized: ref(50, 0xcc),
		},
		{
			// Verified finalized behind cache holds the cache (finalized cannot
			// reorg; monotonic).
			name:                "verified finalized behind cache holds cache",
			safeCand:            headCandidate{kind: crossVerified, ref: ref(100, 0xaa)},
			finalizedCand:       headCandidate{kind: crossVerified, ref: ref(40, 0xbb)},
			cachedFinalized:     ref(50, 0xdd),
			wantSafe:            ref(100, 0xaa),
			wantFinalized:       ref(50, 0xdd),
			wantCachedSafe:      ref(100, 0xaa),
			wantCachedFinalized: ref(50, 0xdd),
		},
		{
			// Fix 5: use-local records the published head in the cache, so a
			// subsequent resolve that holds-previous returns that head rather than
			// a stale/zero cache. This row is the SECOND resolve (the cache here is
			// what the prior use-local resolve persisted).
			name:                "use-local-then-hold-previous: hold-previous returns cached use-local head",
			safeCand:            headCandidate{kind: crossHoldPrevious},
			finalizedCand:       headCandidate{kind: crossHoldPrevious},
			cachedSafe:          ref(100, 0xaa), // persisted by a prior use-local resolve
			cachedFinalized:     ref(30, 0xdd),  // persisted by a prior use-local resolve
			wantSafe:            ref(100, 0xaa),
			wantFinalized:       ref(30, 0xdd),
			wantCachedSafe:      ref(100, 0xaa),
			wantCachedFinalized: ref(30, 0xdd),
		},
		{
			// Finalized (monotonic) same-height/different-hash against the cache is
			// genuinely inconsistent state — finalized cannot reorg: error.
			name:            "monotonic finalized same-height conflict against cache errors",
			safeCand:        headCandidate{kind: crossVerified, ref: ref(50, 0xaa)},
			finalizedCand:   headCandidate{kind: crossVerified, ref: ref(40, 0xbb)},
			cachedSafe:      ref(50, 0xaa),
			cachedFinalized: ref(40, 0xdd),
			wantErr:         true,
		},
		{
			// Safe (non-monotonic) same-height/different-hash against the cache is a
			// legitimate same-height reorg. gatherVerified already re-confirmed the
			// candidate is canonical, so it is adopted (published and cached), not
			// errored.
			name:                "non-monotonic safe same-height reorg adopts candidate",
			safeCand:            headCandidate{kind: crossVerified, ref: ref(50, 0xbb)},
			finalizedCand:       headCandidate{kind: crossHoldPrevious},
			cachedSafe:          ref(50, 0xcc),
			cachedFinalized:     ref(40, 0xdd),
			wantSafe:            ref(50, 0xbb),
			wantFinalized:       ref(40, 0xdd),
			wantCachedSafe:      ref(50, 0xbb),
			wantCachedFinalized: ref(40, 0xdd),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, newCachedSafe, newCachedFinalized, err := resolveCrossHeads(
				tt.safeCand, tt.finalizedCand, tt.cachedSafe, tt.cachedFinalized)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantSafe, resolved.safe, "safe")
			require.Equal(t, tt.wantFinalized, resolved.finalized, "finalized")
			require.Equal(t, tt.wantCachedSafe, newCachedSafe, "cachedSafe")
			require.Equal(t, tt.wantCachedFinalized, newCachedFinalized, "cachedFinalized")
			// Invariant: finalized never exceeds safe.
			require.LessOrEqual(t, resolved.finalized.Number, resolved.safe.Number,
				"invariant finalized <= safe")
		})
	}
}
