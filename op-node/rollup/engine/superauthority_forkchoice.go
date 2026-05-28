package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SuperAuthority forkchoice resolution
//
// When a SuperAuthority (interop verifier) is configured, the published safe
// and finalized heads sent to the EL are NOT a direct read of either the local
// or the cross layer. They are a reconciliation of the two that must uphold a
// single coupled invariant: finalized <= safe. Historically safe and finalized
// were resolved in two independent code paths, and because the invariant
// couples them, the split caused a class of bugs (finalized pinned at genesis;
// finalized ahead of safe).
//
// This file resolves both heads TOGETHER in two steps:
//
//  1. GATHER (superAuthorityForkchoice.gather): turn each SuperAuthority head
//     signal (VerifierHead + ok) into a concrete candidate block, using the EL
//     for canonicality / anchor-timestamp->block resolution and bounding each
//     candidate by the corresponding LOCAL head. Any lookup or canonicality
//     failure, or an ok=false (HoldPrevious) signal, produces a "hold previous"
//     candidate. We NEVER fall back to local-safe or floor finalized on such a
//     failure.
//
//  2. RESOLVE (resolveCrossHeads): a PURE function that takes the two
//     candidates plus the cached cross heads and returns the resolved
//     {safe, finalized} together, enforcing every invariant in one place.

// crossSourceKind classifies what a SuperAuthority head signal resolved to
// during the gather step.
type crossSourceKind uint8

const (
	// crossUseLocal: the verifier is pre-activation for this head; the published
	// head is the local head (already captured as the candidate ref).
	crossUseLocal crossSourceKind = iota
	// crossHoldPrevious: the head could not be resolved (verifier read failure,
	// EL lookup failure, reorg signal, non-canonical). Hold the cached value.
	crossHoldPrevious
	// crossVerified: a concrete verified tip resolved to a canonical EL block.
	crossVerified
	// crossAnchor: the verifier has no entry yet; the candidate is the canonical
	// block at the pre-activation cap timestamp. This is a WEAK signal that must
	// not rewind a stronger cached value.
	crossAnchor
)

func (k crossSourceKind) String() string {
	switch k {
	case crossUseLocal:
		return "use-local"
	case crossHoldPrevious:
		return "hold-previous"
	case crossVerified:
		return "verified"
	case crossAnchor:
		return "anchor"
	default:
		return fmt.Sprintf("unknown(%d)", uint8(k))
	}
}

// headCandidate is the concrete result of gathering one SuperAuthority head
// signal: a resolved block ref together with the kind of source that produced
// it. For crossHoldPrevious the ref is the zero value.
type headCandidate struct {
	kind crossSourceKind
	ref  eth.L2BlockRef
}

// crossHeads is the resolved, mutually-consistent pair sent to the EL.
type crossHeads struct {
	safe      eth.L2BlockRef
	finalized eth.L2BlockRef
}

// superAuthorityForkchoice gathers SuperAuthority head signals into concrete
// candidates. It is stateless apart from its collaborators; all invariant logic
// lives in the pure resolveCrossHeads.
type superAuthorityForkchoice struct {
	log       log.Logger
	rollupCfg *rollup.Config
	engine    ExecEngine
}

// gatherSafe turns the SuperAuthority's FullyVerifiedL2Head signal into a safe
// candidate, bounded by localSafe.
func (r *superAuthorityForkchoice) gatherSafe(ctx context.Context, head rollup.VerifierHead, ok bool, localSafe eth.L2BlockRef) headCandidate {
	return r.gather(ctx, head, ok, localSafe, "safe")
}

// gatherFinalized turns the SuperAuthority's FinalizedL2Head signal into a
// finalized candidate, bounded by localFinalized.
func (r *superAuthorityForkchoice) gatherFinalized(ctx context.Context, head rollup.VerifierHead, ok bool, localFinalized eth.L2BlockRef) headCandidate {
	return r.gather(ctx, head, ok, localFinalized, "finalized")
}

// gather resolves a single VerifierHead+ok into a concrete headCandidate,
// bounded by localBound (the corresponding local head). A read/lookup failure,
// non-canonical block, or ok=false all map to crossHoldPrevious — never to
// local-safe or a finalized floor.
func (r *superAuthorityForkchoice) gather(ctx context.Context, head rollup.VerifierHead, ok bool, localBound eth.L2BlockRef, which string) headCandidate {
	if !ok {
		return headCandidate{kind: crossHoldPrevious}
	}
	switch head.Source {
	case rollup.VerifierHeadPreActivation:
		return headCandidate{kind: crossUseLocal, ref: localBound}
	case rollup.VerifierHeadAnchor:
		return r.gatherAnchor(ctx, head.Timestamp, localBound, which)
	case rollup.VerifierHeadVerified:
		// Safe heads can reorg, so we re-check canonicality against the EL.
		// Finalized heads cannot reorg, so a hash lookup is sufficient.
		return r.gatherVerified(ctx, head.Block, localBound, which, which == "safe")
	default:
		r.log.Error("unhandled VerifierHeadSource; holding previous", "which", which, "source", head.Source)
		return headCandidate{kind: crossHoldPrevious}
	}
}

