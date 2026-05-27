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

// FullyVerifiedL2Head reports the cross-verified safe L2 head.
//
// With no verifiers, returns PreActivation. Per verifier active at the current
// local-safe timestamp, contribute either the verified tip or an activation
// anchor; take the oldest contribution across active verifiers. Not-yet-active
// verifiers are skipped — their anchor block doesn't exist on this chain yet.
// Panics if two active verifiers report distinct Verified tips at the same
// timestamp.
func (c *simpleChainContainer) FullyVerifiedL2Head(ctx context.Context) (rollup.VerifierHead, bool) {
	if len(c.verifiers) == 0 {
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}

	localTS, localTSOk := c.localSafeTimestamp(ctx)

	oldestTimestamp := uint64(math.MaxUint64)
	oldest := rollup.VerifierHead{}
	contributed := 0
	for _, v := range c.verifiers {
		if localTSOk && !v.IsActiveAt(localTS) {
			continue
		}
		contribution, ts, err := c.verifierSafeContribution(ctx, v)
		if err != nil {
			c.log.Warn("FullyVerifiedL2Head: verifier read failed, holding previous",
				"verifier", v.Name(), "err", err)
			return rollup.VerifierHead{}, false
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
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}
	return oldest, true
}

// FinalizedL2Head is the finalized analogue of FullyVerifiedL2Head. Per-verifier
// contribution comes from VerifiedBlockAtL1(chainID, ss.FinalizedL1); empty
// results yield the activation anchor; read errors return HoldPrevious.
func (c *simpleChainContainer) FinalizedL2Head(ctx context.Context) (rollup.VerifierHead, bool) {
	if len(c.verifiers) == 0 {
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}

	ss, err := c.SyncStatus(ctx)
	if err != nil {
		c.log.Warn("FinalizedL2Head: failed to get sync status, holding previous", "err", err)
		return rollup.VerifierHead{}, false
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
			return rollup.VerifierHead{}, false
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
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}
	return oldest, true
}

// verifierSafeContribution: verified tip if present, otherwise activation anchor.
// Caller must filter inactive verifiers — anchor lookup assumes the verifier is active.
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

// anchorContribution resolves the L2 block at `verifier.ActivationTimestamp() - 1`
// on this chain. Engine-not-ready or RPC failure becomes HoldPrevious upstream.
func (c *simpleChainContainer) anchorContribution(ctx context.Context, v activity.VerificationActivity) (rollup.VerifierHead, uint64, error) {
	if c.engine == nil {
		return rollup.VerifierHead{}, 0, engine_controller.ErrNoEngineClient
	}
	activeTS := v.ActivationTimestamp()
	if activeTS == 0 {
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

func (c *simpleChainContainer) localSafeTimestamp(ctx context.Context) (uint64, bool) {
	ss, err := c.SyncStatus(ctx)
	if err != nil {
		c.log.Warn("localSafeTimestamp: failed to get sync status", "err", err)
		return 0, false
	}
	return ss.LocalSafeL2.Time, true
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

var _ rollup.SuperAuthority = (*simpleChainContainer)(nil)
