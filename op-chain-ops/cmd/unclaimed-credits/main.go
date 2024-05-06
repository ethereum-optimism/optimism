package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

var (
	factoryAddressFlag = &cli.StringFlag{
		Name:     "game-factory-address",
		Usage:    "Address of the fault game factory contract.",
		Required: true,
	}
	l1EthRpcFlag = &cli.StringFlag{
		Name:     "l1-eth-rpc",
		Usage:    "HTTP provider URL for L1.",
		Required: true,
	}
	gameAddressFlag = &cli.StringFlag{
		Name:  "game-address",
		Usage: "Address of the FaultDisputeGame proxy contract to limit the search to.",
	}
	gameWindowFlag = &cli.DurationFlag{
		Name:  "game-window",
		Usage: "The time window to limit the search of games containing unclaimed credit.",
	}
)

func unclaimedCreditsApp(ctx *cli.Context) error {
	logger := oplog.NewLogger(os.Stderr, oplog.DefaultCLIConfig())
	oplog.SetGlobalLogHandler(logger.Handler())

	rpcUrl := ctx.String(l1EthRpcFlag.Name)
	if rpcUrl == "" {
		return fmt.Errorf("missing %v", l1EthRpcFlag.Name)
	}
	factoryAddr, err := opservice.ParseAddress(ctx.String(factoryAddressFlag.Name))
	if err != nil {
		return err
	}
	gameWindow := ctx.Duration(gameWindowFlag.Name)
	var gameAddr common.Address
	if ctx.String(gameAddressFlag.Name) != "" {
		gameAddr, err = opservice.ParseAddress(ctx.String(gameAddressFlag.Name))
		if err != nil {
			return err
		}
	}

	l1Client, err := dial.DialEthClientWithTimeout(ctx.Context, dial.DefaultDialTimeout, logger, rpcUrl)
	if err != nil {
		return fmt.Errorf("failed to dial L1: %w", err)
	}
	defer l1Client.Close()

	caller := batching.NewMultiCaller(l1Client.Client(), batching.DefaultBatchSize)
	contract := contracts.NewDisputeGameFactoryContract(metrics.NoopContractMetrics, factoryAddr, caller)
	head, err := l1Client.HeaderByNumber(ctx.Context, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve current head block: %w", err)
	}
	return unclaimedCredits(ctx.Context, caller, contract, head.Hash(), gameWindow, gameAddr)
}

func unclaimedCredits(ctx context.Context, caller *batching.MultiCaller, factory *contracts.DisputeGameFactoryContract, block common.Hash, gameWindow time.Duration, gameFilter common.Address) error {
	earliestTimestamp := clock.MinCheckedTimestamp(clock.SystemClock, gameWindow)
	games, err := factory.GetGamesAtOrAfter(ctx, block, earliestTimestamp)
	if err != nil {
		return fmt.Errorf("failed to retrieve games: %w", err)
	}

	unclaimedCredits := make(map[common.Address]*big.Int)
	for _, game := range games {
		if (gameFilter != common.Address{}) && game.Proxy != gameFilter {
			continue
		}
		gameContract, err := contracts.NewFaultDisputeGameContract(ctx, metrics.NoopContractMetrics, game.Proxy, caller)
		if err != nil {
			return fmt.Errorf("failed to create game contract: %w", err)
		}
		if status, err := gameContract.GetStatus(ctx); err != nil {
			return err
		} else if status == types.GameStatusInProgress {
			continue
		}
		err = unclaimedCreditsForGame(ctx, gameContract, block, unclaimedCredits)
		if err != nil {
			return fmt.Errorf("failed to retrieve unclaimed credits for game: %w", err)
		}
		if game.Proxy == gameFilter {
			break
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(unclaimedCredits)
}

func unclaimedCreditsForGame(ctx context.Context, game contracts.FaultDisputeGameContract, block common.Hash, unclaimedCredits map[common.Address]*big.Int) error {
	claims, err := game.GetAllClaims(ctx, rpcblock.ByHash(block))
	if err != nil {
		return fmt.Errorf("failed to retrieve claims: %w", err)
	}
	players := make(map[common.Address]bool)
	for _, claim := range claims {
		players[claim.Claimant] = true
		if claim.CounteredBy != (common.Address{}) {
			players[claim.CounteredBy] = true
		}
	}
	playerList := maps.Keys(players)
	credits, err := game.GetCredits(ctx, rpcblock.Latest, playerList...)
	if err != nil {
		return fmt.Errorf("failed to retrieve credits: %w", err)
	}
	for i, credit := range credits {
		player := playerList[i]
		total := unclaimedCredits[player]
		if total == nil {
			total = new(big.Int)
			unclaimedCredits[player] = total
		}
		total.Add(total, credit)
	}
	return nil
}

func main() {
	app := &cli.App{
		Name:        "unclaimed-credits",
		Description: "Outputs a JSON containing the unclaimed credits of each player of Fault Proofs. Only resolved games are considered.",
		Flags: []cli.Flag{
			l1EthRpcFlag,
			factoryAddressFlag,
			gameWindowFlag,
			gameAddressFlag,
		},
		Action: unclaimedCreditsApp,
	}
	if err := app.Run(os.Args); err != nil {
		log.Crit("error unclaimed-credits", "err", err)
	}
}
