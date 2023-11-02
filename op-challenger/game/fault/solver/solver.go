package solver

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

var (
	ErrStepNonLeafNode       = errors.New("cannot step on non-leaf claims")
	ErrStepAgreedClaim       = errors.New("cannot step on claims we agree with")
	ErrStepIgnoreInvalidPath = errors.New("cannot step on claims that dispute invalid paths")
)

type TraceLogic interface {
	AgreeWithClaim(ctx context.Context, game types.Game, claim types.Claim) (bool, error)
	AttackClaim(ctx context.Context, game types.Game, claim types.Claim) (*types.Claim, error)
	DefendClaim(ctx context.Context, game types.Game, claim types.Claim) (*types.Claim, error)
	StepAttack(ctx context.Context, game types.Game, claim types.Claim) (StepData, error)
	StepDefend(ctx context.Context, game types.Game, claim types.Claim) (StepData, error)
}

// claimSolver uses a [TraceProvider] to determine the moves to make in a dispute game.
type claimSolver struct {
	trace     TraceLogic
	gameDepth int
}

// newClaimSolver creates a new [claimSolver] using the provided [TraceProvider].
func newClaimSolver(gameDepth int, traceProvider types.TraceProvider) *claimSolver {
	return &claimSolver{
		NewSimpleTraceLogic(traceProvider),
		gameDepth,
	}
}

// NextMove returns the next move to make given the current state of the game.
func (s *claimSolver) NextMove(ctx context.Context, claim types.Claim, game types.Game) (*types.Claim, error) {
	if claim.Depth() == s.gameDepth {
		return nil, types.ErrGameDepthReached
	}

	// Before challenging this claim, first check that the move wasn't warranted.
	// If the parent claim is on a dishonest path, then we would have moved against it anyways. So we don't move.
	// Avoiding dishonest paths ensures that there's always a valid claim available to support ours during step.
	if !claim.IsRoot() {
		parent, err := game.GetParent(claim)
		if err != nil {
			return nil, err
		}
		agreeWithParent, err := s.agreeWithClaimPath(ctx, game, parent)
		if err != nil {
			return nil, err
		}
		if !agreeWithParent {
			return nil, nil
		}
	}

	agree, err := s.trace.AgreeWithClaim(ctx, game, claim)
	if err != nil {
		return nil, err
	}
	if agree {
		if claim.IsRoot() {
			return nil, nil
		}
		return s.trace.DefendClaim(ctx, game, claim)
	} else {
		return s.trace.AttackClaim(ctx, game, claim)
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

	claimCorrect, err := s.trace.AgreeWithClaim(ctx, game, claim)
	if err != nil {
		return StepData{}, err
	}
	if claimCorrect {
		return s.trace.StepDefend(ctx, game, claim)
	} else {
		return s.trace.StepAttack(ctx, game, claim)
	}
}

// agreeWithClaimPath returns true if the every other claim in the path to root is correct according to the internal [TraceProvider].
func (s *claimSolver) agreeWithClaimPath(ctx context.Context, game types.Game, claim types.Claim) (bool, error) {
	agree, err := s.trace.AgreeWithClaim(ctx, game, claim)
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