// gatherVerified resolves a verified tip to a canonical EL block, bounded by the
// local head. A block ahead of the local bound, an unknown block, a lookup
// failure, or a non-canonical block all hold previous.
func (r *superAuthorityForkchoice) gatherVerified(ctx context.Context, block eth.BlockID, localBound eth.L2BlockRef, which string, checkCanonical bool) headCandidate {
	if block.Number > localBound.Number {
		r.log.Warn("super authority head ahead of local head, using local",
			"which", which, "super_authority", block, "local", localBound)
		return headCandidate{kind: crossUseLocal, ref: localBound}
	}
	br, err := r.engine.L2BlockRefByHash(ctx, block.Hash)
	if err != nil {
		r.log.Warn("super authority head unknown to engine (reorg signal); holding previous",
			"which", which, "super_authority", block, "err", err)
		return headCandidate{kind: crossHoldPrevious}
	}
	if !checkCanonical {
		return headCandidate{kind: crossVerified, ref: br}
	}
	if canonical, canonicalRef, err := r.isCanonical(ctx, br); err != nil {
		r.log.Warn("cannot verify super authority head canonicality; holding previous",
			"which", which, "super_authority", br, "err", err)
		return headCandidate{kind: crossHoldPrevious}
	} else if !canonical {
		r.log.Warn("super authority head non-canonical (reorg signal); holding previous",
			"which", which, "super_authority", br, "canonical", canonicalRef)
		return headCandidate{kind: crossHoldPrevious}
	}
	return headCandidate{kind: crossVerified, ref: br}
}

// gatherAnchor resolves the canonical L2 block at the pre-activation cap
// timestamp, bounded by the local head. The block is canonical by construction
// (looked up by number). A target-number or lookup failure holds previous.
func (r *superAuthorityForkchoice) gatherAnchor(ctx context.Context, ts uint64, localBound eth.L2BlockRef, which string) headCandidate {
	num, err := r.rollupCfg.TargetBlockNumber(ts)
	if err != nil {
		r.log.Warn("cannot compute anchor block number; holding previous", "which", which, "ts", ts, "err", err)
		return headCandidate{kind: crossHoldPrevious}
	}
	if num > localBound.Number {
		// Local head hasn't reached the anchor block yet; use the local head.
		return headCandidate{kind: crossUseLocal, ref: localBound}
	}
	br, err := r.engine.L2BlockRefByNumber(ctx, num)
	if err != nil {
		r.log.Warn("cannot resolve anchor block; holding previous", "which", which, "ts", ts, "num", num, "err", err)
		return headCandidate{kind: crossHoldPrevious}
	}
	return headCandidate{kind: crossAnchor, ref: br}
}

// isCanonical reports whether `target` still matches the EL's canonical chain at
// its number.
func (r *superAuthorityForkchoice) isCanonical(ctx context.Context, target eth.L2BlockRef) (ok bool, canonical eth.L2BlockRef, err error) {
	canonical, err = r.engine.L2BlockRefByNumber(ctx, target.Number)
	if err != nil {
		return false, eth.L2BlockRef{}, err
	}
	return canonical.Hash == target.Hash, canonical, nil
}

// errCrossHeadConflict signals genuinely inconsistent state: a freshly resolved
// head conflicts with a cached head at the same height but a different hash.
var errCrossHeadConflict = errors.New("super authority head conflicts with cached head at same height")

