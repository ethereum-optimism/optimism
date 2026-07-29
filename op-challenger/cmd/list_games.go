package main

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	format := ctx.String(FormatFlag.Name)
	switch format {
	case formatText, formatJSON:
	default:
		return fmt.Errorf("invalid %v %q: must be one of %v, %v", FormatFlag.Name, format, formatText, formatJSON)
	}

	gameWindow := ctx.Duration(flags.GameWindowFlag.Name)

	l1Client, err := dial.DialEthClientWithTimeout(ctx.Context, dial.DefaultDialTimeout, logger, rpcUrl)
	if err != nil {
		return fmt.Errorf("failed to dial L1: %w", err)
	}
	defer l1Client.Close()

	caller := batching.NewMultiCaller(l1Client.Client(), batching.DefaultBatchSize)
	contract, err := contracts.NewDisputeGameFactoryContract(ctx.Context, metrics.NoopContractMetrics, factoryAddr, caller)
	if err != nil {
		return fmt.Errorf("failed to create dispute game factory contract: %w", err)
	}
	head, err := l1Client.HeaderByNumber(ctx.Context, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve current head block: %w", err)
	}
	return listGames(ctx.Context, caller, contract, head.Hash(), gameWindow, sortBy, sortOrder, format)
}

type gameInfo struct {
	types.GameMetadata
	claimCount uint64
	l2BlockNum uint64
	rootClaim  common.Hash
	status     types.GameStatus
	err        error
}

func listGames(ctx context.Context, caller *batching.MultiCaller, factory *contracts.DisputeGameFactoryContract, block common.Hash, gameWindow time.Duration, sortBy, sortOrder, format string) error {
	earliestTimestamp := clock.MinCheckedTimestamp(clock.SystemClock, gameWindow)
	games, err := factory.GetGamesAtOrAfter(ctx, block, earliestTimestamp)
	if err != nil {
		return fmt.Errorf("failed to retrieve games: %w", err)
	}
	slices.Reverse(games)

	infos := make([]gameInfo, len(games))
	var wg sync.WaitGroup
	for idx, game := range games {
		idx := idx
		gameContract, err := contracts.NewDisputeGameContractForGame(ctx, metrics.NoopContractMetrics, caller, game)
		if err != nil {
			return fmt.Errorf("failed to create dispute game contract: %w", err)
		}
		infos[idx] = gameInfo{GameMetadata: game}
		gameProxy := game.Proxy
		wg.Add(1)
		go func() {
			defer wg.Done()
			metadata, err := gameContract.GetMetadata(ctx, rpcblock.ByHash(block))
			if err != nil {
				infos[idx].err = fmt.Errorf("failed to retrieve metadata for game %v: %w", gameProxy, err)
				return
			}
			infos[idx].status = metadata.Status
			infos[idx].l2BlockNum = metadata.L2SequenceNum
			infos[idx].rootClaim = metadata.ProposedRoot
			if fdg, ok := gameContract.(contracts.FaultDisputeGameContract); ok {
				claimCount, err := fdg.GetClaimCount(ctx)
				if err != nil {
					infos[idx].err = fmt.Errorf("failed to retrieve claim count for game %v: %w", gameProxy, err)
					return
				}
				infos[idx].claimCount = claimCount
			}
		}()
	}
	wg.Wait()

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

	records := make([]gameRecord, 0, len(infos))
	for _, game := range infos {
		if game.err != nil {
			return game.err
		}
		records = append(records, gameRecord{
			Index:         game.Index,
			Game:          game.Proxy.Hex(),
			GameType:      game.GameType,
			Timestamp:     int64(game.Timestamp),
			Created:       time.Unix(int64(game.Timestamp), 0).Format(time.RFC3339),
			L2BlockNumber: game.l2BlockNum,
			OutputRoot:    game.rootClaim.Hex(),
			ClaimCount:    game.claimCount,
			Status:        game.status.String(),
		})
	}

	switch format {
	case formatJSON:
		return renderGamesJSON(os.Stdout, records)
	default:
		return renderGamesText(os.Stdout, records)
	}
}

// gameRecord is the structured, machine-readable view of a single game.
type gameRecord struct {
	Index         uint64 `json:"index"`
	Game          string `json:"game"` // proxy address
	GameType      uint32 `json:"gameType"`
	Timestamp     int64  `json:"timestamp"` // unix seconds (creation)
	Created       string `json:"created"`   // RFC3339
	L2BlockNumber uint64 `json:"l2BlockNumber"`
	OutputRoot    string `json:"outputRoot"`
	ClaimCount    uint64 `json:"claimCount"`
	Status        string `json:"status"`
}

func renderGamesText(out io.Writer, games []gameRecord) error {
	lineFormat := "%3v %-42v %4v %-21v %14v %-66v %6v %-14v\n"
	if _, err := fmt.Fprintf(out, lineFormat, "Idx", "Game", "Type", "Created (Local)", "L2 Block", "Output Root", "Claims", "Status"); err != nil {
		return err
	}
	for _, g := range games {
		if _, err := fmt.Fprintf(out, lineFormat,
			g.Index, g.Game, g.GameType, time.Unix(g.Timestamp, 0).Format(time.DateTime),
			g.L2BlockNumber, g.OutputRoot, g.ClaimCount, g.Status); err != nil {
			return err
		}
	}
	return nil
}

func renderGamesJSON(out io.Writer, games []gameRecord) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{"games": games})
}

func listGamesFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		SortByFlag,
		SortOrderFlag,
		FormatFlag,
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
