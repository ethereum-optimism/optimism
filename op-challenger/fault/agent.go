package fault

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
)

type Agent struct {
	solver                  *Solver
	trace                   TraceProvider
	loader                  Loader
	responder               Responder
	maxDepth                int
	agreeWithProposedOutput bool
	log                     log.Logger
}

func NewAgent(loader Loader, maxDepth int, trace TraceProvider, responder Responder, agreeWithProposedOutput bool, log log.Logger) Agent {
	return Agent{
		solver:                  NewSolver(maxDepth, trace),
		trace:                   trace,
		loader:                  loader,
		responder:               responder,
		maxDepth:                maxDepth,
		agreeWithProposedOutput: agreeWithProposedOutput,
		log:                     log,
	}
}

// Act iterates the game & performs all of the next actions.
func (a *Agent) Act() error {
	game, err := a.newGameFromContracts(context.Background())
	if err != nil {
		a.log.Error("Failed to create new game", "err", err)
		return err
	}
	// Create counter claims
	for _, claim := range game.Claims() {
		if err := a.move(claim, game); err != nil {
			log.Error("Failed to move", "err", err)
		}
	}
	// Step on all leaf claims
	for _, claim := range game.Claims() {
		if err := a.step(claim, game); err != nil {
			log.Error("Failed to step", "err", err)
		}
	}
	return nil
}

// newGameFromContracts initializes a new game state from the state in the contract
func (a *Agent) newGameFromContracts(ctx context.Context) (Game, error) {
	claims, err := a.loader.FetchClaims(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch claims: %w", err)
	}
	if len(claims) == 0 {
		return nil, errors.New("no claims")
	}
	game := NewGameState(a.agreeWithProposedOutput, claims[0], uint64(a.maxDepth))
	if err := game.PutAll(claims[1:]); err != nil {
		return nil, fmt.Errorf("failed to load claims into the local state: %w", err)
	}
	return game, nil
}

// move determines & executes the next move given a claim
func (a *Agent) move(claim Claim, game Game) error {
	nextMove, err := a.solver.NextMove(claim, game.AgreeWithClaimLevel(claim))
	if err != nil {
		a.log.Warn("Failed to execute the next move", "err", err)
		return err
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
		log.Debug("Duplicate move")
		return nil
	}
	log.Info("Performing move")
	return a.responder.Respond(context.TODO(), move)
}

// step determines & executes the next step against a leaf claim through the responder
func (a *Agent) step(claim Claim, game Game) error {
	if claim.Depth() != a.maxDepth {
		return nil
	}
	a.log.Info("Attempting step", "claim_depth", claim.Depth(), "maxDepth", a.maxDepth)
	step, err := a.solver.AttemptStep(claim)
	if err != nil {
		a.log.Warn("Failed to get a step", "err", err)
		return err
	}
	stateData, err := a.trace.GetPreimage(step.PreStateTraceIndex)
	if err != nil {
		a.log.Warn("Failed to get a state data", "err", err)
		return err
	}

	a.log.Info("Performing step",
		"depth", step.LeafClaim.Depth(), "index_at_depth", step.LeafClaim.IndexAtDepth(), "value", step.LeafClaim.Value,
		"is_attack", step.IsAttack, "prestate_trace_index", step.PreStateTraceIndex)
	callData := StepCallData{
		ClaimIndex: uint64(step.LeafClaim.ContractIndex),
		IsAttack:   step.IsAttack,
		StateData:  stateData,
	}
	return a.responder.Step(context.TODO(), callData)
}
