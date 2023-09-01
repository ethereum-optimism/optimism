package fault

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/responder"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

type actor func(ctx context.Context) error

type GameInfo interface {
	GetGameStatus(context.Context) (types.GameStatus, error)
	GetClaimCount(context.Context) (uint64, error)
}

type GamePlayer struct {
	act                     actor
	agreeWithProposedOutput bool
	loader                  GameInfo
	logger                  log.Logger

	completed bool
}

func NewGamePlayer(
	ctx context.Context,
	logger log.Logger,
	cfg *config.Config,
	dir string,
	addr common.Address,
	txMgr txmgr.TxManager,
	client bind.ContractCaller,
) (*GamePlayer, error) {
	logger = logger.New("game", addr)
	contract, err := bindings.NewFaultDisputeGameCaller(addr, client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind the fault dispute game contract: %w", err)
	}

	loader := NewLoader(contract)

	status, err := loader.GetGameStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game status: %w", err)
	}
	if status != types.GameStatusInProgress {
		logger.Info("Game already resolved", "status", status)
		// Game is already complete so skip creating the trace provider, loading game inputs etc.
		return &GamePlayer{
			logger:                  logger,
			loader:                  loader,
			agreeWithProposedOutput: cfg.AgreeWithProposedOutput,
			completed:               true,
			// Act function does nothing because the game is already complete
			act: func(ctx context.Context) error {
				return nil
			},
		}, nil
	}

	gameDepth, err := loader.FetchGameDepth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the game depth: %w", err)
	}

	var provider types.TraceProvider
	var updater types.OracleUpdater
	switch cfg.TraceType {
	case config.TraceTypeCannon:
		cannonProvider, err := cannon.NewTraceProvider(ctx, logger, cfg, client, dir, addr)
		if err != nil {
			return nil, fmt.Errorf("create cannon trace provider: %w", err)
		}
		provider = cannonProvider
		updater, err = cannon.NewOracleUpdater(ctx, logger, txMgr, addr, client)
		if err != nil {
			return nil, fmt.Errorf("failed to create the cannon updater: %w", err)
		}
	case config.TraceTypeAlphabet:
		provider = alphabet.NewTraceProvider(cfg.AlphabetTrace, gameDepth)
		updater = alphabet.NewOracleUpdater(logger)
	default:
		return nil, fmt.Errorf("unsupported trace type: %v", cfg.TraceType)
	}

	if err := ValidateAbsolutePrestate(ctx, provider, loader); err != nil {
		return nil, fmt.Errorf("failed to validate absolute prestate: %w", err)
	}

	responder, err := responder.NewFaultResponder(logger, txMgr, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create the responder: %w", err)
	}

	return &GamePlayer{
		act:                     NewAgent(loader, int(gameDepth), provider, responder, updater, cfg.AgreeWithProposedOutput, logger).Act,
		agreeWithProposedOutput: cfg.AgreeWithProposedOutput,
		loader:                  loader,
		logger:                  logger,
		completed:               status != types.GameStatusInProgress,
	}, nil
}

func (g *GamePlayer) ProgressGame(ctx context.Context) bool {
	if g.completed {
		// Game is already complete so don't try to perform further actions.
		g.logger.Trace("Skipping completed game")
		return true
	}
	g.logger.Trace("Checking if actions are required")
	if err := g.act(ctx); err != nil {
		g.logger.Error("Error when acting on game", "err", err)
	}
	if status, err := g.loader.GetGameStatus(ctx); err != nil {
		g.logger.Warn("Unable to retrieve game status", "err", err)
	} else {
		g.logGameStatus(ctx, status)
		g.completed = status != types.GameStatusInProgress
		return g.completed
	}
	return false
}

func (g *GamePlayer) logGameStatus(ctx context.Context, status types.GameStatus) {
	if status == types.GameStatusInProgress {
		claimCount, err := g.loader.GetClaimCount(ctx)
		if err != nil {
			g.logger.Error("Failed to get claim count for in progress game", "err", err)
			return
		}
		g.logger.Info("Game info", "claims", claimCount, "status", status)
		return
	}
	var expectedStatus types.GameStatus
	if g.agreeWithProposedOutput {
		expectedStatus = types.GameStatusChallengerWon
	} else {
		expectedStatus = types.GameStatusDefenderWon
	}
	if expectedStatus == status {
		g.logger.Info("Game won", "status", status)
	} else {
		g.logger.Error("Game lost", "status", status)
	}
}

type PrestateLoader interface {
	FetchAbsolutePrestateHash(ctx context.Context) ([]byte, error)
}

// ValidateAbsolutePrestate validates the absolute prestate of the fault game.
func ValidateAbsolutePrestate(ctx context.Context, trace types.TraceProvider, loader PrestateLoader) error {
	providerPrestate, err := trace.AbsolutePreState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the trace provider's absolute prestate: %w", err)
	}
	providerPrestateHash := crypto.Keccak256(providerPrestate)
	onchainPrestate, err := loader.FetchAbsolutePrestateHash(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the onchain absolute prestate: %w", err)
	}
	if !bytes.Equal(providerPrestateHash, onchainPrestate) {
		return fmt.Errorf("trace provider's absolute prestate does not match onchain absolute prestate")
	}
	return nil
}
