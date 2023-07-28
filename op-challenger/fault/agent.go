package fault

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/solver"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum/go-ethereum/log"
)

// Responder takes a response action & executes.
// For full op-challenger this means executing the transaction on chain.
type Responder interface {
	CanResolve(ctx context.Context) bool
	Resolve(ctx context.Context) error
	Respond(ctx context.Context, response types.Claim) error
	Step(ctx context.Context, stepData types.StepCallData) error
}

type Agent struct {
	solver                  *solver.Solver
	loader                  Loader
	responder               Responder
	maxDepth                int
	agreeWithProposedOutput bool
	log                     log.Logger
}

func NewAgent(loader Loader, maxDepth int, trace types.TraceProvider, responder Responder, agreeWithProposedOutput bool, log log.Logger) *Agent {
	return &Agent{
		solver:                  solver.NewSolver(maxDepth, trace),
		loader:                  loader,
		responder:               responder,
		maxDepth:                maxDepth,
		agreeWithProposedOutput: agreeWithProposedOutput,
		log:                     log,
	}
}

// Act iterates the game & performs all of the next actions.
func (a *Agent) Act(ctx context.Context) error {
	if a.tryResolve(ctx) {
		return nil
	}
	game, err := a.newGameFromContracts(ctx)
	if err != nil {
		return fmt.Errorf("create game from contracts: %w", err)
	}
	// Create counter claims
	for _, claim := range game.Claims() {
		if err := a.move(ctx, claim, game); err != nil && !errors.Is(err, types.ErrGameDepthReached) {
			log.Error("Failed to move", "err", err)
		}
	}
	// Step on all leaf claims
	for _, claim := range game.Claims() {
		if err := a.step(ctx, claim, game); err != nil {
			log.Error("Failed to step", "err", err)
		}
	}
	return nil
}

// tryResolve resolves the game if it is in a terminal state
// and returns true if the game resolves successfully.
func (a *Agent) tryResolve(ctx context.Context) bool {
	if !a.responder.CanResolve(ctx) {
		return false
	}
	a.log.Info("Resolving game")
	if err := a.responder.Resolve(ctx); err != nil {
		a.log.Error("Failed to resolve the game", "err", err)
		return false
	}
	return true
}

// newGameFromContracts initializes a new game state from the state in the contract
func (a *Agent) newGameFromContracts(ctx context.Context) (types.Game, error) {
	claims, err := a.loader.FetchClaims(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch claims: %w", err)
	}
	if len(claims) == 0 {
		return nil, errors.New("no claims")
	}
	game := types.NewGameState(a.agreeWithProposedOutput, claims[0], uint64(a.maxDepth))
	if err := game.PutAll(claims[1:]); err != nil {
		return nil, fmt.Errorf("failed to load claims into the local state: %w", err)
	}
	return game, nil
}

// move determines & executes the next move given a claim
func (a *Agent) move(ctx context.Context, claim types.Claim, game types.Game) error {
	nextMove, err := a.solver.NextMove(ctx, claim, game.AgreeWithClaimLevel(claim))
	if err != nil {
		return fmt.Errorf("execute next move: %w", err)
	}
	if nextMove == nil {
		a.log.Debug("No next move")
		return nil
	}
	move := *nextMove
	log := a.log.New("is_defend", move.DefendsParent(), "depth", move.Depth(), "index_at_depth", move.IndexAtDepth(),
		"value", move.Value, "trace_index", move.TraceIndex(a.maxDepth),
		"parent_value", claim.Value, "parent_trace_index", claim.TraceIndex(a.maxDepth))
	if game.IsDuplicate(move) {
		log.Debug("Skipping duplicate move")
		return nil
	}
	log.Info("Performing move")
	return a.responder.Respond(ctx, move)
}

// step determines & executes the next step against a leaf claim through the responder
func (a *Agent) step(ctx context.Context, claim types.Claim, game types.Game) error {
	if claim.Depth() != a.maxDepth {
		return nil
	}

	agreeWithClaimLevel := game.AgreeWithClaimLevel(claim)
	if agreeWithClaimLevel {
		a.log.Debug("Agree with leaf claim, skipping step", "claim_depth", claim.Depth(), "maxDepth", a.maxDepth)
		return nil
	}

	if claim.Countered {
		a.log.Debug("Step already executed against claim", "depth", claim.Depth(), "index_at_depth", claim.IndexAtDepth(), "value", claim.Value)
		return nil
	}

	a.log.Info("Attempting step", "claim_depth", claim.Depth(), "maxDepth", a.maxDepth)
	step, err := a.solver.AttemptStep(ctx, claim, agreeWithClaimLevel)
	if err != nil {
		return fmt.Errorf("attempt step: %w", err)
	}

	a.log.Info("Performing step", "is_attack", step.IsAttack,
		"depth", step.LeafClaim.Depth(), "index_at_depth", step.LeafClaim.IndexAtDepth(), "value", step.LeafClaim.Value)
	callData := types.StepCallData{
		ClaimIndex: uint64(step.LeafClaim.ContractIndex),
		IsAttack:   step.IsAttack,
		StateData:  step.PreState,
		Proof:      step.ProofData,
	}
	return a.responder.Step(ctx, callData)
}
