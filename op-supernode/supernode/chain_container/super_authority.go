package chain_container

import (
	"context"
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum/go-ethereum/common"
)

// FullyVerifiedL2Head reports the cross-verified safe L2 head as a tri-state.
//
// Returns one of:
//   - (VerifierHead{Source: PreActivation}, Ok): no verifiers registered, or
//     every registered verifier is still inactive at the current local-safe
//     timestamp. Caller uses local-safe.
//   - (VerifierHead{Source: Anchor, Block: anchor}, Ok): at least one active
//     verifier has no verified-DB entry for this chain yet. The Block is the
//     per-(chain, verifier) activation-anchor block (the L2 block at
//     timestamp `verifier.ActivationTimestamp() - 1`), selected as the oldest
//     contribution across active verifiers.
//   - (VerifierHead{Source: Verified, Block: tip}, Ok): all active verifiers
//     have verified tips; Block is the oldest tip across them.
//   - (VerifierHead{}, HoldPrevious): a verifier read failed transiently. The
//     caller must not advance and must not fall back to local-safe; floor at
//     FinalizedHead.
//
// Panics if two verifiers report distinct Verified tips at the same timestamp —
// that is a consensus-violating state.
func (c *simpleChainContainer) FullyVerifiedL2Head() (rollup.VerifierHead, rollup.VerifierHeadStatus) {
	if len(c.verifiers) == 0 {
		c.log.Debug("FullyVerifiedL2Head: no verifiers registered, pre-activation")
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, rollup.VerifierHeadOk
	}

	// Pre-activation L2 content is verified by consensus alone; gating it on a
	// not-yet-active interop verifier would stall the head at genesis (#20191).
	localTS, localTSOk := c.localSafeTimestamp()
	if localTSOk && c.allVerifiersPreActivationAt(localTS) {
		c.log.Debug("FullyVerifiedL2Head: all verifiers pre-activation", "localSafeTime", localTS)
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, rollup.VerifierHeadOk
	}

	ctx := context.Background()
	oldestTimestamp := uint64(math.MaxUint64)
	oldest := rollup.VerifierHead{}
	contributed := 0
	for _, v := range c.verifiers {
		// Skip verifiers not yet active at the current local-safe timestamp.
		// Their activation-anchor block doesn't exist on this chain yet, so we
		// must not try to resolve it and must not treat them as errors.
		if localTSOk && !v.IsActiveAt(localTS) {
			continue
		}
		contribution, ts, err := c.verifierSafeContribution(ctx, v)
		if err != nil {
			c.log.Warn("FullyVerifiedL2Head: verifier read failed, holding previous",
				"verifier", v.Name(), "err", err)
			return rollup.VerifierHead{}, rollup.VerifierHeadHoldPrevious
		}
		contributed++
		if ts < oldestTimestamp ||
			(ts == oldestTimestamp && contribution.Source == rollup.VerifierHeadAnchor && oldest.Source == rollup.VerifierHeadVerified) {
			oldestTimestamp = ts
			oldest = contribution
		} else if ts == oldestTimestamp && contribution.Block != oldest.Block && oldest.Source == rollup.VerifierHeadVerified && contribution.Source == rollup.VerifierHeadVerified {
			panic("verifiers disagree on block hash for same timestamp")
		}
	}
	if contributed == 0 {
		// Reachable only when localSafeTimestamp is unavailable AND every verifier
		// is currently inactive. The pre-activation short-circuit above handles
		// the common case; this is the cold-start safety net.
		c.log.Debug("FullyVerifiedL2Head: no active verifiers contributed, pre-activation")
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, rollup.VerifierHeadOk
	}
	c.log.Debug("FullyVerifiedL2Head: returning head", "block", oldest.Block, "source", oldest.Source, "timestamp", oldestTimestamp)
	return oldest, rollup.VerifierHeadOk
}

