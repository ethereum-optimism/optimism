package processors

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const (
	// The data may have changed, and we may have missed a poke, so re-attempt regularly.
	pollCrossUnsafeUpdateDuration = time.Second * 4
	// Make sure to flush cross-unsafe updates to the DB regularly when there are large spans of data
	maxCrossUnsafeUpdateDuration = time.Second * 4
)

type CrossUnsafeDBDeps interface {
	LocalUnsafe() types.HeadPointer
	CrossSafe(chainId types.ChainID) types.HeadPointer
	CrossUnsafe(chainID types.ChainID) types.HeadPointer
	LogsIteratorAt(chainID types.ChainID, at types.HeadPointer) (logs.Iterator, error)
	Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash common.Hash) error
	UpdateCrossUnsafe(chain types.ChainID, crossUnsafe types.HeadPointer) error
}

// CrossUnsafeVerifier iterates the local-safe data of a chain, and promotes blocks to cross-safe once dependencies are cross-safe
type CrossUnsafeVerifier struct {
	log log.Logger

	chain types.ChainID

	deps CrossUnsafeDBDeps

	// current cross-unsafe logs-DB iterator
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

func NewCrossUnsafeVerifier(log log.Logger, chain types.ChainID, deps CrossUnsafeDBDeps) *CrossUnsafeVerifier {
	ctx, cancel := context.WithCancel(context.Background())
	out := &CrossUnsafeVerifier{
		log:     log,
		chain:   chain,
		deps:    deps,
		newHead: make(chan struct{}, 1),
		out:     make(chan struct{}, 1),
		ctx:     ctx,
		cancel:  cancel,
	}
	out.wg.Add(1)
	go out.worker()
	return out
}

func (s *CrossUnsafeVerifier) worker() {
	defer s.wg.Done()

	delay := time.NewTicker(pollCrossUnsafeUpdateDuration)
	for {
		if s.ctx.Err() != nil { // check if we are closing down
			return
		}

		if err := s.update(); err != nil {
			s.log.Error("Failed to process new block", "err", err)
			// idle until next update trigger
		} else {
			s.log.Debug("Continuing cross-unsafe-processing")
			continue
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

func (s *CrossUnsafeVerifier) update() error {
	ctx, cancel := context.WithTimeout(s.ctx, maxCrossUnsafeUpdateDuration)
	defer cancel()

	// TODO init iterator if needed

	iter, err := s.deps.LogsIteratorAt()

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
			if err := s.deps.Check(chainID, execMsg.BlockNum, execMsg.LogIdx, execMsg.Hash); err != nil {
				return fmt.Errorf("failed to check %s: %w", execMsg, err)
			}
		}
		return nil
	})
	if err == nil {
		panic("expected reader to complete with an exit-error")
	}

	crossUnsafe, err := iter.HeadPointer()
	if err != nil {
		return fmt.Errorf("failed to get head pointer: %w", err)
	}

	// register the new cross-safe block as cross-safe up to the current L1 view
	if err := s.deps.UpdateCrossUnsafe(s.chain, crossUnsafe); err != nil {
		return fmt.Errorf("failed to write cross-unsafe update: %w", err)
	}

	// If we stopped iterating after running out of time, instead of out of data, then we can continue immediately
	if errors.Is(err, ctx.Err()) {
		return nil
	}
	return err
}

func (s *CrossUnsafeVerifier) OnNewData() error {
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
