package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	txMetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/urfave/cli/v2"
)

// minChallengeTimeRemaining skips games whose challenge window closes too soon for a challenge to land.
const minChallengeTimeRemaining = 10 * time.Minute

var (
	GameCreatorsFlag = &cli.StringSliceFlag{
		Name:    "game-creators",
		Usage:   "Only challenge games created by these addresses.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "GAME_CREATORS"),
	}
	DryRunFlag = &cli.BoolFlag{
		Name:    "dry-run",
		Usage:   "Log the game that would be challenged without sending the challenge.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "DRY_RUN"),
	}
)

// ChallengeLatestZKGame challenges the newest unchallenged ZK game from one of the given creators.
func ChallengeLatestZKGame(ctx *cli.Context) error {
	logger, err := setupLogging(ctx)
	if err != nil {
		return err
	}
	rpcUrl := ctx.String(flags.L1EthRpcFlag.Name)
	if rpcUrl == "" {
		return fmt.Errorf("missing %v", flags.L1EthRpcFlag.Name)
	}
	factoryAddr, err := flags.FactoryAddress(ctx)
	if err != nil {
		return err
	}
	var creators []common.Address
	for _, value := range ctx.StringSlice(GameCreatorsFlag.Name) {
		creator, err := opservice.ParseAddress(value)
		if err != nil {
			return fmt.Errorf("invalid %v: %w", GameCreatorsFlag.Name, err)
		}
		creators = append(creators, creator)
	}
	if len(creators) == 0 {
		return fmt.Errorf("missing %v", GameCreatorsFlag.Name)
	}

	txMgr, err := txmgr.NewSimpleTxManager("challenger", logger, &txMetrics.NoopTxMetrics{}, txmgr.ReadCLIConfig(ctx))
	if err != nil {
		return fmt.Errorf("failed to create the transaction manager: %w", err)
	}
	defer txMgr.Close()

	l1Client, err := dial.DialEthClientWithTimeout(ctx.Context, dial.DefaultDialTimeout, logger, rpcUrl)
	if err != nil {
		return fmt.Errorf("failed to dial L1: %w", err)
	}
	defer l1Client.Close()
	caller := batching.NewMultiCaller(l1Client.Client(), batching.DefaultBatchSize)
	factory, err := contracts.NewDisputeGameFactoryContract(ctx.Context, contractMetrics.NoopContractMetrics, factoryAddr, caller)
	if err != nil {
		return fmt.Errorf("failed to create dispute game factory contract: %w", err)
	}
	head, err := l1Client.HeaderByNumber(ctx.Context, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve current head block: %w", err)
	}
	earliestTimestamp := clock.MinCheckedTimestamp(clock.SystemClock, ctx.Duration(flags.GameWindowFlag.Name))
	games, err := factory.GetGamesAtOrAfter(ctx.Context, head.Hash(), earliestTimestamp)
	if err != nil {
		return fmt.Errorf("failed to retrieve games: %w", err)
	}
	target, game, err := selectZKGameToChallenge(ctx.Context, logger, caller, games, creators, time.Unix(int64(head.Time), 0))
	if err != nil {
		return err
	}
	if ctx.Bool(DryRunFlag.Name) {
		logger.Info("Dry run: not sending the challenge")
		return nil
	}

	tx, err := game.ChallengeTx(ctx.Context)
	if err != nil {
		return fmt.Errorf("failed to create challenge tx: %w", err)
	}
	rct, err := txMgr.Send(ctx.Context, tx)
	if err != nil {
		return fmt.Errorf("failed to send challenge tx: %w", err)
	}
	if rct.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("challenge tx %v reverted", rct.TxHash)
	}
	logger.Info("Challenged ZK game", "index", target.Index, "game", target.Proxy, "tx", rct.TxHash)
	return nil
}

// selectZKGameToChallenge returns the newest ZK game created by one of creators that is unchallenged and can still be
// challenged for at least minChallengeTimeRemaining after now. games must be ordered newest first.
func selectZKGameToChallenge(
	ctx context.Context,
	logger log.Logger,
	caller *batching.MultiCaller,
	games []gameTypes.GameMetadata,
	creators []common.Address,
	now time.Time,
) (gameTypes.GameMetadata, contracts.ZKDisputeGameContract, error) {
	for _, metadata := range games {
		if metadata.GameType != uint32(gameTypes.ZKDisputeGameType) {
			continue
		}
		game, err := contracts.NewZKDisputeGameContract(contractMetrics.NoopContractMetrics, metadata.Proxy, caller)
		if err != nil {
			return gameTypes.GameMetadata{}, nil, fmt.Errorf("failed to create bindings for game %v: %w", metadata.Proxy, err)
		}
		bonds, err := game.GetBondMetadata(ctx, rpcblock.Latest)
		if err != nil {
			return gameTypes.GameMetadata{}, nil, fmt.Errorf("game %v: %w", metadata.Proxy, err)
		}
		if !slices.Contains(creators, bonds.GameCreator) {
			continue
		}
		proposal, err := game.GetChallengerMetadata(ctx, rpcblock.Latest)
		if err != nil {
			return gameTypes.GameMetadata{}, nil, fmt.Errorf("game %v: %w", metadata.Proxy, err)
		}
		if proposal.ProposalStatus != contracts.ProposalStatusUnchallenged || proposal.Deadline.Sub(now) < minChallengeTimeRemaining {
			logger.Debug("Skipping ZK game", "index", metadata.Index, "game", metadata.Proxy,
				"proposalStatus", proposal.ProposalStatus, "deadline", proposal.Deadline)
			continue
		}
		logger.Info("Selected ZK game to challenge", "index", metadata.Index, "game", metadata.Proxy,
			"creator", bonds.GameCreator, "deadline", proposal.Deadline)
		return metadata, game, nil
	}
	return gameTypes.GameMetadata{}, nil, errors.New("no challengeable ZK game from the given creators")
}

func challengeLatestZKGameFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		flags.NetworkFlag,
		flags.FactoryAddressFlag,
		flags.GameWindowFlag,
		GameCreatorsFlag,
		DryRunFlag,
	}
	cliFlags = append(cliFlags, txmgr.CLIFlagsWithDefaults(flags.EnvVarPrefix, txmgr.DefaultChallengerFlagValues)...)
	cliFlags = append(cliFlags, logcli.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var ChallengeLatestZKGameCommand = &cli.Command{
	Name:  "challenge-latest-zk-game",
	Usage: "Challenges the newest unchallenged ZK game from the given creators",
	Description: "Challenges the newest unchallenged ZK game created by one of --game-creators, skipping games whose " +
		"challenge deadline is less than 10 minutes away. Run on a schedule against an honest op-zk-proposer's games, " +
		"it tests that the proposer defends them: the proposer must prove the game before the prove deadline, or the " +
		"game resolves Challenger Won.",
	Action: Interruptible(ChallengeLatestZKGame),
	Flags:  challengeLatestZKGameFlags(),
}