// FinalizedL2Head reports the cross-verified finalized L2 head with the same
// tri-state semantics as FullyVerifiedL2Head. Per-verifier finalized comes from
// VerifiedBlockAtL1(chainID, ss.FinalizedL1); empty results contribute the
// activation anchor; read errors return HoldPrevious. Finalized is bounded
// below by `min(verified-safe-tip, L1-finalized-reflection)` implicitly via
// VerifiedBlockAtL1's L1-anchored lookup.
func (c *simpleChainContainer) FinalizedL2Head() (rollup.VerifierHead, rollup.VerifierHeadStatus) {
	if len(c.verifiers) == 0 {
		c.log.Debug("FinalizedL2Head: no verifiers registered, pre-activation")
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, rollup.VerifierHeadOk
	}

	ctx := context.Background()
	ss, err := c.SyncStatus(ctx)
	if err != nil {
		c.log.Warn("FinalizedL2Head: failed to get sync status, holding previous", "err", err)
		return rollup.VerifierHead{}, rollup.VerifierHeadHoldPrevious
	}

	// FinalizedL2 <= LocalSafeL2; if local-safe is pre-activation, so is finalized.
	if c.allVerifiersPreActivationAt(ss.LocalSafeL2.Time) {
		c.log.Debug("FinalizedL2Head: all verifiers pre-activation", "localSafeTime", ss.LocalSafeL2.Time)
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, rollup.VerifierHeadOk
	}

	oldestTimestamp := uint64(math.MaxUint64)
	oldest := rollup.VerifierHead{}
	contributed := 0
	for _, v := range c.verifiers {
		if !v.IsActiveAt(ss.LocalSafeL2.Time) {
			continue
		}
		contribution, ts, err := c.verifierFinalizedContribution(ctx, v, ss.FinalizedL1)
		if err != nil {
			c.log.Warn("FinalizedL2Head: verifier read failed, holding previous",
				"verifier", v.Name(), "err", err)
			return rollup.VerifierHead{}, rollup.VerifierHeadHoldPrevious
		}
		contributed++
		if ts < oldestTimestamp ||
			(ts == oldestTimestamp && contribution.Source == rollup.VerifierHeadAnchor && oldest.Source == rollup.VerifierHeadVerified) {
			oldestTimestamp = ts
			oldest = contribution
		} else if ts == oldestTimestamp && contribution.Block != oldest.Block && oldest.Source == rollup.VerifierHeadVerified && contribution.Source == rollup.VerifierHeadVerified {
			panic("verifiers disagree on block hash for same timestamp")
		}
	}
	if contributed == 0 {
		c.log.Debug("FinalizedL2Head: no active verifiers contributed, pre-activation")
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, rollup.VerifierHeadOk
	}
	c.log.Debug("FinalizedL2Head: returning head", "block", oldest.Block, "source", oldest.Source, "timestamp", oldestTimestamp)
	return oldest, rollup.VerifierHeadOk
}

// verifierSafeContribution returns the (head, timestamp) this verifier
// contributes to the safe-head oldest-across-verifiers comparison: its verified
// tip if present, otherwise its per-(chain, verifier) activation anchor. Must
// only be called for verifiers active at the current local-safe timestamp.
func (c *simpleChainContainer) verifierSafeContribution(ctx context.Context, v activity.VerificationActivity) (rollup.VerifierHead, uint64, error) {
	bId, ts, err := v.LatestVerifiedL2Block(c.chainID)
	if err != nil {
		return rollup.VerifierHead{}, 0, err
	}
	if (bId == eth.BlockID{}) || ts == 0 {
		return c.anchorContribution(ctx, v)
	}
	return rollup.VerifierHead{Source: rollup.VerifierHeadVerified, Block: bId}, ts, nil
}

// verifierFinalizedContribution mirrors verifierSafeContribution for the
// finalized-head path, using VerifiedBlockAtL1 with the L1 finalized block.
func (c *simpleChainContainer) verifierFinalizedContribution(ctx context.Context, v activity.VerificationActivity, finalizedL1 eth.L1BlockRef) (rollup.VerifierHead, uint64, error) {
	bId, ts, err := v.VerifiedBlockAtL1(c.chainID, finalizedL1)
	if err != nil {
		return rollup.VerifierHead{}, 0, err
	}
	if (bId == eth.BlockID{}) || ts == 0 {
		return c.anchorContribution(ctx, v)
	}
	return rollup.VerifierHead{Source: rollup.VerifierHeadVerified, Block: bId}, ts, nil
}

