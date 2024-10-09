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

	CrossDerivedFrom(chainID types.ChainID, derived eth.BlockID, logIndex uint32) (derivedFrom eth.BlockID, err error)
	LocalDerivedFrom(chainID types.ChainID, derived eth.BlockID) (derivedFrom eth.BlockID, err error)

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

// cross safe means:
//    - at L2 block X
//    - derived from L1 block Y
//    - with L2 dependencies A, B, C, ...
//    - each L2 dependency is included in cross-safe view
//    - intra-block dependencies are a thing too. We have intra-block cross-safe increments.
//      Transitive dependencies can thus only be resolved if we maintain how many of the logs we have verified in an L2 block.
//      So we can zig-zag between L2 chains until all dependencies have been resolved.
//    - each time we look at a dependency, we have to verify it's derived from the same canonical L1 chain, within view

// within local-safe: iterate
//
// on executing message:
// check where it's included              -> events DB lookup
// check what that is cross-derived from        -> cross DB
// check that it is within view

var errNeedNextL1Data = errors.New("need next L1 data")

func (s *CrossSafeVerifier) update() error {
	ctx, cancel := context.WithTimeout(s.ctx, maxCrossUnsafeUpdateDuration)
	defer cancel()

	// TODO init iterator if needed

	iter, err := s.deps.LogsIteratorAt()

	l1View := eth.BlockID{}
	// TODO acquire a reorg-lock, so that all operations against the
	//  chains DB will be with the ensurance of this L1 block.
	// The acquire might fail if there's a reorg and data is already inconsistent between chains.

	// TODO defer unlock the reorg-lock

	err := s.iter.TraverseConditional(func(state logs.IteratorState) error {
		// we can stop early, to make some progress, and not indefinitely iterate, when there is a lot of unsafe data.
		if ctx.Err() != nil {
			return ctx.Err()
		}

		h, n, ok := state.SealedBlock()
		if !ok {
			return nil // no readable block
		}
		localDerivedFrom, err := s.deps.LocalDerivedFrom(s.chain, eth.BlockID{Hash: h, Number: n})
		if localDerivedFrom.Number > l1View.Number {
			return errNeedNextL1Data
		}

		_, _, ok = state.InitMessage()
		if !ok {
			return nil // no readable message, just an empty block
		}

		// check if it is an executing message. If so, check the dependency.
		if execMsg := state.ExecMessage(); execMsg != nil {
			chainID := types.ChainIDFromUInt64(uint64(execMsg.Chain))

			// Check that the initiating message, which was pulled in by the executing message,
			// does indeed exist in an L2 block.
			includedIn, err := s.deps.Check(chainID, execMsg.BlockNum, execMsg.LogIdx, execMsg.Hash)
			if err != nil {
				// TODO this can be an ErrFuture where we just don't have the data for a definite answer,
				// but can also be an ErrConflict where we know the message isn't valid, and we have to reorg.
				return fmt.Errorf("failed to check %s: %w", execMsg, err)
			}

			// The L2 it was included in must be cross-safe up to and including the message.
			// But not necessarily fully, since we can go back and forth between chains, intra-block.
			// The L2 blockhash ensures we are still checking the same log content.
			crossDerivedFrom, err := s.deps.CrossDerivedFrom(chainID, includedIn, execMsg.LogIdx)
			if err != nil {
				// TODO this can be ErrFuture when we don't know the cross-derivation link of the L2 block yet.
				return fmt.Errorf("failed to inspect cross-derived-from: %w", err)
			}

			// Check if the L1 block is within the view. Even if it exists,
			// it might not be time to accept safety-promotion of the executing-message yet,
			// due to the dependency on later L1 data.
			if crossDerivedFrom.Number > l1View.Number {
				// The executing message depends on an L2 block which is cross-derived from an L1 block that is not within view yet.
				// So we need to traverse L1 to bring it into view.
				return errNeedNextL1Data
			}
		}
		return nil
	})
	if err == nil {
		panic("expected reader to complete with an exit-error")
	}

	preH, preN, ok := iter.SealedBlock()
	logsSince, ok := iter.LogsSince()

	// TODO traverse to next block-seal to find post-state hash

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
