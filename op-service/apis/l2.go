package apis

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2EthBlockRef interface {
	// L2BlockRefByLabel returns the [eth.L2BlockRef] for the given block label.
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)

	// L2BlockRefByNumber returns the [eth.L2BlockRef] for the given block number.
	L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)

	// L2BlockRefByHash returns the [eth.L2BlockRef] for the given block hash.
	// The returned BlockRef may not be in the canonical chain.
	L2BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L2BlockRef, error)
}

type SystemConfigFetcher interface {
	// SystemConfigByL2Hash returns the [eth.SystemConfig] (matching the config updates up to and including the L1 origin) for the given L2 block hash.
	// The returned [eth.SystemConfig] may not be in the canonical chain when the hash is not canonical.
	SystemConfigByL2Hash(ctx context.Context, hash common.Hash) (eth.SystemConfig, error)
}

type OutputRootFetcher interface {
	OutputV0AtBlockNumber(ctx context.Context, blockNum uint64) (*eth.OutputV0, error)
	OutputV0AtBlock(ctx context.Context, blockHash common.Hash) (*eth.OutputV0, error)
}

type L2EthClient interface {
	EthClient
	L2EthBlockRef
}

type L2EthExtendedClient interface {
	EthExtendedClient
	SystemConfigFetcher
	OutputRootFetcher
}
