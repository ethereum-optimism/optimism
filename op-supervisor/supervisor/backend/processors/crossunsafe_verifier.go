package processors

import (
	"context"
	"errors"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// CrossUnsafeVerifier iterates the local-safe data of a chain, and promotes blocks to cross-safe once dependencies are cross-safe
type CrossUnsafeVerifier struct {
	log log.Logger

	chain types.ChainID

	// current cross-safe DB iterator
	iter logs.Iterator

	// the last known local-safe head. May be 0 if not known.
	lastHead atomic.Uint64

	// channel with capacity of 1, full if there is work to do
	newHead chan struct{}

	// channel with capacity of 1, to signal work complete if running in synchroneous mode
	out chan struct{}

	// lifetime management of the chain processor
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewCrossUnsafeVerifier(log log.Logger, client Source, chain types.ChainID, processor LogProcessor, rewinder DatabaseRewinder) *CrossUnsafeVerifier {
	ctx, cancel := context.WithCancel(context.Background())
	out := &CrossUnsafeVerifier{
		log:     log,
		chain:   chain,
		newHead: make(chan struct{}, 1),
		out:     make(chan struct{}, 1),
		ctx:     ctx,
		cancel:  cancel,
	}
	out.wg.Add(1)
	go out.worker()
	return out
}

func (s *CrossUnsafeVerifier) nextNum() uint64 {
	// TODO look at cross-safe tip
	return 0
}

func (s *CrossUnsafeVerifier) worker() {
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

func (s *CrossUnsafeVerifier) update(nextNum uint64) error {
	// TODO init iterator if needed

	// TODO conditional iteration, with checks of cross-unsafe view
	err := vi.iter.TraverseConditional(func(state logs.IteratorState) error {
		hash, num, ok := state.SealedBlock()
		if !ok {
			return entrydb.ErrFuture // maybe a more specific error for no-genesis case?
		}
		// TODO(#11693): reorg check in the future. To make sure that what we traverse is still canonical.
		_ = hash
		// check if L2 block is within view
		if !vi.localView.WithinRange(num, 0) {
			return entrydb.ErrFuture
		}
		_, initLogIndex, ok := state.InitMessage()
		if !ok {
			return nil // no readable message, just an empty block
		}
		// check if the message is within view
		if !vi.localView.WithinRange(num, initLogIndex) {
			return entrydb.ErrFuture
		}
		// check if it is an executing message. If so, check the dependency
		if execMsg := state.ExecMessage(); execMsg != nil {
			// Check if executing message is within cross L2 view,
			// relative to the L1 view of current message.
			// And check if the message is valid to execute at all
			// (i.e. if it exists on the initiating side).
			// TODO(#12187): it's inaccurate to check with the view of the local-unsafe
			// it should be limited to the L1 view at the time of the inclusion of execution of the message.
			err := vi.validWithinView(vi.localDerivedFrom.Number, execMsg)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		panic("expected reader to complete with an exit-error")
	}
	if errors.Is(err, entrydb.ErrFuture) {
		// register the new cross-safe block as cross-safe up to the current L1 view
		return nil
	}

	// TODO check cross-L2 view
	{
		execChainID := types.ChainIDFromUInt64(uint64(execMsg.Chain))
		_, err := r.chains.Check(execChainID, execMsg.BlockNum, execMsg.LogIdx, execMsg.Hash)
		return err
	}
	return nil
}

func (s *CrossUnsafeVerifier) OnNewHead(ctx context.Context, head eth.BlockRef) error {
	// update the latest target
	s.lastHead.Store(head.Number)
	// signal that we have something to process
	select {
	case s.newHead <- struct{}{}:
	default:
		// already requested an update
	}
	return nil
}

func (s *CrossUnsafeVerifier) Close() {
	s.cancel()
	s.wg.Wait()
}
