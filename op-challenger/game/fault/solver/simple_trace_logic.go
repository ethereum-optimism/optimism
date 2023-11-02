package solver

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

type SimpleTraceLogic struct {
	trace types.TraceProvider
}

func NewSimpleTraceLogic(trace types.TraceProvider) *SimpleTraceLogic {
	return &SimpleTraceLogic{
		trace: trace,
	}
}

// AgreeWithClaim returns true if the claim is correct according to the internal [TraceProvider].
func (l *SimpleTraceLogic) AgreeWithClaim(ctx context.Context, _ types.Game, claim types.Claim) (bool, error) {
	ourValue, err := l.trace.Get(ctx, claim.Position)
	return bytes.Equal(ourValue[:], claim.Value[:]), err
}

// AttackClaim returns the claim to respond with to attack the specified claim.
func (l *SimpleTraceLogic) AttackClaim(ctx context.Context, _ types.Game, claim types.Claim) (*types.Claim, error) {
	position := claim.Attack()
	value, err := l.trace.Get(ctx, position)
	if err != nil {
		return nil, fmt.Errorf("attack claim: %w", err)
	}
	return &types.Claim{
		ClaimData:           types.ClaimData{Value: value, Position: position},
		ParentContractIndex: claim.ContractIndex,
	}, nil
}

// DefendClaim returns the claim to respond with to defend the specified claim.
func (l *SimpleTraceLogic) DefendClaim(ctx context.Context, _ types.Game, claim types.Claim) (*types.Claim, error) {
	position := claim.Defend()
	value, err := l.trace.Get(ctx, position)
	if err != nil {
		return nil, fmt.Errorf("defend claim: %w", err)
	}
	return &types.Claim{
		ClaimData:           types.ClaimData{Value: value, Position: position},
		ParentContractIndex: claim.ContractIndex,
	}, nil
}

func (l *SimpleTraceLogic) StepAttack(ctx context.Context, game types.Game, claim types.Claim) (StepData, error) {
	return l.stepData(ctx, game, claim, true)
}

func (l *SimpleTraceLogic) StepDefend(ctx context.Context, game types.Game, claim types.Claim) (StepData, error) {
	return l.stepData(ctx, game, claim, false)
}

func (l *SimpleTraceLogic) stepData(ctx context.Context, _ types.Game, claim types.Claim, attack bool) (StepData, error) {
	var position types.Position
	if attack {
		// Attack the claim by executing step index, so we need to get the pre-state of that index
		position = claim.Position
	} else {
		// Defend and use this claim as the starting point to execute the step after.
		// Thus, we need the pre-state of the next step.
		position = claim.Position.MoveRight()
	}
	preState, proofData, oracleData, err := l.trace.GetStepData(ctx, position)
	if err != nil {
		return StepData{}, err
	}

	return StepData{
		LeafClaim:  claim,
		IsAttack:   attack,
		PreState:   preState,
		ProofData:  proofData,
		OracleData: oracleData,
	}, nil
}
