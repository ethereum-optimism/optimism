package chain_container

import (
	"context"
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// SafeL2Head returns the safe L2 head block identifier.
// Returns local-safe when no verifiers are registered or when verifiers return empty BlockID.
// This provides a graceful fallback for pre-interop blocks.
// Panics if verifiers disagree on the block hash for the same timestamp.
func (c *simpleChainContainer) FullyVerifiedL2Head() eth.BlockID {
	getLocalSafe := func() eth.BlockID {
		if c.rollupClient == nil {
			c.log.Debug("FullyVerifiedL2Head: rollup client not initialized, returning empty")
			return eth.BlockID{}
		}
		status, err := c.rollupClient.SyncStatus(context.Background())
		if err != nil {
			c.log.Debug("FullyVerifiedL2Head: failed to get sync status", "err", err)
			return eth.BlockID{}
		}
		c.log.Debug("FullyVerifiedL2Head: falling back to local-safe", "local_safe", status.LocalSafeL2.ID())
		return status.LocalSafeL2.ID()
	}

	// If no verifiers registered, fall back to local-safe
	if len(c.verifiers) == 0 {
		c.log.Debug("FullyVerifiedL2Head: no verifiers registered, using local-safe fallback")
		return getLocalSafe()
	}

	timestamp := uint64(math.MaxUint64)
	oldestVerifiedBlock := eth.BlockID{}
	for _, v := range c.verifiers {
		bId, ts := v.LatestVerifiedL2Block(c.chainID)
		if (bId == eth.BlockID{} || ts == 0) {
			return bId
		}
		if ts < timestamp {
			timestamp = ts
			oldestVerifiedBlock = bId
		} else if ts == timestamp && bId != oldestVerifiedBlock {
			panic("verifiers disagree on block hash for same timestamp")
		}
	}
	return oldestVerifiedBlock
}

// IsDenied checks if a block hash is on the deny list at the given height.
func (c *simpleChainContainer) IsDenied(height uint64, payloadHash common.Hash) (bool, error) {
	if c.denyList == nil {
		return false, fmt.Errorf("deny list not initialized")
	}
	return c.denyList.Contains(height, payloadHash)
}

// Interface satisfaction static check
var _ rollup.SuperAuthority = (*simpleChainContainer)(nil)
