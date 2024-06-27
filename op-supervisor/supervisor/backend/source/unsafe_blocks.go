package source

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

type BlockByNumberSource interface {
	L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error)
}

type BlockProcessor interface {
	ProcessBlock(ctx context.Context, block eth.L1BlockRef) error
}

type BlockProcessorFn func(ctx context.Context, block eth.L1BlockRef) error

func (fn BlockProcessorFn) ProcessBlock(ctx context.Context, block eth.L1BlockRef) error {
	return fn(ctx, block)
}

// UnsafeBlocksStage is a PipelineEventHandler that watches for UnsafeHeadEvent and backfills any skipped blocks.
// It emits a UnsafeBlockEvent for each block up to the new unsafe head.
type UnsafeBlocksStage struct {
	log       log.Logger
	client    BlockByNumberSource
	lastBlock eth.L1BlockRef
	processor BlockProcessor
}

var _ PipelineEventHandler[eth.L1BlockRef] = (*UnsafeBlocksStage)(nil)

func NewUnsafeBlocksStage(log log.Logger, client BlockByNumberSource, startingHead eth.L1BlockRef, processor BlockProcessor) *UnsafeBlocksStage {
	return &UnsafeBlocksStage{
		log:       log,
		client:    client,
		lastBlock: startingHead,
		processor: processor,
	}
}

func (s *UnsafeBlocksStage) Handle(ctx context.Context, head eth.L1BlockRef) {
	if head.Number <= s.lastBlock.Number {
		return
	}
	for s.lastBlock.Number+1 < head.Number {
		blockNum := s.lastBlock.Number + 1
		nextBlock, err := s.client.L1BlockRefByNumber(ctx, blockNum)
		if err != nil {
			s.log.Error("Failed to fetch block info", "number", blockNum, "err", err)
			return // Don't update the last processed block so we will retry fetching this block on next head update
		}
		if err := s.processor.ProcessBlock(ctx, nextBlock); err != nil {
			s.log.Error("Failed to process block", "block", nextBlock, "err", err)
			return // Don't update the last processed block so we will retry on next update
		}
		s.lastBlock = nextBlock
	}

	if err := s.processor.ProcessBlock(ctx, head); err != nil {
		s.log.Error("Failed to process block", "block", head, "err", err)
		return // Don't update the last processed block so we will retry on next update
	}
	s.lastBlock = head
}
