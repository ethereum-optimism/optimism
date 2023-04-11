package l1

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Source interface {
	InfoByHash(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, error)
	InfoAndTxsByHash(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Transactions, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
}

type FetchingL1Oracle struct {
	ctx    context.Context
	source Source
}

func NewFetchingL1Oracle(ctx context.Context, source Source) *FetchingL1Oracle {
	return &FetchingL1Oracle{
		ctx:    ctx,
		source: source,
	}
}

func (o FetchingL1Oracle) HeaderByHash(blockHash common.Hash) eth.BlockInfo {
	info, err := o.source.InfoByHash(o.ctx, blockHash)
	if err != nil {
		panic(fmt.Errorf("retrieve block %s: %w", blockHash, err))
	}
	if info == nil {
		panic(fmt.Errorf("unknown block: %s", blockHash))
	}
	return info
}

func (o FetchingL1Oracle) TransactionsByHash(blockHash common.Hash) (eth.BlockInfo, types.Transactions) {
	info, txs, err := o.source.InfoAndTxsByHash(o.ctx, blockHash)
	if err != nil {
		panic(fmt.Errorf("retrieve transactions for block %s: %w", blockHash, err))
	}
	if info == nil || txs == nil {
		panic(fmt.Errorf("unknown block: %s", blockHash))
	}
	return info, txs
}

func (o FetchingL1Oracle) ReceiptsByHash(blockHash common.Hash) (eth.BlockInfo, types.Receipts) {
	info, rcpts, err := o.source.FetchReceipts(o.ctx, blockHash)
	if err != nil {
		panic(fmt.Errorf("retrieve receipts for block %s: %w", blockHash, err))
	}
	if info == nil || rcpts == nil {
		panic(fmt.Errorf("unknown block: %s", blockHash))
	}
	return info, rcpts
}
