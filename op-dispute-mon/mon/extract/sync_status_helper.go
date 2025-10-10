package extract

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// lagTolerance defines a small tolerance to absorb minor races between
// output indexing and reported sync status.
const lagTolerance uint64 = 10

// syncStatusProvider is an optional capability that some rollup clients implement.
// It enables classification of a node as "behind" relative to a requested L2 block.
type syncStatusProvider interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
}

// isNodeBehind returns true if the given client reports an UnsafeL2 height that is
// behind the requested l2BlockNum by more than behindSlack. If the client does not
// implement syncStatusProvider, it returns (false, nil) to preserve existing behavior.
func isNodeBehind(ctx context.Context, client any, l2BlockNum uint64) (bool, error) {
	provider, ok := client.(syncStatusProvider)
	if !ok {
		return false, nil
	}
	status, err := provider.SyncStatus(ctx)
	if err != nil || status == nil {
		return false, err
	}
	unsafe := status.UnsafeL2.Number
	if l2BlockNum > unsafe+lagTolerance {
		return true, nil
	}
	return false, nil
}
