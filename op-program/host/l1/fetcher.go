package l1

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type Source interface {
	InfoByHash(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, error)
	InfoAndTxsByHash(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Transactions, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
}

type FetchingL1Oracle struct {
	ctx    context.Context
	logger log.Logger
	source Source
}

func NewFetchingL1Oracle(ctx context.Context, logger log.Logger, source Source) *FetchingL1Oracle {
	return &FetchingL1Oracle{
		ctx:    ctx,
		logger: logger,
		source: source,
	}
}

func (o FetchingL1Oracle) HeaderByBlockHash(blockHash common.Hash) eth.BlockInfo {
	o.logger.Trace("HeaderByBlockHash", "hash", blockHash)
	info, err := o.source.InfoByHash(o.ctx, blockHash)
	if err != nil {
		panic(fmt.Errorf("retrieve block %s: %w", blockHash, err))
	}
	if info == nil {
		panic(fmt.Errorf("unknown block: %s", blockHash))
	}
	return info
}

func (o FetchingL1Oracle) TransactionsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Transactions) {
	o.logger.Trace("TransactionsByBlockHash", "hash", blockHash)
	info, txs, err := o.source.InfoAndTxsByHash(o.ctx, blockHash)
	if err != nil {
		panic(fmt.Errorf("retrieve transactions for block %s: %w", blockHash, err))
	}
	if info == nil || txs == nil {
		panic(fmt.Errorf("unknown block: %s", blockHash))
	}
	return info, txs
}

func (o FetchingL1Oracle) ReceiptsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Receipts) {
	o.logger.Trace("ReceiptsByBlockHash", "hash", blockHash)
	info, rcpts, err := o.source.FetchReceipts(o.ctx, blockHash)
	if err != nil {
		panic(fmt.Errorf("retrieve receipts for block %s: %w", blockHash, err))
	}
	if info == nil || rcpts == nil {
		panic(fmt.Errorf("unknown block: %s", blockHash))
	}
	return info, rcpts
}
