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
func (s *Solver) NextMove(claim Claim) (*Claim, error) {
	// Special case of the root claim
	if claim.IsRoot() {
		return s.handleRoot(claim)
	}
	return s.handleMiddle(claim)
}

type StepData struct {
	LeafClaim  Claim
	StateClaim Claim
	IsAttack   bool
}

// AttemptStep determines what step should occur for a given leaf claim.
// An error will be returned if the claim is not at the max depth.
func (s *Solver) AttemptStep(claim Claim, state Game) (StepData, error) {
	if claim.Depth() != s.gameDepth {
		return StepData{}, errors.New("cannot step on non-leaf claims")
	}
	claimCorrect, err := s.agreeWithClaim(claim.ClaimData)
	if err != nil {
		return StepData{}, err
	}
	var selectorFn func(Claim) (Claim, error)
	if claimCorrect {
		selectorFn = state.PostStateClaim
	} else {
		selectorFn = state.PreStateClaim
	}
	stateClaim, err := selectorFn(claim)
	if err != nil {
		return StepData{}, err
	}
	return StepData{
		LeafClaim:  claim,
		StateClaim: stateClaim,
		IsAttack:   claimCorrect,
	}, nil
}

func (s *Solver) handleRoot(claim Claim) (*Claim, error) {
	agree, err := s.agreeWithClaim(claim.ClaimData)
	if err != nil {
		return nil, err
	}
	// Attack the root claim if we do not agree with it
	if !agree {
		return s.attack(claim)
	} else {
		return nil, nil
	}
}

func (s *Solver) handleMiddle(claim Claim) (*Claim, error) {
	parentCorrect, err := s.agreeWithClaim(claim.Parent)
	if err != nil {
		return nil, err
	}
	claimCorrect, err := s.agreeWithClaim(claim.ClaimData)
	if err != nil {
		return nil, err
	}
	if claim.Depth() == s.gameDepth {
		return nil, errors.New("game depth reached")
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
		return nil, nil
	} else if !parentCorrect && !claimCorrect {
		// We disagree with the parent so want to counter it (which the claim is doing)
		// but we also disagree with the claim so there must be a difference to the left of claim
		// Note that we will create the correct counter-claim for parent when it is evaluated, no need to do it here
		return s.attack(claim)
	}
	// This should not be reached
	return nil, errors.New("no next move")
}

// attack returns a response that attacks the claim.
func (s *Solver) attack(claim Claim) (*Claim, error) {
	position := claim.Attack()
	value, err := s.traceAtPosition(position)
	if err != nil {
		return nil, err
	}
	return &Claim{
		ClaimData: ClaimData{Value: value, Position: position},
		Parent:    claim.ClaimData,
	}, nil
}

// defend returns a response that defends the claim.
func (s *Solver) defend(claim Claim) (*Claim, error) {
	position := claim.Defend()
	value, err := s.traceAtPosition(position)
	if err != nil {
		return nil, err
	}
	return &Claim{
		ClaimData: ClaimData{Value: value, Position: position},
		Parent:    claim.ClaimData,
	}, nil
}

// agreeWithClaim returns true if the claim is correct according to the internal [TraceProvider].
func (s *Solver) agreeWithClaim(claim ClaimData) (bool, error) {
	ourValue, err := s.traceAtPosition(claim.Position)
	return ourValue == claim.Value, err
}

// traceAtPosition returns the [common.Hash] from internal [TraceProvider] at the given [Position].
func (s *Solver) traceAtPosition(p Position) (common.Hash, error) {
	index := p.TraceIndex(s.gameDepth)
	hash, err := s.Get(index)
	return hash, err
}
