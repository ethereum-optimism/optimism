package solver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var challengerAddr = common.Address(bytes.Repeat([]byte{0xaa}, 20))

type actionRule func(game types.Game, action types.Action, correctTrace types.TraceProvider) error

var rules = []actionRule{
	parentMustExist,
	onlyStepAtMaxDepth,
	onlyMoveBeforeMaxDepth,
	doNotDuplicateExistingMoves,
	doNotStepAlreadyCounteredClaims,
	doNotDefendRootClaim,
	avoidPoisonedPrestate,
	detectPoisonedStepPrestate,
	detectFailedStep,
	doNotCounterSelf,
}

func printClaim(claim types.Claim, game types.Game) string {
	return fmt.Sprintf("Claim %v: Pos: %v TraceIdx: %v Depth: %v IndexAtDepth: %v ParentIdx: %v Value: %v Claimant: %v CounteredBy: %v",
		claim.ContractIndex, claim.Position.ToGIndex(), claim.Position.TraceIndex(game.MaxDepth()), claim.Position.Depth(), claim.Position.IndexAtDepth(), claim.ParentContractIndex, claim.Value, claim.Claimant, claim.CounteredBy)
}

func checkRules(game types.Game, action types.Action, correctTrace types.TraceProvider) error {
	var errs []error
	for _, rule := range rules {
		errs = append(errs, rule(game, action, correctTrace))
	}
	return errors.Join(errs...)
}

// parentMustExist checks that every action performed has a valid parent claim
// Rationale: The action would be rejected by the contracts
func parentMustExist(game types.Game, action types.Action, _ types.TraceProvider) error {
	if len(game.Claims()) <= action.ParentClaim.ContractIndex || action.ParentClaim.ContractIndex < 0 {
		return fmt.Errorf("parent claim %v does not exist in game with %v claims", action.ParentClaim.ContractIndex, len(game.Claims()))
	}
	return nil
}

// onlyStepAtMaxDepth verifies that step actions are only performed against leaf claims
// Rationale: The action would be rejected by the contracts
func onlyStepAtMaxDepth(game types.Game, action types.Action, _ types.TraceProvider) error {
	if action.Type == types.ActionTypeStep {
		return nil
	}
	parentDepth := game.Claims()[action.ParentClaim.ContractIndex].Position.Depth()
	if parentDepth >= game.MaxDepth() {
		return fmt.Errorf("parent at max depth (%v) but attempting to perform %v action instead of step",
			parentDepth, action.Type)
	}
	return nil
}

// onlyMoveBeforeMaxDepth verifies that move actions are not performed against leaf claims
// Rationale: The action would be rejected by the contracts
func onlyMoveBeforeMaxDepth(game types.Game, action types.Action, _ types.TraceProvider) error {
	if action.Type == types.ActionTypeMove {
		return nil
	}
	parentDepth := game.Claims()[action.ParentClaim.ContractIndex].Position.Depth()
	if parentDepth < game.MaxDepth() {
		return fmt.Errorf("parent (%v) not at max depth (%v) but attempting to perform %v action instead of move",
			parentDepth, game.MaxDepth(), action.Type)
	}
	return nil
}

// doNotDuplicateExistingMoves verifies that the challenger doesn't attempt to post a duplicate claim
// Rationale: The action would be rejected by the contracts
func doNotDuplicateExistingMoves(game types.Game, action types.Action, _ types.TraceProvider) error {
	newClaimData := types.ClaimData{
		Value:    action.Value,
		Position: resultingPosition(game, action),
	}
	if game.IsDuplicate(types.Claim{ClaimData: newClaimData, ParentContractIndex: action.ParentClaim.ContractIndex}) {
		return fmt.Errorf("creating duplicate claim at %v with value %v", newClaimData.Position.ToGIndex(), newClaimData.Value)
	}
	return nil
}

// doNotStepAlreadyCounteredClaims checks the challenger does not attempt to call step on already countered claims
// Rationale: The step call is redundant and a waste of gas
func doNotStepAlreadyCounteredClaims(game types.Game, action types.Action, _ types.TraceProvider) error {
	claim := game.Claims()[action.ParentClaim.ContractIndex]
	if claim.CounteredBy != (common.Address{}) {
		return fmt.Errorf("attempting to step already countered claim: %v", claim.ContractIndex)
	}
	return nil
}