// anchorContribution computes the per-(chain, verifier) activation-anchor block
// — the L2 block on this chain at timestamp `verifier.ActivationTimestamp() - 1`
// — and returns it together with the anchor timestamp for cross-verifier
// oldest-comparison. Engine-not-ready or RPC failure returns an error which
// the caller surfaces as HoldPrevious.
func (c *simpleChainContainer) anchorContribution(ctx context.Context, v activity.VerificationActivity) (rollup.VerifierHead, uint64, error) {
	if c.engine == nil {
		return rollup.VerifierHead{}, 0, engine_controller.ErrNoEngineClient
	}
	activeTS := v.ActivationTimestamp()
	if activeTS == 0 {
		// Verifier with no activation timestamp configured; treat as a
		// zero-timestamped anchor (oldest possible) carrying no concrete block.
		// This case is informational — production verifiers always configure
		// an activation timestamp.
		return rollup.VerifierHead{Source: rollup.VerifierHeadAnchor}, 0, nil
	}
	num, err := c.vncfg.Rollup.TargetBlockNumber(activeTS - 1)
	if err != nil {
		return rollup.VerifierHead{}, 0, fmt.Errorf("compute anchor block number for activation %d: %w", activeTS, err)
	}
	ref, err := c.engine.L2BlockRefByNumber(ctx, num)
	if err != nil {
		return rollup.VerifierHead{}, 0, fmt.Errorf("resolve anchor block %d: %w", num, err)
	}
	return rollup.VerifierHead{Source: rollup.VerifierHeadAnchor, Block: ref.ID()}, activeTS - 1, nil
}

// localSafeTimestamp returns the timestamp of the current local-safe L2 head.
// The bool is false if SyncStatus is unavailable, in which case callers should
// not attempt the pre-activation short-circuit.
func (c *simpleChainContainer) localSafeTimestamp() (uint64, bool) {
	ss, err := c.SyncStatus(context.Background())
	if err != nil {
		c.log.Warn("localSafeTimestamp: failed to get sync status", "err", err)
		return 0, false
	}
	return ss.LocalSafeL2.Time, true
}

// allVerifiersPreActivationAt reports whether every registered verifier is
// still inactive at the given L2 timestamp. Returns false if there are no
// verifiers; callers are expected to handle that case separately.
func (c *simpleChainContainer) allVerifiersPreActivationAt(ts uint64) bool {
	if len(c.verifiers) == 0 {
		return false
	}
	for _, v := range c.verifiers {
		if v.IsActiveAt(ts) {
			return false
		}
	}
	return true
}

// IsDenied checks if a block hash is on the deny list at the given height.
func (c *simpleChainContainer) IsDenied(height uint64, payloadHash common.Hash) (bool, error) {
	if c.denyList == nil {
		return false, fmt.Errorf("deny list not initialized")
	}
	return c.denyList.Contains(height, payloadHash)
}

// GetDeniedOutput returns the reconstructed OutputV0 for a denied block.
func (c *simpleChainContainer) GetDeniedOutput(height uint64, payloadHash common.Hash) (*eth.OutputV0, error) {
	if c.denyList == nil {
		return nil, fmt.Errorf("deny list not initialized")
	}
	return c.denyList.GetOutputV0(height, payloadHash)
}

// OutputV0AtBlockNumber returns the full OutputV0 for the block at the given number.
func (c *simpleChainContainer) OutputV0AtBlockNumber(ctx context.Context, l2BlockNum uint64) (*eth.OutputV0, error) {
	if c.engine == nil {
		return nil, engine_controller.ErrNoEngineClient
	}
	return c.engine.OutputV0AtBlockNumber(ctx, l2BlockNum)
}

// Interface satisfaction static check
var _ rollup.SuperAuthority = (*simpleChainContainer)(nil)
