package fault

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/solver"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// Responder takes a response action & executes.
// For full op-challenger this means executing the transaction on chain.
type Responder interface {
	CallResolve(ctx context.Context) (gameTypes.GameStatus, error)
	Resolve() error
	CallResolveClaim(ctx context.Context, claimIdx uint64) error
	ResolveClaims(claimIdx ...uint64) error
	PerformAction(ctx context.Context, action types.Action) error
}

type ClaimLoader interface {
	GetAllClaims(ctx context.Context, block rpcblock.Block) ([]types.Claim, error)
	IsL2BlockNumberChallenged(ctx context.Context, block rpcblock.Block) (bool, error)
}

type Agent struct {
	metrics          metrics.Metricer
	systemClock      clock.Clock
	l1Clock          types.ClockReader
	solver           *solver.GameSolver
	loader           ClaimLoader
	responder        Responder
	selective        bool
	claimants        []common.Address
	maxDepth         types.Depth
	maxClockDuration time.Duration
	log              log.Logger
}

func NewAgent(
	m metrics.Metricer,
	systemClock clock.Clock,
	l1Clock types.ClockReader,
	loader ClaimLoader,
	maxDepth types.Depth,
	maxClockDuration time.Duration,
	trace types.TraceAccessor,
	responder Responder,
	log log.Logger,
	selective bool,
	claimants []common.Address,
) *Agent {
	return &Agent{
		metrics:          m,
		systemClock:      systemClock,
		l1Clock:          l1Clock,
		solver:           solver.NewGameSolver(maxDepth, trace),
		loader:           loader,
		responder:        responder,
		selective:        selective,
		claimants:        claimants,
		maxDepth:         maxDepth,
		maxClockDuration: maxClockDuration,
		log:              log,
	}
}

// Act iterates the game & performs all of the next actions.
func (a *Agent) Act(ctx context.Context) error {
	if a.tryResolve(ctx) {
		return nil
	}

	start := a.systemClock.Now()
	defer func() {
		a.metrics.RecordGameActTime(a.systemClock.Since(start).Seconds())
	}()

	if challenged, err := a.loader.IsL2BlockNumberChallenged(ctx, rpcblock.Latest); err != nil {
		return fmt.Errorf("failed to check if L2 block number already challenged: %w", err)
	} else if challenged {
		a.log.Debug("Skipping game with already challenged L2 block number")
		return nil
	}

	game, err := a.newGameFromContracts(ctx)
	if err != nil {
		return fmt.Errorf("create game from contracts: %w", err)
	}

	actions, err := a.solver.CalculateNextActions(ctx, game)
	if err != nil {
		a.log.Error("Failed to calculate all required moves", "err", err)
	}

	var wg sync.WaitGroup
	wg.Add(len(actions))
	for _, action := range actions {
		go a.performAction(ctx, &wg, action)
	}
	wg.Wait()
	return nil
}

func (a *Agent) performAction(ctx context.Context, wg *sync.WaitGroup, action types.Action) {
	defer wg.Done()
	actionLog := a.log.New("action", action.Type)
	if action.Type == types.ActionTypeStep {
		containsOracleData := action.OracleData != nil
		isLocal := containsOracleData && action.OracleData.IsLocal
		actionLog = actionLog.New(
			"is_attack", action.IsAttack,
			"parent", action.ParentClaim.ContractIndex,
			"prestate", common.Bytes2Hex(action.PreState),
			"proof", common.Bytes2Hex(action.ProofData),
			"containsOracleData", containsOracleData,
			"isLocalPreimage", isLocal,
		)
		if action.OracleData != nil {
			actionLog = actionLog.New("oracleKey", common.Bytes2Hex(action.OracleData.OracleKey))
		}
	} else if action.Type == types.ActionTypeMove {
		actionLog = actionLog.New("is_attack", action.IsAttack, "parent", action.ParentClaim.ContractIndex, "value", action.Value)
	}

	switch action.Type {
	case types.ActionTypeMove:
		a.metrics.RecordGameMove()
	case types.ActionTypeStep:
		a.metrics.RecordGameStep()
	case types.ActionTypeChallengeL2BlockNumber:
		a.metrics.RecordGameL2Challenge()
	}
	actionLog.Info("Performing action")
	err := a.responder.PerformAction(ctx, action)
	if err != nil {
		actionLog.Error("Action failed", "err", err)
	}
}

// tryResolve resolves the game if it is in a winning state
// Returns true if the game is resolvable (regardless of whether it was actually resolved)
func (a *Agent) tryResolve(ctx context.Context) bool {
	if err := a.resolveClaims(ctx); err != nil {
		a.log.Error("Failed to resolve claims", "err", err)
		return false
	}
	if a.selective {
		// Never resolve games in selective mode as it won't unlock any bonds for us.
		// Assume the game is still in progress or the player wouldn't have told us to act.
		return false
	}
	status, err := a.responder.CallResolve(ctx)
	if err != nil || status == gameTypes.GameStatusInProgress {
		return false
	}
	a.log.Info("Resolving game")
	if err := a.responder.Resolve(); err != nil {
		a.log.Error("Failed to resolve the game", "err", err)
	}
	return true
}

var errNoResolvableClaims = errors.New("no resolvable claims")

func (a *Agent) tryResolveClaims(ctx context.Context) error {
	claims, err := a.loader.GetAllClaims(ctx, rpcblock.Latest)
	if err != nil {
		return fmt.Errorf("failed to fetch claims: %w", err)
	}
	if len(claims) == 0 {
		return errNoResolvableClaims
	}

	var resolvableClaims []uint64
	for _, claim := range claims {
		var parent types.Claim
		if !claim.IsRootPosition() {
			parent = claims[claim.ParentContractIndex]
		}
		if types.ChessClock(a.l1Clock.Now(), claim, parent) <= a.maxClockDuration {
			continue
		}
		if a.selective {
			a.log.Trace("Selective claim resolution, checking if claim is incentivized", "claimIdx", claim.ContractIndex)
			isUncounteredClaim := slices.Contains(a.claimants, claim.Claimant) && claim.CounteredBy == common.Address{}
			ourCounter := slices.Contains(a.claimants, claim.CounteredBy)
			if !isUncounteredClaim && !ourCounter {
				a.log.Debug("Skipping claim to check resolution", "claimIdx", claim.ContractIndex)
				continue
			}
		}
		a.log.Trace("Checking if claim is resolvable", "claimIdx", claim.ContractIndex)
		if err := a.responder.CallResolveClaim(ctx, uint64(claim.ContractIndex)); err == nil {
			a.log.Info("Resolving claim", "claimIdx", claim.ContractIndex)
			resolvableClaims = append(resolvableClaims, uint64(claim.ContractIndex))
		}
	}
	if len(resolvableClaims) == 0 {
		return errNoResolvableClaims
	}
	a.log.Info("Resolving claims", "numClaims", len(resolvableClaims))

	if err := a.responder.ResolveClaims(resolvableClaims...); err != nil {
		a.log.Error("Failed to resolve claims", "err", err)
	}
	return nil
}

func (a *Agent) resolveClaims(ctx context.Context) error {
	start := a.systemClock.Now()
	defer func() {
		a.metrics.RecordClaimResolutionTime(a.systemClock.Since(start).Seconds())
	}()
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
	claims, err := a.loader.GetAllClaims(ctx, rpcblock.Latest)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch claims: %w", err)
	}
	if len(claims) == 0 {
		return nil, errors.New("no claims")
	}
	game := types.NewGameState(claims, a.maxDepth)
	return game, nil
}
