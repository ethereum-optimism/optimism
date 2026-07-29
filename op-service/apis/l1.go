package apis

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L1EthBlockRef interface {

	// L1BlockRefByLabel returns the [eth.L1BlockRef] for the given block label.
	// Notice, we cannot cache a block reference by label because labels are not guaranteed to be unique.
	L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error)

	// L1BlockRefByNumber returns an [eth.L1BlockRef] for the given block number.
	// Notice, we cannot cache a block reference by number because L1 re-orgs can invalidate the cached block reference.
	L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error)

	// L1BlockRefByHash returns the [eth.L1BlockRef] for the given block hash.
	// We cache the block reference by hash as it is safe to assume collision will not occur.
	L1BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L1BlockRef, error)
}

type L1EthClient interface {
	EthClient
	L1EthBlockRef
}
