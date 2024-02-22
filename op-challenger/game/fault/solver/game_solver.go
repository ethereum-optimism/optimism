package solver

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type GameSolver struct {
	claimSolver *claimSolver
}

func NewGameSolver(gameDepth types.Depth, trace types.TraceAccessor) *GameSolver {
	return &GameSolver{
		claimSolver: newClaimSolver(gameDepth, trace),
	}
}

func (s *GameSolver) AgreeWithRootClaim(ctx context.Context, game types.Game) (bool, error) {
	return s.claimSolver.agreeWithClaim(ctx, game, game.Claims()[0])
}

func (s *GameSolver) CalculateNextActions(ctx context.Context, game types.Game) ([]types.Action, error) {
	agreeWithRootClaim, err := s.AgreeWithRootClaim(ctx, game)
	if err != nil {
		return nil, fmt.Errorf("failed to determine if root claim is correct: %w", err)
	}
	var errs []error
	var actions []types.Action
	agreedClaims := newAgreedClaimTracker()
	if agreeWithRootClaim {
		agreedClaims.MarkAgreed(game.Claims()[0])
	}
	for _, claim := range game.Claims() {
		var action *types.Action
		var err error
		if claim.Depth() == game.MaxDepth() {
			action, err = s.calculateStep(ctx, game, claim, agreedClaims)
		} else {
			action, err = s.calculateMove(ctx, game, claim, agreedClaims)
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if action == nil {
			continue
		}
		actions = append(actions, *action)
	}
	return actions, errors.Join(errs...)
}

func (s *GameSolver) calculateStep(ctx context.Context, game types.Game, claim types.Claim, agreedClaims *agreedClaimTracker) (*types.Action, error) {
	if claim.CounteredBy != (common.Address{}) {
		return nil, nil
	}
	step, err := s.claimSolver.AttemptStep(ctx, game, claim, agreedClaims)
	if err != nil {
		return nil, err
	}
	if step == nil {
		return nil, nil
	}
	return &types.Action{
		Type:           types.ActionTypeStep,
		ParentIdx:      step.LeafClaim.ContractIndex,
		ParentPosition: step.LeafClaim.Position,
		IsAttack:       step.IsAttack,
		PreState:       step.PreState,
		ProofData:      step.ProofData,
		OracleData:     step.OracleData,
	}, nil
}

func (s *GameSolver) calculateMove(ctx context.Context, game types.Game, claim types.Claim, agreedClaims *agreedClaimTracker) (*types.Action, error) {
	move, err := s.claimSolver.NextMove(ctx, claim, game, agreedClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate next move for claim index %v: %w", claim.ContractIndex, err)
	}
	if move == nil {
		return nil, nil
	}
	duplicate, isDupe := game.IsDuplicate(*move)
	if isDupe {
		fmt.Printf("Marking %v as agreed\n", duplicate.ContractIndex)
		agreedClaims.MarkAgreed(duplicate)
		return nil, nil
	}
	return &types.Action{
		Type:           types.ActionTypeMove,
		IsAttack:       !game.DefendsParent(*move),
		ParentIdx:      move.ParentContractIndex,
		ParentPosition: claim.Position,
		Value:          move.Value,
	}, nil
}
