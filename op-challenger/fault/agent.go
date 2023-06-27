package fault

import (
	"context"
	"sync"
)

type Agent struct {
	mu        sync.Mutex
	game      Game
	solver    *Solver
	trace     TraceProvider
	responder Responder
	maxDepth  int
}

func NewAgent(game Game, maxDepth int, trace TraceProvider, responder Responder) Agent {
	return Agent{
		game:      game,
		solver:    NewSolver(maxDepth, trace),
		trace:     trace,
		responder: responder,
		maxDepth:  maxDepth,
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
	for _, pair := range a.game.ClaimPairs() {
		_ = a.move(pair.claim, pair.parent)
	}
}

// move determines & executes the next move given a claim pair
func (a *Agent) move(claim, parent Claim) error {
	move, err := a.solver.NextMove(claim)
	if err != nil || move == nil {
		return err
	}
	// TODO(CLI-4123): Don't send duplicate responses
	// if a.game.IsDuplicate(move) {
	// 	return nil
	// }
	return a.responder.Respond(context.TODO(), *move)
}
