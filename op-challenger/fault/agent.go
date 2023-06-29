package fault

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

type Agent struct {
	mu        sync.Mutex
	game      Game
	solver    *Solver
	trace     TraceProvider
	responder Responder
	maxDepth  int
	log       log.Logger
}

func NewAgent(game Game, maxDepth int, trace TraceProvider, responder Responder, log log.Logger) Agent {
	return Agent{
		game:      game,
		solver:    NewSolver(maxDepth, trace),
		trace:     trace,
		responder: responder,
		maxDepth:  maxDepth,
		log:       log,
	}
}

// AddClaim stores a claim in the local state.
// This function shares a lock with PerformActions.
func (a *Agent) AddClaim(claim Claim) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.game.Put(claim)
}

// PerformActions iterates the game & performs all of the next actions.
// Note: PerformActions & AddClaim share a lock so the responder cannot
// call AddClaim on the same thread.
func (a *Agent) PerformActions() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, claim := range a.game.Claims() {
		_ = a.move(claim)
	}
}

// move determines & executes the next move given a claim pair
func (a *Agent) move(claim Claim) error {
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
	if a.game.IsDuplicate(move) {
		log.Debug("Duplicate move")
		return nil
	}
	log.Info("Performing move")
	return a.responder.Respond(context.TODO(), move)
}
