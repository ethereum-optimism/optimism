package processors

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const (
	// The data may have changed, and we may have missed a poke, so re-attempt regularly.
	pollCrossSafeUpdateDuration = time.Second * 4
	// Make sure to flush cross-unsafe updates to the DB regularly when there are large spans of data
	maxCrossSafeUpdateDuration = time.Second * 4
)

type CrossSafeDBDeps interface {
	LocalUnsafe() types.HeadPointer
	CrossSafe(chainId types.ChainID) types.HeadPointer
	CrossUnsafe(chainID types.ChainID) types.HeadPointer

	Finalized(chainID types.ChainID) (eth.BlockID, error)
	DerivedFrom(chainID types.ChainID, derived eth.BlockID) (derivedFrom eth.BlockID, err error)

	LogsIteratorAt(chainID types.ChainID, at types.HeadPointer) (logs.Iterator, error)
	Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash common.Hash) (includedIn eth.BlockID, err error)
	UpdateCrossSafe(chain types.ChainID, l1View eth.BlockID, crossUnsafe types.HeadPointer) error
}

// CrossSafeVerifier iterates the local-safe data of a chain, and promotes blocks to cross-safe once dependencies are cross-safe
type CrossSafeVerifier struct {
	log log.Logger

	chain types.ChainID

	deps CrossSafeDBDeps

	// current cross-safe DB iterator
	iter logs.Iterator

	// channel with capacity of 1, full if there is work to do
	poke chan struct{}

	// channel with capacity of 1, to signal work complete if running in synchroneous mode
	out chan struct{}

	// lifetime management of the chain processor
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewCrossSafeVerifier(log log.Logger, chain types.ChainID, deps CrossSafeDBDeps) *CrossSafeVerifier {
	ctx, cancel := context.WithCancel(context.Background())
	out := &CrossSafeVerifier{
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

func (s *CrossSafeVerifier) worker() {
	defer s.wg.Done()

	delay := time.NewTicker(pollCrossSafeUpdateDuration)
	for {
		if s.ctx.Err() != nil { // check if we are closing down
			return
		}

		if err := s.update(); err != nil {
			s.log.Error("Failed to process new block", "err", err)
			// idle until next update trigger
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

func (s *CrossSafeVerifier) update() error {
	ctx, cancel := context.WithTimeout(s.ctx, maxCrossUnsafeUpdateDuration)
	defer cancel()

	// TODO init iterator if needed

	iter, err := s.deps.LogsIteratorAt()

	l1View := eth.BlockID{}

	err := s.iter.TraverseConditional(func(state logs.IteratorState) error {
		// we can stop early, to make some progress, and not indefinitely iterate, when there is a lot of unsafe data.
		if ctx.Err() != nil {
			return ctx.Err()
		}

		hash, num, ok := state.SealedBlock()
		if !ok {
			return entrydb.ErrFuture // maybe a more specific error for no-genesis case?
		}
		// TODO(#11693): reorg check in the future. To make sure that what we traverse is still canonical.
		_, _ = hash, num

		_, _, ok = state.InitMessage()
		if !ok {
			return nil // no readable message, just an empty block
		}

		// check if it is an executing message. If so, check the dependency.
		if execMsg := state.ExecMessage(); execMsg != nil {
			chainID := types.ChainIDFromUInt64(uint64(execMsg.Chain))

			// TODO: go over L1 views, transitive dependencies are a problem

			// Check that the initiating message, which was pulled in by the executing message,
			// does indeed exist. And in which L2 block it is included in (if any).
			includedIn, err := s.deps.Check(chainID, execMsg.BlockNum, execMsg.LogIdx, execMsg.Hash)
			if err != nil {
				return fmt.Errorf("failed to check %s: %w", execMsg, err)
			}

			// if the executing message falls within the execFinalized range, then nothing to check
			execFinalized, err := s.deps.Finalized(chainID)
			if err != nil {
				return fmt.Errorf("failed to check finalized block: %s", err)
			}
			if execFinalized.Number > execMsg.BlockNum {
				return nil
			}
			// check if the L1 block of the executing message is known
			derivedFrom, err := s.deps.DerivedFrom(chainID, includedIn)
			if err != nil {
				return err
			}
			// check if the L1 block is within the view
			if derivedFrom.Number > l1View.Number {
				return fmt.Errorf("exec message depends on L2 block %s, derived from L1 block %s, not within view %s yet: %w",
					includedIn, derivedFrom, l1View, entrydb.ErrFuture)
			}
		}
		return nil
	})
	if err == nil {
		panic("expected reader to complete with an exit-error")
	}

	crossSafe, err := iter.HeadPointer()
	if err != nil {
		return fmt.Errorf("failed to get head pointer: %w", err)
	}

	// register the new cross-safe block as cross-safe up to the current L1 view
	if err := s.deps.UpdateCrossSafe(s.chain, l1View, crossSafe); err != nil {
		return fmt.Errorf("failed to write cross-safe update: %w", err)
	}

	// If we stopped iterating after running out of time, instead of out of data, then we can continue immediately
	if errors.Is(err, ctx.Err()) {
		return nil
	}
	return err
}

func (s *CrossSafeVerifier) OnNewData() error {
	// signal that we have something to process
	select {
	case s.poke <- struct{}{}:
	default:
		// already requested an update
	}
	return nil
}

func (s *CrossSafeVerifier) Close() {
	s.cancel()
	s.wg.Wait()
}
