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

	// safe and unsafe dependencies
	// only one of these is expected to be used by the worker
	// and would be used in conjunction with a matching workFn
	safeDeps   CrossSafeDeps
	unsafeDeps CrossUnsafeDeps
	// workFn is the function to call to process the scope
	workFn workFn

	// channel with capacity of 1, full if there is work to do
	poke chan struct{}

	// channel with capacity of 1, to signal work complete if running in synchroneous mode
	out chan struct{}

	// lifetime management of the chain processor
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewCrossSafeVerifier creates a new worker to process cross-safe updates
// it sets the safeDeps, and the workFn to a crossSafeWorkFn
func NewCrossSafeVerifier(log log.Logger, chain types.ChainID, deps CrossSafeDeps) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	out := &Worker{
		log:      log,
		chain:    chain,
		safeDeps: deps,
		poke:     make(chan struct{}, 1),
		out:      make(chan struct{}, 1),
		ctx:      ctx,
		cancel:   cancel,
	}
	out.workFn = crossSafeWorkFn(out)
	out.wg.Add(1)
	go out.worker()
	return out
}

// NewCrossUnsafeVerifier creates a new worker to process cross-unsafe updates
// it sets the unsafeDeps, and the workFn to a crossUnsafeWorkFn
func NewCrossUnsafeVerifier(log log.Logger, chain types.ChainID, deps CrossUnsafeDeps) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	out := &Worker{
		log:        log,
		chain:      chain,
		unsafeDeps: deps,
		poke:       make(chan struct{}, 1),
		out:        make(chan struct{}, 1),
		ctx:        ctx,
		cancel:     cancel,
	}
	out.workFn = crossUnsafeWorkFn(out)
	out.wg.Add(1)
	go out.worker()
	return out
}

// workFn is a function used by the worker
// it is opaque to the worker, and is set by the constructor
type workFn func() error

// crossUnsafeWorkFn creates a workFn to process cross-unsafe updates for a worker
func crossUnsafeWorkFn(s *Worker) workFn {
	return func() error {
		ctx, cancel := context.WithTimeout(s.ctx, maxCrossSafeUpdateDuration)
		// TODO: rather than "scope.Process", we need an actual function to process the scope
		// as far as I can tell, no such thing exists as "scope" in this context
		err := s.scope.Process(ctx)
		cancel()
		if err != nil {
			if errors.Is(err, ctx.Err()) {
				s.log.Debug("Processed some, but not all data", "err", err)
				return err
			} else {
				s.log.Error("Failed to process new block", "err", err)
				return err
			}
		}
		s.log.Debug("Continuing cross-safe-processing")
		return nil
	}
}

// crossSafeWorkFn creates a workFn to process cross-safe updates for a worker
func crossSafeWorkFn(s *Worker) workFn {
	return func() error {
		ctx, cancel := context.WithTimeout(s.ctx, maxCrossSafeUpdateDuration)
		// TODO: rather than "scope.Process", we need an actual function to process the scope
		// as far as I can tell, no such thing exists as "scope" in this context
		err := s.scope.Process(ctx)
		cancel()
		if err != nil {
			if errors.Is(err, ctx.Err()) {
				s.log.Debug("Processed some, but not all data", "err", err)
				return err
			} else {
				s.log.Error("Failed to process new block", "err", err)
				return err
			}
		}
		s.log.Debug("Continuing cross-safe-processing")
		return nil
	}
}

func (s *Worker) worker() {
	defer s.wg.Done()

	delay := time.NewTicker(pollCrossSafeUpdateDuration)
	for {
		if s.ctx.Err() != nil { // check if we are closing down
			return
		}

		// do the work
		err := s.workFn()
		if err == nil {
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
