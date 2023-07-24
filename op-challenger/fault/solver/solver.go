package solver

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
)

var (
	ErrStepNonLeafNode = errors.New("cannot step on non-leaf claims")
	ErrStepAgreedClaim = errors.New("cannot step on claims we agree with")
)

// Solver uses a [ProviderWrapper] to determine the moves to make in a dispute game.
type Solver struct {
	provider  ProviderWrapper
	gameDepth int
}

// NewSolver creates a new [Solver] using the provided [TraceProvider].
func NewSolver(gameDepth int, traceProvider types.TraceProvider) *Solver {
	return &Solver{
		provider:  NewProviderWrapper(traceProvider),
		gameDepth: gameDepth,
	}
}

// NextMove returns the next move to make given the current state of the game.
func (s *Solver) NextMove(claim types.Claim, agreeWithClaimLevel bool) (*types.Claim, error) {
	if agreeWithClaimLevel {
		return nil, nil
	}
	if claim.Depth() == s.gameDepth {
		return nil, types.ErrGameDepthReached
	}
	agree, err := s.provider.AgreeWithClaim(claim.ClaimData, s.gameDepth)
	if err != nil {
		return nil, err
	}
	if agree {
		return s.provider.Defend(claim, s.gameDepth)
	} else {
		return s.provider.Attack(claim, s.gameDepth)
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
	claimCorrect, err := s.provider.AgreeWithClaim(claim.ClaimData, s.gameDepth)
	if err != nil {
		return StepData{}, err
	}
	index := claim.TraceIndex(s.gameDepth)
	var preState []byte
	var proofData []byte
	// If we are attacking index 0, we provide the absolute pre-state, not an intermediate state
	if index == 0 && !claimCorrect {
		preState = s.provider.AbsolutePreState()
	} else {
		// If attacking, get the state just before, other get the state after
		if !claimCorrect {
			index = index - 1
		}
		preState, proofData, err = s.provider.GetPreimage(index)
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
