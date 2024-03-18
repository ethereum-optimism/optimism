package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/urfave/cli/v2"
)

var (
	GameAddressFlag = &cli.StringFlag{
		Name:    "game-address",
		Usage:   "Address of the fault game contract.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "GAME_ADDRESS"),
	}
)

func ListClaims(ctx *cli.Context) error {
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
	contract, err := contracts.NewFaultDisputeGameContract(gameAddr, caller)
	if err != nil {
		return fmt.Errorf("failed to create dispute game bindings: %w", err)
	}
	return listClaims(ctx.Context, contract)
}

func listClaims(ctx context.Context, game *contracts.FaultDisputeGameContract) error {
	maxDepth, err := game.GetMaxGameDepth(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve max depth: %w", err)
	}
	splitDepth, err := game.GetSplitDepth(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve split depth: %w", err)
	}
	status, err := game.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve status: %w", err)
	}
	_, l2BlockNum, err := game.GetBlockRange(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve status: %w", err)
	}

	claims, err := game.GetAllClaims(ctx, rpcblock.Latest)
	if err != nil {
		return fmt.Errorf("failed to retrieve claims: %w", err)
	}

	info := fmt.Sprintf("Claim count: %v\n", len(claims))
	for i, claim := range claims {
		pos := claim.Position
		info = info + fmt.Sprintf("%v - Position: %v, Depth: %v, IndexAtDepth: %v Trace Index: %v, Value: %v, Countered: %v, ParentIndex: %v\n",
			i, pos.ToGIndex(), pos.Depth(), pos.IndexAtDepth(), pos.TraceIndex(maxDepth), claim.Value.Hex(), claim.CounteredBy, claim.ParentContractIndex)
	}
	fmt.Printf("Status: %v - L2 Block: %v - Split Depth: %v - Max Depth: %v:\n%v\n",
		status, l2BlockNum, splitDepth, maxDepth, info)
	return nil
}

func listClaimsFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		GameAddressFlag,
	}
	cliFlags = append(cliFlags, oplog.CLIFlags("OP_CHALLENGER")...)
	return cliFlags
}

var ListClaimsCommand = &cli.Command{
	Name:        "list-claims",
	Usage:       "List the claims in a dispute game",
	Description: "Lists the claims in a dispute game",
	Action:      ListClaims,
	Flags:       listClaimsFlags(),
	Hidden:      true,
}
