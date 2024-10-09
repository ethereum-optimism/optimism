package processors

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// CrossSafeVerifier iterates the local-safe data of a chain, and promotes blocks to cross-safe once dependencies are cross-safe
type CrossSafeVerifier struct {
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

func NewCrossSafeVerifier(log log.Logger, client Source, chain types.ChainID, processor LogProcessor, rewinder DatabaseRewinder) *CrossSafeVerifier {
	ctx, cancel := context.WithCancel(context.Background())
	out := &CrossSafeVerifier{
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

func (s *CrossSafeVerifier) nextNum() uint64 {
	// TODO look at cross-safe tip
	return 0
}

func (s *CrossSafeVerifier) worker() {
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

func (s *CrossSafeVerifier) update(nextNum uint64) error {
	// TODO init iterator if needed

	// TODO conditional iteration, with checks of cross-safe view

	execChainID := types.ChainIDFromUInt64(uint64(execMsg.Chain))

	// TODO check cross-L2 view
	// Check that the initiating message, which was pulled in by the executing message,
	// does indeed exist. And in which L2 block it exists (if any).
	l2BlockHash, err := r.chains.Check(execChainID, execMsg.BlockNum, execMsg.LogIdx, execMsg.Hash)
	if err != nil {
		return err
	}
	// if the executing message falls within the execFinalized range, then nothing to check
	execFinalized, ok := r.finalized[execChainID]
	if ok && execFinalized.Number > execMsg.BlockNum {
		return nil
	}
	// check if the L1 block of the executing message is known
	execL1Block, ok := r.derivedFrom[execChainID][l2BlockHash]
	if !ok {
		return entrydb.ErrFuture // TODO(#12185) need to distinguish between same-data future, and new-data future
	}
	// check if the L1 block is within the view
	if execL1Block.Number > l1View {
		return fmt.Errorf("exec message depends on L2 block %s:%d, derived from L1 block %s, not within view yet: %w",
			l2BlockHash, execMsg.BlockNum, execL1Block, entrydb.ErrFuture)
	}
	return nil

	return nil
}

func (s *CrossSafeVerifier) OnNewHead(ctx context.Context, head eth.BlockRef) error {
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

func (s *CrossSafeVerifier) Close() {
	s.cancel()
	s.wg.Wait()
}
