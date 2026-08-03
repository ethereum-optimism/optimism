package chain_container

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
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
	return c.floorAtReplacement(contribution), true
}

// floorAtReplacement lifts a verifier head that sits below the latest applied
// block replacement up to the replacement height (as an Anchor, resolved to the
// canonical block by the engine controller).
//
// Rationale: after an invalidation replaces block N, a derivation walk anchored
// BELOW N re-proposes the denied block, which triggers a deposits-only
// replacement plus a channel flush — discarding the remainder of the span that
// the rest of the network kept, and stranding the node on an abandoned batch
// lineage (see devnet interop-reorg-5, chain 420120192, blocks 13460/14245, and
// TestReplacedBlockLineageSelection in op-node/rollup/derive). A walk anchored
// AT the replacement past-skips the denied block, matching the behavior of
// nodes that performed the invalidation with a current verifier state. The
// denylist is the record of replacements this node has applied, so its max
// height is a safe derivation floor: everything at or below it is
// authority-decided local chain.
func (c *simpleChainContainer) floorAtReplacement(head rollup.VerifierHead) rollup.VerifierHead {
	h, any, err := c.MaxDeniedHeight()
	if err != nil {
		c.recordFloorDecision(floorDecision{outcome: floorOutcomeReadFailed},
			"anchor floor: deny list read failed, not flooring", "err", err)
		return head
	}
	c.met().DenyListMaxHeight.WithLabelValues(c.chainID.String()).Set(float64(h))
	if !any {
		// The post-wipe case: no replacement is recorded locally, so there is no
		// floor and the walk anchors wherever the verifier points — which is how a
		// fresh node re-proposes a block the rest of the cluster already replaced.
		c.recordFloorDecision(floorDecision{outcome: floorOutcomeNoDenyList},
			"anchor floor: no deny list entries, nothing to floor",
			"verifier_source", head.Source, "verifier_ts", head.Timestamp)
		return head
	}
	rcfg := c.vncfg.Rollup
	replacementTs := rcfg.Genesis.L2Time + h*rcfg.BlockTime
	switch head.Source {
	case rollup.VerifierHeadAnchor, rollup.VerifierHeadVerified:
		if head.Timestamp >= replacementTs {
			c.recordFloorDecision(floorDecision{outcome: floorOutcomeAtOrAbove, deniedHeight: h},
				"anchor floor: verifier head at or above latest applied replacement",
				"denied_height", h, "replacement_ts", replacementTs,
				"verifier_source", head.Source, "verifier_ts", head.Timestamp)
			return head
		}
		c.recordFloorDecision(floorDecision{outcome: floorOutcomeFloored, deniedHeight: h},
			"flooring verifier head at latest applied replacement",
			"denied_height", h, "replacement_ts", replacementTs,
			"verifier_source", head.Source, "verifier_ts", head.Timestamp)
		return rollup.VerifierHead{Source: rollup.VerifierHeadAnchor, Timestamp: replacementTs}
	default:
		// PreActivation resolves to local-safe directly, which already includes
		// any applied replacement.
		c.recordFloorDecision(floorDecision{outcome: floorOutcomePreActivation, deniedHeight: h},
			"anchor floor: pre-activation verifier head, using local safe",
			"denied_height", h, "replacement_ts", replacementTs)
		return head
	}
}

// defaultMetrics backs met() for containers built without metrics (tests).
var defaultMetrics = sync.OnceValue(resources.NewSupernodeMetrics)

// met returns the container's metrics, or a detached default when none were
// supplied, matching the nil-metrics contract documented on SupernodeMetrics.
func (c *simpleChainContainer) met() *resources.SupernodeMetrics {
	if c.metrics != nil {
		return c.metrics
	}
	return defaultMetrics()
}

// Outcomes of floorAtReplacement, also used as the metric label value.
const (
	floorOutcomeFloored       = "floored"
	floorOutcomeAtOrAbove     = "at_or_above_replacement"
	floorOutcomeNoDenyList    = "no_denylist_entries"
	floorOutcomeReadFailed    = "denylist_read_failed"
	floorOutcomePreActivation = "pre_activation"
)

// floorDecision is the dedupe key for floorAtReplacement logging. It
// deliberately excludes the verifier timestamp: that advances every block
// during normal sync, so including it made every block a new "transition" and
// emitted a line per block per chain. Outcome plus replacement height changes
// only when something actually changes.
type floorDecision struct {
	outcome      string
	deniedHeight uint64
}

// recordFloorDecision counts every decision but logs only transitions: the
// verifier head is polled continuously, so logging every call would emit
// several lines per second per chain and bury the derivation logs around it.
func (c *simpleChainContainer) recordFloorDecision(d floorDecision, msg string, ctx ...any) {
	c.met().AnchorFloorDecisions.WithLabelValues(c.chainID.String(), d.outcome).Inc()
	if prev, ok := c.lastFloorDecision.Load().(floorDecision); ok && prev == d {
		return
	}
	c.lastFloorDecision.Store(d)
	c.log.Info(msg, ctx...)
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
	denied, err := c.denyList.Contains(height, payloadHash)
	if err == nil && denied {
		// A hit means derivation re-proposed a block this node already replaced:
		// the walk anchored below the replacement instead of past-skipping it.
		c.met().DenyListHits.WithLabelValues(c.chainID.String()).Inc()
		c.log.Info("deny list hit, block re-proposed after replacement",
			"height", height, "payload_hash", payloadHash)
	}
	return denied, err
}

// MaxDeniedHeight returns the highest denied block height and whether any
// denial exists.
func (c *simpleChainContainer) MaxDeniedHeight() (uint64, bool, error) {
	if c.denyList == nil {
		return 0, false, nil
	}
	return c.denyList.MaxDeniedHeight()
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
