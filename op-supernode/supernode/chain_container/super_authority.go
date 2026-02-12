package chain_container

import (
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// SafeL2Head returns the safe L2 head block identifier.
// It returns an empty BlockID if no fully verified head can be determined.
// Panics if verifiers disagree on the block hash for the same timestamp.
func (c *simpleChainContainer) FullyVerifiedL2Head() eth.BlockID {
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
