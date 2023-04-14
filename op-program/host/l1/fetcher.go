package l1

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type Source interface {
	HeaderByHash(ctx context.Context, blockHash common.Hash) (*types.Header, error)
	BlockByHash(ctx context.Context, blockHash common.Hash) (*types.Block, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

var _ Source = (*ethclient.Client)(nil)

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

func (o *FetchingL1Oracle) HeaderByBlockHash(blockHash common.Hash) *types.Header {
	header := o.RawHeaderByBlockHash(blockHash)
	return header
}

func (o *FetchingL1Oracle) RawHeaderByBlockHash(blockHash common.Hash) *types.Header {
	o.logger.Trace("HeaderByBlockHash", "hash", blockHash)
	header, err := o.source.HeaderByHash(o.ctx, blockHash)
	if err != nil {
		panic(fmt.Errorf("retrieve block %s: %w", blockHash, err))
	}
	if header == nil {
		panic(fmt.Errorf("unknown block: %s", blockHash))
	}
	return header
}

func (o *FetchingL1Oracle) TransactionsByBlockHash(blockHash common.Hash) (*types.Header, types.Transactions) {
	o.logger.Trace("TransactionsByBlockHash", "hash", blockHash)
	block, err := o.source.BlockByHash(o.ctx, blockHash)
	//info, txs, err := o.source.InfoAndTxsByHash(o.ctx, blockHash)
	if err != nil {
		panic(fmt.Errorf("retrieve transactions for block %s: %w", blockHash, err))
	}
	if block == nil {
		panic(fmt.Errorf("unknown block: %s", blockHash))
	}
	return block.Header(), block.Transactions()
}

func (o *FetchingL1Oracle) ReceiptsByBlockHash(blockHash common.Hash) (*types.Header, types.Receipts) {
	o.logger.Trace("ReceiptsByBlockHash", "hash", blockHash)
	header, transactions := o.TransactionsByBlockHash(blockHash)

	var receipts []*types.Receipt
	for _, transaction := range transactions {
		receipt, err := o.source.TransactionReceipt(o.ctx, transaction.Hash())
		if err != nil {
			panic(fmt.Errorf("loading receipt for tx %s: %w", transaction.Hash(), err))
		}
		receipts = append(receipts, receipt)
	}
	return header, receipts
}
