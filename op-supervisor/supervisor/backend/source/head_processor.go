package source

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

type HeadProcessor interface {
	OnNewHead(ctx context.Context, head eth.L1BlockRef)
}

type HeadProcessorFn func(ctx context.Context, head eth.L1BlockRef)

func (f HeadProcessorFn) OnNewHead(ctx context.Context, head eth.L1BlockRef) {
	f(ctx, head)
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
		processor.OnNewHead(ctx, block)
	}
}

func (n *headUpdateProcessor) OnNewSafeHead(ctx context.Context, block eth.L1BlockRef) {
	n.log.Debug("New safe head", "block", block)
	for _, processor := range n.safeProcessors {
		processor.OnNewHead(ctx, block)
	}
}
func (n *headUpdateProcessor) OnNewFinalizedHead(ctx context.Context, block eth.L1BlockRef) {
	n.log.Debug("New finalized head", "block", block)
	for _, processor := range n.finalizedProcessors {
		processor.OnNewHead(ctx, block)
	}
}
