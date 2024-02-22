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

func isChallengingClaim(claim types.Claim, game types.Game, agreedClaims *agreedClaimTracker) (types.Claim, bool, error) {
	levelsToAgreedClaim := 0
	var err error
	for {
		if agreedClaims.IsAgreed(claim) {
			break
		}
		levelsToAgreedClaim++
		if claim.IsRoot() {
			break
		}
		claim, err = game.GetParent(claim)
		if err != nil {
			return types.Claim{}, false, err
		}
	}
	return claim, levelsToAgreedClaim%2 == 1, nil
}

func (s *claimSolver) respond(ctx context.Context, claim types.Claim, game types.Game, agreedClaims *agreedClaimTracker) (moveType, error) {
	if agreedClaims.IsAgreed(claim) {
		return moveNop, nil
	}
	// Root case is simple - attack since we have established it's not a claim we'd post
	if claim.IsRoot() {
		return moveAttack, nil
	}

	parent, err := game.GetParent(claim)
	if err != nil {
		return moveNop, err
	}
	subGameRoot, challenging, err := isChallengingClaim(claim, game, agreedClaims)
	if err != nil {
		return moveNop, nil
	}
	if challenging {
		agreeWithParent, err := s.agreeWithClaimPath(ctx, game, parent, subGameRoot)
		if err != nil {
			return moveNop, err
		}
		if agreeWithParent {
			return s.respondClaim(ctx, claim, game)
		} else {
			return moveNop, nil
		}
	} else {
		correctResponse, err := s.respond(ctx, parent, game, agreedClaims)
		if err != nil {
			return moveNop, err
		}
		claimResponse := moveDefend
		if !game.DefendsParent(claim) {
			claimResponse = moveAttack
		}
		invalidDefense := claimResponse == moveDefend && correctResponse == moveAttack
		if !invalidDefense {
			return s.respondClaim(ctx, claim, game)
		} else {
			return moveNop, nil
		}
	}
}

// NextMove returns the next move to make given the current state of the game.
func (s *claimSolver) NextMove(ctx context.Context, claim types.Claim, game types.Game, agreedClaims *agreedClaimTracker) (*types.Claim, error) {
	if claim.Depth() == s.gameDepth {
		return nil, types.ErrGameDepthReached
	}
	responseType, err := s.respond(ctx, claim, game, agreedClaims)
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
func (s *claimSolver) AttemptStep(ctx context.Context, game types.Game, claim types.Claim, agreedClaims *agreedClaimTracker) (*StepData, error) {
	if claim.Depth() != s.gameDepth {
		return nil, ErrStepNonLeafNode
	}

	responseType, err := s.respond(ctx, claim, game, agreedClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to determine correct step type: %w", err)
	}
	if responseType == moveNop {
		return nil, nil
	}

	var position types.Position
	if responseType == moveAttack {
		// Attack the claim by executing step index, so we need to get the pre-state of that index
		position = claim.Position
	} else {
		// Defend and use this claim as the starting point to execute the step after.
		// Thus, we need the pre-state of the next step.
		position = claim.Position.MoveRight()
	}

	preState, proofData, oracleData, err := s.trace.GetStepData(ctx, game, claim, position)
	if err != nil {
		return nil, err
	}

	return &StepData{
		LeafClaim:  claim,
		IsAttack:   responseType == moveAttack,
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
func (s *claimSolver) agreeWithClaimPath(ctx context.Context, game types.Game, claim types.Claim, subGameRoot types.Claim) (bool, error) {
	agree, err := s.agreeWithClaim(ctx, game, claim)
	if err != nil {
		return false, err
	}
	if !agree {
		return false, nil
	}
	if claim.IsRoot() || claim == subGameRoot {
		return true, nil
	}
	parent, err := game.GetParent(claim)
	if err != nil {
		return false, fmt.Errorf("failed to get parent of claim %v: %w", claim.ContractIndex, err)
	}
	if parent.IsRoot() || claim == subGameRoot {
		return true, nil
	}
	grandParent, err := game.GetParent(parent)
	if err != nil {
		return false, err
	}
	return s.agreeWithClaimPath(ctx, game, grandParent, subGameRoot)
}
