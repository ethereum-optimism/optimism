package solver

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

var (
	ErrStepNonLeafNode       = errors.New("cannot step on non-leaf claims")
	ErrStepIgnoreInvalidPath = errors.New("cannot step on claims that dispute invalid paths")
)

type moveType uint8

const (
	moveAttack moveType = iota
	moveDefend
	moveNop
)

// claimSolver uses a [TraceProvider] to determine the moves to make in a dispute game.
type claimSolver struct {
	trace     types.TraceAccessor
	gameDepth types.Depth
}

// newClaimSolver creates a new [claimSolver] using the provided [TraceProvider].
func newClaimSolver(gameDepth types.Depth, trace types.TraceAccessor) *claimSolver {
	return &claimSolver{
		trace,
		gameDepth,
	}
}

func (s *claimSolver) respondClaim(ctx context.Context, claim types.Claim, game types.Game) (moveType, error) {
	agree, err := s.agreeWithClaim(ctx, game, claim)
	if err != nil {
		return moveNop, err
	}
	if agree {
		return moveDefend, nil
	} else {
		return moveAttack, nil
	}
}

func (s *claimSolver) respond(ctx context.Context, claim types.Claim, game types.Game, agreeWithRootClaim bool) (moveType, error) {
	// Root case is simple - attack if we disagree, do nothing if we agree
	if claim.IsRoot() {
		if !game.AgreeWithClaimLevel(claim, agreeWithRootClaim) {
			return moveAttack, nil
		} else {
			return moveNop, nil
		}
	}

	parent, err := game.GetParent(claim)
	if err != nil {
		return moveNop, err
	}
	if !game.AgreeWithClaimLevel(claim, agreeWithRootClaim) {
		agreeWithParent, err := s.agreeWithClaimPath(ctx, game, parent)
		if err != nil {
			return moveNop, err
		}
		if agreeWithParent {
			return s.respondClaim(ctx, claim, game)
		} else {
			return moveNop, nil
		}
	} else {
		correctResponse, err := s.respond(ctx, parent, game, agreeWithRootClaim)
		if err != nil {
			return moveNop, err
		}
		claimResponse := moveDefend
		if !game.DefendsParent(claim) {
			claimResponse = moveAttack
		}
		invalidDefense := claimResponse == moveDefend && correctResponse == moveAttack

		// Note this check for a claim that matches what we'd do is not in the spec
		claimIsCorrectMove := correctResponse == claimResponse
		if claimIsCorrectMove {
			agreeWithClaim, err := s.agreeWithClaim(ctx, game, claim)
			if err != nil {
				return moveNop, err
			}
			if agreeWithClaim {
				// Don't counter moves that we would have made.
				return moveNop, nil
			}
		}

		// Resume bits that are in the spec
		if !invalidDefense {
			return s.respondClaim(ctx, claim, game)
		} else {
			return moveNop, nil
		}
	}
}

func (s *claimSolver) respondToClaim(ctx context.Context, claim types.Claim, game types.Game) (*types.Claim, error) {
	agree, err := s.agreeWithClaim(ctx, game, claim)
	if err != nil {
		return nil, err
	}
	if agree {
		return s.defend(ctx, game, claim)
	} else {
		return s.attack(ctx, game, claim)
	}
}

// NextMove returns the next move to make given the current state of the game.
func (s *claimSolver) NextMove(ctx context.Context, claim types.Claim, game types.Game, agreeWithRootClaim bool) (*types.Claim, error) {
	if claim.Depth() == s.gameDepth {
		return nil, types.ErrGameDepthReached
	}
	responseType, err := s.respond(ctx, claim, game, agreeWithRootClaim)
	if err != nil {
		return nil, fmt.Errorf("failed to determine correct move type: %w", err)
	}
	switch responseType {
	case moveNop:
		return nil, nil
	case moveAttack:
		return s.attack(ctx, game, claim)
	case moveDefend:
		return s.defend(ctx, game, claim)
	default:
		panic(fmt.Errorf("unknown move type: %v", responseType))
	}
}

