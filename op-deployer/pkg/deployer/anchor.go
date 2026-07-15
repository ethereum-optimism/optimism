package deployer

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type l1BlockFetcher interface {
	CallContext(ctx context.Context, result any, method string, args ...any) error
}

// selectAnchorBlock returns safe when no override is set. An override must be
// no newer than safe and canonical at its height.
func selectAnchorBlock(ctx context.Context, l1 l1BlockFetcher, safe *state.L1BlockRefJSON, overrideHash *common.Hash) (*state.L1BlockRefJSON, error) {
	if overrideHash == nil {
		return safe, nil
	}

	anchor, err := fetchL1BlockRefByHash(ctx, l1, *overrideHash)
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
	canonical, err := fetchL1BlockRefByNumber(ctx, l1, hexutil.EncodeUint64(uint64(anchor.Number)))
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

func fetchL1BlockRefByNumber(ctx context.Context, l1 l1BlockFetcher, blockNumber string) (*state.L1BlockRefJSON, error) {
	return fetchL1BlockRef(ctx, l1, "eth_getBlockByNumber", blockNumber)
}

func fetchL1BlockRefByHash(ctx context.Context, l1 l1BlockFetcher, hash common.Hash) (*state.L1BlockRefJSON, error) {
	return fetchL1BlockRef(ctx, l1, "eth_getBlockByHash", hash.Hex())
}

// fetchL1BlockRef fetches an L1 block header by tag/number or by hash and returns
// its block ref. A null RPC result (an unknown block, or a node that does not serve
// the requested tag) is returned as an error.
func fetchL1BlockRef(ctx context.Context, l1 l1BlockFetcher, method string, arg string) (*state.L1BlockRefJSON, error) {
	var ref state.L1BlockRefJSON
	if err := l1.CallContext(ctx, &ref, method, arg, false); err != nil {
		return nil, err
	}
	if ref.Hash == (common.Hash{}) {
		return nil, fmt.Errorf("L1 block %q not found", arg)
	}
	return &ref, nil
}
