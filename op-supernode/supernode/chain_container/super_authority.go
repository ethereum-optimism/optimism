package chain_container

import (
	"context"
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum/go-ethereum/common"
)

// FullyVerifiedL2Head reports the cross-verified safe L2 head.
//
// Per active verifier, contribute either the verified tip or an Anchor with
// the pre-activation cap timestamp (returned by the verifier when it has no
// entry for this chain). Take the oldest contribution. Not-yet-active
// verifiers are skipped. The caller (engine_controller) resolves Anchor
// timestamps to canonical L2 blocks.
//
// Panics if two active verifiers report distinct Verified tips at the same
// timestamp.
func (c *simpleChainContainer) FullyVerifiedL2Head(ctx context.Context) (rollup.VerifierHead, bool) {
	if len(c.verifiers) == 0 {
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}

	localTS, localTSOk := c.localSafeTimestamp(ctx)

	oldest := rollup.VerifierHead{Timestamp: math.MaxUint64}
	contributed := 0
	for _, v := range c.verifiers {
		if localTSOk && !v.IsActiveAt(localTS) {
			continue
		}
		contribution, err := c.verifierContribution(v.LatestVerifiedL2Block(c.chainID))
		if err != nil {
			c.log.Warn("FullyVerifiedL2Head: verifier read failed, holding previous",
				"verifier", v.Name(), "err", err)
			return rollup.VerifierHead{}, false
		}
		contributed++
		oldest = pickOldest(oldest, contribution)
	}
	if contributed == 0 {
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}
	return oldest, true
}

// FinalizedL2Head is the finalized analogue of FullyVerifiedL2Head.
func (c *simpleChainContainer) FinalizedL2Head(ctx context.Context) (rollup.VerifierHead, bool) {
	if len(c.verifiers) == 0 {
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}

	ss, err := c.SyncStatus(ctx)
	if err != nil {
		c.log.Warn("FinalizedL2Head: failed to get sync status, holding previous", "err", err)
		return rollup.VerifierHead{}, false
	}

	oldest := rollup.VerifierHead{Timestamp: math.MaxUint64}
	contributed := 0
	for _, v := range c.verifiers {
		if !v.IsActiveAt(ss.LocalSafeL2.Time) {
			continue
		}
		contribution, err := c.verifierContribution(v.VerifiedBlockAtL1(c.chainID, ss.FinalizedL1))
		if err != nil {
			c.log.Warn("FinalizedL2Head: verifier read failed, holding previous",
				"verifier", v.Name(), "err", err)
			return rollup.VerifierHead{}, false
		}
		contributed++
		oldest = pickOldest(oldest, contribution)
	}
	if contributed == 0 {
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}
	return oldest, true
}

// verifierContribution classifies a verifier's (block, ts) return:
//   - empty block → Anchor (caller resolves the canonical L2 block at ts).
//   - non-empty block → Verified tip.
func (c *simpleChainContainer) verifierContribution(bId eth.BlockID, ts uint64, err error) (rollup.VerifierHead, error) {
	if err != nil {
		return rollup.VerifierHead{}, err
	}
	if (bId == eth.BlockID{}) {
		return rollup.VerifierHead{Source: rollup.VerifierHeadAnchor, Timestamp: ts}, nil
	}
	return rollup.VerifierHead{Source: rollup.VerifierHeadVerified, Block: bId, Timestamp: ts}, nil
}

// pickOldest returns the older of two contributions. Ties break toward Anchor
// (more conservative). Panics if two Verified tips disagree at the same timestamp.
func pickOldest(a, b rollup.VerifierHead) rollup.VerifierHead {
	if b.Timestamp < a.Timestamp {
		return b
	}
	if b.Timestamp > a.Timestamp {
		return a
	}
	// Equal timestamps.
	if a.Source == rollup.VerifierHeadVerified && b.Source == rollup.VerifierHeadVerified && a.Block != b.Block {
		panic("verifiers disagree on block hash for same timestamp")
	}
	if b.Source == rollup.VerifierHeadAnchor && a.Source == rollup.VerifierHeadVerified {
		return b
	}
	return a
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
