package derive

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L1BlockRefByNumber interface {
	L1BlockRefByNumber(context.Context, uint64) (eth.L1BlockRef, error)
}

// VerifyNewL1Origin checks that the L2 unsafe head still has a L1 origin that is on the canonical chain.
// If the unsafe head origin is after the new L1 origin it is assumed to still be canonical.
// The check is only required when moving to a new L1 origin.
func VerifyNewL1Origin(ctx context.Context, unsafeOrigin eth.L1BlockRef, l1 L1BlockRefByNumber, newOrigin eth.L1BlockRef) error {
	if newOrigin.Number == unsafeOrigin.Number && newOrigin != unsafeOrigin {
		return NewResetError(fmt.Errorf("l1 origin was inconsistent with l2 unsafe head origin, need reset to resolve: l1 origin: %v; unsafe origin: %v",
			newOrigin.ID(), unsafeOrigin))
	}
	// Avoid requesting an older block by checking against the parent hash
	if newOrigin.Number == unsafeOrigin.Number+1 && newOrigin.ParentHash != unsafeOrigin.Hash {
		return NewResetError(fmt.Errorf("l2 unsafe head origin is no longer canonical, need reset to resolve: canonical hash: %v; unsafe origin hash: %v",
			newOrigin.ParentHash, unsafeOrigin.Hash))
	}
	if newOrigin.Number > unsafeOrigin.Number+1 {
		// If unsafe origin is further behind new origin, check it's still on the canonical chain.
		canonical, err := l1.L1BlockRefByNumber(ctx, unsafeOrigin.Number)
		if err != nil {
			return NewTemporaryError(fmt.Errorf("failed to fetch canonical L1 block at slot: %v; err: %w", unsafeOrigin.Number, err))
		}
		if canonical != unsafeOrigin {
			return NewResetError(fmt.Errorf("l2 unsafe head origin is no longer canonical, need reset to resolve: canonical: %v; unsafe origin: %v",
				canonical, unsafeOrigin))
		}
	}
	return nil
}
