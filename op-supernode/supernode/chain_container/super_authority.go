package chain_container

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum/go-ethereum/common"
)

// FullyVerifiedL2Head returns the fully-verified L2 head from the registered
// verifier. The bool is true to signal use-local-safe (no verifier registered,
// pre-activation, or transient verifier error); false carries the verifier
// result (block, or empty when the verifier is operational but has nothing
// verified yet).
func (c *simpleChainContainer) FullyVerifiedL2Head() (eth.BlockID, bool) {
	v := c.registeredVerifier()
	if v == nil {
		c.log.Debug("FullyVerifiedL2Head: no verifier registered, signaling local-safe fallback")
		return eth.BlockID{}, true
	}

	// Pre-activation L2 content is verified by consensus alone; gating it on
	// a not-yet-active interop verifier would stall the head at genesis (#20191).
	if activeTS, ok := c.localSafeTimestamp(); ok && !v.IsActiveAt(activeTS) {
		c.log.Debug("FullyVerifiedL2Head: verifier pre-activation, signaling local-safe fallback", "localSafeTime", activeTS)
		return eth.BlockID{}, true
	}

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

	c.log.Debug("FullyVerifiedL2Head: returning verified block", "block", bId, "timestamp", ts)
	return bId, false
}

// FinalizedL2Head returns the finalized L2 head from the registered verifier.
// The bool is true to signal use-local-finalized (no verifier registered,
// pre-activation, sync status unavailable, or transient verifier error);
// false carries the verifier result.
func (c *simpleChainContainer) FinalizedL2Head() (eth.BlockID, bool) {
	v := c.registeredVerifier()
	if v == nil {
		c.log.Debug("FinalizedL2Head: no verifier registered, signaling local-finalized fallback")
		return eth.BlockID{}, true
	}

	ss, err := c.SyncStatus(context.Background())
	if err != nil {
		c.log.Error("FinalizedL2Head: failed to get sync status", "err", err)
		return eth.BlockID{}, true
	}

	// FinalizedL2 <= LocalSafeL2; if local-safe is pre-activation, so is finalized.
	if !v.IsActiveAt(ss.LocalSafeL2.Time) {
		c.log.Debug("FinalizedL2Head: verifier pre-activation, signaling local-finalized fallback", "localSafeTime", ss.LocalSafeL2.Time)
		return eth.BlockID{}, true
	}

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

	c.log.Debug("FinalizedL2Head: returning finalized block", "block", bId, "timestamp", ts)
	return bId, false
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
