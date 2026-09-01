package generic

import (
	"context"
	"errors"
	"fmt"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type PrestateValidator interface {
	Validate(ctx context.Context) error
}

type Actor interface {
	Act(ctx context.Context) error
	AdditionalStatus(ctx context.Context) ([]any, error)
}

type GenericGameLoader interface {
	GetL1Head(context.Context) (common.Hash, error)
	GetStatus(context.Context) (gameTypes.GameStatus, error)
}

type L1HeaderSource interface {
	BlockRefByHash(ctx context.Context, hash common.Hash) (eth.BlockRef, error)
}

// WithdrawalDeleter deletes the withdrawal proofs invalidated by a game resolving as a challenger
// win. It is nil unless the OptimismPortal address is configured.
type WithdrawalDeleter interface {
	// DeleteInvalidatedWithdrawals deletes the proofs against game that were proven from scanFrom up
	// to and including toBlock, reporting whether every invalidated proof has now been deleted.
	DeleteInvalidatedWithdrawals(ctx context.Context, game common.Address, scanFrom uint64, toBlock uint64) (bool, error)
}

type ActorCreator func(ctx context.Context, logger log.Logger, l1Head eth.BlockID) (Actor, error)

type GamePlayer struct {
	addr               common.Address
	actor              Actor
	loader             GenericGameLoader
	logger             log.Logger
	syncValidator      gameTypes.SyncValidator
	prestateValidators []PrestateValidator
	status             gameTypes.GameStatus
	gameL1Head         eth.BlockID
	withdrawals        WithdrawalDeleter
	// withdrawalScanFrom is the first L1 block that may hold a proof this game invalidated but that
	// has not been deleted yet.
	withdrawalScanFrom uint64
	// done reports whether the game requires no further work beyond its current status. It is false
	// while withdrawal proofs invalidated by a challenger win are still outstanding.
	done bool
}

type actNoop struct{}

func (a *actNoop) Act(_ context.Context) error                       { return nil }
func (a *actNoop) AdditionalStatus(_ context.Context) ([]any, error) { return nil, nil }

func NewGenericGamePlayer(
	ctx context.Context,
	logger log.Logger,
	addr common.Address,
	loader GenericGameLoader,
	syncValidator gameTypes.SyncValidator,
	validators []PrestateValidator,
	l1HeaderSource L1HeaderSource,
	withdrawalDeleter WithdrawalDeleter,
	createActor ActorCreator,
) (*GamePlayer, error) {
	logger = logger.New("game", addr)

	status, err := loader.GetStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game status: %w", err)
	}
	// Withdrawal proofs against a game the challenger won are invalidated and must be deleted. That
	// is done by ProgressGame, on a worker thread, so the game L1 head is loaded here to scan from.
	deleteWithdrawals := withdrawalDeleter != nil && status == gameTypes.GameStatusChallengerWon
	resolved := status != gameTypes.GameStatusInProgress
	if resolved {
		logger.Info("Game already resolved", "status", status)
		if !deleteWithdrawals {
			// Game is already complete so skip creating the trace provider, loading game inputs etc.
			return &GamePlayer{
				logger:             logger,
				loader:             loader,
				prestateValidators: validators,
				status:             status,
				done:               true,
				// Act function does nothing because the game is already complete
				actor: &actNoop{},
			}, nil
		}
	}
	l1HeadHash, err := loader.GetL1Head(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load game L1 head: %w", err)
	}
	l1Header, err := l1HeaderSource.BlockRefByHash(ctx, l1HeadHash)
	if err != nil {
		return nil, fmt.Errorf("failed to load L1 header %v: %w", l1HeadHash, err)
	}
	l1Head := l1Header.ID()

	player := &GamePlayer{
		addr:               addr,
		loader:             loader,
		logger:             logger,
		status:             status,
		gameL1Head:         l1Head,
		syncValidator:      syncValidator,
		prestateValidators: validators,
		withdrawals:        withdrawalDeleter,
		withdrawalScanFrom: l1Head.Number,
		done:               !deleteWithdrawals,
	}
	if resolved {
		// Skip creating the trace provider, loading game inputs etc. Act does nothing because a
		// resolved game is never acted on; the outstanding withdrawal deletions are done by
		// ProgressGame, which keeps done false until they complete.
		player.actor = &actNoop{}
		return player, nil
	}
	actor, err := createActor(ctx, logger, l1Head)
	if err != nil {
		return nil, fmt.Errorf("failed to create actor: %w", err)
	}
	player.actor = actor
	return player, nil
}

