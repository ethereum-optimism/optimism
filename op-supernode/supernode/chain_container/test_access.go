package chain_container

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type finalizedPromoterForTest interface {
	PromoteFinalizedForTest(ctx context.Context, ref eth.L2BlockRef) error
}

// PromoteFinalizedForTest drives the contained virtual node's engine-controller
// finalization path. It is intended for integration tests only.
func (c *simpleChainContainer) PromoteFinalizedForTest(ctx context.Context, ref eth.L2BlockRef) error {
	vn := c.getVN()
	if vn == nil {
		return errors.New("chain container virtual node is not initialized")
	}
	promoter, ok := vn.(finalizedPromoterForTest)
	if !ok {
		return fmt.Errorf("virtual node does not support finalized promotion test hook")
	}
	return promoter.PromoteFinalizedForTest(ctx, ref)
}
