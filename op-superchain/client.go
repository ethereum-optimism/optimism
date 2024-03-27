package superchain

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
)

// In order to allow `op-node/rollup/derive` to depend on this, we cannot have a dependency
// on op-service/sources as that creates a circular dependency on op-node. Until we fix this,
// we create an alternative head source here.

type L1BlockRefsSource interface {
	L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error)
}

type blockRefsClient struct {
	clnt client.RPC
}

func (b *blockRefsClient) L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error) {
	var head *types.Header
	err := b.clnt.CallContext(ctx, &head, "eth_getBlockByNumber", label.Arg(), false)
	if err != nil {
		return eth.L1BlockRef{}, err
	}
	if head == nil {
		return eth.L1BlockRef{}, ethereum.NotFound
	}

	return eth.L1BlockRef{Hash: head.Hash(), Number: head.Number.Uint64(), ParentHash: head.ParentHash, Time: head.Time}, nil
}
