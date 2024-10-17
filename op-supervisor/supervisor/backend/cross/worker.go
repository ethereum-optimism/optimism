package cross

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const (
	// The data may have changed, and we may have missed a poke, so re-attempt regularly.
	pollCrossSafeUpdateDuration = time.Second * 4
	// Make sure to flush cross-unsafe updates to the DB regularly when there are large spans of data
	maxCrossSafeUpdateDuration = time.Second * 4
)

// Worker iterates and promotes blocks to cross-safe once dependencies are met
// TODO: generics to make the worker function for both cross-safe and cross-unsafe
type Worker struct {
	log log.Logger

	chain types.ChainID

	deps CrossSafeDeps

	// channel with capacity of 1, full if there is work to do
	poke chan struct{}

	// channel with capacity of 1, to signal work complete if running in synchroneous mode
	out chan struct{}

	// lifetime management of the chain processor
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewCrossSafeVerifier(log log.Logger, chain types.ChainID, deps CrossSafeDeps) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	out := &Worker{
		log:    log,
		chain:  chain,
		deps:   deps,
		poke:   make(chan struct{}, 1),
		out:    make(chan struct{}, 1),
		ctx:    ctx,
		cancel: cancel,
	}
	out.wg.Add(1)
	go out.worker()
	return out
}

func (s *Worker) worker() {
	defer s.wg.Done()

	delay := time.NewTicker(pollCrossSafeUpdateDuration)
	for {
		if s.ctx.Err() != nil { // check if we are closing down
			return
		}

		ctx, cancel := context.WithTimeout(s.ctx, maxCrossSafeUpdateDuration)
		err := s.scope.Process(ctx)
		cancel()
		if err != nil {
			if errors.Is(err, ctx.Err()) {
				s.log.Debug("Processed some, but not all data", "err", err)
			} else {
				s.log.Error("Failed to process new block", "err", err)
			}
			// idle until next update trigger (or resource-context may make the worker stop)
		} else {
			s.log.Debug("Continuing cross-safe-processing")
			continue
		}

		// await next time we process, or detect shutdown
		select {
		case <-s.ctx.Done():
			delay.Stop()
			return
		case <-s.poke:
			s.log.Debug("Continuing cross-safe verification after hint of new data")
			continue
		case <-delay.C:
			s.log.Debug("Checking for cross-safe updates")
			continue
		}
	}
}

func (s *Worker) OnNewData() error {
	// signal that we have something to process
	select {
	case s.poke <- struct{}{}:
	default:
		// already requested an update
	}
	return nil
}

func (s *Worker) Close() {
	s.cancel()
	s.wg.Wait()
}
