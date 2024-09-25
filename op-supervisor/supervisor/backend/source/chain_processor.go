package source

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/log"
)

type BlockByNumberSource interface {
	L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error)
}

type BlockProcessor interface {
	ProcessBlock(ctx context.Context, block eth.L1BlockRef) error
}

type DatabaseRewinder interface {
	Rewind(chain types.ChainID, headBlockNum uint64) error
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
	chain     types.ChainID
	lastBlock eth.L1BlockRef
	processor BlockProcessor
	rewinder  DatabaseRewinder
}

func NewChainProcessor(log log.Logger, client BlockByNumberSource, chain types.ChainID, startingHead eth.L1BlockRef, processor BlockProcessor, rewinder DatabaseRewinder) *ChainProcessor {
	return &ChainProcessor{
		log:       log,
		client:    client,
		chain:     chain,
		lastBlock: startingHead,
		processor: processor,
		rewinder:  rewinder,
	}
}

func (s *ChainProcessor) OnNewHead(ctx context.Context, head eth.L1BlockRef) {
	s.log.Debug("Processing chain", "chain", s.chain, "head", head, "last", s.lastBlock)
	if head.Number <= s.lastBlock.Number {
		s.log.Info("head is not newer than last processed block", "head", head, "lastBlock", s.lastBlock)
		return
	}
	for s.lastBlock.Number+1 < head.Number {
		s.log.Debug("Filling in skipped block", "chain", s.chain, "lastBlock", s.lastBlock, "head", head)
		blockNum := s.lastBlock.Number + 1
		nextBlock, err := s.client.L1BlockRefByNumber(ctx, blockNum)
		if err != nil {
			s.log.Error("Failed to fetch block info", "number", blockNum, "err", err)
			return
		}
		if ok := s.processBlock(ctx, nextBlock); !ok {
			return
		}
	}

	s.processBlock(ctx, head)
}

func (s *ChainProcessor) processBlock(ctx context.Context, block eth.L1BlockRef) bool {
	if err := s.processor.ProcessBlock(ctx, block); err != nil {
		s.log.Error("Failed to process block", "block", block, "err", err)
		// Try to rewind the database to the previous block to remove any logs from this block that were written
		if err := s.rewinder.Rewind(s.chain, s.lastBlock.Number); err != nil {
			// If any logs were written, our next attempt to write will fail and we'll retry this rewind.
			// If no logs were written successfully then the rewind wouldn't have done anything anyway.
			s.log.Error("Failed to rewind after error processing block", "block", block, "err", err)
		}
		return false // Don't update the last processed block so we will retry on next update
	}
	s.lastBlock = block
	return true
}
