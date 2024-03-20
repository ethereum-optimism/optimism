package main

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
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
	l2StartBlockNum, l2BlockNum, err := game.GetBlockRange(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve status: %w", err)
	}

	claims, err := game.GetAllClaims(ctx, rpcblock.Latest)
	if err != nil {
		return fmt.Errorf("failed to retrieve claims: %w", err)
	}

	// The top game runs from depth 0 to split depth *inclusive*.
	// The - 1 here accounts for the fact that the split depth is included in the top game.
	bottomDepth := maxDepth - splitDepth - 1

	gameState := types.NewGameState(claims, maxDepth)
	lineFormat := "%3v %-7v %6v %5v %14v %-66v %-42v %-42v\n"
	info := fmt.Sprintf(lineFormat, "Idx", "Move", "Parent", "Depth", "Index", "Value", "Claimant", "Countered By")
	for i, claim := range claims {
		pos := claim.Position
		parent := strconv.Itoa(claim.ParentContractIndex)
		if claim.IsRoot() {
			parent = ""
		}
		countered := claim.CounteredBy.Hex()
		if claim.CounteredBy == (common.Address{}) {
			countered = "-"
		}
		move := "Attack"
		if gameState.DefendsParent(claim) {
			move = "Defend"
		}
		var traceIdx *big.Int
		if claim.Depth() <= splitDepth {
			traceIdx = claim.TraceIndex(splitDepth)
		} else {
			relativePos, err := claim.Position.RelativeToAncestorAtDepth(splitDepth)
			if err != nil {
				fmt.Printf("Error calculating relative position for claim %v: %v", claim.ContractIndex, err)
				traceIdx = big.NewInt(-1)
			} else {
				traceIdx = relativePos.TraceIndex(bottomDepth)
			}
		}
		info = info + fmt.Sprintf(lineFormat,
			i, move, parent, pos.Depth(), traceIdx, claim.Value.Hex(), claim.Claimant, countered)
	}
	fmt.Printf("Status: %v • L2 Blocks: %v to %v • Split Depth: %v • Max Depth: %v • Claim Count: %v\n%v\n",
		status, l2StartBlockNum, l2BlockNum, splitDepth, maxDepth, len(claims), info)
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
