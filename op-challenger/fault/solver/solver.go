package solver

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrStepNonLeafNode = errors.New("cannot step on non-leaf claims")
	ErrStepAgreedClaim = errors.New("cannot step on claims we agree with")
)

// Solver uses a [TraceProvider] to determine the moves to make in a dispute game.
type Solver struct {
	trace     types.TraceProvider
	gameDepth int
}

// NewSolver creates a new [Solver] using the provided [TraceProvider].
func NewSolver(gameDepth int, traceProvider types.TraceProvider) *Solver {
	return &Solver{
		traceProvider,
		gameDepth,
	}
}

// NextMove returns the next move to make given the current state of the game.
func (s *Solver) NextMove(claim types.Claim, agreeWithClaimLevel bool) (*types.Claim, error) {
	if agreeWithClaimLevel {
		return nil, nil
	}

	// Special case of the root claim
	if claim.IsRoot() {
		return s.handleRoot(claim)
	}
	return s.handleMiddle(claim)
}

func (s *Solver) handleRoot(claim types.Claim) (*types.Claim, error) {
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

func (s *Solver) handleMiddle(claim types.Claim) (*types.Claim, error) {
	claimCorrect, err := s.agreeWithClaim(claim.ClaimData)
	if err != nil {
		return nil, err
	}
	if claim.Depth() == s.gameDepth {
		return nil, types.ErrGameDepthReached
	}
	if claimCorrect {
		return s.defend(claim)
	} else {
		return s.attack(claim)
	}
}

type StepData struct {
	LeafClaim types.Claim
	IsAttack  bool
	PreState  []byte
	ProofData []byte
}

// AttemptStep determines what step should occur for a given leaf claim.
// An error will be returned if the claim is not at the max depth.
func (s *Solver) AttemptStep(claim types.Claim, agreeWithClaimLevel bool) (StepData, error) {
	if claim.Depth() != s.gameDepth {
		return StepData{}, ErrStepNonLeafNode
	}
	if agreeWithClaimLevel {
		return StepData{}, ErrStepAgreedClaim
	}
	claimCorrect, err := s.agreeWithClaim(claim.ClaimData)
	if err != nil {
		return StepData{}, err
	}
	index := claim.TraceIndex(s.gameDepth)
	var preState []byte
	var proofData []byte
	// If we are attacking index 0, we provide the absolute pre-state, not an intermediate state
	if index == 0 && !claimCorrect {
		preState = s.trace.AbsolutePreState()
	} else {
		// If attacking, get the state just before, other get the state after
		if !claimCorrect {
			index = index - 1
		}
		preState, proofData, err = s.trace.GetPreimage(index)
		if err != nil {
			return StepData{}, err
		}
	}

	return StepData{
		LeafClaim: claim,
		IsAttack:  !claimCorrect,
		PreState:  preState,
		ProofData: proofData,
	}, nil
}

// attack returns a response that attacks the claim.
func (s *Solver) attack(claim types.Claim) (*types.Claim, error) {
	position := claim.Attack()
	value, err := s.traceAtPosition(position)
	if err != nil {
		return nil, fmt.Errorf("attack claim: %w", err)
	}
	return &types.Claim{
		ClaimData:           types.ClaimData{Value: value, Position: position},
		Parent:              claim.ClaimData,
		ParentContractIndex: claim.ContractIndex,
	}, nil
}

// defend returns a response that defends the claim.
func (s *Solver) defend(claim types.Claim) (*types.Claim, error) {
	position := claim.Defend()
	value, err := s.traceAtPosition(position)
	if err != nil {
		return nil, fmt.Errorf("defend claim: %w", err)
	}
	return &types.Claim{
		ClaimData:           types.ClaimData{Value: value, Position: position},
		Parent:              claim.ClaimData,
		ParentContractIndex: claim.ContractIndex,
	}, nil
}

// agreeWithClaim returns true if the claim is correct according to the internal [TraceProvider].
func (s *Solver) agreeWithClaim(claim types.ClaimData) (bool, error) {
	ourValue, err := s.traceAtPosition(claim.Position)
	return ourValue == claim.Value, err
}

// traceAtPosition returns the [common.Hash] from internal [TraceProvider] at the given [Position].
func (s *Solver) traceAtPosition(p types.Position) (common.Hash, error) {
	index := p.TraceIndex(s.gameDepth)
	hash, err := s.trace.Get(index)
	return hash, err
}
