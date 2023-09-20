package solver

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

type GameSolver struct {
	claimSolver *claimSolver
}

func NewGameSolver(gameDepth int, trace types.TraceProvider) *GameSolver {
	return &GameSolver{
		claimSolver: newClaimSolver(gameDepth, trace),
	}
}

func (s *GameSolver) CalculateNextActions(ctx context.Context, game types.Game) ([]types.Action, error) {
	var errs []error
	var actions []types.Action
	for _, claim := range game.Claims() {
		var action *types.Action
		var err error
		if uint64(claim.Depth()) == game.MaxDepth() {
			action, err = s.calculateStep(ctx, game, claim)
		} else {
			action, err = s.calculateMove(ctx, game, claim)
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

func (s *GameSolver) calculateStep(ctx context.Context, game types.Game, claim types.Claim) (*types.Action, error) {
	if claim.Countered {
		return nil, nil
	}
	if game.AgreeWithClaimLevel(claim) {
		return nil, nil
	}
	step, err := s.claimSolver.AttemptStep(ctx, game, claim)
	if err == ErrStepIgnoreInvalidPath {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &types.Action{
		Type:       types.ActionTypeStep,
		ParentIdx:  step.LeafClaim.ContractIndex,
		IsAttack:   step.IsAttack,
		PreState:   step.PreState,
		ProofData:  step.ProofData,
		OracleData: step.OracleData,
	}, nil
}

func (s *GameSolver) calculateMove(ctx context.Context, game types.Game, claim types.Claim) (*types.Action, error) {
	if game.AgreeWithClaimLevel(claim) {
		return nil, nil
	}
	move, err := s.claimSolver.NextMove(ctx, claim, game)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate next move for claim index %v: %w", claim.ContractIndex, err)
	}
	if move == nil || game.IsDuplicate(*move) {
		return nil, nil
	}
	return &types.Action{
		Type:      types.ActionTypeMove,
		IsAttack:  !move.DefendsParent(),
		ParentIdx: move.ParentContractIndex,
		Value:     move.Value,
	}, nil
}
