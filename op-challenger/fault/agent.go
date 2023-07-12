package fault

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/log"
)

type Agent struct {
	solver    *Solver
	trace     TraceProvider
	loader    Loader
	responder Responder
	maxDepth  int
	log       log.Logger
}

func NewAgent(loader Loader, maxDepth int, trace TraceProvider, responder Responder, log log.Logger) Agent {
	return Agent{
		solver:    NewSolver(maxDepth, trace),
		trace:     trace,
		loader:    loader,
		responder: responder,
		maxDepth:  maxDepth,
		log:       log,
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
		_ = a.move(claim, game)
	}
	// Step on all leaf claims
	for _, claim := range game.Claims() {
		_ = a.step(claim, game)
	}
	return nil
}

// newGameFromContracts initializes a new game state from the state in the contract
func (a *Agent) newGameFromContracts(ctx context.Context) (Game, error) {
	claims, err := a.loader.FetchClaims(ctx)
	if err != nil {
		return nil, err
	}
	if len(claims) == 0 {
		return nil, errors.New("no claims")
	}
	game := NewGameState(claims[0], uint64(a.maxDepth))
	if err := game.PutAll(claims[1:]); err != nil {
		return nil, err
	}
	return game, nil
}

// move determines & executes the next move given a claim pair
func (a *Agent) move(claim Claim, game Game) error {
	a.log.Info("Fetching claims")
	nextMove, err := a.solver.NextMove(claim)
	if err != nil {
		a.log.Warn("Failed to execute the next move", "err", err)
		return err
	}
	if nextMove == nil {
		a.log.Info("No next move")
		return nil
	}
	move := *nextMove
	log := a.log.New("is_defend", move.DefendsParent(), "depth", move.Depth(), "index_at_depth", move.IndexAtDepth(), "value", move.Value,
		"letter", string(move.Value[31:]), "trace_index", move.Value[30],
		"parent_letter", string(claim.Value[31:]), "parent_trace_index", claim.Value[30])
	if game.IsDuplicate(move) {
		log.Debug("Duplicate move")
		return nil
	}
	log.Info("Performing move")
	return a.responder.Respond(context.TODO(), move)
}

// step attempts to execute the step through the responder
func (a *Agent) step(claim Claim, game Game) error {
	if claim.Depth() != a.maxDepth {
		return nil
	}
	a.log.Info("Attempting step", "claim_depth", claim.Depth(), "maxDepth", a.maxDepth)

	step, err := a.solver.AttemptStep(claim)
	if err != nil {
		a.log.Info("Failed to get a step", "err", err)
		return err
	}

	a.log.Info("Performing step",
		"depth", step.LeafClaim.Depth(), "index_at_depth", step.LeafClaim.IndexAtDepth(), "value", step.LeafClaim.Value,
		"is_attack", step.IsAttack)
	callData := StepCallData{
		ClaimIndex: uint64(step.LeafClaim.ContractIndex),
		IsAttack:   step.IsAttack,
	}
	return a.responder.Step(context.TODO(), callData)
}
