package chain_container

import (
	"context"
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

// FinalizedL2Head returns the finalized L2 head block identifier.
// The second return value indicates whether the caller should fall back to local-finalized.
// Returns (empty, true) only when no verifiers are registered.
// Returns (empty, false) when verifiers are registered but haven't finalized anything yet.
// Panics if verifiers disagree on the block hash for the same timestamp.
func (c *simpleChainContainer) FinalizedL2Head() (eth.BlockID, bool) {
	// If no verifiers registered, signal fallback to local-finalized
	if len(c.verifiers) == 0 {
		c.log.Debug("FinalizedL2Head: no verifiers registered, signaling local-finalized fallback")
		return eth.BlockID{}, true
	}

	ss, err := c.vn.SyncStatus(context.Background())
	if err != nil {
		c.log.Error("FinalizedL2Head: failed to get sync status", "err", err)
		return eth.BlockID{}, true
	}
	timestamp := uint64(math.MaxUint64)
	oldestFinalizedBlock := eth.BlockID{}
	for _, v := range c.verifiers {
		bId, ts := v.VerifiedBlockAtL1(c.chainID, ss.FinalizedL1)
		// If any verifier returns empty, return empty but don't signal fallback
		// The verifier exists but hasn't finalized anything yet
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

// IsDenied checks if a block hash is on the deny list at the given height.
func (c *simpleChainContainer) IsDenied(height uint64, payloadHash common.Hash) (bool, error) {
	if c.denyList == nil {
		return false, fmt.Errorf("deny list not initialized")
	}
	return c.denyList.Contains(height, payloadHash)
}

// Interface satisfaction static check
var _ rollup.SuperAuthority = (*simpleChainContainer)(nil)
