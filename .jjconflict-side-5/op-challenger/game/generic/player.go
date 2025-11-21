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

type ActorCreator func(ctx context.Context, logger log.Logger, l1Head eth.BlockID) (Actor, error)

type GamePlayer struct {
	actor              Actor
	loader             GenericGameLoader
	logger             log.Logger
	syncValidator      gameTypes.SyncValidator
	prestateValidators []PrestateValidator
	status             gameTypes.GameStatus
	gameL1Head         eth.BlockID
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
	createActor ActorCreator,
) (*GamePlayer, error) {
	logger = logger.New("game", addr)

	status, err := loader.GetStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game status: %w", err)
	}
	if status != gameTypes.GameStatusInProgress {
		logger.Info("Game already resolved", "status", status)
		// Game is already complete so skip creating the trace provider, loading game inputs etc.
		return &GamePlayer{
			logger:             logger,
			loader:             loader,
			prestateValidators: validators,
			status:             status,
			// Act function does nothing because the game is already complete
			actor: &actNoop{},
		}, nil
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

	actor, err := createActor(ctx, logger, l1Head)
	if err != nil {
		return nil, fmt.Errorf("failed to create actor: %w", err)
	}

	return &GamePlayer{
		actor:              actor,
		loader:             loader,
		logger:             logger,
		status:             status,
		gameL1Head:         l1Head,
		syncValidator:      syncValidator,
		prestateValidators: validators,
	}, nil
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

func (g *GamePlayer) ProgressGame(ctx context.Context) gameTypes.GameStatus {
	if g.status != gameTypes.GameStatusInProgress {
		// Game is already complete so don't try to perform further actions.
		g.logger.Trace("Skipping completed game")
		return g.status
	}
	if err := g.syncValidator.ValidateNodeSynced(ctx, g.gameL1Head); errors.Is(err, gameTypes.ErrNotInSync) {
		g.logger.Warn("Local node not sufficiently up to date", "err", err)
		return g.status
	} else if err != nil {
		g.logger.Error("Could not check local node was in sync", "err", err)
		return g.status
	}
	g.logger.Trace("Checking if actions are required")
	if err := g.actor.Act(ctx); err != nil {
		g.logger.Error("Error when acting on game", "err", err)
	}
	status, err := g.loader.GetStatus(ctx)
	if err != nil {
		g.logger.Error("Unable to retrieve game status", "err", err)
		return gameTypes.GameStatusInProgress
	}
	g.logGameStatus(ctx, status)
	g.status = status
	if status != gameTypes.GameStatusInProgress {
		// Release the agent as we will no longer need to act on this game.
		g.actor = &actNoop{}
	}
	return status
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
