package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
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
	var games []game.FaultDisputeGame
	if len(cfg.GameAllowlist) == 0 {
		games, err = loader.FetchAllGamesAtBlock(ctx.Context, minGameTimestamp(cfg.GameWindow), nil)
		if err != nil {
			return fmt.Errorf("failed to load games: %w", err)
		}
	} else {
		for _, address := range cfg.GameAllowlist {
			games = append(games, game.FaultDisputeGame{Proxy: address})
		}
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
		logActions(logger, data.state, data.missingActions)
		logger.Info("Game Status", "status", data.status)
	}
	return errors.Join(errs...)
}

func logGameMoves(ctx *cli.Context, logger log.Logger, addr common.Address, client *ethclient.Client, cfg *config.Config) (gameData, error) {
	logger = logger.New("game", addr)
	logger.Info("Fetching game state")
	dir := filepath.Join(cfg.Datadir, fmt.Sprintf("game-%v", addr.Hex()))
	if err := os.MkdirAll(dir, 0755); err != nil && !errors.Is(err, os.ErrExist) {
		return gameData{}, err
	}
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
	writeClaimsGraph(logger, gameState, dir)
	actions, err := gameSolver.CalculateNextActions(ctx.Context, gameState)
	if err != nil {
		return gameData{}, fmt.Errorf("failed to calculate actions for game %v: %w", addr, err)
	}
	return gameData{
		state:          gameState,
		missingActions: actions,
		status:         status,
	}, nil
}

func logActions(logger log.Logger, gameState types.Game, actions []types.Action) {
	for _, action := range actions {
		logger := logger.New("type", action.Type, "parentIdx", action.ParentIdx, "attack", action.IsAttack)
		logger.Info("Missing honest action",
			"value", action.Value, "prestate", hex.EncodeToString(action.PreState), "proof", hex.EncodeToString(action.ProofData))
		if err := solver.CheckRules(gameState, action); err != nil {
			logger.Error("Invalid action proposed", "err", err)
			continue
		}
		if action.Type == types.ActionTypeStep {
			logStepInfo(logger, gameState, action)
		}
	}
	if len(actions) == 0 {
		logger.Info("All honest actions played")
	}
}

func logStepInfo(logger log.Logger, state types.Game, action types.Action) {
	witness := mipsevm.StateWitness(action.PreState)
	hash, err := witness.StateHash()
	if err != nil {
		logger.Error("Failed to calculate pre-state hash", "err", err)
		return
	}
	logger.Info("Actual pre-state hash", "hash", hash)
	ancestorClaim := state.Claims()[action.ParentIdx]
	var postStateClaim types.Claim
	var preStateClaim types.Claim
	depth := int(state.MaxDepth())
	parentTraceIdx := ancestorClaim.Position.TraceIndex(depth)
	var traceIdxToFind uint64
	if action.IsAttack {
		postStateClaim = ancestorClaim
		traceIdxToFind = parentTraceIdx - 1
	} else {
		traceIdxToFind = parentTraceIdx + 1
		preStateClaim = ancestorClaim
	}
	for ancestorClaim.Position.TraceIndex(depth) != traceIdxToFind {
		if ancestorClaim.IsRoot() {
			logger.Error("Failed to find ancestor claim for pre/post state", "requiredTraceIdx", traceIdxToFind)
			return
		}
		ancestorClaim = state.Claims()[ancestorClaim.ParentContractIndex]
	}
	if action.IsAttack {
		preStateClaim = ancestorClaim
	} else {
		postStateClaim = ancestorClaim
	}
	logger.Info("Required pre-state hash",
		"hash", preStateClaim.Value,
		"sourceClaimIdx", preStateClaim.ContractIndex,
		"traceIdx", preStateClaim.Position.TraceIndex(depth))
	logger.Info("Post-state hash",
		"hash", postStateClaim.Value,
		"sourceClaimIdx", postStateClaim.ContractIndex,
		"traceIdx", postStateClaim.Position.TraceIndex(depth))
}

type gameData struct {
	state          types.Game
	missingActions []types.Action
	status         gameTypes.GameStatus
}

func logClaims(logger log.Logger, gameState types.Game) {
	claims := gameState.Claims()
	slices.SortFunc(claims, func(a, b types.Claim) bool {
		return a.ContractIndex < b.ContractIndex
	})
	for _, claim := range claims {
		logger.Info("Claim",
			"idx", claim.ContractIndex,
			"parentIdx", claim.ParentContractIndex,
			"pos", claim.Position.ToGIndex(),
			"depth", claim.Position.Depth(),
			"traceIdx", claim.Position.TraceIndex(int(gameState.MaxDepth())),
			"countered", claim.Countered,
			"value", claim.Value)
	}
}

func writeClaimsGraph(logger log.Logger, gameState types.Game, dir string) {
	claims := gameState.Claims()
	depth := int(gameState.MaxDepth())
	// Order by trace index so branches are ordered based on trace index
	slices.SortFunc(claims, func(a, b types.Claim) bool {
		return a.Position.TraceIndex(depth) < b.Position.TraceIndex(depth)
	})
	graph := "digraph G {\n"
	graph += "ordering=\"out\"\n"
	for _, claim := range claims {
		if !claim.IsRoot() {
			label := "Attack"
			if claim.DefendsParent() {
				label = "Defend"
			}
			graph = graph + fmt.Sprintf("%v->%v[label=\"%v\"]\n", claim.ParentContractIndex, claim.ContractIndex, label)
		}
		var color string
		if claim.Position.Depth()%2 == 0 { // Supporting root claim
			color = "red"
		} else {
			color = "green"
		}
		label := fmt.Sprintf("Claim: %v\\n%v\\nTrace: %v", claim.ContractIndex, claim.Value.TerminalString(), claim.Position.TraceIndex(depth))
		var style string
		if !claim.Countered {
			style = "filled"
		}
		graph = graph + fmt.Sprintf("%v[color=\"%v\" label=\"%v\" style=\"%v\"]\n", claim.ContractIndex, color, label, style)
	}
	graph = graph + "}"

	// Doesn't use a logger because we need this to be easy to copy/paste
	path := filepath.Join(dir, "graph.dot")
	if err := os.WriteFile(path, []byte(graph), 0644); err != nil {
		logger.Error("Failed to write graph data", "err", err)
		return
	}
	logger.Info("Graphviz visualisation", "path", path)
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
