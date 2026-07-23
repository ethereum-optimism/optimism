package pipeline

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
)

// L1BlockFetcher is the subset of an L1 RPC client used to fetch block refs.
type L1BlockFetcher interface {
	CallContext(ctx context.Context, result any, method string, args ...any) error
}

func FetchL1BlockRefByNumber(ctx context.Context, l1 L1BlockFetcher, blockNumberOrTag string) (*state.L1BlockRefJSON, error) {
	return FetchL1BlockRef(ctx, l1, "eth_getBlockByNumber", blockNumberOrTag)
}

func FetchL1BlockRefByHash(ctx context.Context, l1 L1BlockFetcher, hash common.Hash) (*state.L1BlockRefJSON, error) {
	return FetchL1BlockRef(ctx, l1, "eth_getBlockByHash", hash.Hex())
}

// FetchL1BlockRef fetches an L1 block header by tag/number or by hash and returns
// its block ref. A null RPC result (an unknown block, or a node that does not serve
// the requested tag) is returned as an error.
func FetchL1BlockRef(ctx context.Context, l1 L1BlockFetcher, method string, arg string) (*state.L1BlockRefJSON, error) {
	var ref state.L1BlockRefJSON
	if err := l1.CallContext(ctx, &ref, method, arg, false); err != nil {
		return nil, err
	}
	if ref.Hash == (common.Hash{}) {
		return nil, fmt.Errorf("L1 block %q not found", arg)
	}
	return &ref, nil
}
