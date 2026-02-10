package chain_container

import (
	"math"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SuperAuthority is an interface for supernode-level authority operations.
// It is passed to op-node instances during initialization to provide
// supernode-specific functionality and coordination.
type SuperAuthority interface {
	FullyVerifiedL2Head() eth.BlockID
}

// SafeL2Head returns the safe L2 head block identifier.
// It returns an empty BlockID if no fully verified head can be determined.
func (c *simpleChainContainer) FullyVerifiedL2Head() eth.BlockID {
	timestamp := uint64(math.MaxUint64)
	oldestVerifiedBlock := eth.BlockID{}
	for _, v := range c.verifiers {
		bId, ts := v.LatestVerifiedL2Block(c.chainID)
		if (bId == eth.BlockID{} || ts == 0) {
			return bId
		}
		if ts < timestamp {
			oldestVerifiedBlock = bId
		}
	}
	return oldestVerifiedBlock
}

// Interface satisfaction static check
var _ SuperAuthority = (*simpleChainContainer)(nil)
