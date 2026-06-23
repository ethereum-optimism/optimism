package chain_container

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum/go-ethereum/common"
)

// FullyVerifiedL2Head reports the cross-verified safe L2 head.
//
// With a single registered verifier:
//   - No verifier registered → PreActivation; caller uses local-safe.
//   - Verifier not yet active at local-safe time → PreActivation.
//   - Verifier read error → HoldPrevious (ok=false).
//   - Verifier has no entry for this chain → Anchor with cap timestamp
//     (`activationTimestamp - 1`); engine controller resolves to a canonical
//     block.
//   - Verifier has a verified tip → Verified.
func (c *simpleChainContainer) FullyVerifiedL2Head(ctx context.Context) (rollup.VerifierHead, bool) {
	v := c.registeredVerifier()
	if v == nil {
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}

	if activeTS, ok := c.localSafeTimestamp(ctx); ok && !v.IsActiveAt(activeTS) {
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}

	contribution, err := c.verifierContribution(v.LatestVerifiedL2Block(c.chainID))
	if err != nil {
		c.log.Warn("FullyVerifiedL2Head: verifier read failed, holding previous",
			"verifier", v.Name(), "err", err)
		return rollup.VerifierHead{}, false
	}
	return contribution, true
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

	// FinalizedL2 <= LocalSafeL2; if local-safe is pre-activation, so is finalized.
	if !v.IsActiveAt(ss.LocalSafeL2.Time) {
		return rollup.VerifierHead{Source: rollup.VerifierHeadPreActivation}, true
	}

	contribution, err := c.verifierContribution(v.VerifiedBlockAtL1(c.chainID, ss.FinalizedL1))
	if err != nil {
		c.log.Warn("FinalizedL2Head: verifier read failed, holding previous",
			"verifier", v.Name(), "err", err)
		return rollup.VerifierHead{}, false
	}
	return contribution, true
}

// verifierContribution classifies a verifier's (block, ts) return:
//   - empty block → Anchor (caller resolves the canonical L2 block at ts).
//   - non-empty block → Verified tip.
//
// Anchor timestamps are clamped up to L2 genesis: the verifier's raw cap is
// activationTimestamp - 1, which is pre-genesis when interop activates at
// genesis and has no resolvable block downstream.
func (c *simpleChainContainer) verifierContribution(bId eth.BlockID, ts uint64, err error) (rollup.VerifierHead, error) {
	if err != nil {
		return rollup.VerifierHead{}, err
	}
	if (bId == eth.BlockID{}) {
		if genesisTs := c.vncfg.Rollup.Genesis.L2Time; ts < genesisTs {
			ts = genesisTs
		}
		return rollup.VerifierHead{Source: rollup.VerifierHeadAnchor, Timestamp: ts}, nil
	}
	return rollup.VerifierHead{Source: rollup.VerifierHeadVerified, Block: bId, Timestamp: ts}, nil
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
