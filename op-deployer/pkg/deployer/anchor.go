package deployer

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// selectAnchorBlock returns safe when no override is set. An override must be
// no newer than safe and canonical at its height.
func selectAnchorBlock(ctx context.Context, l1 pipeline.L1BlockFetcher, safe *state.L1BlockRefJSON, overrideHash *common.Hash) (*state.L1BlockRefJSON, error) {
	if overrideHash == nil {
		return safe, nil
	}

	anchor, err := pipeline.FetchL1BlockRefByHash(ctx, l1, *overrideHash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch anchor block %s: %w", overrideHash.Hex(), err)
	}

	if uint64(anchor.Number) > uint64(safe.Number) {
		return nil, fmt.Errorf(
			"anchor block %s (height %d) is above the L1 safe head %d; it is not yet reorg-safe",
			overrideHash.Hex(), uint64(anchor.Number), uint64(safe.Number),
		)
	}

	// Hash lookups may return reorged-out blocks, so verify canonicality by number.
	canonical, err := pipeline.FetchL1BlockRefByNumber(ctx, l1, hexutil.EncodeUint64(uint64(anchor.Number)))
	if err != nil {
		return nil, fmt.Errorf("failed to verify anchor block %s is canonical: %w", overrideHash.Hex(), err)
	}
	if canonical.Hash != anchor.Hash {
		return nil, fmt.Errorf(
			"anchor block %s is not canonical at height %d (canonical block is %s); it may have been reorged out",
			overrideHash.Hex(), uint64(anchor.Number), canonical.Hash.Hex(),
		)
	}

	return anchor, nil
}
