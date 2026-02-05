package chain_container

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SuperAuthority is an interface for supernode-level authority operations.
// It is passed to op-node instances during initialization to provide
// supernode-specific functionality and coordination.
type SuperAuthority interface {
	SafeL2Head() eth.L2BlockRef
}

func (c *simpleChainContainer) latestFullyVerifiedTimestamp() uint64 {
	timestamp := uint64(0)
	for _, v := range c.verifiers {
		ts, ok := v.LatestVerifiedTimestamp()
		if ok {
			return c.vncfg.Rollup.Genesis.L2Time
		}
		// We need the oldest timestamp across all verifiers
		if ts < timestamp {
			timestamp = ts
		}
	}
	return timestamp
}

func (c *simpleChainContainer) SafeL2Head() eth.L2BlockRef {
	timestamp := c.latestFullyVerifiedTimestamp()
	ctx := context.Background()
	block, err := c.BlockAtTimestamp(ctx, timestamp, eth.Safe)
	if err != nil {
		// Handle error appropriately
		return eth.L2BlockRef{}
	}
	return block
}

// Interface satisfaction static check
var _ SuperAuthority = (*simpleChainContainer)(nil)
