package source

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type LogSource interface {
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
}

type ReceiptProcessor interface {
	ProcessLogs(ctx context.Context, block eth.L1BlockRef, rcpts types.Receipts) error
}

type ReceiptProcessorFn func(ctx context.Context, block eth.L1BlockRef, rcpts types.Receipts) error

func (r ReceiptProcessorFn) ProcessLogs(ctx context.Context, block eth.L1BlockRef, rcpts types.Receipts) error {
	return r(ctx, block, rcpts)
}

type logFetcher struct {
	client    LogSource
	processor ReceiptProcessor
}

func newLogFetcher(client LogSource, processor ReceiptProcessor) *logFetcher {
	return &logFetcher{
		client:    client,
		processor: processor,
	}
}

var _ BlockProcessor = (*logFetcher)(nil)

func (l *logFetcher) ProcessBlock(ctx context.Context, block eth.L1BlockRef) error {
	_, rcpts, err := l.client.FetchReceipts(ctx, block.Hash)
	if err != nil {
		return fmt.Errorf("failed to fetch receipts for block %v: %w", block, err)
	}
	return l.processor.ProcessLogs(ctx, block, rcpts)
}
