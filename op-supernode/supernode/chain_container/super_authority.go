package chain_container

import (
	"math"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SuperAuthority is an interface for supernode-level authority operations.
// It is passed to op-node instances during initialization to provide
// supernode-specific functionality and coordination.
type SuperAuthority interface {
	SafeL2Head() eth.L2BlockRef
}

// SafeL2Head returns the safe L2 head block reference.
// It returns an empty L2BlockRef if no safe head can be determined.
func (c *simpleChainContainer) SafeL2Head() eth.L2BlockRef {
	timestamp := uint64(math.MaxUint64)
	oldestVerifiedBlock := eth.BlockID{}
	for _, v := range c.verifiers {
		bId, ts := v.LatestVerifiedL2Block(c.chainID)
		if (bId == eth.BlockID{} || ts == 0) {
			return eth.L2BlockRef{}
		}
		if ts < timestamp {
			oldestVerifiedBlock = bId
		}
	}
	// TODO we need to store the full block ref in the verifier's DB
	// the following struct is missing data:
	return eth.L2BlockRef{
		Hash:   oldestVerifiedBlock.Hash,
		Number: oldestVerifiedBlock.Number,
	}
}

// Interface satisfaction static check
var _ SuperAuthority = (*simpleChainContainer)(nil)
