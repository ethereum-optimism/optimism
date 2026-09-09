package dsl

import (
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// WaitForL1Origin waits until the unsafe chain has adopted the given L1 height.
func (el *L2ELNode) WaitForL1Origin(number uint64) eth.L2BlockRef {
	var head eth.L2BlockRef
	el.require.Eventually(func() bool {
		var err error
		head, err = el.blockRefByLabel(eth.Unsafe)
		el.log.Info("Waiting for L2 to adopt L1 origin", "target", number, "head", head, "err", err)
		return err == nil && head.L1Origin.Number >= number
	}, 2*time.Minute, time.Second, fmt.Sprintf("L2 must adopt L1 origin at height %d", number))
	return head
}

// VerifyReorgRecovery checks that a replaced block and the whole suffix above it
// have canonical L1 origins. Check both tips again to avoid accepting a mixed
// snapshot while either chain is changing.
func (el *L2ELNode) VerifyReorgRecovery(l1 *L1ELNode, replaced eth.L2BlockRef) {
	el.require.Eventually(func() bool {
		ctx := el.ctx
		l1Head, err := l1.inner.EthClient().BlockRefByLabel(ctx, eth.Unsafe)
		if err != nil {
			return false
		}
		head, err := el.blockRefByLabel(eth.Unsafe)
		el.log.Info("Waiting for L2 reorg recovery", "replaced", replaced, "head", head, "err", err)
		if err != nil || head.Number <= replaced.Number || head.L1Origin.Number < replaced.L1Origin.Number {
			return false
		}
		ref := head
		for {
			origin, err := l1.inner.EthClient().BlockRefByNumber(ctx, ref.L1Origin.Number)
			if err != nil || origin.ID() != ref.L1Origin {
				return false
			}
			if ref.Number == replaced.Number {
				if ref.Hash == replaced.Hash {
					return false
				}
				break
			}
			ref, err = el.inner.L2EthClient().L2BlockRefByHash(ctx, ref.ParentHash)
			if err != nil {
				return false
			}
		}
		canonical, err := el.inner.L2EthClient().L2BlockRefByNumber(ctx, head.Number)
		if err != nil || canonical.Hash != head.Hash {
			return false
		}
		canonicalL1, err := l1.inner.EthClient().BlockRefByNumber(ctx, l1Head.Number)
		return err == nil && canonicalL1.Hash == l1Head.Hash
	}, 4*time.Minute, time.Second, fmt.Sprintf("L2 block %s must be replaced and its suffix must use canonical L1 origins", replaced))
}
