package virtual_node

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type finalizedPromoterForTest interface {
	PromoteFinalizedForTest(ctx context.Context, ref eth.L2BlockRef) error
}

// PromoteFinalizedForTest drives the inner op-node engine-controller
// finalization path. It is intended for integration tests only.
func (v *simpleVirtualNode) PromoteFinalizedForTest(ctx context.Context, ref eth.L2BlockRef) error {
	v.mu.Lock()
	inner := v.inner
	v.mu.Unlock()
	if inner == nil {
		return ErrVirtualNodeNotRunning
	}
	promoter, ok := inner.(finalizedPromoterForTest)
	if !ok {
		return fmt.Errorf("inner node does not support finalized promotion test hook")
	}
	if err := promoter.PromoteFinalizedForTest(ctx, ref); err != nil {
		if errors.Is(err, context.Canceled) {
			return ErrVirtualNodeNotRunning
		}
		return err
	}
	return nil
}
