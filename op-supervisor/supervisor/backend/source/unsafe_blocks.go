package source

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

type BlockByNumberSource interface {
	L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error)
}

// UnsafeBlocksStage is a PipelineEventHandler that watches for UnsafeHeadEvent and backfills any skipped blocks.
// It emits a UnsafeBlockEvent for each block up to the new unsafe head.
type UnsafeBlocksStage struct {
	log       log.Logger
	client    BlockByNumberSource
	lastBlock eth.L1BlockRef
}

var _ PipelineEventHandler = (*UnsafeBlocksStage)(nil)

func NewUnsafeBlocksStage(log log.Logger, client BlockByNumberSource, startingHead eth.L1BlockRef) *UnsafeBlocksStage {
	return &UnsafeBlocksStage{
		log:       log,
		client:    client,
		lastBlock: startingHead,
	}
}

func (s *UnsafeBlocksStage) Handle(ctx context.Context, evt PipelineEvent, out chan<- PipelineEvent) {
	headEvt, ok := evt.(UnsafeHeadEvent)
	if !ok {
		return
	}
	if headEvt.Block.Number <= s.lastBlock.Number {
		return
	}
	for s.lastBlock.Number+1 < headEvt.Block.Number {
		blockNum := s.lastBlock.Number + 1
		nextBlock, err := s.client.L1BlockRefByNumber(ctx, blockNum)
		if err != nil {
			s.log.Error("Failed to fetch block info", "number", blockNum, "err", err)
			return // Don't update the last processed block so we will retry fetching this block on next head update
		}
		out <- UnsafeBlockEvent{nextBlock}
		s.lastBlock = nextBlock
	}
	out <- UnsafeBlockEvent(headEvt)
	s.lastBlock = headEvt.Block
}
