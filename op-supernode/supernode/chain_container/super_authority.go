package chain_container

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum/go-ethereum/common"
)

// FullyVerifiedL2Head reports the cross-verified safe L2 head.
//
// The verifier contributes either the verified tip or no entry. If the verifier
// is active but has no entry for this chain, SuperAuthority contributes an
// Anchor at the pre-activation cap timestamp. A not-yet-active verifier returns
// PreActivation. The caller decides whether to resolve Anchor timestamps or
// fall back more conservatively.
func (c *simpleChainContainer) FullyVerifiedL2Head(ctx context.Context) (rollup.VerifierHead, bool) {
	v := c.registeredVerifier()
	if v == nil {
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}

	localTS, localTSOk := c.localSafeTimestamp(ctx)
	if localTSOk && !v.IsActiveAt(localTS) {
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}

	block, ts, err := v.LatestVerifiedL2Block(c.chainID)
	head, err := c.verifierContribution(v, block, ts, err)
	if err != nil {
		c.log.Warn("FullyVerifiedL2Head: verifier read failed, holding previous",
			"verifier", v.Name(), "err", err)
		return rollup.VerifierHead{}, false
	}
	return head, true
}

// FinalizedL2Head is the finalized analogue of FullyVerifiedL2Head.
func (c *simpleChainContainer) FinalizedL2Head(ctx context.Context) (rollup.VerifierHead, bool) {
	v := c.registeredVerifier()
	if v == nil {
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}

	ss, err := c.SyncStatus(ctx)
	if err != nil {
		c.log.Warn("FinalizedL2Head: failed to get sync status, holding previous", "err", err)
		return rollup.VerifierHead{}, false
	}

	if !v.IsActiveAt(ss.LocalSafeL2.Time) {
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}

	block, ts, err := v.VerifiedBlockAtL1(c.chainID, ss.FinalizedL1)
	head, err := c.verifierContribution(v, block, ts, err)
	if err != nil {
		c.log.Warn("FinalizedL2Head: verifier read failed, holding previous",
			"verifier", v.Name(), "err", err)
		return rollup.VerifierHead{}, false
	}
	return head, true
}

// verifierContribution classifies a verifier's (block, ts) return:
//   - empty block -> Anchor (activationTimestamp - 1).
//   - non-empty block -> Verified tip.
func (c *simpleChainContainer) verifierContribution(v activity.VerificationActivity, bID eth.BlockID, ts uint64, err error) (rollup.VerifierHead, error) {
	if err != nil {
		return rollup.VerifierHead{}, err
	}
	if (bID == eth.BlockID{}) {
		return rollup.VerifierHead{Source: rollup.VerifierHeadAnchor, Timestamp: activationCap(v.ActivationTimestamp())}, nil
	}
	return rollup.VerifierHead{Source: rollup.VerifierHeadVerified, Block: bID, Timestamp: ts}, nil
}

func activationCap(activationTimestamp uint64) uint64 {
	if activationTimestamp == 0 {
		return 0
	}
	return activationTimestamp - 1
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