// doNotDefendRootClaim checks the challenger doesn't attempt to defend the root claim
// Rationale: The action would be rejected by the contracts
func doNotDefendRootClaim(game types.Game, action types.Action, _ types.TraceProvider) error {
	if game.Claims()[action.ParentClaim.ContractIndex].IsRootPosition() && !action.IsAttack {
		return fmt.Errorf("defending the root claim at idx %v", action.ParentClaim.ContractIndex)
	}
	return nil
}

// doNotCounterSelf checks the challenger doesn't counter its own claims
// Rationale: The challenger should not disagree with itself
func doNotCounterSelf(game types.Game, action types.Action, _ types.TraceProvider) error {
	claim := game.Claims()[action.ParentClaim.ContractIndex]
	if claim.Claimant == challengerAddr {
		return fmt.Errorf("countering own claim at idx %v", action.ParentClaim.ContractIndex)
	}
	return nil
}

// avoidPoisonedPrestate checks the challenger does not perform a move that results in a claim where the ancestor
// with the largest trace index less than the new claim's trace index is invalid.
// Rationale: If such a claim were posted, an attacker could attack with invalid values down to max depth and setup a
// step call which uses the invalid claim as the pre-state. The challenger could not call step because it does not have
// the preimage of the invalid state. If the attacker should call step, they could provide a carefully crafted state
// that allows it to successfully step against the challenger's claim.
func avoidPoisonedPrestate(game types.Game, action types.Action, correctTrace types.TraceProvider) error {
	if action.Type == types.ActionTypeStep {
		return nil
	}
	ancestors := ""
	movePosition := resultingPosition(game, action)
	honestTraceIndex := movePosition.TraceIndex(game.MaxDepth())
	// Walk back up the claims and find the claim with highest trace index < honestTraceIndex
	claim := game.Claims()[action.ParentClaim.ContractIndex]
	var preStateClaim types.Claim
	for {
		ancestors += printClaim(claim, game) + "\n"
		claimTraceIdx := claim.TraceIndex(game.MaxDepth())
		if claimTraceIdx.Cmp(honestTraceIndex) < 0 { // Check it's left of the honest claim
			if preStateClaim == (types.Claim{}) || claimTraceIdx.Cmp(preStateClaim.TraceIndex(game.MaxDepth())) > 0 {
				preStateClaim = claim
			}
		}
		if claim.IsRoot() {
			break
		}
		parent, err := game.GetParent(claim)
		if err != nil {
			return fmt.Errorf("no parent of claim %v: %w", claim.ContractIndex, err)
		}
		claim = parent
	}
	if preStateClaim == (types.Claim{}) {
		// No claim to the left of the honest claim, so can't have been poisoned
		return nil
	}
	correctValue, err := correctTrace.Get(context.Background(), preStateClaim.Position)
	if err != nil {
		return fmt.Errorf("failed to get correct trace at position %v: %w", preStateClaim.Position, err)
	}
	if correctValue != preStateClaim.Value {
		err = fmt.Errorf("prestate poisoned claim %v has invalid prestate and is left of honest claim countering %v at trace index %v", preStateClaim.ContractIndex, action.ParentClaim.ContractIndex, honestTraceIndex)
		return err
	}
	return nil
}

