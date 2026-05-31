package engine

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// crossSafeCache caches the most recently resolved cross-safe head so that
// transient verifier outages and reorg signals don't drop cross-safe all the
// way to FinalizedHead. Cross-safe CAN reorg, so the cache is re-validated
// against the EL on every read and cleared if stale or non-canonical.
type crossSafeCache struct {
	cached eth.L2BlockRef
	log    log.Logger
}

func newCrossSafeCache(log log.Logger) *crossSafeCache {
	return &crossSafeCache{log: log}
}

// Store records br as the latest known canonical cross-safe head. Callers
// must only Store blocks they have just verified canonical on the EL.
func (c *crossSafeCache) Store(br eth.L2BlockRef) {
	c.cached = br
}

// Get returns the cached cross-safe head if it is still canonical on the EL
// and not ahead of localSafeHead. Otherwise clears the cache and returns
// (zero, false).
func (c *crossSafeCache) Get(ctx context.Context, engine ExecEngine, localSafeHead eth.L2BlockRef) (eth.L2BlockRef, bool) {
	cached := c.cached
	if cached == (eth.L2BlockRef{}) {
		return eth.L2BlockRef{}, false
	}
	if cached.Number > localSafeHead.Number {
		c.log.Info("cached cross-safe ahead of local-safe; clearing cache",
			"cached", cached, "local_safe", localSafeHead)
		c.cached = eth.L2BlockRef{}
		return eth.L2BlockRef{}, false
	}
	canonical, err := engine.L2BlockRefByNumber(ctx, cached.Number)
	if err != nil {
		c.log.Warn("cannot validate cached cross-safe canonicality; clearing cache",
			"cached", cached, "err", err)
		c.cached = eth.L2BlockRef{}
		return eth.L2BlockRef{}, false
	}
	if canonical.Hash != cached.Hash {
		c.log.Info("cached cross-safe non-canonical (reorg); clearing cache",
			"cached", cached, "canonical", canonical)
		c.cached = eth.L2BlockRef{}
		return eth.L2BlockRef{}, false
	}
	return cached, true
}
