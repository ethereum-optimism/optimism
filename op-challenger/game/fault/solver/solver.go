package solver

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

var (
	ErrStepNonLeafNode = errors.New("cannot step on non-leaf claims")
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

func (s *claimSolver) shouldCounter(game types.Game, claim types.Claim, honestClaims *honestClaimTracker) (bool, error) {
	// Do not counter honest claims
	if honestClaims.IsHonest(claim) {
		return false, nil
	}

	if claim.IsRoot() {
		// Always counter the root claim if it is not honest
		return true, nil
	}

	parent, err := game.GetParent(claim)
	if err != nil {
		return false, fmt.Errorf("no parent for claim %v: %w", claim.ContractIndex, err)
	}

	// Counter all claims that are countering an honest claim
	if honestClaims.IsHonest(parent) {
		return true, nil
	}

	counter, hasCounter := honestClaims.HonestCounter(parent)
	// Do not respond to any claim countering a claim the honest actor ignored
	if !hasCounter {
		return false, nil
	}

	// Do not counter sibling to an honest claim that are right of the honest claim.
	honestIdx := counter.TraceIndex(game.MaxDepth())
	claimIdx := claim.TraceIndex(game.MaxDepth())
	return claimIdx.Cmp(honestIdx) <= 0, nil
}

// NextMove returns the next move to make given the current state of the game.
func (s *claimSolver) NextMove(ctx context.Context, claim types.Claim, game types.Game, honestClaims *honestClaimTracker) (*types.Claim, error) {
	if claim.Depth() == s.gameDepth {
		return nil, types.ErrGameDepthReached
	}

	if counter, err := s.shouldCounter(game, claim, honestClaims); err != nil {
		return nil, fmt.Errorf("failed to determine if claim should be countered: %w", err)
	} else if !counter {
		return nil, nil
	}

	if agree, err := s.agreeWithClaim(ctx, game, claim); err != nil {
		return nil, err
	} else if agree {
		return s.defend(ctx, game, claim)
	} else {
		return s.attack(ctx, game, claim)
	}
}

type StepData struct {
	LeafClaim  types.Claim
	IsAttack   bool
	PreState   []byte
	ProofData  []byte
	OracleData *types.PreimageOracleData
}

// AttemptStep determines what step, if any, should occur for a given leaf claim.
// An error will be returned if the claim is not at the max depth.
// Returns nil, nil if no step should be performed.
func (s *claimSolver) AttemptStep(ctx context.Context, game types.Game, claim types.Claim, honestClaims *honestClaimTracker) (*StepData, error) {
	if claim.Depth() != s.gameDepth {
		return nil, ErrStepNonLeafNode
	}

	if counter, err := s.shouldCounter(game, claim, honestClaims); err != nil {
		return nil, fmt.Errorf("failed to determine if claim should be countered: %w", err)
	} else if !counter {
		return nil, nil
	}

	claimCorrect, err := s.agreeWithClaim(ctx, game, claim)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return &StepData{
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
