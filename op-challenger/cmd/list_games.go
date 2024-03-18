package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
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
	factoryAddr, err := opservice.ParseAddress(ctx.String(flags.FactoryAddressFlag.Name))
	if err != nil {
		return err
	}

	l1Client, err := dial.DialEthClientWithTimeout(ctx.Context, dial.DefaultDialTimeout, logger, rpcUrl)
	if err != nil {
		return fmt.Errorf("failed to dial L1: %w", err)
	}
	defer l1Client.Close()

	caller := batching.NewMultiCaller(l1Client.Client(), batching.DefaultBatchSize)
	contract, err := contracts.NewDisputeGameFactoryContract(factoryAddr, caller)
	if err != nil {
		return fmt.Errorf("failed to create dispute game bindings: %w", err)
	}
	head, err := l1Client.HeaderByNumber(ctx.Context, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve current head block: %w", err)
	}
	return listGames(ctx.Context, caller, contract, head.Hash())
}

type gameInfo struct {
	types.GameMetadata
	claimCount uint64
	status     types.GameStatus
	err        error
}

func listGames(ctx context.Context, caller *batching.MultiCaller, factory *contracts.DisputeGameFactoryContract, block common.Hash) error {
	games, err := factory.GetAllGames(ctx, block)
	if err != nil {
		return fmt.Errorf("failed to retrieve games: %w", err)
	}

	infos := make([]*gameInfo, len(games))
	var wg sync.WaitGroup
	for idx, game := range games {
		gameContract, err := contracts.NewFaultDisputeGameContract(game.Proxy, caller)
		if err != nil {
			return fmt.Errorf("failed to bind game contract at %v: %w", game.Proxy, err)
		}
		info := gameInfo{GameMetadata: game}
		infos[idx] = &info
		gameProxy := game.Proxy
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimCount, err := gameContract.GetClaimCount(ctx)
			if err != nil {
				info.err = fmt.Errorf("failed to retrieve claim count for game %v: %w", gameProxy, err)
				return
			}
			info.claimCount = claimCount
			status, err := gameContract.GetStatus(ctx)
			if err != nil {
				info.err = fmt.Errorf("failed to retrieve status for game %v: %w", gameProxy, err)
				return
			}
			info.status = status
		}()
	}
	wg.Wait()
	for idx, game := range infos {
		if game.err != nil {
			return err
		}
		fmt.Printf("%v Game: %v Type: %v Created: %v Claims: %v Status: %v\n",
			idx, game.Proxy, game.GameType, time.Unix(int64(game.Timestamp), 0), game.claimCount, game.status)
	}
	return nil
}

func listGamesFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		flags.FactoryAddressFlag,
	}
	cliFlags = append(cliFlags, oplog.CLIFlags("OP_CHALLENGER")...)
	return cliFlags
}

var ListGamesCommand = &cli.Command{
	Name:        "list-games",
	Usage:       "List the games created by a dispute game factory",
	Description: "Lists the games created by a dispute game factory",
	Action:      ListGames,
	Flags:       listGamesFlags(),
	Hidden:      true,
}
