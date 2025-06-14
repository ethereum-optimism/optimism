package backend

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
)

// IndexingAdapter is a workaround to make the supervisor chain indexer work with a regular L2 client binding
type IndexingAdapter struct {
	Source interface {
		apis.ReceiptsFetcher
		apis.EthBlockRef
	}
}

var _ processors.Source = (*IndexingAdapter)(nil)

func (a *IndexingAdapter) BlockRefByNumber(ctx context.Context, number uint64) (eth.BlockRef, error) {
	return a.Source.BlockRefByNumber(ctx, number)
}

func (a *IndexingAdapter) FetchReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	_, receipts, err := a.Source.FetchReceipts(ctx, blockHash)
	return receipts, err
}
