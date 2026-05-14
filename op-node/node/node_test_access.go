package node

import (
	"context"
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// PromoteFinalizedForTest drives the engine-controller finalization path for
// integration tests that need to exercise supernode/op-node internals directly.
func (n *OpNode) PromoteFinalizedForTest(ctx context.Context, ref eth.L2BlockRef) error {
	if n.l2Driver == nil || n.l2Driver.SyncDeriver == nil || n.l2Driver.SyncDeriver.Engine == nil {
		return errors.New("op-node engine controller is not initialized")
	}
	n.l2Driver.SyncDeriver.Engine.PromoteFinalized(ctx, ref)
	return nil
}
