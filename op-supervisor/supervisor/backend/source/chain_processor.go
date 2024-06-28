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

// ChainProcessor is a HeadProcessor that fills in any skipped blocks between head update events.
// It ensures that, absent reorgs, every block in the chain is processed even if some head advancements are skipped.
type ChainProcessor struct {
	log       log.Logger
	client    BlockByNumberSource
	lastBlock eth.L1BlockRef
	processor BlockProcessor
}

func NewChainProcessor(log log.Logger, client BlockByNumberSource, startingHead eth.L1BlockRef, processor BlockProcessor) *ChainProcessor {
	return &ChainProcessor{
		log:       log,
		client:    client,
		lastBlock: startingHead,
		processor: processor,
	}
}

func (s *ChainProcessor) OnNewHead(ctx context.Context, head eth.L1BlockRef) {
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