// resolveCrossHeads is the single pure function that reconciles the gathered
// safe and finalized candidates with the cached cross heads, enforcing ALL
// invariants in one place:
//
//   - hold-previous on a zero/hold candidate: keep the cached value.
//   - never rewind below the cached value (anchor/pre-activation are WEAK and
//     must not rewind a stronger cached value; verified IS authoritative and may
//     sit behind a startup-seeded cache).
//   - finalized <= safe: clamp finalized down to safe.
//
// It returns the resolved heads and the new cached values to persist. A
// same-height/different-hash conflict against the cache returns
// errCrossHeadConflict rather than panicking.
func resolveCrossHeads(safeCand, finalizedCand headCandidate, cachedSafe, cachedFinalized eth.L2BlockRef) (resolved crossHeads, newCachedSafe, newCachedFinalized eth.L2BlockRef, err error) {
	// Safe and finalized differ on how a verified tip relates to its cache:
	//   - safe: the cache is seeded from local-safe at startup, so an
	//     authoritative verified tip BEHIND the cache is the true cross-safe and
	//     must be adopted (lowered). Hence non-monotonic.
	//   - finalized: the cache is the last verified finalized and finalized
	//     cannot reorg, so a verified tip behind the cache is a regression and
	//     must hold the cache. Hence monotonic.
	safe, newCachedSafe, err := resolveOne(safeCand, cachedSafe, false)
	if err != nil {
		return crossHeads{}, eth.L2BlockRef{}, eth.L2BlockRef{}, fmt.Errorf("resolving safe: %w", err)
	}
	finalized, newCachedFinalized, err := resolveOne(finalizedCand, cachedFinalized, true)
	if err != nil {
		return crossHeads{}, eth.L2BlockRef{}, eth.L2BlockRef{}, fmt.Errorf("resolving finalized: %w", err)
	}

	// Couple the two heads: finalized must never exceed safe. Clamp finalized
	// down to safe rather than publishing finalized > safe. We do not raise the
	// cache here — the clamp is a per-publish bound, not a regression of what
	// the verifier proved.
	//
	// NOTE: this clamp is intentionally only a per-resolve bound. The
	// FCU-level monotonicity guard (published finalized never rewinds below the
	// last published finalized) lives in EngineController.reconcilePublished,
	// because it depends on the last PUBLISHED pair, which the pure resolver
	// does not see. Both layers re-apply the clamp so the published pair is
	// consistent even on the error/hold-previous fallback.
	return crossHeads{safe: safe, finalized: finalized}.clampFinalizedToSafe(), newCachedSafe, newCachedFinalized, nil
}

// clampFinalizedToSafe returns h with finalized clamped down to safe so the
// finalized <= safe invariant always holds. A zero safe leaves finalized
// untouched (there is nothing to clamp against). Used by both the pure resolver
// and the error/hold-previous fallback so neither can emit finalized > safe.
func (h crossHeads) clampFinalizedToSafe() crossHeads {
	if h.safe != (eth.L2BlockRef{}) && h.finalized.Number > h.safe.Number {
		h.finalized = h.safe
	}
	return h
}

// resolveOne reconciles a single candidate against its cache, returning the head
// to publish and the cache value to persist.
//
// The cache tracks the last PUBLISHED cross head so that a later "hold previous"
// returns the value we actually published rather than a stale/zero cache. It is
// updated for "use local" (recording the published local head) and for verified/
// anchor adoptions. It is NOT raised on "hold previous" (the cache is reused
// verbatim).
func resolveOne(cand headCandidate, cached eth.L2BlockRef, monotonic bool) (head, newCached eth.L2BlockRef, err error) {
	switch cand.kind {
	case crossHoldPrevious:
		// Use the cache verbatim; never regress to local or to a floor.
		return cached, cached, nil

	case crossUseLocal:
		// Pre-activation / clamp-to-local: publish the local head but never
		// rewind a non-zero cross cache below it. Record the published head in
		// the cache so a subsequent hold-previous cannot rewind below what we
		// just published (the use-local -> hold-previous gap). For finalized
		// (monotonic) only advance the cache, never lower it.
		if cached != (eth.L2BlockRef{}) && cand.ref.Number < cached.Number {
			return cached, cached, nil
		}
		return cand.ref, cand.ref, nil

	case crossAnchor:
		// Anchor is a WEAK pre-verification signal: it must not rewind a
		// stronger cached value, but while it is the strongest signal we have it
		// is recorded so a later transient failure can hold it.
		return adoptIfAhead(cand.ref, cached)

	case crossVerified:
		if cached != (eth.L2BlockRef{}) {
			if cand.ref.ID() == cached.ID() {
				return cached, cached, nil
			}
			if cand.ref.Number == cached.Number {
				return eth.L2BlockRef{}, eth.L2BlockRef{}, errCrossHeadConflict
			}
			if monotonic && cand.ref.Number < cached.Number {
				// Finalized cannot reorg: a verified tip behind the cache is a
				// regression. Hold the cache.
				return cached, cached, nil
			}
		}
		// Verified is the authoritative cross signal. For safe it may
		// legitimately sit behind a startup-seeded cache (cache=local at
		// startup, then the verifier reports its true, lower cross tip), so we
		// adopt and lower it.
		return cand.ref, cand.ref, nil

	default:
		return eth.L2BlockRef{}, eth.L2BlockRef{}, fmt.Errorf("unhandled crossSourceKind %v", cand.kind)
	}
}

// adoptIfAhead adopts cand (publishing and caching it) only when it does not
// rewind a non-zero cache. A same-height/different-hash conflict against the
// cache is inconsistent state.
func adoptIfAhead(cand, cached eth.L2BlockRef) (head, newCached eth.L2BlockRef, err error) {
	if cached == (eth.L2BlockRef{}) {
		return cand, cand, nil
	}
	if cand.ID() == cached.ID() {
		return cached, cached, nil
	}
	if cand.Number < cached.Number {
		// Would rewind; hold the cache.
		return cached, cached, nil
	}
	if cand.Number == cached.Number {
		return eth.L2BlockRef{}, eth.L2BlockRef{}, errCrossHeadConflict
	}
	return cand, cand, nil
}
