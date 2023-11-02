package solver

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type SplitTraceLogic struct {
	topDepth      uint64
	upper         types.TraceProvider
	bottomFactory func(ctx context.Context, pre common.Hash, post common.Hash) (types.TraceProvider, error)
}

func (l *SplitTraceLogic) providerForPosition(ctx context.Context, game types.Game, ref types.Claim, pos types.Position) (types.TraceProvider, error) {
	if uint64(pos.Depth()) <= l.topDepth {
		return l.upper, nil
	}
	// TODO: Walk back up from claim, pulling ancestors from game, until we find the pre and post claim for the top level
	var pre common.Hash
	var post common.Hash
	// TODO: Cache the bottom providers
	bottom, err := l.bottomFactory(ctx, pre, post)
	if err != nil {
		return nil, fmt.Errorf("create provider for pre %v and post %v: %w", pre, post, err)
	}
	return trace.Translate(bottom, l.topDepth), nil
}

// AgreeWithClaim returns true if the claim is correct according to the internal [TraceProvider].
func (l *SplitTraceLogic) AgreeWithClaim(ctx context.Context, game types.Game, claim types.Claim) (bool, error) {
	trace, err := l.providerForPosition(ctx, game, claim, claim.Position)
	if err != nil {
		return false, err
	}
	ourValue, err := trace.Get(ctx, claim.Position)
	return bytes.Equal(ourValue[:], claim.Value[:]), err
}

// AttackClaim returns the claim to respond with to attack the specified claim.
func (l *SplitTraceLogic) AttackClaim(ctx context.Context, game types.Game, claim types.Claim) (*types.Claim, error) {
	position := claim.Attack()
	trace, err := l.providerForPosition(ctx, game, claim, position)
	if err != nil {
		return nil, err
	}
	value, err := trace.Get(ctx, position)
	if err != nil {
		return nil, fmt.Errorf("attack claim: %w", err)
	}
	return &types.Claim{
		ClaimData:           types.ClaimData{Value: value, Position: position},
		ParentContractIndex: claim.ContractIndex,
	}, nil
}

// DefendClaim returns the claim to respond with to defend the specified claim.
func (l *SplitTraceLogic) DefendClaim(ctx context.Context, game types.Game, claim types.Claim) (*types.Claim, error) {
	position := claim.Defend()
	trace, err := l.providerForPosition(ctx, game, claim, position)
	if err != nil {
		return nil, err
	}
	value, err := trace.Get(ctx, position)
	if err != nil {
		return nil, fmt.Errorf("defend claim: %w", err)
	}
	return &types.Claim{
		ClaimData:           types.ClaimData{Value: value, Position: position},
		ParentContractIndex: claim.ContractIndex,
	}, nil
}

func (l *SplitTraceLogic) StepAttack(ctx context.Context, game types.Game, claim types.Claim) (StepData, error) {
	return l.stepData(ctx, game, claim, true)
}

func (l *SplitTraceLogic) StepDefend(ctx context.Context, game types.Game, claim types.Claim) (StepData, error) {
	return l.stepData(ctx, game, claim, false)
}

func (l *SplitTraceLogic) stepData(ctx context.Context, game types.Game, claim types.Claim, attack bool) (StepData, error) {
	var position types.Position
	if attack {
		// Attack the claim by executing step index, so we need to get the pre-state of that index
		position = claim.Position
	} else {
		// Defend and use this claim as the starting point to execute the step after.
		// Thus, we need the pre-state of the next step.
		position = claim.Position.MoveRight()
	}
	trace, err := l.providerForPosition(ctx, game, claim, position)
	if err != nil {
		return StepData{}, err
	}
	preState, proofData, oracleData, err := trace.GetStepData(ctx, position)
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
