package solver

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type ActionType string

const (
	ActionTypeMove ActionType = "move"
	ActionTypeStep ActionType = "step"
)

func (a ActionType) String() string {
	return string(a)
}

type Action struct {
	Type      ActionType
	ParentIdx int
	IsAttack  bool

	// Moves
	Value common.Hash

	// Steps
	PreState   []byte
	ProofData  []byte
	OracleData *types.PreimageOracleData
}

type GameSolver struct {
	claimSolver *claimSolver
	gameDepth   int
}

func NewGameSolver(gameDepth int, trace types.TraceProvider) *GameSolver {
	return &GameSolver{
		claimSolver: newClaimSolver(gameDepth, trace),
		gameDepth:   gameDepth,
	}
}

func (s *GameSolver) CalculateNextActions(ctx context.Context, game types.Game) ([]Action, error) {
	var errs []error
	var actions []Action
	for _, claim := range game.Claims() {
		var action *Action
		var err error
		if claim.Depth() == s.gameDepth {
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

func (s *GameSolver) calculateStep(ctx context.Context, game types.Game, claim types.Claim) (*Action, error) {
	if claim.Countered {
		return nil, nil
	}
	if game.AgreeWithClaimLevel(claim) {
		return nil, nil
	}
	step, err := s.claimSolver.AttemptStep(ctx, claim, game.AgreeWithClaimLevel(claim))
	if err != nil {
		return nil, err
	}
	return &Action{
		Type:       ActionTypeStep,
		ParentIdx:  step.LeafClaim.ContractIndex,
		IsAttack:   step.IsAttack,
		PreState:   step.PreState,
		ProofData:  step.ProofData,
		OracleData: step.OracleData,
	}, nil
}

func (s *GameSolver) calculateMove(ctx context.Context, game types.Game, claim types.Claim) (*Action, error) {
	move, err := s.claimSolver.NextMove(ctx, claim, game.AgreeWithClaimLevel(claim))
	if err != nil {
		return nil, fmt.Errorf("failed to calculate next move for claim index %v: %w", claim.ContractIndex, err)
	}
	if move == nil || game.IsDuplicate(*move) {
		return nil, nil
	}
	return &Action{
		Type:      ActionTypeMove,
		IsAttack:  !move.DefendsParent(),
		ParentIdx: move.ParentContractIndex,
		Value:     move.Value,
	}, nil
}
