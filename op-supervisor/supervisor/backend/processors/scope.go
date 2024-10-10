package processors

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type EndStatus uint8

const (
	// OpenEnd is an end where we can make progress from.
	OpenEnd EndStatus = iota

	// BlockedEnd is a piece of work that needs to be unblocked by something else.
	BlockedEnd

	// ExhaustedEnd is a previously open end where we ran out of L1 data.
	ExhaustedEnd

	// InvalidEnd is an open end that is blocked on something that is not there
	InvalidEnd
)

type Deps interface {
	Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash common.Hash) (includedIn eth.BlockID, err error)

	LocalDerivedFrom(chainID types.ChainID, derived eth.BlockID) (derivedFrom eth.BlockID, err error)
}

type End struct {
	chainID types.ChainID

	// Cached iterator, so we can proceed to efficiently get the next data when we need to.
	iter logs.Iterator

	// Parent block that the logs are building on top of.
	parent types.BlockSeal
	// Verified logs thus far. Index of the log yet to be verified (if any).
	logsSince uint32
	// Block that contains the logs
	current types.BlockSeal
	// Number of logs in current.
	logsTotal uint32

	// What current was locally-derived from.
	localDerivedFrom types.BlockSeal

	// Non-nil if the current log is trying to execute anything.
	// Set to nil when the executing message has been verified.
	Executing *types.ExecutingMessage

	Status EndStatus
}

func (e *End) TryNext() error {
	if e.Executing != nil {
		panic("cannot continue before resolving execution link")
	}
	// TODO: read from iterator / DB, and update the End state

	return nil
}

type Scope struct {
	// Point in L1 that may consumed up till.
	// The L1 view is bounded, so we can process accurate cross-safe increments.
	L1Bound types.BlockSeal

	// The ends, one per chain, that we want to synchronously resolve dependencies of.
	Ends []*End

	// Chains we are tracking an End of
	InvolvedChains map[types.ChainID]struct{}

	deps Deps
}

func (s *Scope) Process(ctx context.Context) error {
	if len(s.Ends) == 0 { // nothing to process
		return nil
	}
	// We keep revisiting the set of ends, until there's no more change.
	for {
		anyChange := false
		for _, end := range s.Ends {
			// Only open ends may be processed.
			// Other ends are done, or need to be unblocked by others.
			if end.Status != OpenEnd {
				continue
			}
			if err := s.ProcessOpenEnd(end); err != nil {
				if errors.Is(err, entrydb.ErrFuture) {
					// Insufficient data to proceed with this end. Continue with the next end.
					continue
				} else {
					return fmt.Errorf("failed to process chain %s: %w", end.chainID, err)
				}
			} else {
				anyChange = true
			}
		}
		if !anyChange {
			break
		}
	}
	var openEnds, blockedEnds, exhaustedEnds, invalidEnds int
	for _, end := range s.Ends {
		switch end.Status {
		case OpenEnd:
			openEnds += 1
		case BlockedEnd:
			blockedEnds += 1
		case ExhaustedEnd:
			exhaustedEnds += 1
		case InvalidEnd:
			invalidEnds += 1
		}
	}
	// If any ends are still open: we stopped early.
	if openEnds > 0 {
		// if we use a context for processing, we can hit this
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("incomplete data, unable to proceed to next scope")
	}
	if invalidEnds > 0 {
		return errors.New("found invalid end, blocked")
	}
	// If all ends are blocked: we found a cycle that cannot be resolved with additional L1 data,
	// we'll need to reorg an L2 chain.
	if blockedEnds == len(s.Ends) {
		return errors.New("every end is blocked")
	}
	if exhaustedEnds == 0 {
		return errors.New("expected to have exhausted L1 view")
	}

	// TODO increment L1 bound

	// TODO Everything that was exhausted for L1 data should be marked as Open again
	return nil
}

func (s *Scope) AddChain(chainID types.ChainID) {
	end := &End{
		chainID:          chainID,
		iter:             nil,
		parent:           types.BlockSeal{},
		logsSince:        0,
		current:          types.BlockSeal{},
		logsTotal:        0,
		localDerivedFrom: types.BlockSeal{},
		Executing:        nil,
		Status:           OpenEnd,
	}
	// TODO
	s.Ends = append(s.Ends, end)
	s.InvolvedChains[chainID] = struct{}{}
}

func (s *Scope) ProcessOpenEnd(end *End) error {

	// If not derived within the L1Bound: put it in Exhausted.
	// This is L2 data that we cannot yet touch, it's outside of view.
	if end.localDerivedFrom.Number > s.L1Bound.Number {
		end.Status = ExhaustedEnd
		return nil
	}

	// If we run into the end of a block:
	// -> mark it as cross-safe derived-from current L1 view
	if end.logsSince == end.logsTotal {
		// TODO: write cross-safe update.
		//  But subtle bug: if transitive block dependency is not cross-safe up to and including the seal,
		//  then it might never become cross-safe as a whole, and thus invalidate this cross-safe update.
		return nil
	}

	// if we run into an executing message:
	if end.Executing != nil {
		execChID := types.ChainIDFromUInt64(uint64(end.Executing.Chain))

		// Check that the message exists
		includedIn, err := s.deps.Check(execChID, end.Executing.BlockNum, end.Executing.LogIdx, end.Executing.Hash)
		if err != nil {
			if errors.Is(err, entrydb.ErrConflict) {
				end.Status = InvalidEnd
				return err
			}
		}
		// Check if within L1 view: checking it is locally derived within L1 view should be enough,
		// since within this Scope it would not be counted as cross-safe in L2 if it wasn't transitively within L1 view.
		localDerivedFrom, err := s.deps.LocalDerivedFrom(execChID, includedIn)
		if err != nil {
			return err
		}
		if localDerivedFrom.Number > s.L1Bound.Number {
			end.Status = ExhaustedEnd
			return nil
		}

		// Check that we are tracking the end of the requested chain
		if _, ok := s.InvolvedChains[execChID]; !ok {
			s.AddChain(execChID)
		}

		// Check if the message is within L2 view
		for _, other := range s.Ends {
			if other.chainID != execChID {
				continue
			}
			// By checking the logsSince, we can resolve intra-block messaging.
			if end.Executing.BlockNum < other.current.Number ||
				(end.Executing.BlockNum == other.current.Number && end.Executing.LogIdx <= other.logsSince) {
				// covered within tentative cross-safe range!
				end.Executing = nil
				return nil
			}
		}

		end.Status = BlockedEnd
		return nil
	}

	// Try to traverse on the open end
	return end.TryNext()
}
