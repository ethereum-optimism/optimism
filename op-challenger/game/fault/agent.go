package fault

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/solver"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// Responder takes a response action & executes.
// For full op-challenger this means executing the transaction on chain.
type Responder interface {
	CallResolve(ctx context.Context) (gameTypes.GameStatus, error)
	Resolve(ctx context.Context) error
	CallResolveClaim(ctx context.Context, claimIdx uint64) error
	ResolveClaim(ctx context.Context, claimIdx uint64) error
	PerformAction(ctx context.Context, action types.Action) error
}

type ClaimLoader interface {
	GetAllClaims(ctx context.Context) ([]types.Claim, error)
}

type Agent struct {
	metrics   metrics.Metricer
	solver    *solver.GameSolver
	loader    ClaimLoader
	responder Responder
	maxDepth  int
	log       log.Logger
}

func NewAgent(m metrics.Metricer, loader ClaimLoader, maxDepth int, trace types.TraceAccessor, responder Responder, log log.Logger) *Agent {
	return &Agent{
		metrics:   m,
		solver:    solver.NewGameSolver(maxDepth, trace),
		loader:    loader,
		responder: responder,
		maxDepth:  maxDepth,
		log:       log,
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

	// Calculate the actions to take
	actions, err := a.solver.CalculateNextActions(ctx, game)
	if err != nil {
		log.Error("Failed to calculate all required moves", "err", err)
	}

	// Perform the actions
	for _, action := range actions {
		log := a.log.New("action", action.Type, "is_attack", action.IsAttack, "parent", action.ParentIdx)
		if action.Type == types.ActionTypeStep {
			log = log.New("prestate", common.Bytes2Hex(action.PreState), "proof", common.Bytes2Hex(action.ProofData))
		} else {
			log = log.New("value", action.Value)
		}

		switch action.Type {
		case types.ActionTypeMove:
			a.metrics.RecordGameMove()
		case types.ActionTypeStep:
			a.metrics.RecordGameStep()
		}
		log.Info("Performing action")
		err := a.responder.PerformAction(ctx, action)
		if err != nil {
			log.Error("Action failed", "err", err)
		}
	}
	return nil
}

// tryResolve resolves the game if it is in a winning state
// Returns true if the game is resolvable (regardless of whether it was actually resolved)
func (a *Agent) tryResolve(ctx context.Context) bool {
	if err := a.resolveClaims(ctx); err != nil {
		a.log.Error("Failed to resolve claims", "err", err)
		return false
	}
	status, err := a.responder.CallResolve(ctx)
	if err != nil || status == gameTypes.GameStatusInProgress {
		return false
	}
	a.log.Info("Resolving game")
	if err := a.responder.Resolve(ctx); err != nil {
		a.log.Error("Failed to resolve the game", "err", err)
	}
	return true
}

var errNoResolvableClaims = errors.New("no resolvable claims")

func (a *Agent) tryResolveClaims(ctx context.Context) error {
	claims, err := a.loader.GetAllClaims(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch claims: %w", err)
	}
	if len(claims) == 0 {
		return errNoResolvableClaims
	}

	var resolvableClaims []int64
	for _, claim := range claims {
		a.log.Debug("checking if claim is resolvable", "claimIdx", claim.ContractIndex)
		if err := a.responder.CallResolveClaim(ctx, uint64(claim.ContractIndex)); err == nil {
			a.log.Info("Resolving claim", "claimIdx", claim.ContractIndex)
			resolvableClaims = append(resolvableClaims, int64(claim.ContractIndex))
		}
	}
	if len(resolvableClaims) == 0 {
		return errNoResolvableClaims
	}
	a.log.Info("Resolving claims", "numClaims", len(resolvableClaims))

	var wg sync.WaitGroup
	wg.Add(len(resolvableClaims))
	for _, claimIdx := range resolvableClaims {
		claimIdx := claimIdx
		go func() {
			defer wg.Done()
			err := a.responder.ResolveClaim(ctx, uint64(claimIdx))
			if err != nil {
				a.log.Error("Failed to resolve claim", "err", err)
			}
		}()
	}
	wg.Wait()
	return nil
}

func (a *Agent) resolveClaims(ctx context.Context) error {
	for {
		err := a.tryResolveClaims(ctx)
		switch err {
		case errNoResolvableClaims:
			return nil
		case nil:
			continue
		default:
			return err
		}
	}
}

// newGameFromContracts initializes a new game state from the state in the contract
func (a *Agent) newGameFromContracts(ctx context.Context) (types.Game, error) {
	claims, err := a.loader.GetAllClaims(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch claims: %w", err)
	}
	if len(claims) == 0 {
		return nil, errors.New("no claims")
	}
	game := types.NewGameState(claims, uint64(a.maxDepth))
	return game, nil
}
