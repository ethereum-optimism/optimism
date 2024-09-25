package source

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type HeadProcessor interface {
	OnNewHead(ctx context.Context, head eth.L1BlockRef) error
}

type HeadProcessorFn func(ctx context.Context, head eth.L1BlockRef) error

func (f HeadProcessorFn) OnNewHead(ctx context.Context, head eth.L1BlockRef) error {
	return f(ctx, head)
}

// headUpdateProcessor handles head update events and routes them to the appropriate handlers
type headUpdateProcessor struct {
	log                 log.Logger
	unsafeProcessors    []HeadProcessor
	safeProcessors      []HeadProcessor
	finalizedProcessors []HeadProcessor
}

func newHeadUpdateProcessor(log log.Logger, unsafeProcessors []HeadProcessor, safeProcessors []HeadProcessor, finalizedProcessors []HeadProcessor) *headUpdateProcessor {
	return &headUpdateProcessor{
		log:                 log,
		unsafeProcessors:    unsafeProcessors,
		safeProcessors:      safeProcessors,
		finalizedProcessors: finalizedProcessors,
	}
}

func (n *headUpdateProcessor) OnNewUnsafeHead(ctx context.Context, block eth.L1BlockRef) {
	n.log.Debug("New unsafe head", "block", block)
	for _, processor := range n.unsafeProcessors {
		if err := processor.OnNewHead(ctx, block); err != nil {
			n.log.Error("unsafe-head processing failed", "err", err)
		}
	}
}

func (n *headUpdateProcessor) OnNewSafeHead(ctx context.Context, block eth.L1BlockRef) {
	n.log.Debug("New safe head", "block", block)
	for _, processor := range n.safeProcessors {
		if err := processor.OnNewHead(ctx, block); err != nil {
			n.log.Error("safe-head processing failed", "err", err)
		}
	}
}

func (n *headUpdateProcessor) OnNewFinalizedHead(ctx context.Context, block eth.L1BlockRef) {
	n.log.Debug("New finalized head", "block", block)
	for _, processor := range n.finalizedProcessors {
		if err := processor.OnNewHead(ctx, block); err != nil {
			n.log.Error("finalized-head processing failed", "err", err)
		}
	}
}

// OnNewHead is a util function to turn a head-signal processor into head-pointer updater
func OnNewHead(id types.ChainID, apply func(id types.ChainID, v heads.HeadPointer) error) HeadProcessorFn {
	return func(ctx context.Context, head eth.L1BlockRef) error {
		return apply(id, heads.HeadPointer{
			LastSealedBlockHash: head.Hash,
			LastSealedBlockNum:  head.Number,
			LogsSince:           0,
		})
	}
}
