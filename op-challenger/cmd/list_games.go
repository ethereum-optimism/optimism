package main

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	openum "github.com/ethereum-optimism/optimism/op-service/enum"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

var ColumnTypes = []string{"time", "claimCount", "l2BlockNum"}

var (
	SortByFlag = &cli.StringFlag{
		Name:    "sort-by",
		Usage:   "Sort games by column. Valid options: " + openum.EnumString(ColumnTypes),
		Value:   "time",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "SORT_BY"),
	}
	SortOrderFlag = &cli.StringFlag{
		Name:    "sort-order",
		Usage:   "Sort order for games. Valid options: 'asc' or 'desc'.",
		Value:   "asc",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "SORT_ORDER"),
	}
)

func ListGames(ctx *cli.Context) error {
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
	sortBy := ctx.String(SortByFlag.Name)
	if sortBy != "" && !slices.Contains(ColumnTypes, sortBy) {
		return fmt.Errorf("invalid sort-by value: %v", sortBy)
	}
	sortOrder := ctx.String(SortOrderFlag.Name)
	if sortOrder != "" && sortOrder != "asc" && sortOrder != "desc" {
		return fmt.Errorf("invalid sort-order value: %v", sortOrder)
	}

	gameWindow := ctx.Duration(flags.GameWindowFlag.Name)

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
	return listGames(ctx.Context, caller, contract, head.Hash(), gameWindow, sortBy, sortOrder)
}

type gameInfo struct {
	types.GameMetadata
	claimCount uint64
	l2BlockNum uint64
	rootClaim  common.Hash
	status     types.GameStatus
	err        error
}

func listGames(ctx context.Context, caller *batching.MultiCaller, factory *contracts.DisputeGameFactoryContract, block common.Hash, gameWindow time.Duration, sortBy, sortOrder string) error {
	earliestTimestamp := clock.MinCheckedTimestamp(clock.SystemClock, gameWindow)
	games, err := factory.GetGamesAtOrAfter(ctx, block, earliestTimestamp)
	if err != nil {
		return fmt.Errorf("failed to retrieve games: %w", err)
	}
	slices.Reverse(games)

	infos := make([]gameInfo, len(games))
	var wg sync.WaitGroup
	for idx, game := range games {
		gameContract, err := contracts.NewFaultDisputeGameContract(ctx, metrics.NoopContractMetrics, game.Proxy, caller)
		if err != nil {
			return fmt.Errorf("failed to create dispute game contract: %w", err)
		}
		info := gameInfo{GameMetadata: game}
		infos[idx] = info
		gameProxy := game.Proxy
		currIndex := idx
		wg.Add(1)
		go func() {
			defer wg.Done()
			metadata, err := gameContract.GetGameMetadata(ctx, rpcblock.ByHash(block))
			if err != nil {
				info.err = fmt.Errorf("failed to retrieve metadata for game %v: %w", gameProxy, err)
				return
			}
			infos[currIndex].status = metadata.Status
			infos[currIndex].l2BlockNum = metadata.L2BlockNum
			infos[currIndex].rootClaim = metadata.RootClaim
			claimCount, err := gameContract.GetClaimCount(ctx)
			if err != nil {
				info.err = fmt.Errorf("failed to retrieve claim count for game %v: %w", gameProxy, err)
				return
			}
			infos[currIndex].claimCount = claimCount
		}()
	}
	wg.Wait()
	lineFormat := "%3v %-42v %4v %-21v %14v %-66v %6v %-14v\n"
	fmt.Printf(lineFormat, "Idx", "Game", "Type", "Created (Local)", "L2 Block", "Output Root", "Claims", "Status")

	// Sort infos by the specified column
	switch sortBy {
	case "time":
		slices.SortFunc(infos, func(i, j gameInfo) int {
			if sortOrder == "desc" {
				return cmp.Compare(j.Timestamp, i.Timestamp)
			}
			return cmp.Compare(i.Timestamp, j.Timestamp)
		})
	case "claimCount":
		slices.SortFunc(infos, func(i, j gameInfo) int {
			if sortOrder == "desc" {
				return cmp.Compare(j.claimCount, i.claimCount)
			}
			return cmp.Compare(i.claimCount, j.claimCount)
		})
	case "l2BlockNum":
		slices.SortFunc(infos, func(i, j gameInfo) int {
			if sortOrder == "desc" {
				return cmp.Compare(j.l2BlockNum, i.l2BlockNum)
			}
			return cmp.Compare(i.l2BlockNum, j.l2BlockNum)
		})
	}

	for _, game := range infos {
		if game.err != nil {
			return game.err
		}
		created := time.Unix(int64(game.Timestamp), 0).Format(time.DateTime)
		fmt.Printf(lineFormat,
			game.Index, game.Proxy, game.GameType, created, game.l2BlockNum, game.rootClaim, game.claimCount, game.status)
	}
	return nil
}

func listGamesFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		SortByFlag,
		SortOrderFlag,
		flags.L1EthRpcFlag,
		flags.NetworkFlag,
		flags.FactoryAddressFlag,
		flags.GameWindowFlag,
	}
	cliFlags = append(cliFlags, oplog.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var ListGamesCommand = &cli.Command{
	Name:        "list-games",
	Usage:       "List the games created by a dispute game factory",
	Description: "Lists the games created by a dispute game factory",
	Action:      Interruptible(ListGames),
	Flags:       listGamesFlags(),
}
