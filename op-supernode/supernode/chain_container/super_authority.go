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

// FullyVerifiedL2Head returns the oldest fully-verified L2 head across all
// verifiers. The bool is true to signal use-local-safe (no verifiers, all
// pre-activation, or transient verifier error); false carries the verifier
// result (block, or empty when verifiers are operational but have nothing
// verified yet). Panics if verifiers disagree at the same timestamp.
func (c *simpleChainContainer) FullyVerifiedL2Head() (eth.BlockID, bool) {
	if len(c.verifiers) == 0 {
		c.log.Debug("FullyVerifiedL2Head: no verifiers registered, signaling local-safe fallback")
		return eth.BlockID{}, true
	}

	// Pre-activation L2 content is verified by consensus alone; gating it on
	// a not-yet-active interop verifier would stall the head at genesis (#20191).
	if activeTS, ok := c.localSafeTimestamp(); ok && c.allVerifiersPreActivationAt(activeTS) {
		c.log.Debug("FullyVerifiedL2Head: all verifiers pre-activation, signaling local-safe fallback", "localSafeTime", activeTS)
		return eth.BlockID{}, true
	}

	timestamp := uint64(math.MaxUint64)
	oldestVerifiedBlock := eth.BlockID{}
	for _, v := range c.verifiers {
		bId, ts, err := v.LatestVerifiedL2Block(c.chainID)
		if err != nil {
			c.log.Warn("FullyVerifiedL2Head: verifier read failed, signaling local-safe fallback",
				"verifier", v.Name(), "err", err)
			return eth.BlockID{}, true
		}
		if (bId == eth.BlockID{} || ts == 0) {
			c.log.Debug("FullyVerifiedL2Head: verifier returned empty, returning empty without fallback", "verifier", v.Name())
			return eth.BlockID{}, false
		}
		if ts < timestamp {
			timestamp = ts
			oldestVerifiedBlock = bId
		} else if ts == timestamp && bId != oldestVerifiedBlock {
			panic("verifiers disagree on block hash for same timestamp")
		}
	}

	c.log.Debug("FullyVerifiedL2Head: returning verified block", "block", oldestVerifiedBlock, "timestamp", timestamp)
	return oldestVerifiedBlock, false
}

// FinalizedL2Head returns the oldest finalized L2 head across all verifiers.
// The bool is true to signal use-local-finalized (no verifiers, all
// pre-activation, sync status unavailable, or transient verifier error);
// false carries the verifier result. Panics if verifiers disagree at the
// same timestamp.
func (c *simpleChainContainer) FinalizedL2Head() (eth.BlockID, bool) {
	if len(c.verifiers) == 0 {
		c.log.Debug("FinalizedL2Head: no verifiers registered, signaling local-finalized fallback")
		return eth.BlockID{}, true
	}

	ss, err := c.SyncStatus(context.Background())
	if err != nil {
		c.log.Error("FinalizedL2Head: failed to get sync status", "err", err)
		return eth.BlockID{}, true
	}

	// FinalizedL2 <= LocalSafeL2; if local-safe is pre-activation, so is finalized.
	if c.allVerifiersPreActivationAt(ss.LocalSafeL2.Time) {
		c.log.Debug("FinalizedL2Head: all verifiers pre-activation, signaling local-finalized fallback", "localSafeTime", ss.LocalSafeL2.Time)
		return eth.BlockID{}, true
	}

	timestamp := uint64(math.MaxUint64)
	oldestFinalizedBlock := eth.BlockID{}
	for _, v := range c.verifiers {
		bId, ts, err := v.VerifiedBlockAtL1(c.chainID, ss.FinalizedL1)
		if err != nil {
			c.log.Warn("FinalizedL2Head: verifier read failed, signaling local-finalized fallback",
				"verifier", v.Name(), "err", err)
			return eth.BlockID{}, true
		}
		if (bId == eth.BlockID{} || ts == 0) {
			c.log.Debug("FinalizedL2Head: verifier returned empty, returning empty without fallback", "verifier", v.Name())
			return eth.BlockID{}, false
		}
		if ts < timestamp {
			timestamp = ts
			oldestFinalizedBlock = bId
		} else if ts == timestamp && bId != oldestFinalizedBlock {
			panic("verifiers disagree on block hash for same timestamp")
		}
	}

	c.log.Debug("FinalizedL2Head: returning finalized block", "block", oldestFinalizedBlock, "timestamp", timestamp)
	return oldestFinalizedBlock, false
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
