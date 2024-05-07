package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/urfave/cli/v2"
)

func CheckResolution(ctx *cli.Context) error {
	logger, err := setupLogging(ctx)
	if err != nil {
		return err
	}
	rpcUrl := ctx.String(flags.L1EthRpcFlag.Name)
	if rpcUrl == "" {
		return fmt.Errorf("missing %v", flags.L1EthRpcFlag.Name)
	}
	gameAddr, err := opservice.ParseAddress(ctx.String(GameAddressFlag.Name))
	if err != nil {
		return err
	}

	l1Client, err := dial.DialEthClientWithTimeout(ctx.Context, dial.DefaultDialTimeout, logger, rpcUrl)
	if err != nil {
		return fmt.Errorf("failed to dial L1: %w", err)
	}
	defer l1Client.Close()

	caller := batching.NewMultiCaller(l1Client.Client(), batching.DefaultBatchSize)
	contract, err := contracts.NewFaultDisputeGameContract(ctx.Context, metrics.NoopContractMetrics, gameAddr, caller)
	if err != nil {
		return err
	}
	return checkResolution(ctx.Context, contract)
}

func checkResolution(ctx context.Context, game contracts.FaultDisputeGameContract) error {
	status, err := game.GetStatus(ctx)
	if err != nil {
		return err
	}
	if status != gameTypes.GameStatusInProgress {
		fmt.Printf("Game Resolved with status: %v\n", status)
		return nil
	}
	fmt.Println("Game is still in progress")

	claims, err := game.GetAllClaims(ctx, rpcblock.Latest)
	if err != nil {
		return fmt.Errorf("failed to retrieve claims: %w", err)
	}

	resolved, err := game.IsResolved(ctx, rpcblock.Latest, claims...)
	if err != nil {
		return fmt.Errorf("failed to retrieve claim resolution: %w", err)
	}

	unresolvedClaims := 0
	resolvedClaims := 0
	for i := range claims {
		if resolved[i] {
			resolvedClaims += 1
		} else {
			unresolvedClaims += 1
		}
	}

	fmt.Printf("Resolved: %d, Unresolved: %d\n", resolvedClaims, unresolvedClaims)

	return nil
}

func resolutionFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		GameAddressFlag,
	}
	cliFlags = append(cliFlags, oplog.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var ResolutionCommand = &cli.Command{
	Name:        "resolution",
	Usage:       "Checks resolution status for the specified dispute game",
	Description: "Checks resolution status for the specified dispute game",
	Action:      CheckResolution,
	Flags:       resolutionFlags(),
}
