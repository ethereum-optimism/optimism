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
func (s *Solver) NextMove(claim Claim, agreeWithClaimLevel bool) (*Claim, error) {
	if agreeWithClaimLevel {
		return nil, nil
	}

	// Special case of the root claim
	if claim.IsRoot() {
		return s.handleRoot(claim)
	}
	return s.handleMiddle(claim)
}

func (s *Solver) handleRoot(claim Claim) (*Claim, error) {
	agree, err := s.agreeWithClaim(claim.ClaimData)
	if err != nil {
		return nil, err
	}
	// Attack the root claim if we do not agree with it
	// Note: We always disagree with the claim level at this point,
	// so if we agree with claim maybe we should also attack?
	if !agree {
		return s.attack(claim)
	} else {
		return nil, nil
	}
}

func (s *Solver) handleMiddle(claim Claim) (*Claim, error) {
	claimCorrect, err := s.agreeWithClaim(claim.ClaimData)
	if err != nil {
		return nil, err
	}
	if claim.Depth() == s.gameDepth {
		return nil, errors.New("game depth reached")
	}
	if claimCorrect {
		return s.defend(claim)
	} else {
		return s.attack(claim)
	}
}

type StepData struct {
	LeafClaim          Claim
	IsAttack           bool
	PreStateTraceIndex uint64
}

// AttemptStep determines what step should occur for a given leaf claim.
// An error will be returned if the claim is not at the max depth.
func (s *Solver) AttemptStep(claim Claim) (StepData, error) {
	if claim.Depth() != s.gameDepth {
		return StepData{}, errors.New("cannot step on non-leaf claims")
	}
	claimCorrect, err := s.agreeWithClaim(claim.ClaimData)
	if err != nil {
		return StepData{}, err
	}
	index := claim.TraceIndex(s.gameDepth)
	// TODO(CLI-4198): Handle case where we dispute trace index 0
	if !claimCorrect {
		index -= 1
	}
	return StepData{
		LeafClaim:          claim,
		IsAttack:           !claimCorrect,
		PreStateTraceIndex: index,
	}, nil
}

// attack returns a response that attacks the claim.
func (s *Solver) attack(claim Claim) (*Claim, error) {
	position := claim.Attack()
	value, err := s.traceAtPosition(position)
	if err != nil {
		return nil, err
	}
	return &Claim{
		ClaimData:           ClaimData{Value: value, Position: position},
		Parent:              claim.ClaimData,
		ParentContractIndex: claim.ContractIndex,
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
		ClaimData:           ClaimData{Value: value, Position: position},
		Parent:              claim.ClaimData,
		ParentContractIndex: claim.ContractIndex,
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