func (g *GamePlayer) ValidatePrestate(ctx context.Context) error {
	for _, validator := range g.prestateValidators {
		if err := validator.Validate(ctx); err != nil {
			return fmt.Errorf("failed to validate prestate: %w", err)
		}
	}
	return nil
}

func (g *GamePlayer) Status() gameTypes.GameStatus {
	return g.status
}

// Done reports whether the game requires no further work beyond its current status.
func (g *GamePlayer) Done() bool {
	return g.done
}

func (g *GamePlayer) ProgressGame(ctx context.Context, l1BlockNumber uint64) (gameTypes.GameStatus, bool) {
	if g.status != gameTypes.GameStatusInProgress {
		// Game is already complete so the only outstanding work is deleting invalidated withdrawals.
		g.logger.Trace("Skipping completed game")
		g.onResolved(ctx, l1BlockNumber)
		return g.status, g.done
	}
	if err := g.syncValidator.ValidateNodeSynced(ctx, g.gameL1Head); errors.Is(err, gameTypes.ErrNotInSync) {
		g.logger.Warn("Local node not sufficiently up to date", "err", err)
		return g.status, g.done
	} else if err != nil {
		g.logger.Error("Could not check local node was in sync", "err", err)
		return g.status, g.done
	}
	g.logger.Trace("Checking if actions are required")
	if err := g.actor.Act(ctx); err != nil {
		g.logger.Error("Error when acting on game", "err", err)
	}
	status, err := g.loader.GetStatus(ctx)
	if err != nil {
		g.logger.Error("Unable to retrieve game status", "err", err)
		return gameTypes.GameStatusInProgress, g.done
	}
	g.logGameStatus(ctx, status)
	g.status = status
	if status != gameTypes.GameStatusInProgress {
		g.onResolved(ctx, l1BlockNumber)
	}
	return status, g.done
}

// onResolved deletes any withdrawal proofs the game invalidated and, once they have all been
// deleted, releases the actor as the game will never need to be acted on again.
func (g *GamePlayer) onResolved(ctx context.Context, l1BlockNumber uint64) {
	g.done = g.deleteInvalidatedWithdrawals(ctx, l1BlockNumber)
	if g.done {
		g.actor = &actNoop{}
	}
}

func (g *GamePlayer) deleteInvalidatedWithdrawals(ctx context.Context, l1BlockNumber uint64) bool {
	if g.withdrawals == nil || g.status != gameTypes.GameStatusChallengerWon {
		return true
	}
	done, err := g.withdrawals.DeleteInvalidatedWithdrawals(ctx, g.addr, g.withdrawalScanFrom, l1BlockNumber)
	if err != nil {
		g.logger.Error("Failed to delete withdrawal proofs invalidated by the game", "err", err)
		return false
	}
	if done {
		g.withdrawalScanFrom = l1BlockNumber + 1
	}
	return done
}

func (g *GamePlayer) logGameStatus(ctx context.Context, status gameTypes.GameStatus) {
	if status == gameTypes.GameStatusInProgress {
		additionalStatus, err := g.actor.AdditionalStatus(ctx)
		if err != nil {
			g.logger.Error("Failed to get additional status info for in progress game", "err", err)
			return
		}
		additionalStatus = append(additionalStatus, "status", g.status)
		g.logger.Info("Game info", additionalStatus...)
		return
	}
	g.logger.Info("Game resolved", "status", status)
}
