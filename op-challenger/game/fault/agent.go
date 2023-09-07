package fault

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/solver"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum/go-ethereum/log"
)

// Responder takes a response action & executes.
// For full op-challenger this means executing the transaction on chain.
type Responder interface {
	CallResolve(ctx context.Context) (gameTypes.GameStatus, error)
	Resolve(ctx context.Context) error
	Respond(ctx context.Context, response types.Claim) error
	Step(ctx context.Context, stepData types.StepCallData) error
}

type ClaimLoader interface {
	FetchClaims(ctx context.Context) ([]types.Claim, error)
}

type Agent struct {
	metrics                 metrics.Metricer
	solver                  *solver.Solver
	loader                  ClaimLoader
	responder               Responder
	updater                 types.OracleUpdater
	maxDepth                int
	agreeWithProposedOutput bool
	log                     log.Logger
}

func NewAgent(m metrics.Metricer, loader ClaimLoader, maxDepth int, trace types.TraceProvider, responder Responder, updater types.OracleUpdater, agreeWithProposedOutput bool, log log.Logger) *Agent {
	return &Agent{
		metrics:                 m,
		solver:                  solver.NewSolver(maxDepth, trace),
		loader:                  loader,
		responder:               responder,
		updater:                 updater,
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

// shouldResolve returns true if the agent should resolve the game.
// This method will return false if the game is still in progress.
func (a *Agent) shouldResolve(status gameTypes.GameStatus) bool {
	expected := gameTypes.GameStatusDefenderWon
	if a.agreeWithProposedOutput {
		expected = gameTypes.GameStatusChallengerWon
	}
	if expected != status {
		a.log.Warn("Game will be lost", "expected", expected, "actual", status)
	}
	return expected == status
}

// tryResolve resolves the game if it is in a winning state
// Returns true if the game is resolvable (regardless of whether it was actually resolved)
func (a *Agent) tryResolve(ctx context.Context) bool {
	status, err := a.responder.CallResolve(ctx)
	if err != nil || status == gameTypes.GameStatusInProgress {
		return false
	}
	if !a.shouldResolve(status) {
		return true
	}
	a.log.Info("Resolving game")
	if err := a.responder.Resolve(ctx); err != nil {
		a.log.Error("Failed to resolve the game", "err", err)
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
	a.metrics.RecordGameMove()
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

	if step.OracleData != nil {
		a.log.Info("Updating oracle data", "oracleKey", step.OracleData.OracleKey, "oracleData", step.OracleData.OracleData)
		if err := a.updater.UpdateOracle(ctx, step.OracleData); err != nil {
			return fmt.Errorf("failed to load oracle data: %w", err)
		}
	}

	a.log.Info("Performing step", "is_attack", step.IsAttack,
		"depth", step.LeafClaim.Depth(), "index_at_depth", step.LeafClaim.IndexAtDepth(), "value", step.LeafClaim.Value)
	a.metrics.RecordGameStep()
	callData := types.StepCallData{
		ClaimIndex: uint64(step.LeafClaim.ContractIndex),
		IsAttack:   step.IsAttack,
		StateData:  step.PreState,
		Proof:      step.ProofData,
	}
	return a.responder.Step(ctx, callData)
}
