package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/solver"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"
)

var HonestMovesCommand = &cli.Command{
	Name:        "honest-moves",
	Usage:       "Print the honest moves that should be made against a game",
	Description: "Calculates and prints the moves that the honest actor would perform against a dispute game",
	Action:      HonestMoves,
	Flags:       flags.Flags,
}

func HonestMoves(ctx *cli.Context) error {
	logger, err := setupLogging(ctx)
	if err != nil {
		return err
	}

	cfg, err := flags.NewConfigFromCLI(ctx)
	if err != nil {
		return err
	}

	client, err := client.DialEthClientWithTimeout(client.DefaultDialTimeout, logger, cfg.L1EthRpc)
	if err != nil {
		return fmt.Errorf("failed to dial L1: %w", err)
	}

	factory, err := bindings.NewDisputeGameFactory(cfg.GameFactoryAddress, client)
	if err != nil {
		return fmt.Errorf("failed to bind the fault dispute game factory contract: %w", err)
	}
	loader := game.NewGameLoader(factory)

	logger.Info("Fetching games", "factory", cfg.GameFactoryAddress)
	games, err := loader.FetchAllGamesAtBlock(ctx.Context, minGameTimestamp(cfg.GameWindow), nil)
	if err != nil {
		return fmt.Errorf("failed to load games: %w", err)
	}

	allData := make(map[common.Address]gameData)
	var errs []error
	for _, g := range games {
		if len(cfg.GameAllowlist) > 0 && !slices.Contains(cfg.GameAllowlist, g.Proxy) {
			continue
		}

		data, err := logGameMoves(ctx, logger, g.Proxy, client, cfg)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		allData[g.Proxy] = data
	}

	for addr, data := range allData {
		logger := logger.New("game", addr)
		logClaims(logger, data.state)
		logActions(logger, data.missingActions)
		logger.Info("Game Status", "status", data.status)
	}
	return errors.Join(errs...)
}

func logGameMoves(ctx *cli.Context, logger log.Logger, addr common.Address, client *ethclient.Client, cfg *config.Config) (gameData, error) {
	logger = logger.New("game", addr)
	logger.Info("Fetching game state")
	dir := filepath.Join(cfg.Datadir, addr.Hex())
	gameLoader, err := fault.NewLoaderFromBindings(addr, client)
	if err != nil {
		return gameData{}, fmt.Errorf("failed to create claim loader for game %v: %w", addr, err)
	}
	status, err := gameLoader.GetGameStatus(ctx.Context)
	if err != nil {
		return gameData{}, fmt.Errorf("failed to load game status: %v: %w", addr, err)
	}
	depth, err := gameLoader.FetchGameDepth(ctx.Context)
	if err != nil {
		return gameData{}, fmt.Errorf("failed to load depth of game %v: %w", addr, err)
	}
	trace, err := fault.NewTraceProvider(ctx.Context, logger, metrics.NoopMetrics, cfg, dir, addr, client, depth)
	if err != nil {
		return gameData{}, fmt.Errorf("failed to create trace provider for game %v: %w", addr, err)
	}
	gameSolver := solver.NewGameSolver(int(depth), trace)
	gameState, err := gameLoader.FetchGameState(ctx.Context, cfg.AgreeWithProposedOutput, depth)
	if err != nil {
		return gameData{}, fmt.Errorf("failed to load state of game %v: %w", addr, err)
	}
	logClaims(logger, gameState)
	actions, err := gameSolver.CalculateNextActions(ctx.Context, gameState)
	if err != nil {
		return gameData{}, fmt.Errorf("failed to calculate actions for game %v: %w", addr, err)
	}
	logClaims(logger, gameState)
	logActions(logger, actions)
	return gameData{
		state:          gameState,
		missingActions: actions,
		status:         status,
	}, nil
}

func logActions(logger log.Logger, actions []types.Action) {
	for _, action := range actions {
		logger.Info("Missing honest action", "type", action.Type, "parentIdx", action.ParentIdx, "attack", action.IsAttack, "value", action.Value,
			"prestate", hex.EncodeToString(action.PreState), "proof", hex.EncodeToString(action.ProofData))
	}
	if len(actions) == 0 {
		logger.Info("All honest actions played")
	}
}

type gameData struct {
	state          types.Game
	missingActions []types.Action
	status         gameTypes.GameStatus
}

func logClaims(logger log.Logger, gameState types.Game) {
	for i, claim := range gameState.Claims() {
		logger.Info("Claim",
			"idx", i,
			"pos", claim.Position.ToGIndex(),
			"traceIdx", claim.Position.TraceIndex(int(gameState.MaxDepth())),
			"parentIdx", claim.ParentContractIndex,
			"countered", claim.Countered,
			"value", claim.Value)
	}
}

func minGameTimestamp(gameWindow time.Duration) uint64 {
	if gameWindow.Seconds() == 0 {
		return 0
	}
	// time: "To compute t-d for a duration d, use t.Add(-d)."
	// https://pkg.go.dev/time#Time.Sub
	if time.Now().Unix() > int64(gameWindow.Seconds()) {
		return uint64(time.Now().Add(-gameWindow).Unix())
	}
	return 0
}