type StepData struct {
	LeafClaim  types.Claim
	IsAttack   bool
	PreState   []byte
	ProofData  []byte
	OracleData *types.PreimageOracleData
}

// AttemptStep determines what step should occur for a given leaf claim.
// An error will be returned if the claim is not at the max depth.
// Returns ErrStepIgnoreInvalidPath if the claim disputes an invalid path
func (s *claimSolver) AttemptStep(ctx context.Context, game types.Game, claim types.Claim) (StepData, error) {
	if claim.Depth() != s.gameDepth {
		return StepData{}, ErrStepNonLeafNode
	}

	// Step only on claims that dispute a valid path
	parent, err := game.GetParent(claim)
	if err != nil {
		return StepData{}, err
	}
	parentValid, err := s.agreeWithClaimPath(ctx, game, parent)
	if err != nil {
		return StepData{}, err
	}
	if !parentValid {
		return StepData{}, ErrStepIgnoreInvalidPath
	}

	claimCorrect, err := s.agreeWithClaim(ctx, game, claim)
	if err != nil {
		return StepData{}, err
	}

	var position types.Position
	if !claimCorrect {
		// Attack the claim by executing step index, so we need to get the pre-state of that index
		position = claim.Position
	} else {
		// Defend and use this claim as the starting point to execute the step after.
		// Thus, we need the pre-state of the next step.
		position = claim.Position.MoveRight()
	}

	preState, proofData, oracleData, err := s.trace.GetStepData(ctx, game, claim, position)
	if err != nil {
		return StepData{}, err
	}

	return StepData{
		LeafClaim:  claim,
		IsAttack:   !claimCorrect,
		PreState:   preState,
		ProofData:  proofData,
		OracleData: oracleData,
	}, nil
}

// attack returns a response that attacks the claim.
func (s *claimSolver) attack(ctx context.Context, game types.Game, claim types.Claim) (*types.Claim, error) {
	position := claim.Attack()
	value, err := s.trace.Get(ctx, game, claim, position)
	if err != nil {
		return nil, fmt.Errorf("attack claim: %w", err)
	}
	return &types.Claim{
		ClaimData:           types.ClaimData{Value: value, Position: position},
		ParentContractIndex: claim.ContractIndex,
	}, nil
}

// defend returns a response that defends the claim.
func (s *claimSolver) defend(ctx context.Context, game types.Game, claim types.Claim) (*types.Claim, error) {
	if claim.IsRoot() {
		return nil, nil
	}
	position := claim.Defend()
	value, err := s.trace.Get(ctx, game, claim, position)
	if err != nil {
		return nil, fmt.Errorf("defend claim: %w", err)
	}
	return &types.Claim{
		ClaimData:           types.ClaimData{Value: value, Position: position},
		ParentContractIndex: claim.ContractIndex,
	}, nil
}

// agreeWithClaim returns true if the claim is correct according to the internal [TraceProvider].
func (s *claimSolver) agreeWithClaim(ctx context.Context, game types.Game, claim types.Claim) (bool, error) {
	ourValue, err := s.trace.Get(ctx, game, claim, claim.Position)
	return bytes.Equal(ourValue[:], claim.Value[:]), err
}

// agreeWithClaimPath returns true if the every other claim in the path to root is correct according to the internal [TraceProvider].
func (s *claimSolver) agreeWithClaimPath(ctx context.Context, game types.Game, claim types.Claim) (bool, error) {
	agree, err := s.agreeWithClaim(ctx, game, claim)
	if err != nil {
		return false, err
	}
	if !agree {
		return false, nil
	}
	if claim.IsRoot() {
		return true, nil
	}
	parent, err := game.GetParent(claim)
	if err != nil {
		return false, fmt.Errorf("failed to get parent of claim %v: %w", claim.ContractIndex, err)
	}
	if parent.IsRoot() {
		return true, nil
	}
	grandParent, err := game.GetParent(parent)
	if err != nil {
		return false, err
	}
	return s.agreeWithClaimPath(ctx, game, grandParent)
}
