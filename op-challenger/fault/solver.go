package fault

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

// Solver uses a [TraceProvider] to determine the moves to make in a dispute game.
type Solver struct {
	TraceProvider

	gameDepth int
}

// NewSolver creates a new [Solver] using the provided [TraceProvider].
func NewSolver(gameDepth int, traceProvider TraceProvider) *Solver {
	return &Solver{
		traceProvider,
		gameDepth,
	}
}

// NextMove returns the next move to make given the current state of the game.
func (s *Solver) NextMove(claim Claim, parent Claim) (*Response, error) {
	parentCorrect, err := s.agreeWithClaim(parent)
	if err != nil {
		return nil, err
	}
	claimCorrect, err := s.agreeWithClaim(claim)
	if err != nil {
		return nil, err
	}
	if parentCorrect && claimCorrect {
		// We agree with the parent, but the claim is disagreeing with it.
		// Since we agree with the claim, the difference must be to the right of the claim
		return s.defend(claim)
	} else if parentCorrect && !claimCorrect {
		// We agree with the parent, but the claim disagrees with it.
		// Since we disagree with the claim, the difference must be to the left of the claim
		return s.attack(claim)
	} else if !parentCorrect && claimCorrect {
		// Do nothing, we disagree with the parent, but this claim has correctly countered it
		return s.doNothing()
	} else if !parentCorrect && !claimCorrect {
		// We disagree with the parent so want to counter it (which the claim is doing)
		// but we also disagree with the claim so there must be a difference to the left of claim
		// Note that we will create the correct counter-claim for parent when it is evaluated, no need to do it here
		return s.attack(claim)
	}
	// This should not be reached
	return nil, errors.New("no next move")
}

func (s *Solver) doNothing() (*Response, error) {
	return nil, nil
}

// attack returns a response that attacks the claim.
func (s *Solver) attack(claim Claim) (*Response, error) {
	value, err := s.traceAtPosition(claim.Position.Attack())
	if err != nil {
		return nil, err
	}
	return &Response{Attack: true, Value: value}, nil
}

// defend returns a response that defends the claim.
func (s *Solver) defend(claim Claim) (*Response, error) {
	value, err := s.traceAtPosition(claim.Position.Defend())
	if err != nil {
		return nil, err
	}
	return &Response{Attack: false, Value: value}, nil
}

// agreeWithClaim returns true if the [Claim] is correct according to the internal [TraceProvider].
func (s *Solver) agreeWithClaim(claim Claim) (bool, error) {
	ourValue, err := s.traceAtPosition(claim.Position)
	return ourValue == claim.Value, err
}

// traceAtPosition returns the [common.Hash] from internal [TraceProvider] at the given [Position].
func (s *Solver) traceAtPosition(p Position) (common.Hash, error) {
	index := p.TraceIndex(s.gameDepth)
	hash, err := s.Get(index)
	return hash, err
}
