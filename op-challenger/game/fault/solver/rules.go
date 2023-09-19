package solver

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

type actionRule func(game types.Game, action types.Action) error

var rules = []actionRule{
	parentMustExist,
	onlyStepAtMaxDepth,
	onlyMoveBeforeMaxDepth,
	doNotDuplicateExistingMoves,
	doNotDefendRootClaim,
}

func checkRules(game types.Game, action types.Action) error {
	var errs []error
	for _, rule := range rules {
		errs = append(errs, rule(game, action))
	}
	return errors.Join(errs...)
}

func parentMustExist(game types.Game, action types.Action) error {
	if len(game.Claims()) <= action.ParentIdx || action.ParentIdx < 0 {
		return fmt.Errorf("parent claim %v does not exist in game with %v claims", action.ParentIdx, len(game.Claims()))
	}
	return nil
}

func onlyStepAtMaxDepth(game types.Game, action types.Action) error {
	if action.Type == types.ActionTypeStep {
		return nil
	}
	parentDepth := uint64(game.Claims()[action.ParentIdx].Position.Depth())
	if parentDepth >= game.MaxDepth() {
		return fmt.Errorf("parent at max depth (%v) but attempting to perform %v action instead of step",
			parentDepth, action.Type)
	}
	return nil
}

func onlyMoveBeforeMaxDepth(game types.Game, action types.Action) error {
	if action.Type == types.ActionTypeMove {
		return nil
	}
	parentDepth := uint64(game.Claims()[action.ParentIdx].Position.Depth())
	if parentDepth < game.MaxDepth() {
		return fmt.Errorf("parent (%v) not at max depth (%v) but attempting to perform %v action instead of move",
			parentDepth, game.MaxDepth(), action.Type)
	}
	return nil
}

func doNotDuplicateExistingMoves(game types.Game, action types.Action) error {
	newClaimData := types.ClaimData{
		Value:    action.Value,
		Position: resultingPosition(game, action),
	}
	if game.IsDuplicate(types.Claim{ClaimData: newClaimData, ParentContractIndex: action.ParentIdx}) {
		return fmt.Errorf("creating duplicate claim at %v with value %v", newClaimData.Position.ToGIndex(), newClaimData.Value)
	}
	return nil
}

func doNotDefendRootClaim(game types.Game, action types.Action) error {
	if game.Claims()[action.ParentIdx].IsRootPosition() && !action.IsAttack {
		return fmt.Errorf("defending the root claim at idx %v", action.ParentIdx)
	}
	return nil
}

func resultingPosition(game types.Game, action types.Action) types.Position {
	parentPos := game.Claims()[action.ParentIdx].Position
	if action.Type == types.ActionTypeStep {
		return parentPos
	}
	if action.IsAttack {
		return parentPos.Attack()
	}
	return parentPos.Defend()
}