// detectFailedStep checks that step actions will succeed.
// Rationale: The action would be rejected by the contracts
//
// INVARIANT: If a step is an attack, the poststate is valid if the step produces
//
//	the same poststate hash as the parent claim's value.
//	If a step is a defense:
//	  1. If the parent claim and the found post state agree with each other
//	     (depth diff % 2 == 0), the step is valid if it produces the same
//	     state hash as the post state's claim.
//	  2. If the parent claim and the found post state disagree with each other
//	     (depth diff % 2 != 0), the parent cannot be countered unless the step
//	     produces the same state hash as `postState.claim`.
func detectFailedStep(game types.Game, action types.Action, correctTrace types.TraceProvider) error {
	if action.Type != types.ActionTypeStep {
		// An invalid post state is not an issue if we are moving, only if the honest challenger has to call step.
		return nil
	}
	position := resultingPosition(game, action)
	if position.Depth() != game.MaxDepth() {
		// Not at max depth yet
		return nil
	}
	honestTraceIndex := position.TraceIndex(game.MaxDepth())
	poststateIndex := honestTraceIndex
	if !action.IsAttack {
		poststateIndex = new(big.Int).Add(honestTraceIndex, big.NewInt(1))
	}
	// Walk back up the claims and find the claim required post state index
	claim := game.Claims()[action.ParentClaim.ContractIndex]
	poststateClaim, ok := game.AncestorWithTraceIndex(claim, poststateIndex)
	if !ok {
		return fmt.Errorf("did not find required poststate at %v to counter claim %v", poststateIndex, action.ParentClaim.ContractIndex)
	}
	correctValue, err := correctTrace.Get(context.Background(), poststateClaim.Position)
	if err != nil {
		return fmt.Errorf("failed to get correct trace at position %v: %w", poststateClaim.Position, err)
	}
	validStep := correctValue == poststateClaim.Value
	parentPostAgree := (claim.Depth()-poststateClaim.Depth())%2 == 0
	if parentPostAgree == validStep {
		return fmt.Errorf("failed step against claim at %v using poststate from claim %v post state is correct? %v parentPostAgree? %v",
			action.ParentClaim.ContractIndex, poststateClaim.ContractIndex, validStep, parentPostAgree)
	}
	return nil
}

// detectPoisonedStepPrestate checks that:
// 1. step actions performed by the challenger always have a valid prestate
// 2. move actions that create a claim a max depth would have a valid prestate if they are attacked
// 3. the actual prestate provided matches the prestate claim's commitment
// Rationale: A step against an invalid prestate will fail because the preimage of the prestate claim is unknown
// and claims at max depth with an invalid prestate could be stepped against because the prestate is invalid so a VM
// step will not result in the correct post-state.
func detectPoisonedStepPrestate(game types.Game, action types.Action, correctTrace types.TraceProvider) error {
	position := resultingPosition(game, action)
	if position.Depth() != game.MaxDepth() {
		// Not at max depth yet
		return nil
	}
	honestTraceIndex := position.TraceIndex(game.MaxDepth())
	prestateIndex := honestTraceIndex
	// If we're performing a move to post a leaf claim, assume the attacker will try to attack it from their
	// poisoned prestate
	if action.IsAttack || action.Type == types.ActionTypeMove {
		prestateIndex = new(big.Int).Sub(prestateIndex, big.NewInt(1))
	}
	if prestateIndex.Cmp(big.NewInt(0)) < 0 {
		// Absolute prestate is not poisoned
		return nil
	}
	// Walk back up the claims and find the claim with highest trace index < honestTraceIndex
	claim := game.Claims()[action.ParentClaim.ContractIndex]
	preStateClaim, ok := game.AncestorWithTraceIndex(claim, prestateIndex)
	if !ok {
		return fmt.Errorf("performing step against claim %v with no prestate available at %v", claim.ContractIndex, prestateIndex)
	}
	correctValue, err := correctTrace.Get(context.Background(), preStateClaim.Position)
	if err != nil {
		return fmt.Errorf("failed to get correct trace at position %v: %w", preStateClaim.Position, err)
	}
	if correctValue != preStateClaim.Value {
		if action.Type == types.ActionTypeStep {
			return fmt.Errorf("stepping from poisoned prestate at claim %v when countering %v", preStateClaim.ContractIndex, action.ParentClaim.ContractIndex)
		} else {
			return fmt.Errorf("posting leaf claim with poisoned prestate from claim %v when countering %v", preStateClaim.ContractIndex, action.ParentClaim.ContractIndex)
		}
	}
	if action.Type == types.ActionTypeStep {
		prestateHash := crypto.Keccak256Hash(action.PreState)
		if !slices.Equal(prestateHash[1:], preStateClaim.Value[1:]) {
			return fmt.Errorf("prestate hash %v does not match expected prestate claim %v from claim %v", prestateHash, preStateClaim.Value, preStateClaim.ContractIndex)
		}
	}
	return nil
}

func resultingPosition(game types.Game, action types.Action) types.Position {
	parentPos := game.Claims()[action.ParentClaim.ContractIndex].Position
	if action.Type == types.ActionTypeStep {
		return parentPos
	}
	if action.IsAttack {
		return parentPos.Attack()
	}
	return parentPos.Defend()
}
