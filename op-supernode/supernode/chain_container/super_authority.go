package chain_container

import (
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// FullyVerifiedL2Head returns the fully verified L2 head block identifier.
// The second return value indicates whether the caller should fall back to local-safe.
// Returns (empty, true) only when no verifiers are registered.
// Returns (empty, false) when verifiers are registered but haven't verified anything yet.
// Panics if verifiers disagree on the block hash for the same timestamp.
func (c *simpleChainContainer) FullyVerifiedL2Head() (eth.BlockID, bool) {
	// If no verifiers registered, signal fallback to local-safe
	if len(c.verifiers) == 0 {
		c.log.Debug("FullyVerifiedL2Head: no verifiers registered, signaling local-safe fallback")
		return eth.BlockID{}, true
	}

	timestamp := uint64(math.MaxUint64)
	oldestVerifiedBlock := eth.BlockID{}
	for _, v := range c.verifiers {
		bId, ts := v.LatestVerifiedL2Block(c.chainID)
		// If any verifier returns empty, return empty but don't signal fallback
		// The verifier exists but hasn't verified anything yet
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

// IsDenied checks if a block hash is on the deny list at the given height.
func (c *simpleChainContainer) IsDenied(height uint64, payloadHash common.Hash) (bool, error) {
	if c.denyList == nil {
		return false, fmt.Errorf("deny list not initialized")
	}
	return c.denyList.Contains(height, payloadHash)
}

// Interface satisfaction static check
var _ rollup.SuperAuthority = (*simpleChainContainer)(nil)
