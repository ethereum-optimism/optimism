package source

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type Source interface {
	L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, gethtypes.Receipts, error)
}

type LogProcessor interface {
	ProcessLogs(ctx context.Context, block eth.L2BlockRef, receipts gethtypes.Receipts) error
}

type DatabaseRewinder interface {
	Rewind(chain types.ChainID, headBlockNum uint64) error
	LatestBlockNum(chain types.ChainID) (num uint64, ok bool)
}

type BlockProcessorFn func(ctx context.Context, block eth.L1BlockRef) error

func (fn BlockProcessorFn) ProcessBlock(ctx context.Context, block eth.L1BlockRef) error {
	return fn(ctx, block)
}

// ChainProcessor is a HeadProcessor that fills in any skipped blocks between head update events.
// It ensures that, absent reorgs, every block in the chain is processed even if some head advancements are skipped.
type ChainProcessor struct {
	log    log.Logger
	client Source

	chain types.ChainID

	processor LogProcessor
	rewinder  DatabaseRewinder

	// the last known head. May be 0 if not known.
	lastHead atomic.Uint64
	// channel with capacity of 1, full if there is work to do
	newHead chan struct{}

	// bool to indicate if calls are synchronous
	synchronous bool
	// channel with capacity of 1, to signal work complete if running in synchroneous mode
	out chan struct{}

	// lifetime management of the chain processor
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewChainProcessor(log log.Logger, client Source, chain types.ChainID, processor LogProcessor, rewinder DatabaseRewinder) *ChainProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	out := &ChainProcessor{
		log:       log,
		client:    client,
		chain:     chain,
		processor: processor,
		rewinder:  rewinder,
		newHead:   make(chan struct{}, 1),
		// default to synchronous because we want other processors to wait for this
		// in the future we could make this async and have a separate mechanism which forwards the work signal to other processors
		synchronous: true,
		out:         make(chan struct{}, 1),
		ctx:         ctx,
		cancel:      cancel,
	}
	out.wg.Add(1)
	go out.worker()
	return out
}

func (s *ChainProcessor) nextNum() uint64 {
	headNum, ok := s.rewinder.LatestBlockNum(s.chain)
	if !ok {
		return 0 // genesis. We could change this to start at a later block.
	}
	return headNum + 1
}

func (s *ChainProcessor) worker() {
	defer s.wg.Done()

	delay := time.NewTicker(time.Second * 5)
	for {
		if s.ctx.Err() != nil { // check if we are closing down
			return
		}
		target := s.nextNum()
		if err := s.update(target); err != nil {
			s.log.Error("Failed to process new block", "err", err)
			// idle until next update trigger
		} else if x := s.lastHead.Load(); target+1 <= x {
			s.log.Debug("Continuing with next block",
				"newTarget", target+1, "lastHead", x)
			continue // instantly continue processing, no need to idle
		} else {
			s.log.Debug("Idling block-processing, reached latest block", "head", target)
		}
		if s.synchronous {
			s.out <- struct{}{}
		}
		// await next time we process, or detect shutdown
		select {
		case <-s.ctx.Done():
			delay.Stop()
			return
		case <-s.newHead:
			s.log.Debug("Responding to new head signal")
			continue
		case <-delay.C:
			s.log.Debug("Checking for updates")
			continue
		}
	}
}

func (s *ChainProcessor) update(nextNum uint64) error {
	ctx, cancel := context.WithTimeout(s.ctx, time.Second*10)
	nextL1, err := s.client.L1BlockRefByNumber(ctx, nextNum)
	next := eth.L2BlockRef{
		Hash:       nextL1.Hash,
		ParentHash: nextL1.ParentHash,
		Number:     nextL1.Number,
		Time:       nextL1.Time,
	}
	cancel()
	if err != nil {
		return fmt.Errorf("failed to fetch next block: %w", err)
	}

	// Try and fetch the receipts
	ctx, cancel = context.WithTimeout(s.ctx, time.Second*10)
	_, receipts, err := s.client.FetchReceipts(ctx, next.Hash)
	cancel()
	if err != nil {
		return fmt.Errorf("failed to fetch receipts of block: %w", err)
	}
	if err := s.processor.ProcessLogs(ctx, next, receipts); err != nil {
		s.log.Error("Failed to process block", "block", next, "err", err)

		if next.Number == 0 { // cannot rewind genesis
			return nil
		}

		// Try to rewind the database to the previous block to remove any logs from this block that were written
		if err := s.rewinder.Rewind(s.chain, nextNum-1); err != nil {
			// If any logs were written, our next attempt to write will fail and we'll retry this rewind.
			// If no logs were written successfully then the rewind wouldn't have done anything anyway.
			s.log.Error("Failed to rewind after error processing block", "block", next, "err", err)
		}
	}
	return nil
}

func (s *ChainProcessor) OnNewHead(ctx context.Context, head eth.L1BlockRef) error {
	// update the latest target
	s.lastHead.Store(head.Number)
	// signal that we have something to process
	select {
	case s.newHead <- struct{}{}:
	default:
		// already requested an update
	}
	// if we are running synchronously, wait for the work to complete
	if s.synchronous {
		<-s.out
	}
	return nil
}

func (s *ChainProcessor) Close() {
	s.cancel()
	s.wg.Wait()
}
